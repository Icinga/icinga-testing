package icinga2

import (
	_ "embed"
	"github.com/icinga/icinga-testing/services"
)

type Creator interface {
	CreateIcinga2(name string) services.Icinga2Base
	Cleanup()
}

// info provides a partial implementation of the services.Icinga2Base interface.
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
