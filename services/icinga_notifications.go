package services

import (
	_ "embed"
	"io"
	"text/template"
)

type IcingaNotificationsBase interface {
	// Host returns the host on which Icinga Notification's listener can be reached.
	Host() string

	// Port return the port on which Icinga Notification's listener can be reached.
	Port() string

	// RelationalDatabase returns the instance information of the relational database this instance is using.
	RelationalDatabase() RelationalDatabase

	// Cleanup stops the instance and removes everything that was created to start it.
	Cleanup()
}

// IcingaNotifications wraps the IcingaNotificationsBase interface and adds some more helper functions.
type IcingaNotifications struct {
	IcingaNotificationsBase
	config string
}

//go:embed icinga_notifications.yml
var icingaNotificationsYmlRawTemplate string
var icingaNotificationsYmlTemplate = template.Must(template.New("icinga_notifications.yml").Parse(icingaNotificationsYmlRawTemplate))

func (i IcingaNotifications) WriteConfig(w io.Writer) error {
	return icingaNotificationsYmlTemplate.Execute(w, i)
}

// Config returns additional raw YAML configuration, if any.
func (i IcingaNotifications) Config() string {
	return i.config
}

// IcingaNotificationsOption configures IcingaNotifications.
type IcingaNotificationsOption func(*IcingaNotifications)

// WithIcingaNotificationsConfig sets additional raw YAML configuration.
func WithIcingaNotificationsConfig(config string) func(notifications *IcingaNotifications) {
	return func(db *IcingaNotifications) {
		db.config = config
	}
}
