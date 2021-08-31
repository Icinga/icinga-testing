package icingadb

import (
	_ "embed"
	"github.com/icinga/icinga-testing/services"
)

type Creator interface {
	CreateIcingaDb(redis services.RedisServerBase, mysql services.MysqlDatabaseBase) services.IcingaDbBase
	Cleanup()
}

// info provides a partial implementation of the services.IcingaDbBase interface.
type info struct {
	redis services.RedisServerBase
	mysql services.MysqlDatabaseBase
}

func (i *info) Redis() services.RedisServerBase {
	return i.redis
}

func (i *info) Mysql() services.MysqlDatabaseBase {
	return i.mysql
}
