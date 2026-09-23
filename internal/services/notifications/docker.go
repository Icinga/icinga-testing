package notifications

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/icinga/icinga-testing/services"
	"github.com/icinga/icinga-testing/utils"
	"go.uber.org/zap"
	"sync"
	"sync/atomic"
)

type dockerCreator struct {
	logger              *zap.Logger
	dockerClient        *client.Client
	dockerNetworkId     string
	containerNamePrefix string
	containerCounter    uint32

	runningMutex sync.Mutex
	running      map[*dockerInstance]struct{}
}

var _ Creator = (*dockerCreator)(nil)

func NewDockerCreator(
	logger *zap.Logger,
	dockerClient *client.Client,
	containerNamePrefix string,
	dockerNetworkId string,
) Creator {
	return &dockerCreator{
		logger:              logger.With(zap.Bool("icinga_notifications", true)),
		dockerClient:        dockerClient,
		dockerNetworkId:     dockerNetworkId,
		containerNamePrefix: containerNamePrefix,
		running:             make(map[*dockerInstance]struct{}),
	}
}

func (i *dockerCreator) CreateIcingaNotifications(
	rdb services.RelationalDatabase,
	options ...services.IcingaNotificationsOption,
) services.IcingaNotificationsBase {
	inst := &dockerInstance{
		info: info{
			port: defaultPort,
			rdb:  rdb,
		},
		logger:                    i.logger,
		icingaNotificationsDocker: i,
	}

	idb := &services.IcingaNotifications{IcingaNotificationsBase: inst}
	services.WithIcingaNotificationsDefaultsEnvConfig(inst.info.rdb, ":"+defaultPort)(idb)
	for _, option := range options {
		option(idb)
	}

	containerName := fmt.Sprintf("%s-%d", i.containerNamePrefix, atomic.AddUint32(&i.containerCounter, 1))
	inst.logger = inst.logger.With(zap.String("container-name", containerName))
	networkName, err := utils.DockerNetworkName(context.Background(), i.dockerClient, i.dockerNetworkId)
	if err != nil {
		panic(err)
	}

	dockerImage := utils.GetEnvDefault("ICINGA_TESTING_NOTIFICATIONS_IMAGE", "icinga-notifications:latest")
	err = utils.DockerImagePull(context.Background(), inst.logger, i.dockerClient, dockerImage, false)
	if err != nil {
		panic(err)
	}

	cont, err := i.dockerClient.ContainerCreate(
		context.Background(),
		&container.Config{
			Image: dockerImage,
			Env:   idb.ConfEnviron(),
		},
		&container.HostConfig{},
		&network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				networkName: {
					NetworkID: i.dockerNetworkId,
				},
			},
		},
		nil,
		containerName)
	if err != nil {
		inst.logger.Fatal("failed to create icinga-notifications container", zap.Error(err))
	}
	inst.containerId = cont.ID
	inst.logger = inst.logger.With(zap.String("container-id", cont.ID))
	inst.logger.Debug("created container")

	err = utils.ForwardDockerContainerOutput(context.Background(), i.dockerClient, cont.ID,
		false, utils.NewLineWriter(func(line []byte) {
			inst.logger.Debug("container output",
				zap.ByteString("line", line))
		}))
	if err != nil {
		inst.logger.Fatal("failed to attach to container output", zap.Error(err))
	}

	err = i.dockerClient.ContainerStart(context.Background(), cont.ID, types.ContainerStartOptions{})
	if err != nil {
		inst.logger.Fatal("failed to start container", zap.Error(err))
	}
	inst.logger.Debug("started container")

	inst.info.host = utils.MustString(utils.DockerContainerAddress(context.Background(), i.dockerClient, cont.ID))

	i.runningMutex.Lock()
	i.running[inst] = struct{}{}
	i.runningMutex.Unlock()

	return inst
}

func (i *dockerCreator) Cleanup() {
	i.runningMutex.Lock()
	instances := make([]*dockerInstance, 0, len(i.running))
	for inst := range i.running {
		instances = append(instances, inst)
	}
	i.runningMutex.Unlock()

	for _, inst := range instances {
		inst.Cleanup()
	}
}

type dockerInstance struct {
	info
	icingaNotificationsDocker *dockerCreator
	logger                    *zap.Logger
	containerId               string
}

var _ services.IcingaNotificationsBase = (*dockerInstance)(nil)

func (i *dockerInstance) Cleanup() {
	i.icingaNotificationsDocker.runningMutex.Lock()
	delete(i.icingaNotificationsDocker.running, i)
	i.icingaNotificationsDocker.runningMutex.Unlock()

	err := i.icingaNotificationsDocker.dockerClient.ContainerRemove(context.Background(), i.containerId, types.ContainerRemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	})
	if err != nil {
		panic(err)
	}
	i.logger.Debug("removed container")
}
