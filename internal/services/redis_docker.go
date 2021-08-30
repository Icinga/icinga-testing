package services

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
	"time"
)

type redisDocker struct {
	logger              *zap.Logger
	dockerClient        *client.Client
	dockerNetworkId     string
	containerNamePrefix string
	containerCounter    uint32

	runningMutex sync.Mutex
	running      map[*redisDockerServer]struct{}
}

var _ Redis = (*redisDocker)(nil)

func NewRedisDocker(
	logger *zap.Logger, dockerClient *client.Client, containerNamePrefix string, dockerNetworkId string,
) Redis {
	return &redisDocker{
		logger:              logger.With(zap.Bool("redis", true)),
		dockerClient:        dockerClient,
		dockerNetworkId:     dockerNetworkId,
		containerNamePrefix: containerNamePrefix,
		running:             make(map[*redisDockerServer]struct{}),
	}
}

func (r *redisDocker) Server() services.RedisServer {
	containerName := fmt.Sprintf("%s-%d", r.containerNamePrefix, atomic.AddUint32(&r.containerCounter, 1))
	logger := r.logger.With(zap.String("container-name", containerName))

	networkName, err := utils.DockerNetworkName(context.Background(), r.dockerClient, r.dockerNetworkId)
	if err != nil {
		panic(err)
	}

	dockerImage := "redis:latest"
	err = utils.DockerImagePull(context.Background(), logger, r.dockerClient, dockerImage, false)
	if err != nil {
		panic(err)
	}

	cont, err := r.dockerClient.ContainerCreate(context.Background(), &container.Config{
		Image: dockerImage,
	}, nil, &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			networkName: {
				NetworkID: r.dockerNetworkId,
			},
		},
	}, nil, containerName)
	if err != nil {
		logger.Fatal("failed to create redis container")
	}
	logger = logger.With(zap.String("container-id", cont.ID))
	logger.Debug("started redis container")

	err = utils.ForwardDockerContainerOutput(context.Background(), r.dockerClient, cont.ID,
		false, utils.NewLineWriter(func(line []byte) {
			logger.Debug("container output",
				zap.ByteString("line", line))
		}))
	if err != nil {
		logger.Fatal("failed to attach to container output",
			zap.Error(err))
	}

	err = r.dockerClient.ContainerStart(context.Background(), cont.ID, types.ContainerStartOptions{})
	if err != nil {
		logger.Fatal("failed to start container")
	}
	logger.Debug("started container")

	s := &redisDockerServer{
		redisServerInfo: redisServerInfo{
			host: utils.MustString(utils.DockerContainerAddress(context.Background(), r.dockerClient, cont.ID)),
			port: "6379",
		},
		redisDocker: r,
		logger:      logger,
		containerId: cont.ID,
	}

	c := services.RedisServerOpen(s)
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

	r.runningMutex.Lock()
	r.running[s] = struct{}{}
	r.runningMutex.Unlock()

	return s
}

func (r *redisDocker) Cleanup() {
	r.runningMutex.Lock()
	servers := make([]*redisDockerServer, 0, len(r.running))
	for s, _ := range r.running {
		servers = append(servers, s)
	}
	r.runningMutex.Unlock()

	for _, s := range servers {
		s.Cleanup()
	}
}

type redisDockerServer struct {
	redisServerInfo
	redisDocker *redisDocker
	logger      *zap.Logger
	containerId string
}

func (s *redisDockerServer) Cleanup() {
	s.redisDocker.runningMutex.Lock()
	delete(s.redisDocker.running, s)
	s.redisDocker.runningMutex.Unlock()

	err := s.redisDocker.dockerClient.ContainerRemove(context.Background(), s.containerId, types.ContainerRemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	})
	if err != nil {
		panic(err)
	}
	s.logger.Debug("removed container")
}

var _ services.RedisServer = (*redisDockerServer)(nil)
