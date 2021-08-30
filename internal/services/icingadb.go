package services

import (
	_ "embed"
	"github.com/icinga/icinga-testing/services"
)

type IcingaDb interface {
	Instance(redis services.RedisServerBase, mysql services.MysqlDatabaseBase) services.IcingaDbBase
	Cleanup()
}

type icingaDbInstanceInfo struct {
	redis services.RedisServerBase
	mysql services.MysqlDatabaseBase
}

func (i *icingaDbInstanceInfo) Redis() services.RedisServerBase {
	return i.redis
}

func (i *icingaDbInstanceInfo) Mysql() services.MysqlDatabaseBase {
	return i.mysql
}
