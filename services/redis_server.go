package services

import (
	"fmt"
	"github.com/go-redis/redis/v8"
)

type RedisServer interface {
	Host() string
	Port() string
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
