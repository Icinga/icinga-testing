package services

type RelationalDatabase interface {
	// Host returns the host for connecting to this database.
	Host() string

	// Port returns the port for connecting to this database.
	Port() string

	// Username returns the username for connecting to this database.
	Username() string

	// Password returns the password for connecting to the database.
	Password() string

	// Database returns the name of the database.
	Database() string

	// IcingaDbType returns the database type for use in config.yml for Icinga DB.
	IcingaDbType() string

	// Driver returns the sql driver name to connect to this database from Go.
	Driver() string

	// DSN returns the data source name (DSN) to connect to this database from Go.
	DSN() string

	// ImportIcingaDbSchema imports the Icinga DB schema into this database.
	ImportIcingaDbSchema()

	// ImportIcingaNotificationsSchema imports the Icinga Notifications schema into this database.
	ImportIcingaNotificationsSchema()

	// Cleanup removes the database.
	Cleanup()
}
