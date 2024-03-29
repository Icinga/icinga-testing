package redis

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
	"os"
	"sync"
	"sync/atomic"
	"time"
)

type dockerCreator struct {
	logger              *zap.Logger
	dockerClient        *client.Client
	dockerNetworkId     string
	containerNamePrefix string
	containerCounter    uint32

	runningMutex sync.Mutex
	running      map[*dockerServer]struct{}
}

var _ Creator = (*dockerCreator)(nil)

func NewDockerCreator(
	logger *zap.Logger, dockerClient *client.Client, containerNamePrefix string, dockerNetworkId string,
) Creator {
	return &dockerCreator{
		logger:              logger.With(zap.Bool("redis", true)),
		dockerClient:        dockerClient,
		dockerNetworkId:     dockerNetworkId,
		containerNamePrefix: containerNamePrefix,
		running:             make(map[*dockerServer]struct{}),
	}
}

func (r *dockerCreator) CreateRedisServer() services.RedisServerBase {
	containerName := fmt.Sprintf("%s-%d", r.containerNamePrefix, atomic.AddUint32(&r.containerCounter, 1))
	logger := r.logger.With(zap.String("container-name", containerName))

	networkName, err := utils.DockerNetworkName(context.Background(), r.dockerClient, r.dockerNetworkId)
	if err != nil {
		panic(err)
	}

	dockerImage := utils.GetEnvDefault("ICINGA_TESTING_REDIS_IMAGE", "redis:latest")
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
		logger.Fatal("failed to start container", zap.Error(err))
	}
	logger.Debug("started container")

	s := &dockerServer{
		info: info{
			host: utils.MustString(utils.DockerContainerAddress(context.Background(), r.dockerClient, cont.ID)),
			port: "6379",
		},
		redisDocker: r,
		logger:      logger,
		containerId: cont.ID,
	}

	c := services.RedisServer{RedisServerBase: s}.Open()
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

	if os.Getenv("ICINGA_TESTING_REDIS_MONITOR") == "1" {
		go func() {
			stdout := utils.NewLineWriter(func(line []byte) {
				r.logger.Debug("redis-cli monitor", zap.ByteString("command", line))
			})
			stderr := utils.NewLineWriter(func(line []byte) {
				r.logger.Debug("redis-cli monitor", zap.ByteString("error", line))
			})

			cmd := []string{"redis-cli", "monitor"}
			err := utils.DockerExec(context.Background(), r.dockerClient, logger, cont.ID, cmd, nil, stdout, stderr)
			if err != nil {
				r.logger.Debug("redis-cli monitor exited with an error", zap.Error(err))
			}
		}()
	}

	r.runningMutex.Lock()
	r.running[s] = struct{}{}
	r.runningMutex.Unlock()

	return s
}

func (r *dockerCreator) Cleanup() {
	r.runningMutex.Lock()
	servers := make([]*dockerServer, 0, len(r.running))
	for s := range r.running {
		servers = append(servers, s)
	}
	r.runningMutex.Unlock()

	for _, s := range servers {
		s.Cleanup()
	}
}

type dockerServer struct {
	info
	redisDocker *dockerCreator
	logger      *zap.Logger
	containerId string
}

func (s *dockerServer) Cleanup() {
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

var _ services.RedisServerBase = (*dockerServer)(nil)
