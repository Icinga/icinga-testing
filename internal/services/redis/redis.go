package redis

import "github.com/icinga/icinga-testing/services"

type Creator interface {
	CreateRedisServer() services.RedisServerBase
	Cleanup()
}

// info provides a partial implementation of the services.RedisServerBase interface.
type info struct {
	host string
	port string
}

func (r *info) Host() string {
	return r.host
}

func (r *info) Port() string {
	return r.port
}
