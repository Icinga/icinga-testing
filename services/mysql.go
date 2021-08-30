package services

import (
	"database/sql"
	"fmt"
	"github.com/go-sql-driver/mysql"
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

func (m MysqlDatabase) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", m.Username(), m.Password(), m.Host(), m.Port(), m.Database())
}

func (m MysqlDatabase) Open() (*sql.DB, error) {
	return sql.Open("mysql", m.DSN())
}

func (m MysqlDatabase) ImportIcingaDbSchema() {
	key := "ICINGA_TESTING_ICINGADB_SCHEMA"
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
				if mysqlErr, ok := err.(*mysql.MySQLError); ok && mysqlErr.Number == 1050 {
					// ignore 'table already exists' errors for now, probably the schema was already imported
					// TODO(jb): find a proper solution for this
					return
				} else {
					panic(err)
				}
			}
		}
	}
}
