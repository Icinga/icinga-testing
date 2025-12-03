// Package icingatesting contains helpers to facilitate performing integration tests between components of the Icinga
// stack using the Go testing package. The general idea is to write test cases in Go that can dynamically spawn
// individual components as required, connect them and then perform checks on this setup. This is implemented by using
// the Docker API to start and stop containers locally as required by the tests.
//
// The following environment variables are used by icinga-testing:
//   - ICINGA_TESTING_ICINGA2_IMAGE: Icinga 2 container image to use (default: "icinga/icinga2:edge")
//   - ICINGA_TESTING_MYSQL_IMAGE: MySQL/MariaDB container image to use (default: "mysql:latest")
//   - ICINGA_TESTING_PGSQL_IMAGE: PostgreSQL container image to use (default: "postgres:latest")
//   - ICINGA_TESTING_REDIS_IMAGE: Redis container image to use (default: "redis:latest")
//   - ICINGA_TESTING_REDIS_MONITOR: If set to "1", log all Redis commands to the debug log using redis-cli monitor
//   - ICINGA_TESTING_ICINGADB_BINARY: Path to the Icinga DB binary to test. It will run in a container and therefore
//     must be compiled using CGO_ENABLED=0
//   - ICINGA_TESTING_ICINGADB_SCHEMA_MYSQL: Path to the full Icinga DB schema file for MySQL/MariaDB
//   - ICINGA_TESTING_ICINGADB_SCHEMA_PGSQL: Path to the full Icinga DB schema file for PostgreSQL
package icingatesting

import (
	"context"
	"flag"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/icinga/icinga-testing/internal/services/icinga2"
	"github.com/icinga/icinga-testing/internal/services/icingadb"
	"github.com/icinga/icinga-testing/internal/services/mysql"
	"github.com/icinga/icinga-testing/internal/services/postgresql"
	"github.com/icinga/icinga-testing/internal/services/redis"
	"github.com/icinga/icinga-testing/services"
	"github.com/icinga/icinga-testing/utils"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
	"os"
	"sync"
	"testing"
)

// IT is the core type to start interacting with this module.
//
// The intended use is to create a global variable of type *IT in the test package and then initialize it in TestMain
// to allow the individual Test* functions to make use of it to dynamically start services as required:
//
//	var it *icingatesting.IT
//
//	func TestMain(m *testing.M) {
//		it = icingatesting.NewIT()
//		defer it.Cleanup()
//
//		m.Run()
//	}
type IT struct {
	mutex           sync.Mutex
	deferredCleanup []func()
	prefix          string
	dockerClient    *client.Client
	dockerNetworkId string
	mysql           mysql.Creator
	postgresql      postgresql.Creator
	redis           redis.Creator
	icinga2         icinga2.Creator
	icingaDb        icingadb.Creator
	logger          *zap.Logger
	loggerDebugCore zapcore.Core
}

var flagDebugLog = flag.String("icingatesting.debuglog", "", "file to write debug log to")

