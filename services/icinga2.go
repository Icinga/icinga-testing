package services

type Icinga2 interface {
	Node(name string) Icinga2Node
	Cleanup()
}

type Icinga2Node interface {
	Host() string
	Port() string
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
