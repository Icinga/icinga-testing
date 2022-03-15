package postgresql

import (
	"github.com/icinga/icinga-testing/services"
)

type Creator interface {
	CreatePostgresqlDatabase() services.PostgresqlDatabaseBase
	Cleanup()
}

// info serves as a base for implementing the PostgresqlDatabaseBase interface. Another struct can embed it and
// initialize it with values to implement all interface functions except Cleanup.
type info struct {
	host     string
	port     string
	username string
	password string
	database string
}

func (p *info) Host() string {
	return p.host
}

func (p *info) Port() string {
	return p.port
}

func (p *info) Username() string {
	return p.username
}

func (p *info) Password() string {
	return p.password
}

func (p *info) Database() string {
	return p.database
}

type postgresqlDatabaseNopCleanup struct {
	info
}

func (_ *postgresqlDatabaseNopCleanup) Cleanup() {}

var _ services.PostgresqlDatabaseBase = (*postgresqlDatabaseNopCleanup)(nil)
