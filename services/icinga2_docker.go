package services

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/icinga/icinga-testing/utils"
	"go.uber.org/zap"
	"io"
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

	net, err := i.dockerClient.NetworkInspect(context.Background(), i.dockerNetworkId, types.NetworkInspectOptions{})
	if err != nil {
		panic(err)
	}

	cont, err := i.dockerClient.ContainerCreate(context.Background(), &container.Config{
		Image: "icinga/icinga2:master",
		Env:   []string{"ICINGA_MASTER=1"},
	}, nil, &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			net.Name: {
				NetworkID: i.dockerNetworkId,
			},
		},
	}, nil, containerName)
	if err != nil {
		i.logger.Fatal("failed to create icinga2 container",
			zap.String("name", containerName),
			zap.Error(err))
	}
	i.logger.Debug("created icinga2 container",
		zap.String("name", containerName),
		zap.String("id", cont.ID))

	err = utils.ForwardDockerContainerOutput(context.Background(), i.dockerClient, cont.ID,
		false, utils.NewLineWriter(func(line []byte) {
			i.logger.Debug("container output",
				zap.String("name", containerName),
				zap.String("id", cont.ID),
				zap.ByteString("line", line))
		}))
	if err != nil {
		i.logger.Fatal("failed to attach to container output",
			zap.String("name", containerName),
			zap.String("id", cont.ID),
			zap.Error(err))
	}

	err = i.dockerClient.ContainerStart(context.Background(), cont.ID, types.ContainerStartOptions{})
	if err != nil {
		i.logger.Fatal("failed to start container",
			zap.String("name", containerName),
			zap.String("id", cont.ID),
			zap.Error(err))
	}
	i.logger.Debug("started container",
		zap.String("name", containerName),
		zap.String("id", cont.ID))

	n := &icinga2DockerNode{
		icinga2NodeInfo: icinga2NodeInfo{
			host: utils.MustString(utils.DockerContainerAddress(context.Background(), i.dockerClient, cont.ID)),
			port: "5665",
		},
		icinga2Docker: i,
		containerId:   cont.ID,
		containerName: containerName,
	}

	for attempt := 1; ; attempt++ {
		time.Sleep(100 * time.Millisecond)
		err := Icinga2NodePing(n)
		if err == nil {
			break
		} else if attempt == 100 {
			i.logger.Fatal("icinga2 failed to start in time",
				zap.String("name", containerName),
				zap.String("id", cont.ID),
				zap.Error(err))
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
	containerId   string
	containerName string
}

var _ Icinga2Node = (*icinga2DockerNode)(nil)

func (n *icinga2DockerNode) Reload() {
	err := n.icinga2Docker.dockerClient.ContainerKill(context.Background(), n.containerId, "HUP")
	if err != nil {
		n.icinga2Docker.logger.Fatal("failed to send reload signal to container",
			zap.String("name", n.containerName),
			zap.String("id", n.containerId))
	}
	n.icinga2Docker.logger.Debug("sent reload signal to icinga2",
		zap.String("name", n.containerName),
		zap.String("id", n.containerId))

	// TODO(jb): wait for successful reload?
}

func (n *icinga2DockerNode) WriteConfig(file string, data []byte) {
	logger := n.icinga2Docker.logger.With(
		zap.String("name", n.containerName),
		zap.String("id", n.containerName),
		zap.String("file", file),
	)

	logger.Debug("writing file to container", zap.ByteString("data", data))
	c := n.icinga2Docker.dockerClient
	exec, err := c.ContainerExecCreate(context.Background(), n.containerId, types.ExecConfig{
		Cmd:          []string{"tee", "/" + file},
		AttachStdin:  true,
		AttachStderr: true,
		Detach:       true,
	})
	if err != nil {
		logger.Fatal("failed to create container exec", zap.Error(err))
	}
	attach, err := c.ContainerExecAttach(context.Background(), exec.ID, types.ExecStartCheck{})
	if err != nil {
		logger.Fatal("failed to attach container exec", zap.Error(err))
	}
	_, err = attach.Conn.Write(data)
	if err != nil {
		logger.Fatal("failed to write data to container exec", zap.Error(err))
	}
	err = attach.CloseWrite()
	if err != nil {
		logger.Fatal("failed to close container exec", zap.Error(err))
	}
	for {
		inspect, err := c.ContainerExecInspect(context.Background(), exec.ID)
		if err != nil {
			panic(err)
		}
		if !inspect.Running {
			w := utils.NewLineWriter(func(line []byte) {
				msg := "container output"
				field := zap.ByteString("line", line)
				logger.Error(msg, field)
			})
			_, err = io.Copy(w, attach.Reader)
			if err != nil {
				logger.Fatal("failed to copy container output", zap.Error(err))
			}

			if inspect.ExitCode != 0 {
				logger.Fatal("writing file to container failed", zap.Int("exit-code", inspect.ExitCode))
			} else {
				return
			}
		}
		time.Sleep(25 * time.Millisecond)
	}
}

func (n *icinga2DockerNode) EnableIcingaDb(redis RedisServer) {
	Icinga2NodeWriteIcingaDbConf(n, redis)
	c := n.icinga2Docker.dockerClient
	exec, err := c.ContainerExecCreate(context.Background(), n.containerId, types.ExecConfig{
		Cmd: []string{"icinga2", "feature", "enable", "icingadb"},
	})
	if err != nil {
		panic(err)
	}
	err = c.ContainerExecStart(context.Background(), exec.ID, types.ExecStartCheck{})
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
	n.icinga2Docker.logger.Debug("removed icinga2 container",
		zap.String("name", n.containerName),
		zap.String("id", n.containerId))
}
