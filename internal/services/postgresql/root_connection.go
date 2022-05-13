package postgresql

import (
	"database/sql"
	"fmt"
	"github.com/icinga/icinga-testing/services"
	"github.com/icinga/icinga-testing/utils"
	_ "github.com/lib/pq"
	"sync/atomic"
)

type rootConnection struct {
	host     string
	port     string
	username string
	password string
	counter  uint32
}

func newRootConnection(host string, port string, rootUsername string, rootPassword string) *rootConnection {
	return &rootConnection{
		host:     host,
		port:     port,
		username: rootUsername,
		password: rootPassword,
	}
}

func (c *rootConnection) CreatePostgresqlDatabase() services.PostgresqlDatabaseBase {
	id := atomic.AddUint32(&c.counter, 1)
	username := fmt.Sprintf("u%d", id)
	password := utils.RandomString(16)
	database := fmt.Sprintf("d%d", id)

	db, err := c.openAsRoot("postgres")
	defer func() { _ = db.Close() }()

	// I'm sorry for making the following queries look like they are prone to SQL-injections, but it seems like
	// PostgreSQL does not support prepared statements for these queries. The values are not user-controlled, so it's
	// fine.
	_, err = db.Exec(fmt.Sprintf("CREATE USER %s WITH PASSWORD '%s'", username, password))
	if err != nil {
		panic(err)
	}
	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s WITH OWNER %s", database, username))
	if err != nil {
		panic(err)
	}

	// The citext extension is required by Icinga DB.
	err = c.createExtension(database, "citext")
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

func (c *rootConnection) openAsRoot(database string) (*sql.DB, error) {
	d := postgresqlDatabaseNopCleanup{info{
		host:     c.host,
		port:     c.port,
		username: c.username,
		password: c.password,
		database: database,
	}}
	return services.PostgresqlDatabase{PostgresqlDatabaseBase: &d}.Open()
}

func (c *rootConnection) createExtension(database string, extension string) error {
	userDb, err := c.openAsRoot(database)
	if err != nil {
		return err
	}
	defer func() { _ = userDb.Close() }()

	_, err = userDb.Exec("CREATE EXTENSION IF NOT EXISTS " + extension)
	return err
}

type rootConnectionDatabase struct {
	info
	server *rootConnection
}

func (d *rootConnectionDatabase) Cleanup() {
	db, err := d.server.openAsRoot("postgres")
	if err != nil {
		panic(err)
	}
	defer func() { _ = db.Close() }()

	var serverVersion int
	err = db.QueryRow("SHOW server_version_num").Scan(&serverVersion)
	if err != nil {
		panic(err)
	}

	// Support for `WITH (FORCE)` was only added in PostgreSQL 13. For older versions,
	// open connections have to be terminated explicitly using another query.
	if serverVersion >= 130000 {
		_, err := db.Exec(fmt.Sprintf("DROP DATABASE %s WITH (FORCE)", d.database))
		if err != nil {
			panic(err)
		}
	} else {
		_, err := db.Exec("SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1", d.database)
		if err != nil {
			panic(err)
		}
		_, err = db.Exec(fmt.Sprintf(`DROP DATABASE %s`, d.database))
		if err != nil {
			panic(err)
		}
	}

	_, err = db.Exec(fmt.Sprintf("DROP USER %s", d.username))
	if err != nil {
		panic(err)
	}
}
