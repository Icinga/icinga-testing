package postgresql

import (
	"database/sql"
	"fmt"
	"github.com/icinga/icinga-testing/services"
	"github.com/icinga/icinga-testing/utils"
	_ "github.com/lib/pq"
	"net"
	"net/url"
	"sync/atomic"
)

type rootConnection struct {
	host     string
	port     string
	username string
	password string
	db       *sql.DB
	counter  uint32
}

func newRootConnection(host string, port string, rootUsername string, rootPassword string) *rootConnection {
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(rootUsername, rootPassword),
		Host:     net.JoinHostPort(host, port),
		Path:     "/postgres",
		RawQuery: "sslmode=disable",
	}

	db, err := sql.Open("postgres", u.String())
	if err != nil {
		panic(err)
	}
	return &rootConnection{
		host:     host,
		port:     port,
		username: rootUsername,
		password: rootPassword,
		db:       db,
	}
}

func (c *rootConnection) CreatePostgresqlDatabase() services.PostgresqlDatabaseBase {
	id := atomic.AddUint32(&c.counter, 1)
	username := fmt.Sprintf("u%d", id)
	password := utils.RandomString(16)
	database := fmt.Sprintf("d%d", id)

	// I'm sorry for making the following queries look like they are prone to SQL-injections, but it seems like
	// PostgreSQL does not support prepared statements for these queries. The values are not user-controlled, so it's
	// fine.
	_, err := c.db.Exec(fmt.Sprintf("CREATE USER %s WITH PASSWORD '%s'", username, password))
	if err != nil {
		panic(err)
	}
	_, err = c.db.Exec(fmt.Sprintf("CREATE DATABASE %s WITH OWNER %s", database, username))
	if err != nil {
		panic(err)
	}

	return &rootConnectionDatabase{
		info: info{
			host:     c.host,
			port:     c.port,
			username: username,
			password: password,
			database: database,
		},
		server: c,
	}
}

func (c *rootConnection) rootConnection() (*sql.DB, error) {
	d := postgresqlDatabaseNopCleanup{info{
		host:     c.host,
		port:     c.port,
		username: c.username,
		password: c.password,
		database: "postgres",
	}}
	return services.PostgresqlDatabase{PostgresqlDatabaseBase: &d}.Open()
}

type rootConnectionDatabase struct {
	info
	server *rootConnection
}

func (d *rootConnectionDatabase) Cleanup() {
	_, err := d.server.db.Exec(fmt.Sprintf("DROP DATABASE %s", d.database))
	if err != nil {
		panic(err)
	}
	_, err = d.server.db.Exec(fmt.Sprintf("DROP USER %s", d.username))
	if err != nil {
		panic(err)
	}
}
