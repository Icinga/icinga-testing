package services

import (
	"bytes"
	_ "embed"
	"fmt"
	"github.com/icinga/icinga-testing/utils"
	"text/template"
)

type Icinga2Base interface {
	// Host returns the host on which the Icinga 2 API can be reached.
	Host() string

	// Port return the port on which the Icinga 2 API can be reached.
	Port() string

	// Reload sends a reload signal to the Icinga 2 node.
	Reload()

	// WriteConfig writes a config file to the file system of the Icinga 2 node.
	//
	// Example usage:
	//
	//   i.WriteConfig("etc/icinga2/conf.d/api-users.conf", []byte("var answer = 42"))
	WriteConfig(file string, data []byte)

	// EnableIcingaDb enables the icingadb feature on this node using the connection details of redis.
	EnableIcingaDb(redis RedisServerBase)

	// Cleanup stops the node and removes everything that was created to start this node.
	Cleanup()
}

// Icinga2 wraps the Icinga2Base interface and adds some more helper functions.
type Icinga2 struct {
	Icinga2Base
}

func (i Icinga2) ApiClient() *utils.Icinga2Client {
	// TODO: API credentials
	return utils.NewIcinga2Client(i.Host()+":"+i.Port(), "root", "root")
}

// Ping tries to connect to the API port of an Icinga 2 instance to see if it is running.
func (i Icinga2) Ping() error {
	response, err := i.ApiClient().Get("/")
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

func (i Icinga2) WriteIcingaDbConf(r RedisServerBase) {
	b := bytes.NewBuffer(nil)
	err := icinga2IcingaDbConfTemplate.Execute(b, r)
	if err != nil {
		panic(err)
	}
	i.WriteConfig("etc/icinga2/features-available/icingadb.conf", b.Bytes())
}
