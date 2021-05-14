package services

import (
	"bytes"
	_ "embed"
	"fmt"
	"github.com/icinga/icinga-testing/utils"
	"text/template"
)

type Icinga2 interface {
	Node(name string) Icinga2Node
	Cleanup()
}

type Icinga2Node interface {
	Host() string
	Port() string
	Reload()
	WriteConfig(file string, data []byte)
	EnableIcingaDb(redis RedisServer)
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

func Icinga2NodeApiClient(n Icinga2Node) *utils.Icinga2Client {
	// TODO: API credentials
	return utils.NewIcinga2Client(n.Host()+":"+n.Port(), "root", "root")
}

// Icinga2NodePing tries to connect to the API port of an Icinga 2 instance to see if it is running.
func Icinga2NodePing(n Icinga2Node) error {
	response, err := Icinga2NodeApiClient(n).Get("/")
	if err != nil {
		return err
	}
	if response.StatusCode != 401 {
		return fmt.Errorf("received unexpected status code %d (expected 401)", response.StatusCode)
	}
	return nil
}

//go:embed icinga2_icingadb.conf
var icinga2IcingaDbConfRawTemplate string
var icinga2IcingaDbConfTemplate = template.Must(template.New("icingadb.conf").Parse(icinga2IcingaDbConfRawTemplate))

func Icinga2NodeWriteIcingaDbConf(n Icinga2Node, r RedisServer) {
	b := bytes.NewBuffer(nil)
	err := icinga2IcingaDbConfTemplate.Execute(b, r)
	if err != nil {
		panic(err)
	}
	n.WriteConfig("etc/icinga2/features-available/icingadb.conf", b.Bytes())
}
