package services

import (
	"database/sql"
	"fmt"
)

type MysqlDatabase interface {
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

func MysqlDatabaseDSN(m MysqlDatabase) string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", m.Username(), m.Password(), m.Host(), m.Port(), m.Database())
}

func MysqlDatabaseOpen(m MysqlDatabase) (*sql.DB, error) {
	return sql.Open("mysql", MysqlDatabaseDSN(m))
}
