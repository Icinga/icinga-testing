package services

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/icinga/icinga-testing/utils"
	"log"
	"sync/atomic"
	"time"
)

type redisDocker struct {
	dockerClient        *client.Client
	dockerNetworkId     string
	containerNamePrefix string
	containerCounter    uint32
}

var _ Redis = (*redisDocker)(nil)

func NewRedisDocker(dockerClient *client.Client, containerNamePrefix string, dockerNetworkId string) Redis {
	return &redisDocker{
		dockerClient:        dockerClient,
		dockerNetworkId:     dockerNetworkId,
		containerNamePrefix: containerNamePrefix,
	}
}

func (r *redisDocker) Server() RedisServer {
	containerName := fmt.Sprintf("%s-%d", r.containerNamePrefix, atomic.AddUint32(&r.containerCounter, 1))
	networkName := "net"

	cont, err := r.dockerClient.ContainerCreate(context.Background(), &container.Config{
		Image: "redis:latest",
	}, nil, &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			networkName: {
				NetworkID: r.dockerNetworkId,
			},
		},
	}, nil, containerName)
	if err != nil {
		panic(err)
	}
	log.Printf("created redis container: %s (%s)", containerName, cont.ID)

	err = utils.ForwardDockerContainerOutput(context.Background(), r.dockerClient, cont.ID,
		false, utils.NewLogWriter(containerName))
	if err != nil {
		panic(err)
	}

	err = r.dockerClient.ContainerStart(context.Background(), cont.ID, types.ContainerStartOptions{})
	if err != nil {
		panic(err)
	}
	log.Printf("started redis container: %s (%s)", containerName, cont.ID)

	s := &redisDockerServer{
		redisServerInfo: redisServerInfo{
			host: utils.MustString(utils.DockerContainerAddress(context.Background(), r.dockerClient, cont.ID)),
			port: "6379",
		},
		redisDocker: r,
		containerId: cont.ID,
	}

	c := RedisServerOpen(s)
	for attempt := 1; ; attempt++ {
		time.Sleep(1 * time.Second)
		_, err := c.Ping(context.Background()).Result()
		if err == nil {
			break
		} else if attempt == 20 {
			panic(err)
		}
	}
	if err := c.Close(); err != nil {
		panic(err)
	}

	return s
}

func (r *redisDocker) Cleanup() {
	// TODO(jb): store all spawned containers and kill them
}

type redisDockerServer struct {
	redisServerInfo
	redisDocker *redisDocker
	containerId string
}

func (s *redisDockerServer) Cleanup() {
	err := s.redisDocker.dockerClient.ContainerRemove(context.Background(), s.containerId, types.ContainerRemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	})
	if err != nil {
		panic(err)
	}
	log.Printf("removed redis container: %s", s.containerId)
}

var _ RedisServer = (*redisDockerServer)(nil)
