package services

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync/atomic"
)

type icingaDbDockerBinary struct {
	dockerClient        *client.Client
	dockerNetworkId     string
	containerNamePrefix string
	binaryPath          string
	containerCounter    uint32
}

var _ IcingaDb = (*icingaDbDockerBinary)(nil)

func NewIcingaDbDockerBinary(
	dockerClient *client.Client, containerNamePrefix string, dockerNetworkId string, binaryPath string,
) IcingaDb {
	binaryPath, err := filepath.Abs(binaryPath)
	if err != nil {
		panic(err)
	}
	return &icingaDbDockerBinary{
		dockerClient:        dockerClient,
		dockerNetworkId:     dockerNetworkId,
		containerNamePrefix: containerNamePrefix,
		binaryPath:          binaryPath,
	}
}

func (i *icingaDbDockerBinary) Instance(redis RedisServer, mysql MysqlDatabase) IcingaDbInstance {
	inst := &icingaDbDockerBinaryInstance{
		icingaDbInstanceInfo: icingaDbInstanceInfo{
			redis: redis,
			mysql: mysql,
		},
		icingaDbDockerBinary: i,
	}

	IcingaDbInstanceImportSchema(inst)

	configFile, err := ioutil.TempFile("", "icingadb.yml")
	if err != nil {
		panic(err)
	}
	IcingaDbInstanceWriteConfig(inst, configFile)
	if err != nil {
		panic(err)
	}
	inst.configFileName = configFile.Name()
	err = configFile.Close()
	if err != nil {
		panic(err)
	}

	containerName := fmt.Sprintf("%s-%d", i.containerNamePrefix, atomic.AddUint32(&i.containerCounter, 1))
	networkName := "net"

	cont, err := i.dockerClient.ContainerCreate(context.Background(), &container.Config{
		Image: "alpine:latest",
		Cmd:   []string{"/icingadb", "--config", "/icingadb.yml"},
	}, &container.HostConfig{
		Mounts: []mount.Mount{{
			Type:     mount.TypeBind,
			Source:   i.binaryPath,
			Target:   "/icingadb",
			ReadOnly: true,
		}, {
			Type:     mount.TypeBind,
			Source:   inst.configFileName,
			Target:   "/icingadb.yml",
			ReadOnly: true,
		}},
	}, &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			networkName: {
				NetworkID: i.dockerNetworkId,
			},
		},
	}, nil, containerName)
	if err != nil {
		panic(err)
	}
	log.Printf("created icingadb container: %s (%s)", containerName, cont.ID)

	err = i.dockerClient.ContainerStart(context.Background(), cont.ID, types.ContainerStartOptions{})
	if err != nil {
		panic(err)
	}
	inst.containerId = cont.ID
	log.Printf("started icingadb container: %s (%s)", containerName, cont.ID)

	return inst
}

func (i *icingaDbDockerBinary) Cleanup() {
	// TODO(jb): remove all instances
}

type icingaDbDockerBinaryInstance struct {
	icingaDbInstanceInfo
	icingaDbDockerBinary *icingaDbDockerBinary
	containerId          string
	configFileName       string
}

var _ IcingaDbInstance = (*icingaDbDockerBinaryInstance)(nil)

func (i *icingaDbDockerBinaryInstance) Cleanup() {
	err := i.icingaDbDockerBinary.dockerClient.ContainerRemove(context.Background(), i.containerId, types.ContainerRemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	})
	if err != nil {
		panic(err)
	}
	log.Printf("removed icingadb container: %s", i.containerId)

	err = os.Remove(i.configFileName)
	if err != nil {
		panic(err)
	}
}
