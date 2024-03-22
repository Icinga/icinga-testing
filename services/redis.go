package services

import (
	"fmt"
	"github.com/redis/go-redis/v9"
)

type RedisServerBase interface {
	// Host returns the host for connecting to this Redis server.
	Host() string

	// Port returns the port for connecting to this Redis server.
	Port() string

	// Cleanup stops and removes this Redis server.
	Cleanup()
}

// RedisServer wraps the RedisServerBase interface and adds some helper functions.
type RedisServer struct {
	RedisServerBase
}

func (r RedisServer) Address() string {
	return fmt.Sprintf("%s:%s", r.Host(), r.Port())
}

func (r RedisServer) Open() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr: r.Address(),
	})
}
