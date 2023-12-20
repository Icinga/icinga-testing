package services

import (
	"database/sql"
	"fmt"
	icingasql "github.com/Icinga/go-libs/sql"
	"os"
)

type MysqlDatabaseBase interface {
	// Host returns the host for connecting to this database.
	Host() string

	// Port returns the port for connecting to this database.
	Port() string

	// Username returns the username for connecting to this database.
	Username() string

	// Password returns the password for connecting to the database.
	Password() string

	// Database returns the name of the database on the MySQL server.
	Database() string

	// Cleanup removes the MySQL database.
	Cleanup()
}

// MysqlDatabase wraps the MysqlDatabaseBase interface and adds some helper functions.
type MysqlDatabase struct {
	MysqlDatabaseBase
}

var _ RelationalDatabase = MysqlDatabase{}

func (m MysqlDatabase) IcingaDbType() string {
	return "mysql"
}

func (m MysqlDatabase) Driver() string {
	return "mysql"
}

func (m MysqlDatabase) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?sql_mode=ANSI_QUOTES", m.Username(), m.Password(), m.Host(), m.Port(), m.Database())
}

func (m MysqlDatabase) Open() (*sql.DB, error) {
	return sql.Open(m.Driver(), m.DSN())
}

func (m MysqlDatabase) ImportIcingaDbSchema() {
	key := "ICINGA_TESTING_ICINGADB_SCHEMA_MYSQL"
	schemaFile, ok := os.LookupEnv(key)
	if !ok {
		panic(fmt.Errorf("environment variable %s must be set", key))
	}

	schema, err := os.ReadFile(schemaFile)
	if err != nil {
		panic(fmt.Errorf("failed to read icingadb schema file %q: %w", schemaFile, err))
	}

	db, err := MysqlDatabase{m}.Open()
	if err != nil {
		panic(err)
	}

	for _, stmt := range icingasql.MysqlSplitStatements(string(schema)) {
		if _, err := db.Exec(stmt); err != nil {
			panic(err)
		}
	}
}

func (m MysqlDatabase) ImportIcingaNotificationsSchema() {
	panic("icinga-notifications does not support MySQL yet")
}
