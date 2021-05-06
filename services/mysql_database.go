package services

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
)

type MysqlDatabase interface {
	Host() string
	Port() string
	Username() string
	Password() string
	Database() string
	Cleanup()
}

// mysqlDatabaseInfo serves as a base for implementing the MysqlDatabase interface. Another struct can embed it and
// initialize it with values to implement all interface functions except Cleanup.
type mysqlDatabaseInfo struct {
	host     string
	port     string
	username string
	password string
	database string
}

func (m *mysqlDatabaseInfo) Host() string {
	return m.host
}

func (m *mysqlDatabaseInfo) Port() string {
	return m.port
}

func (m *mysqlDatabaseInfo) Username() string {
	return m.username
}

func (m *mysqlDatabaseInfo) Password() string {
	return m.password
}

func (m *mysqlDatabaseInfo) Database() string {
	return m.database
}

func MysqlDatabaseDSN(m MysqlDatabase) string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", m.Username(), m.Password(), m.Host(), m.Port(), m.Database())
}

func MysqlDatabaseOpen(m MysqlDatabase) (*sql.DB, error) {
	return sql.Open("mysql", MysqlDatabaseDSN(m))
}

type mysqlDatabaseNopCleanup struct {
	mysqlDatabaseInfo
}

func (_ *mysqlDatabaseNopCleanup) Cleanup() {}

var _ MysqlDatabase = (*mysqlDatabaseNopCleanup)(nil)
