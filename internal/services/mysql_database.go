package services

import (
	_ "github.com/go-sql-driver/mysql"
	"github.com/icinga/icinga-testing/services"
)

// mysqlDatabaseInfo serves as a base for implementing the MysqlDatabaseBase interface. Another struct can embed it and
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

type mysqlDatabaseNopCleanup struct {
	mysqlDatabaseInfo
}

func (_ *mysqlDatabaseNopCleanup) Cleanup() {}

var _ services.MysqlDatabaseBase = (*mysqlDatabaseNopCleanup)(nil)
