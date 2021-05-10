package services

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

type MysqlDocker struct {
	*mysqlServerWithRootCreds
	logger        *zap.Logger
	client        *client.Client
	containerId   string
	containerName string
}

var _ MysqlServer = (*MysqlDocker)(nil)

func NewMysqlDocker(logger *zap.Logger, dockerClient *client.Client, containerName string, dockerNetworkId string) *MysqlDocker {
	logger = logger.With(zap.Bool("mysql", true))

	networkName, err := utils.DockerNetworkName(context.Background(), dockerClient, dockerNetworkId)
	if err != nil {
		panic(err)
	}

	rootPassword := utils.RandomString(16)
	cont, err := dockerClient.ContainerCreate(context.Background(), &container.Config{
		ExposedPorts: nil,
		Env:          []string{"MYSQL_ROOT_PASSWORD=" + rootPassword},
		Cmd:          nil,
		Image:        "mysql:latest",
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
			zap.String("name", containerName),
			zap.Error(err))
	}
	logger.Debug("created mysql container",
		zap.String("name", containerName),
		zap.String("id", cont.ID))

	err = utils.ForwardDockerContainerOutput(context.Background(), dockerClient, cont.ID,
		false, utils.NewLineWriter(func(line []byte) {
			logger.Debug("container output",
				zap.String("name", containerName),
				zap.String("id", cont.ID),
				zap.ByteString("line", line))
		}))
	if err != nil {
		logger.Fatal("failed to attach to container output",
			zap.String("name", containerName),
			zap.String("id", cont.ID),
			zap.Error(err))
	}

	err = dockerClient.ContainerStart(context.Background(), cont.ID, types.ContainerStartOptions{})
	if err != nil {
		logger.Fatal("failed to start container",
			zap.String("name", containerName),
			zap.String("id", cont.ID),
			zap.Error(err))
	}
	logger.Debug("started mysql container",
		zap.String("name", containerName),
		zap.String("id", cont.ID))

	containerAddress := utils.MustString(utils.DockerContainerAddress(context.Background(), dockerClient, cont.ID))

	d := &MysqlDocker{
		mysqlServerWithRootCreds: NewMysqlServerWithRootCreds(containerAddress, "3306", "root", rootPassword),
		logger:                   logger,
		client:                   dockerClient,
		containerId:              cont.ID,
		containerName:            containerName,
	}

	for attempt := 1; ; attempt++ {
		time.Sleep(1 * time.Second)
		err := d.mysqlServerWithRootCreds.db.Ping()
		if err == nil {
			break
		} else if attempt == 20 {
			logger.Fatal("mysql failed to start in time",
				zap.String("name", containerName),
				zap.String("id", cont.ID),
				zap.Error(err))
		}
	}

	return d
}

func (m *MysqlDocker) Cleanup() {
	err := m.client.ContainerRemove(context.Background(), m.containerId, types.ContainerRemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	})
	if err != nil {
		m.logger.Error("failed to remove mysql container",
			zap.String("name", m.containerName),
			zap.String("id", m.containerId),
			zap.Error(err))
	} else {
		m.logger.Debug("removed mysql container",
			zap.String("name", m.containerName),
			zap.String("id", m.containerId))
	}
}
