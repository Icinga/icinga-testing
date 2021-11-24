package icingadb

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/icinga/icinga-testing/services"
	"github.com/icinga/icinga-testing/utils"
	"go.uber.org/zap"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
)

type dockerBinaryCreator struct {
	logger              *zap.Logger
	dockerClient        *client.Client
	dockerNetworkId     string
	containerNamePrefix string
	binaryPath          string
	containerCounter    uint32

	runningMutex sync.Mutex
	running      map[*dockerBinaryInstance]struct{}
}

var _ Creator = (*dockerBinaryCreator)(nil)

func NewDockerBinaryCreator(
	logger *zap.Logger,
	dockerClient *client.Client,
	containerNamePrefix string,
	dockerNetworkId string,
	binaryPath string,
) Creator {
	binaryPath, err := filepath.Abs(binaryPath)
	if err != nil {
		panic(err)
	}
	return &dockerBinaryCreator{
		logger:              logger.With(zap.Bool("icingadb", true)),
		dockerClient:        dockerClient,
		dockerNetworkId:     dockerNetworkId,
		containerNamePrefix: containerNamePrefix,
		binaryPath:          binaryPath,
		running:             make(map[*dockerBinaryInstance]struct{}),
	}
}

func (i *dockerBinaryCreator) CreateIcingaDb(
	redis services.RedisServerBase,
	mysql services.MysqlDatabaseBase,
) services.IcingaDbBase {
	inst := &dockerBinaryInstance{
		info: info{
			redis: redis,
			mysql: mysql,
		},
		logger:               i.logger,
		icingaDbDockerBinary: i,
	}

	services.MysqlDatabase{MysqlDatabaseBase: mysql}.ImportIcingaDbSchema()

	configFile, err := ioutil.TempFile("", "icingadb.yml")
	if err != nil {
		panic(err)
	}
	err = services.IcingaDb{IcingaDbBase: inst}.WriteConfig(configFile)
	if err != nil {
		panic(err)
	}
	inst.configFileName = configFile.Name()
	err = configFile.Close()
	if err != nil {
		panic(err)
	}

	containerName := fmt.Sprintf("%s-%d", i.containerNamePrefix, atomic.AddUint32(&i.containerCounter, 1))
	inst.logger = inst.logger.With(zap.String("container-name", containerName))
	networkName, err := utils.DockerNetworkName(context.Background(), i.dockerClient, i.dockerNetworkId)
	if err != nil {
		panic(err)
	}

	dockerImage := "alpine:latest"
	err = utils.DockerImagePull(context.Background(), inst.logger, i.dockerClient, dockerImage, false)
	if err != nil {
		panic(err)
	}

	cont, err := i.dockerClient.ContainerCreate(context.Background(), &container.Config{
		Image: dockerImage,
		Cmd:   []string{"/icingadb", "--config", "/icingadb.yml"},
	}, &container.HostConfig{
		Mounts: []mount.Mount{{
			Type:     mount.TypeBind,
			Source:   i.binaryPath,
			Target:   "/icingadb",
			ReadOnly: true,
		}, {
			Type:     mount.TypeBind,
			Source:   inst.configFileName,
			Target:   "/icingadb.yml",
			ReadOnly: true,
		}},
	}, &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			networkName: {
				NetworkID: i.dockerNetworkId,
			},
		},
	}, nil, containerName)
	if err != nil {
		inst.logger.Fatal("failed to create icingadb container", zap.Error(err))
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
		inst.logger.Fatal("failed to attach to container output",
			zap.Error(err))
	}

	err = i.dockerClient.ContainerStart(context.Background(), cont.ID, types.ContainerStartOptions{})
	if err != nil {
		inst.logger.Fatal("failed to start container")
	}
	inst.logger.Debug("started container")

	i.runningMutex.Lock()
	i.running[inst] = struct{}{}
	i.runningMutex.Unlock()

	return inst
}

func (i *dockerBinaryCreator) Cleanup() {
	i.runningMutex.Lock()
	instances := make([]*dockerBinaryInstance, 0, len(i.running))
	for inst, _ := range i.running {
		instances = append(instances, inst)
	}
	i.runningMutex.Unlock()

	for _, inst := range instances {
		inst.Cleanup()
	}
}

type dockerBinaryInstance struct {
	info
	icingaDbDockerBinary *dockerBinaryCreator
	logger               *zap.Logger
	containerId          string
	configFileName       string
}

var _ services.IcingaDbBase = (*dockerBinaryInstance)(nil)

func (i *dockerBinaryInstance) Cleanup() {
	i.icingaDbDockerBinary.runningMutex.Lock()
	delete(i.icingaDbDockerBinary.running, i)
	i.icingaDbDockerBinary.runningMutex.Unlock()

	err := i.icingaDbDockerBinary.dockerClient.ContainerRemove(context.Background(), i.containerId, types.ContainerRemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	})
	if err != nil {
		panic(err)
	}
	i.logger.Debug("removed container")

	err = os.Remove(i.configFileName)
	if err != nil {
		panic(err)
	}
}
