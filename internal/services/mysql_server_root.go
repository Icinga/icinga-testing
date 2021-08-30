package services

import (
	"database/sql"
	"fmt"
	"github.com/icinga/icinga-testing/services"
	"github.com/icinga/icinga-testing/utils"
	"sync/atomic"
)

type mysqlServerWithRootCreds struct {
	host         string
	port         string
	rootUsername string
	rootPassword string
	db           *sql.DB
	counter      uint32
}

func NewMysqlServerWithRootCreds(host string, port string, rootUsername string, rootPassword string) *mysqlServerWithRootCreds {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/information_schema", rootUsername, rootPassword, host, port)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}
	return &mysqlServerWithRootCreds{
		host:         host,
		port:         port,
		rootUsername: rootUsername,
		rootPassword: rootPassword,
		db:           db,
	}
}

func (m *mysqlServerWithRootCreds) Database() services.MysqlDatabase {
	id := atomic.AddUint32(&m.counter, 1)
	username := fmt.Sprintf("u%d", id)
	password := utils.RandomString(16)
	database := fmt.Sprintf("d%d", id)

	// I'm sorry for making the following three queries look like they are prone to SQL-injections, but itesting seems like
	// MySQL does not support prepared statements for these queries. The values are not user-controlled, so itesting's fine.
	_, err := m.db.Exec(fmt.Sprintf("CREATE USER %s IDENTIFIED WITH mysql_native_password BY '%s'", username, password))
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
		panic(err)
	}

	return &mysqlServerWithRootCredsDatabase{
		mysqlDatabaseInfo: mysqlDatabaseInfo{
			host:     m.host,
			port:     m.port,
			username: username,
			password: password,
			database: database,
		},
		server: m,
	}
}

type mysqlServerWithRootCredsDatabase struct {
	mysqlDatabaseInfo
	server *mysqlServerWithRootCreds
}

func (m *mysqlServerWithRootCreds) rootConnection() (*sql.DB, error) {
	d := mysqlDatabaseNopCleanup{mysqlDatabaseInfo{
		host:     m.host,
		port:     m.port,
		username: m.rootUsername,
		password: m.rootPassword,
		database: "information_schema",
	}}
	return services.MysqlDatabaseOpen(&d)
}

func (d *mysqlServerWithRootCredsDatabase) Cleanup() {
	_, err := d.server.db.Exec(fmt.Sprintf("DROP DATABASE %s", d.database))
	if err != nil {
		panic(err)
	}
	_, err = d.server.db.Exec(fmt.Sprintf("DROP USER %s", d.username))
	if err != nil {
		panic(err)
	}
}
