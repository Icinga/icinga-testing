package services

import (
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"os"
	"regexp"
	"strings"
)

type PostgresqlDatabaseBase interface {
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
type PostgresqlDatabase struct {
	PostgresqlDatabaseBase
}

func (p PostgresqlDatabase) DSN() string {
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(p.Username(), p.Password()),
		Host:     net.JoinHostPort(p.Host(), p.Port()),
		Path:     "/" + url.PathEscape(p.Database()),
		RawQuery: "sslmode=disable",
	}

	return u.String()
}

func (p PostgresqlDatabase) Open() (*sql.DB, error) {
	return sql.Open("postgres", p.DSN())
}

func (p PostgresqlDatabase) ImportIcingaDbSchema() {
	key := "ICINGA_TESTING_ICINGADB_SCHEMA_POSTGRESQL"
	schemaFile, ok := os.LookupEnv(key)
	if !ok {
		panic(fmt.Errorf("environment variable %s must be set", key))
	}

	schema, err := os.ReadFile(schemaFile)
	if err != nil {
		panic(fmt.Errorf("failed to read icingadb schema file %q: %w", schemaFile, err))
	}

	db, err := PostgresqlDatabase{PostgresqlDatabaseBase: p}.Open()
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
