package icingadb

import (
	_ "embed"
	"github.com/icinga/icinga-testing/services"
)

type Creator interface {
	CreateIcingaDb(redis services.RedisServerBase, mysql services.RelationalDatabase) services.IcingaDbBase
	Cleanup()
}

// info provides a partial implementation of the services.IcingaDbBase interface.
type info struct {
	redis services.RedisServerBase
	rdb   services.RelationalDatabase
}

func (i *info) Redis() services.RedisServerBase {
	return i.redis
}

func (i *info) RelationalDatabase() services.RelationalDatabase {
	return i.rdb
}
