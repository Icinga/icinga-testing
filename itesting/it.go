package itesting

import (
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/icinga/icinga-testing/services"
	"github.com/icinga/icinga-testing/utils"
	"log"
	"sync"
)

type IT struct {
	mutex           sync.Mutex
	deferredCleanup []func()
	prefix          string
	dockerClient    *client.Client
	dockerNetworkId string
	mysqlServer     services.MysqlServer
}

func NewIT() *IT {
	it := &IT{}

	it.prefix = "icinga-testing-" + utils.RandomString(8)

	c, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("failed to create docker client: %v", err)
	}
	it.dockerClient = c
	it.deferCleanup(func() {
		if err := it.dockerClient.Close(); err != nil {
			log.Fatalf("failed to close docker client: %v", err)
		}
	})

	network, err := c.NetworkCreate(context.Background(), it.prefix, types.NetworkCreate{})
	if err != nil {
		log.Fatalf("failed to create docker network: %v", err)
	}
	log.Printf("created network %s (%s)", it.prefix, network.ID)
	it.dockerNetworkId = network.ID
	it.deferCleanup(func() {
		if err := it.dockerClient.NetworkRemove(context.Background(), it.dockerNetworkId); err != nil {
			log.Fatalf("failed to remove docker network %s (%s): %v", it.prefix, it.dockerNetworkId, err)
		}
	})

	return it
}

// deferCleanup registers a cleanup function that is called when Cleanup is called on the IT object. The caller must
// ensure that IT.mutex is locked. Cleanup functions are called in reversed registration order (just like the defer
// keyword in Go does).
func (it *IT) deferCleanup(f func()) {
	it.deferredCleanup = append(it.deferredCleanup, f)
}

func (it *IT) Cleanup() {
	it.mutex.Lock()
	defer it.mutex.Unlock()

	for i := len(it.deferredCleanup) - 1; i >= 0; i-- {
		f := it.deferredCleanup[i]
		f()
	}
}

func (it *IT) DockerClient() *client.Client {
	return it.dockerClient
}

func (it *IT) DockerNetworkId() string {
	return it.dockerNetworkId
}

func (it *IT) MysqlServer() services.MysqlServer {
	it.mutex.Lock()
	defer it.mutex.Unlock()

	if it.mysqlServer == nil {
		it.mysqlServer = services.NewMysqlDocker(it.DockerClient(), it.prefix+"-mysql", it.DockerNetworkId())
		it.deferCleanup(it.mysqlServer.Cleanup)
	}

	return it.mysqlServer
}

func (it *IT) MysqlDatabase() services.MysqlDatabase {
	return it.MysqlServer().Database()
}
