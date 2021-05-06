package services

type Redis interface {
	Server() RedisServer
	Cleanup()
}
