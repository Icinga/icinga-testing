package services

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/icinga/icinga-testing/utils"
	"io"
	"log"
	"os"
	"time"
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
		panic(err)
	}
	log.Printf("created icinga2 container: %s (%s)", containerName, cont.ID)

	err = i.dockerClient.ContainerStart(context.Background(), cont.ID, types.ContainerStartOptions{})
	if err != nil {
		panic(err)
	}
	log.Printf("started icinga2 container: %s (%s)", containerName, cont.ID)

	return &icinga2DockerNode{
		icinga2NodeInfo: icinga2NodeInfo{
			host: utils.MustString(utils.DockerContainerAddress(context.Background(), i.dockerClient, cont.ID)),
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

func (n *icinga2DockerNode) Reload() {
	// TODO(jb): debug why signal doesn't work
	//err := n.icinga2Docker.dockerClient.ContainerKill(context.Background(), n.containerId, "HUP")
	// TODO(jb): there seems to be some race condition here
	time.Sleep(2*time.Second)
	err := n.icinga2Docker.dockerClient.ContainerRestart(context.Background(), n.containerId, nil)
	if err != nil {
		panic(err)
	}
}

func (n *icinga2DockerNode) WriteConfig(file string, data []byte) {
	log.Printf("writing %q to container %s (%q)", file, n.containerId, data)
	c := n.icinga2Docker.dockerClient
	exec, err := c.ContainerExecCreate(context.Background(), n.containerId, types.ExecConfig{
		Cmd:          []string{"tee", "/" + file},
		AttachStdin:  true,
		AttachStderr: true,
		Detach:       true,
	})
	if err != nil {
		panic(err)
	}
	attach, err := c.ContainerExecAttach(context.Background(), exec.ID, types.ExecStartCheck{})
	if err != nil {
		panic(err)
	}
	_, err = attach.Conn.Write(data)
	if err != nil {
		panic(err)
	}
	err = attach.CloseWrite()
	if err != nil {
		panic(err)
	}
	for {
		log.Printf("waiting for file write")
		inspect, err := c.ContainerExecInspect(context.Background(), exec.ID)
		if err != nil {
			panic(err)
		}
		if !inspect.Running {
			if inspect.ExitCode == 0 {
				break
			} else {
				panic(fmt.Errorf("writing %q in container failed with exit code %d", file, inspect.ExitCode))
			}
		}
		time.Sleep(25 * time.Millisecond)
	}
	// TODO(jb): proper error handling
	log.Print("file written (hopefully)")
	io.Copy(os.Stderr, attach.Reader)
	log.Print("file written")
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
	err := n.icinga2Docker.dockerClient.ContainerRemove(context.Background(), n.containerId, types.ContainerRemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	})
	if err != nil {
		panic(err)
	}
	log.Printf("removed redis container: %s", n.containerId)
}