// NewIT allocates a new IT instance and initializes it.
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

	if n, err := it.dockerClient.NetworkCreate(context.Background(), it.prefix, types.NetworkCreate{Labels: map[string]string{"icinga": "testing"}}); err != nil {
		it.logger.Fatal("failed to create docker network", zap.String("network-name", it.prefix), zap.Error(err))
	} else {
		it.logger.Debug("created docker network", zap.String("network-name", it.prefix), zap.String("network-id", n.ID))
		it.dockerNetworkId = n.ID
		it.deferCleanup(func() {
			if err := it.dockerClient.NetworkRemove(context.Background(), it.dockerNetworkId); err != nil {
				it.logger.Error("failed to remove docker network",
					zap.String("network-name", it.prefix), zap.String("network-id", n.ID), zap.Error(err))
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
			panic(fmt.Errorf("failed to open debug log %q: %w", *flagDebugLog, err))
		}
		it.loggerDebugCore = zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewDevelopmentEncoderConfig()),
			w, zapcore.DebugLevel)
		cores = append(cores, it.loggerDebugCore)
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

// Cleanup tears down everything that was started during the tests by this IT instance.
func (it *IT) Cleanup() {
	it.mutex.Lock()
	defer it.mutex.Unlock()

	for i := len(it.deferredCleanup) - 1; i >= 0; i-- {
		f := it.deferredCleanup[i]
		f()
	}
}

func (it *IT) getMysqlServer() mysql.Creator {
	it.mutex.Lock()
	defer it.mutex.Unlock()

	if it.mysql == nil {
		it.mysql = mysql.NewDockerCreator(it.logger, it.dockerClient, it.prefix+"-mysql", it.dockerNetworkId)
		it.deferCleanup(it.mysql.Cleanup)
	}

	return it.mysql
}

func (it *IT) getPostgresqlServer() postgresql.Creator {
	it.mutex.Lock()
	defer it.mutex.Unlock()

	if it.postgresql == nil {
		it.postgresql = postgresql.NewDockerCreator(it.logger, it.dockerClient,
			it.prefix+"-postgresql", it.dockerNetworkId)
		it.deferCleanup(it.postgresql.Cleanup)
	}

	return it.postgresql
}

// MysqlDatabase creates a new MySQL database and a user to access it.
//
// The IT object will start a single MySQL Docker container on demand using the mysql:latest image and then creates
// multiple databases in it.
func (it *IT) MysqlDatabase() services.MysqlDatabase {
	return services.MysqlDatabase{MysqlDatabaseBase: it.getMysqlServer().CreateMysqlDatabase()}
}

// MysqlDatabaseT creates a new MySQL database and registers its cleanup function with testing.T.
func (it *IT) MysqlDatabaseT(t testing.TB) services.MysqlDatabase {
	m := it.MysqlDatabase()
	t.Cleanup(m.Cleanup)
	return m
}

// PostgresqlDatabase creates a new PostgreSQL database and a user to access it.
//
// The IT object will start a single PostgreSQL Docker container on demand using the postgres:latest image and then
// creates multiple databases in it.
func (it *IT) PostgresqlDatabase() services.PostgresqlDatabase {
	return services.PostgresqlDatabase{PostgresqlDatabaseBase: it.getPostgresqlServer().CreatePostgresqlDatabase()}
}

// PostgresDatabaseT creates a new MySQL database and registers its cleanup function with testing.T.
func (it *IT) PostgresqlDatabaseT(t testing.TB) services.PostgresqlDatabase {
	p := it.PostgresqlDatabase()
	t.Cleanup(p.Cleanup)
	return p
}

func (it *IT) getRedis() redis.Creator {
	it.mutex.Lock()
	defer it.mutex.Unlock()

	if it.redis == nil {
		it.redis = redis.NewDockerCreator(it.logger, it.dockerClient, it.prefix+"-redis", it.dockerNetworkId)
		it.deferCleanup(it.redis.Cleanup)
	}

	return it.redis
}

// RedisServer creates a new Redis server.
//
// Each call to this function will spawn a dedicated Redis Docker container using the redis:latest image.
func (it *IT) RedisServer() services.RedisServer {
	return services.RedisServer{RedisServerBase: it.getRedis().CreateRedisServer()}
}

// RedisServerT creates a new Redis server and registers its cleanup function with testing.T.
func (it *IT) RedisServerT(t testing.TB) services.RedisServer {
	r := it.RedisServer()
	t.Cleanup(r.Cleanup)
	return r
}

func (it *IT) getIcinga2() icinga2.Creator {
	it.mutex.Lock()
	defer it.mutex.Unlock()

	if it.icinga2 == nil {
		it.icinga2 = icinga2.NewDockerCreator(it.logger, it.dockerClient, it.prefix+"-icinga2", it.dockerNetworkId)
		it.deferCleanup(it.icinga2.Cleanup)
	}

	return it.icinga2
}

// Icinga2Node creates a new Icinga 2 node.
//
// Each call to this function will spawn a dedicated Icinga 2 Docker container using the icinga/icinga2:edge image.
func (it *IT) Icinga2Node(name string) services.Icinga2 {
	return services.Icinga2{Icinga2Base: it.getIcinga2().CreateIcinga2(name)}
}

// Icinga2NodeT creates a new Icinga 2 node and registers its cleanup function with testing.T.
func (it *IT) Icinga2NodeT(t testing.TB, name string) services.Icinga2 {
	n := it.Icinga2Node(name)
	t.Cleanup(n.Cleanup)
	return n
}

func (it *IT) getIcingaDb() icingadb.Creator {
	key := "ICINGA_TESTING_ICINGADB_BINARY"
	path, ok := os.LookupEnv(key)
	if !ok {
		panic(fmt.Errorf("environment variable %s must be set", key))
	}

	it.mutex.Lock()
	defer it.mutex.Unlock()

	if it.icingaDb == nil {
		it.icingaDb = icingadb.NewDockerBinaryCreator(it.logger, it.dockerClient, it.prefix+"-icingadb",
			it.dockerNetworkId, path)
		it.deferCleanup(it.icingaDb.Cleanup)
	}

	return it.icingaDb
}

// IcingaDbInstance starts a new Icinga DB instance.
//
// It expects the ICINGA_TESTING_ICINGADB_BINARY environment variable to be set to the path of a precompiled icingadb
// binary which is then started in a new Docker container when this function is called.
func (it *IT) IcingaDbInstance(redis services.RedisServer, rdb services.RelationalDatabase, options ...services.IcingaDbOption) services.IcingaDb {
	return services.IcingaDb{IcingaDbBase: it.getIcingaDb().CreateIcingaDb(redis, rdb, options...)}
}

// IcingaDbInstanceT creates a new Icinga DB instance and registers its cleanup function with testing.T.
func (it *IT) IcingaDbInstanceT(
	t testing.TB, redis services.RedisServer, rdb services.RelationalDatabase, options ...services.IcingaDbOption,
) services.IcingaDb {
	i := it.IcingaDbInstance(redis, rdb, options...)
	t.Cleanup(i.Cleanup)
	return i
}

// Logger returns a *zap.Logger which additionally logs the current test case name.
func (it *IT) Logger(t testing.TB) *zap.Logger {
	cores := []zapcore.Core{zaptest.NewLogger(t, zaptest.WrapOptions(zap.IncreaseLevel(zap.InfoLevel))).Core()}
	if it.loggerDebugCore != nil {
		cores = append(cores, it.loggerDebugCore)
	}
	return zap.New(zapcore.NewTee(cores...)).With(zap.String("testcase", t.Name()))
}
