package services

import (
	_ "embed"
	"github.com/icinga/icinga-testing/services"
)

type IcingaDb interface {
	Instance(redis services.RedisServer, mysql services.MysqlDatabase) services.IcingaDbInstance
	Cleanup()
}

type icingaDbInstanceInfo struct {
	redis services.RedisServer
	mysql services.MysqlDatabase
}

func (i *icingaDbInstanceInfo) Redis() services.RedisServer {
	return i.redis
}

func (i *icingaDbInstanceInfo) Mysql() services.MysqlDatabase {
	return i.mysql
}
