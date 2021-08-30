package services

import "github.com/icinga/icinga-testing/services"

type Redis interface {
	Server() services.RedisServerBase
	Cleanup()
}
