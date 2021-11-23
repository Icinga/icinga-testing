package services

import (
	"database/sql"
	"fmt"
	"os"
	"regexp"
	"strings"
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
	// duplicated from https://github.com/Icinga/docker-icingadb/blob/master/entrypoint/main.go
	sqlComment := regexp.MustCompile(`(?m)^--.*`)
	sqlStmtSep := regexp.MustCompile(`(?m);$`)
	for _, ddl := range sqlStmtSep.Split(string(sqlComment.ReplaceAll(schema, nil)), -1) {
		if ddl = strings.TrimSpace(ddl); ddl != "" {
			if _, err := db.Exec(ddl); err != nil {
				panic(err)
			}
		}
	}
}
