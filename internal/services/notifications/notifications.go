package notifications

import (
	"github.com/icinga/icinga-testing/services"
)

// defaultPort of the Icinga Notifications Web Listener.
const defaultPort string = "5680"

type Creator interface {
	CreateIcingaNotifications(rdb services.RelationalDatabase, options ...services.IcingaNotificationsOption) services.IcingaNotificationsBase
	Cleanup()
}

// info provides a partial implementation of the services.IcingaNotificationsBase interface.
type info struct {
	host string
	port string

	rdb services.RelationalDatabase
}

func (i *info) Host() string {
	return i.host
}

func (i *info) Port() string {
	return i.port
}

func (i *info) RelationalDatabase() services.RelationalDatabase {
	return i.rdb
}
