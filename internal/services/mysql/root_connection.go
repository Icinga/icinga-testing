package mysql

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"github.com/icinga/icinga-testing/services"
	"github.com/icinga/icinga-testing/utils"
	"sync/atomic"
)

type rootConnection struct {
	host         string
	port         string
	rootUsername string
	rootPassword string
	db           *sql.DB
	counter      uint32
}

func newRootConnection(host string, port string, rootUsername string, rootPassword string) *rootConnection {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/information_schema", rootUsername, rootPassword, host, port)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}
	return &rootConnection{
		host:         host,
		port:         port,
		rootUsername: rootUsername,
		rootPassword: rootPassword,
		db:           db,
	}
}

func (m *rootConnection) CreateMysqlDatabase() services.MysqlDatabaseBase {
	id := atomic.AddUint32(&m.counter, 1)
	username := fmt.Sprintf("u%d", id)
	password := utils.RandomString(16)
	database := fmt.Sprintf("d%d", id)

	// I'm sorry for making the following three queries look like they are prone to SQL-injections, but it seems like
	// MySQL does not support prepared statements for these queries. The values are not user-controlled, so it's fine.
	_, err := m.db.Exec(fmt.Sprintf("CREATE USER %s IDENTIFIED BY '%s'", username, password))
	if err != nil {
		panic(err)
	}
	_, err = m.db.Exec(fmt.Sprintf("CREATE DATABASE %s", database))
	if err != nil {
		panic(err)
	}
	_, err = m.db.Exec(fmt.Sprintf("GRANT ALL PRIVILEGES ON %s.* TO %s", database, username))
	if err != nil {
		panic(err)
	}
	_, err = m.db.Exec(fmt.Sprintf("GRANT SESSION_VARIABLES_ADMIN ON *.* TO %s", username))
	if err != nil {
		// SESSION_VARIABLES_ADMIN is only needed and supported on MySQL 8+, others return a syntax error (1064).
		var mysqlError *mysql.MySQLError
		if !errors.As(err, &mysqlError) || mysqlError.Number != 1064 {
			panic(err)
		}
	}

	return &rootConnectionDatabase{
		info: info{
			host:     m.host,
			port:     m.port,
			username: username,
			password: password,
			database: database,
		},
		server: m,
	}
}

func (m *rootConnection) rootConnection() (*sql.DB, error) {
	d := mysqlDatabaseNopCleanup{info{
		host:     m.host,
		port:     m.port,
		username: m.rootUsername,
		password: m.rootPassword,
		database: "information_schema",
	}}
	return services.MysqlDatabase{MysqlDatabaseBase: &d}.Open()
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
