package mysql

import (
	_ "github.com/go-sql-driver/mysql"
	"github.com/icinga/icinga-testing/services"
)

type Creator interface {
	CreateMysqlDatabase() services.MysqlDatabaseBase
	Cleanup()
}

// info serves as a base for implementing the MysqlDatabaseBase interface. Another struct can embed it and
// initialize it with values to implement all interface functions except Cleanup.
type info struct {
	host     string
	port     string
	username string
	password string
	database string
}

func (m *info) Host() string {
	return m.host
}

func (m *info) Port() string {
	return m.port
}

func (m *info) Username() string {
	return m.username
}

func (m *info) Password() string {
	return m.password
}

func (m *info) Database() string {
	return m.database
}

type mysqlDatabaseNopCleanup struct {
	info
}

func (_ *mysqlDatabaseNopCleanup) Cleanup() {}

var _ services.MysqlDatabaseBase = (*mysqlDatabaseNopCleanup)(nil)
