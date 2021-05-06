package services

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"log"
)

type icinga2Docker struct {
	dockerClient        *client.Client
	dockerNetworkId     string
	containerNamePrefix string
}

var _ Icinga2 = (*icinga2Docker)(nil)

func NewIcinga2Docker(dockerClient *client.Client, containerNamePrefix string, dockerNetworkId string) Icinga2 {
	return &icinga2Docker{
		dockerClient:        dockerClient,
		dockerNetworkId:     dockerNetworkId,
		containerNamePrefix: containerNamePrefix,
	}
}

func (i *icinga2Docker) Node(name string) Icinga2Node {
	containerName := fmt.Sprintf("%s-%s", i.containerNamePrefix, name)
	networkName := "net"

	cont, err := i.dockerClient.ContainerCreate(context.Background(), &container.Config{
		Image: "icinga/icinga2:latest",
		Env:   []string{"ICINGA_MASTER=1"},
	}, nil, &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			networkName: {
				NetworkID: i.dockerNetworkId,
			},
		},
	}, nil, containerName)
	if err != nil {
		panic(err)
	}
	log.Printf("created icinga2 container: %s (%s)", containerName, cont.ID)

	err = i.dockerClient.ContainerStart(context.Background(), cont.ID, types.ContainerStartOptions{})
	if err != nil {
		panic(err)
	}
	log.Printf("started icinga2 container: %s (%s)", containerName, cont.ID)

	inspect, err := i.dockerClient.ContainerInspect(context.Background(), cont.ID)
	if err != nil {
		panic(err)
	}
	containerAddress := inspect.NetworkSettings.Networks[networkName].IPAddress

	return &icinga2DockerNode{
		icinga2NodeInfo: icinga2NodeInfo{
			host: containerAddress,
			port: "5665",
		},
		icinga2Docker: i,
		containerId:   cont.ID,
	}
}

func (i *icinga2Docker) Cleanup() {
	// TODO(jb): remove all containers
}

type icinga2DockerNode struct {
	icinga2NodeInfo
	icinga2Docker *icinga2Docker
	containerId   string
}

var _ Icinga2Node = (*icinga2DockerNode)(nil)

func (n *icinga2DockerNode) Cleanup() {
	err := n.icinga2Docker.dockerClient.ContainerRemove(context.Background(), n.containerId, types.ContainerRemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	})
	if err != nil {
		panic(err)
	}
	log.Printf("removed redis container: %s", n.containerId)
}
