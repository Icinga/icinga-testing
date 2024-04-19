package services

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/icinga/icinga-testing/internal"
	"github.com/icinga/icinga-testing/utils"
	"net/http"
	"text/template"
	"time"
)

type Icinga2Base interface {
	// Host returns the host on which the Icinga 2 API can be reached.
	Host() string

	// Port return the port on which the Icinga 2 API can be reached.
	Port() string

	// TriggerReload sends a reload signal to the Icinga 2 node.
	TriggerReload()

	// WriteConfig writes a config file to the file system of the Icinga 2 node.
	//
	// Example usage:
	//
	//   i.WriteConfig("etc/icinga2/conf.d/api-users.conf", []byte("var answer = 42"))
	WriteConfig(file string, data []byte)

	// DeleteConfigGlob deletes all configs file matching a glob from the file system of the Icinga 2 node.
	//
	// Example usage:
	//
	//   i.DeleteConfigGlob("etc/icinga2/zones.d/test/*.conf")
	DeleteConfigGlob(glob string)

	// EnableIcingaDb enables the icingadb feature on this node using the connection details of redis.
	EnableIcingaDb(redis RedisServerBase)

	// EnableIcingaNotifications enables the Icinga Notifications integration with the custom configuration.
	EnableIcingaNotifications(IcingaNotificationsBase)

	// Cleanup stops the node and removes everything that was created to start this node.
	Cleanup()
}

// Icinga2 wraps the Icinga2Base interface and adds some more helper functions.
type Icinga2 struct {
	Icinga2Base
}

func (i Icinga2) ApiClient() *utils.Icinga2Client {
	return utils.NewIcinga2Client(i.Host()+":"+i.Port(),
		internal.Icinga2DefaultUsername,
		internal.Icinga2DefaultPassword)
}

// Reload sends a reload signal to icinga2 and waits for the new config to become active.
func (i Icinga2) Reload() error {
	variable := "IcingaTestingStartupId"
	startupId := utils.RandomString(32)
	i.WriteConfig("etc/icinga2/conf.d/icinga-testing-startup-id.conf",
		[]byte(fmt.Sprintf("const %s = %q", variable, startupId)))

	i.TriggerReload()

	c := i.ApiClient()
	timeout := time.NewTimer(20 * time.Second)
	defer timeout.Stop()
	interval := time.NewTicker(100 * time.Millisecond)
	defer interval.Stop()

	var err error
	for {
		select {
		case <-interval.C:
			var res *http.Response
			res, err = c.GetJson("/v1/variables/" + variable)
			if err != nil {
				continue
			}
			if res.StatusCode != http.StatusOK {
				err = fmt.Errorf("icinga2 responded with HTTP %s", res.Status)
				continue
			}
			var data struct {
				Results []struct {
					Value string `json:"value"`
				} `json:"results"`
			}
			err = json.NewDecoder(res.Body).Decode(&data)
			if err != nil {
				continue
			}
			if len(data.Results) > 0 && data.Results[0].Value == startupId {
				// New configuration is loaded.
				return nil
			} else {
				err = errors.New("icinga2 is still using an old configuration")
			}
		case <-timeout.C:
			return fmt.Errorf("icinga2 did not reload with new config in time: %w", err)
		}
	}
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
	i.WriteConfig(fmt.Sprintf("etc/icinga2/features-enabled/icingadb_%s_%s.conf", r.Host(), r.Port()), b.Bytes())
}

//go:embed icinga2_icinga_notifications.conf
var icinga2IcingaNotificationsConfRawTemplate string
var icinga2IcingaNotificationsConfTemplate = template.Must(template.New("icinga-notifications.conf").Parse(icinga2IcingaNotificationsConfRawTemplate))

func (i Icinga2) WriteIcingaNotificationsConf(notis IcingaNotificationsBase) {
	b := bytes.NewBuffer(nil)
	err := icinga2IcingaNotificationsConfTemplate.Execute(b, notis)
	if err != nil {
		panic(err)
	}
	i.WriteConfig("etc/icinga2/features-enabled/icinga_notifications.conf", b.Bytes())
}
