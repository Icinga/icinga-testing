package services

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/icinga/icinga-testing/utils"
	"sync"
)

type Redis struct {
	host    string
	port    string
	started bool
}

// Currently there is no support for multiple Redis databases.
// Just use one server with one database for now and use it exclusively.
var redisMutex sync.Mutex

func NewRedis() *Redis {
	return &Redis{
		host: utils.GetEnvDefault("ICINGADB_TESTING_REDIS_HOST", "localhost"),
		port: utils.GetEnvDefault("ICINGADB_TESTING_REDIS_PORT", "6379"),
	}
}

func (r *Redis) Start() error {
	redisMutex.Lock()
	r.started = true
	return r.flush()
}

func (r *Redis) MustStart() {
	if err := r.Start(); err != nil {
		panic(err)
	}
}

func (r *Redis) Stop() error {
	if r.started {
		r.started = false
		redisMutex.Unlock()
	}
	return nil
}

func (r *Redis) MustStop() {
	if err := r.Stop(); err != nil {
		panic(err)
	}
}

func (r *Redis) Host() string {
	return r.host
}

func (r *Redis) Port() string {
	return r.port
}

func (r *Redis) Address() string {
	return fmt.Sprintf("%s:%s", r.Host(), r.Port())
}

func (r *Redis) flush() error {
	c := redis.NewClient(&redis.Options{
		Addr: r.Address(),
	})
	_, err := c.FlushAll(context.Background()).Result()
	return err
}
