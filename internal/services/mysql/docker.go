package mysql

import (
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/icinga/icinga-testing/utils"
	"go.uber.org/zap"
	"time"
)

type dockerCreator struct {
	*rootConnection
	logger        *zap.Logger
	client        *client.Client
	containerId   string
	containerName string
}

var _ Creator = (*dockerCreator)(nil)

func NewDockerCreator(logger *zap.Logger, dockerClient *client.Client, containerName string, dockerNetworkId string) *dockerCreator {
	logger = logger.With(
		zap.Bool("mysql", true),
		zap.String("container-name", containerName),
	)

	networkName, err := utils.DockerNetworkName(context.Background(), dockerClient, dockerNetworkId)
	if err != nil {
		panic(err)
	}

	dockerImage := utils.GetEnvDefault("ICINGA_TESTING_MYSQL_IMAGE", "mysql:latest")
	err = utils.DockerImagePull(context.Background(), logger, dockerClient, dockerImage, false)
	if err != nil {
		panic(err)
	}

	rootPassword := utils.RandomString(16)
	cont, err := dockerClient.ContainerCreate(context.Background(), &container.Config{
		ExposedPorts: nil,
		Env:          []string{"MYSQL_ROOT_PASSWORD=" + rootPassword},
		Cmd:          nil,
		Image:        dockerImage,
	}, nil, &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			networkName: {
				Aliases:   []string{"mysql"},
				NetworkID: dockerNetworkId,
			},
		},
	}, nil, containerName)
	if err != nil {
		logger.Fatal("failed to create mysql container",
			zap.Error(err))
	}
	logger = logger.With(zap.String("container-id", cont.ID))
	logger.Debug("created mysql container")

	err = utils.ForwardDockerContainerOutput(context.Background(), dockerClient, cont.ID,
		false, utils.NewLineWriter(func(line []byte) {
			logger.Debug("container output", zap.ByteString("line", line))
		}))
	if err != nil {
		logger.Fatal("failed to attach to container output", zap.Error(err))
	}

	err = dockerClient.ContainerStart(context.Background(), cont.ID, types.ContainerStartOptions{})
	if err != nil {
		logger.Fatal("failed to start container", zap.Error(err))
	}
	logger.Debug("started mysql container")

	containerAddress := utils.MustString(utils.DockerContainerAddress(context.Background(), dockerClient, cont.ID))

	d := &dockerCreator{
		rootConnection: newRootConnection(containerAddress, "3306", "root", rootPassword),
		logger:         logger,
		client:         dockerClient,
		containerId:    cont.ID,
		containerName:  containerName,
	}

	for attempt := 1; ; attempt++ {
		time.Sleep(1 * time.Second)
		err := d.rootConnection.db.Ping()
		if err == nil {
			break
		} else if attempt == 60 {
			logger.Fatal("mysql failed to start in time", zap.Error(err))
		}
	}

	return d
}

func (m *dockerCreator) Cleanup() {
	err := m.client.ContainerRemove(context.Background(), m.containerId, types.ContainerRemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	})
	if err != nil {
		m.logger.Error("failed to remove mysql container", zap.Error(err))
	} else {
		m.logger.Debug("removed mysql container")
	}
}
