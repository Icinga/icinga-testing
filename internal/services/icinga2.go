package services

import (
	_ "embed"
	"github.com/icinga/icinga-testing/services"
)

type Icinga2 interface {
	Node(name string) services.Icinga2Node
	Cleanup()
}

type icinga2NodeInfo struct {
	host string
	port string
}

func (r *icinga2NodeInfo) Host() string {
	return r.host
}

func (r *icinga2NodeInfo) Port() string {
	return r.port
}
