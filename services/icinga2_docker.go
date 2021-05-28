package services

import (
	"bytes"
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/icinga/icinga-testing/utils"
	"go.uber.org/zap"
	"sync"
	"time"
)

type icinga2Docker struct {
	logger              *zap.Logger
	dockerClient        *client.Client
	dockerNetworkId     string
	containerNamePrefix string

	runningMutex sync.Mutex
	running      map[*icinga2DockerNode]struct{}
}

var _ Icinga2 = (*icinga2Docker)(nil)

func NewIcinga2Docker(logger *zap.Logger, dockerClient *client.Client, containerNamePrefix string, dockerNetworkId string) Icinga2 {
	return &icinga2Docker{
		logger:              logger.With(zap.Bool("icinga2", true)),
		dockerClient:        dockerClient,
		dockerNetworkId:     dockerNetworkId,
		containerNamePrefix: containerNamePrefix,
		running:             make(map[*icinga2DockerNode]struct{}),
	}
}

func (i *icinga2Docker) Node(name string) Icinga2Node {
	containerName := fmt.Sprintf("%s-%s", i.containerNamePrefix, name)
	logger := i.logger.With(zap.String("container-name", containerName))

	networkName, err := utils.DockerNetworkName(context.Background(), i.dockerClient, i.dockerNetworkId)
	if err != nil {
		panic(err)
	}

	cont, err := i.dockerClient.ContainerCreate(context.Background(), &container.Config{
		Image: "icinga/icinga2:master",
		Env:   []string{"ICINGA_MASTER=1"},
	}, nil, &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			networkName: {
				NetworkID: i.dockerNetworkId,
			},
		},
	}, nil, containerName)
	if err != nil {
		logger.Fatal("failed to create icinga2 container", zap.Error(err))
	}
	logger = logger.With(zap.String("container-id", cont.ID))
	logger.Debug("created icinga2 container")

	err = utils.ForwardDockerContainerOutput(context.Background(), i.dockerClient, cont.ID,
		false, utils.NewLineWriter(func(line []byte) {
			logger.Debug("container output", zap.ByteString("line", line))
		}))
	if err != nil {
		logger.Fatal("failed to attach to container output", zap.Error(err))
	}

	err = i.dockerClient.ContainerStart(context.Background(), cont.ID, types.ContainerStartOptions{})
	if err != nil {
		logger.Fatal("failed to start container", zap.Error(err))
	}
	logger.Debug("started container")

	n := &icinga2DockerNode{
		icinga2NodeInfo: icinga2NodeInfo{
			host: utils.MustString(utils.DockerContainerAddress(context.Background(), i.dockerClient, cont.ID)),
			port: "5665",
		},
		icinga2Docker: i,
		logger:        logger,
		containerId:   cont.ID,
		containerName: containerName,
	}

	for attempt := 1; ; attempt++ {
		time.Sleep(100 * time.Millisecond)
		err := Icinga2NodePing(n)
		if err == nil {
			break
		} else if attempt == 100 {
			logger.Fatal("icinga2 failed to start in time", zap.Error(err))
		}
	}

	i.runningMutex.Lock()
	i.running[n] = struct{}{}
	i.runningMutex.Unlock()

	return n
}

func (i *icinga2Docker) Cleanup() {
	i.runningMutex.Lock()
	nodes := make([]*icinga2DockerNode, 0, len(i.running))
	for n, _ := range i.running {
		nodes = append(nodes, n)
	}
	i.runningMutex.Unlock()

	for _, n := range nodes {
		n.Cleanup()
	}
}

type icinga2DockerNode struct {
	icinga2NodeInfo
	icinga2Docker *icinga2Docker
	logger        *zap.Logger
	containerId   string
	containerName string
}

var _ Icinga2Node = (*icinga2DockerNode)(nil)

func (n *icinga2DockerNode) Reload() {
	err := n.icinga2Docker.dockerClient.ContainerKill(context.Background(), n.containerId, "HUP")
	if err != nil {
		n.logger.Fatal("failed to send reload signal to container")
	}
	n.logger.Debug("sent reload signal to icinga2")

	// TODO(jb): wait for successful reload?
}

func (n *icinga2DockerNode) WriteConfig(file string, data []byte) {
	logger := n.logger.With(zap.String("file", file))

	stderr := utils.NewLineWriter(func(line []byte) {
		logger.Error("error from container while writing file", zap.ByteString("line", line))
	})

	err := utils.DockerExec(context.Background(), n.icinga2Docker.dockerClient, n.logger, n.containerId,
		[]string{"tee", "/" + file}, bytes.NewReader(data), nil, stderr)
	if err != nil {
		panic(err)
	}
}

func (n *icinga2DockerNode) EnableIcingaDb(redis RedisServer) {
	Icinga2NodeWriteIcingaDbConf(n, redis)

	stdout := utils.NewLineWriter(func(line []byte) {
		n.logger.Debug("exec stdout", zap.ByteString("line", line))
	})
	stderr := utils.NewLineWriter(func(line []byte) {
		n.logger.Error("exec stderr", zap.ByteString("line", line))
	})

	err := utils.DockerExec(context.Background(), n.icinga2Docker.dockerClient, n.logger, n.containerId,
		[]string{"icinga2", "feature", "enable", "icingadb"}, nil, stdout, stderr)
	if err != nil {
		panic(err)
	}
}

func (n *icinga2DockerNode) Cleanup() {
	n.icinga2Docker.runningMutex.Lock()
	delete(n.icinga2Docker.running, n)
	n.icinga2Docker.runningMutex.Unlock()

	err := n.icinga2Docker.dockerClient.ContainerRemove(context.Background(), n.containerId, types.ContainerRemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	})
	if err != nil {
		panic(err)
	}
	n.logger.Debug("removed icinga2 container")
}
