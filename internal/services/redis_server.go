package services

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
