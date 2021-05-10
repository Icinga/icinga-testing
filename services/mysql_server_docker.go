package services

import (
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/icinga/icinga-testing/utils"
	"log"
	"time"
)

type MysqlDocker struct {
	*mysqlServerWithRootCreds
	client      *client.Client
	containerId string
}

var _ MysqlServer = (*MysqlDocker)(nil)

func NewMysqlDocker(dockerClient *client.Client, containerName string, dockerNetworkId string) *MysqlDocker {
	networkName := "net"
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
		panic(err)
	}
	log.Printf("created mysql container: %s", cont.ID)

	err = utils.ForwardDockerContainerOutput(context.Background(), dockerClient, cont.ID,
		false, utils.NewLogWriter(containerName))
	if err != nil {
		panic(err)
	}

	err = dockerClient.ContainerStart(context.Background(), cont.ID, types.ContainerStartOptions{})
	if err != nil {
		panic(err)
	}
	log.Printf("started mysql container: %s", cont.ID)

	containerAddress := utils.MustString(utils.DockerContainerAddress(context.Background(), dockerClient, cont.ID))

	d := &MysqlDocker{
		mysqlServerWithRootCreds: NewMysqlServerWithRootCreds(containerAddress, "3306", "root", rootPassword),
		client:                   dockerClient,
		containerId:              cont.ID,
	}

	for attempt := 1; ; attempt++ {
		time.Sleep(1 * time.Second)
		err := d.mysqlServerWithRootCreds.db.Ping()
		if err == nil {
			break
		} else if attempt == 20 {
			panic(err)
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
		panic(err)
	}
	log.Printf("removed mysql container: %s", m.containerId)
}
