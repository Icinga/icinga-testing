package services

import (
	_ "embed"
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
	Environ map[string]string
}

// ConfEnviron returns configuration environment variables.
func (i IcingaNotifications) ConfEnviron() []string {
	envs := make([]string, 0, len(i.Environ))
	for k, v := range i.Environ {
		envs = append(envs, k+"="+v)
	}
	return envs
}

// IcingaNotificationsOption configures IcingaNotifications.
type IcingaNotificationsOption func(*IcingaNotifications)

// WithIcingaNotificationsDefaultsEnvConfig populates the configuration environment variables with useful defaults.
//
// This will always be applied before any other IcingaNotificationsOption.
func WithIcingaNotificationsDefaultsEnvConfig(rdb RelationalDatabase, listenAddr string) IcingaNotificationsOption {
	return func(notifications *IcingaNotifications) {
		if notifications.Environ == nil {
			notifications.Environ = make(map[string]string)
		}

		notifications.Environ["ICINGA_NOTIFICATIONS_LISTEN"] = listenAddr
		notifications.Environ["ICINGA_NOTIFICATIONS_CHANNEL-PLUGIN-DIR"] = "/usr/libexec/icinga-notifications/channel"
		notifications.Environ["ICINGA_NOTIFICATIONS_DATABASE_TYPE"] = rdb.IcingaDbType()
		notifications.Environ["ICINGA_NOTIFICATIONS_DATABASE_HOST"] = rdb.Host()
		notifications.Environ["ICINGA_NOTIFICATIONS_DATABASE_PORT"] = rdb.Port()
		notifications.Environ["ICINGA_NOTIFICATIONS_DATABASE_DATABASE"] = rdb.Database()
		notifications.Environ["ICINGA_NOTIFICATIONS_DATABASE_USER"] = rdb.Username()
		notifications.Environ["ICINGA_NOTIFICATIONS_DATABASE_PASSWORD"] = rdb.Password()
		notifications.Environ["ICINGA_NOTIFICATIONS_LOGGING_LEVEL"] = "debug"
	}
}

// WithIcingaNotificationsEnvConfig sets an environment variable configuration for icinga-notifications.
func WithIcingaNotificationsEnvConfig(key, value string) IcingaNotificationsOption {
	return func(notifications *IcingaNotifications) {
		if notifications.Environ == nil {
			notifications.Environ = make(map[string]string)
		}

		notifications.Environ[key] = value
	}
}
