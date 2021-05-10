package icingatesting

import (
	"context"
	"flag"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/icinga/icinga-testing/services"
	"github.com/icinga/icinga-testing/utils"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"sync"
)

type IT struct {
	mutex           sync.Mutex
	deferredCleanup []func()
	prefix          string
	dockerClient    *client.Client
	dockerNetworkId string
	mysqlServer     services.MysqlServer
	redis           services.Redis
	icinga2         services.Icinga2
	icingaDb        services.IcingaDb
	logger          *zap.Logger
}

var flagDebugLog = flag.String("icingatesting.debuglog", "", "file to write debug log to")

func NewIT() *IT {
	flag.Parse()

	it := &IT{
		prefix: "icinga-testing-" + utils.RandomString(8),
	}

	it.setupLogging()

	if c, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation()); err != nil {
		it.logger.Fatal("failed to create docker client", zap.Error(err))
	} else {
		it.dockerClient = c
		it.deferCleanup(func() {
			if err := it.dockerClient.Close(); err != nil {
				it.logger.Error("failed to close docker client", zap.Error(err))
			}
		})
	}

	if n, err := it.dockerClient.NetworkCreate(context.Background(), it.prefix, types.NetworkCreate{}); err != nil {
		it.logger.Fatal("failed to create docker network", zap.String("name", it.prefix), zap.Error(err))
	} else {
		it.logger.Debug("created docker network", zap.String("name", it.prefix), zap.String("id", n.ID))
		it.dockerNetworkId = n.ID
		it.deferCleanup(func() {
			if err := it.dockerClient.NetworkRemove(context.Background(), it.dockerNetworkId); err != nil {
				it.logger.Error("failed to remove docker network",
					zap.String("name", it.prefix), zap.String("id", n.ID), zap.Error(err))
			}
		})
	}

	return it
}

func (it *IT) setupLogging() {
	cores := []zapcore.Core{
		// Log INFO and higher as console log to stderr
		zapcore.NewCore(zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()),
			zapcore.Lock(os.Stderr), zapcore.InfoLevel),
	}

	if *flagDebugLog != "" {
		w, closeLogs, err := zap.Open(*flagDebugLog)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to open debug log %q: %v", *flagDebugLog, err)
		}
		c := zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewDevelopmentEncoderConfig()),
			w, zapcore.DebugLevel)
		cores = append(cores, c)
		it.deferCleanup(func() {
			it.logger.Debug("closing logs")
			closeLogs()
		})
	}

	it.logger = zap.New(zapcore.NewTee(cores...))
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

func (it *IT) MysqlServer() services.MysqlServer {
	it.mutex.Lock()
	defer it.mutex.Unlock()

	if it.mysqlServer == nil {
		it.mysqlServer = services.NewMysqlDocker(it.logger, it.dockerClient, it.prefix+"-mysql", it.dockerNetworkId)
		it.deferCleanup(it.mysqlServer.Cleanup)
	}

	return it.mysqlServer
}

func (it *IT) MysqlDatabase() services.MysqlDatabase {
	return it.MysqlServer().Database()
}

func (it *IT) Redis() services.Redis {
	it.mutex.Lock()
	defer it.mutex.Unlock()

	if it.redis == nil {
		it.redis = services.NewRedisDocker(it.logger, it.dockerClient, it.prefix+"-redis", it.dockerNetworkId)
		it.deferCleanup(it.redis.Cleanup)
	}

	return it.redis
}

func (it *IT) RedisServer() services.RedisServer {
	return it.Redis().Server()
}

func (it *IT) Icinga2() services.Icinga2 {
	it.mutex.Lock()
	defer it.mutex.Unlock()

	if it.icinga2 == nil {
		it.icinga2 = services.NewIcinga2Docker(it.logger, it.dockerClient, it.prefix+"-icinga2", it.dockerNetworkId)
		it.deferCleanup(it.icinga2.Cleanup)
	}

	return it.icinga2
}

func (it *IT) Icinga2Node(name string) services.Icinga2Node {
	return it.Icinga2().Node(name)
}

func (it *IT) IcingaDb() services.IcingaDb {
	key := "ICINGA_TESTING_ICINGADB_BINARY"
	path, ok := os.LookupEnv(key)
	if !ok {
		panic(fmt.Errorf("environment variable %s must be set", key))
	}

	it.mutex.Lock()
	defer it.mutex.Unlock()

	if it.icingaDb == nil {
		it.icingaDb = services.NewIcingaDbDockerBinary(it.logger, it.dockerClient, it.prefix+"-icingadb",
			it.dockerNetworkId, path)
		it.deferCleanup(it.icingaDb.Cleanup)
	}

	return it.icingaDb
}

func (it *IT) IcingaDbInstance(redis services.RedisServer, mysql services.MysqlDatabase) services.IcingaDbInstance {
	return it.IcingaDb().Instance(redis, mysql)
}
