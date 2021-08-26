package services

import (
	"fmt"
	"github.com/go-redis/redis/v8"
)

type RedisServer interface {
	// Host returns the host for connecting to this Redis server.
	Host() string

	// Port returns the port for connecting to this Redis server.
	Port() string

	// Cleanup stops and removes this Redis server.
	Cleanup()
}

type redisServerInfo struct {
	host string
	port string
}

func (r *redisServerInfo) Host() string {
	return r.host
}

func (r *redisServerInfo) Port() string {
	return r.port
}

func RedisServerAddress(r RedisServer) string {
	return fmt.Sprintf("%s:%s", r.Host(), r.Port())
}

func RedisServerOpen(r RedisServer) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr: RedisServerAddress(r),
	})
}
