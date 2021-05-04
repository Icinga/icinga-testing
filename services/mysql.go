package services

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/icinga/icinga-testing/utils"
	"sync/atomic"
)

type MySQL struct {
	host         string
	port         string
	rootUsername string
	rootPassword string
	username     string
	password     string
	database     string
}

// Counter for usernames (e.g. u42) and databases (e.g. d42)
var mysqlId uint32

func NewMySQL() *MySQL {
	return &MySQL{
		host:         utils.GetEnvDefault("ICINGADB_TESTING_MYSQL_HOST", "localhost"),
		port:         utils.GetEnvDefault("ICINGADB_TESTING_MYSQL_PORT", "3306"),
		rootUsername: utils.GetEnvDefault("ICINGADB_TESTING_MYSQL_ROOT_USERNAME", "root"),
		rootPassword: utils.GetEnvDefault("ICINGADB_TESTING_MYSQL_ROOT_PASSWORD", "root"),
	}
}

func (m *MySQL) Start() error {
	id := atomic.AddUint32(&mysqlId, 1)
	m.username = fmt.Sprintf("u%d", id)
	m.password = utils.RandomString(16)
	m.database = fmt.Sprintf("d%d", id)

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/information_schema", m.rootUsername, m.rootPassword, m.Host(), m.Port())
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	// I'm sorry for making the following three queries look like they are prone to SQL-injections, but it seems like
	// MySQL does not support prepared statements for these queries. The values are not user-controlled, so it's fine.
	_, err = db.Exec(fmt.Sprintf("CREATE USER %s IDENTIFIED WITH mysql_native_password BY '%s'", m.username, m.password))
	if err != nil {
		return err
	}
	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", m.database))
	if err != nil {
		return err
	}
	_, err = db.Exec(fmt.Sprintf("GRANT ALL PRIVILEGES ON %s.* TO %s", m.database, m.username))
	if err != nil {
		return err
	}

	return nil
}

func (m *MySQL) MustStart() {
	if err := m.Start(); err != nil {
		panic(err)
	}
}

func (m *MySQL) Stop() error {
	// TODO(jb): remove user and database
	return nil
}

func (m *MySQL) MustStop() {
	if err := m.Stop(); err != nil {
		panic(err)
	}
}

func (m *MySQL) Host() string {
	return m.host
}

func (m *MySQL) Port() string {
	return m.port
}

func (m *MySQL) Username() string {
	return m.username
}

func (m *MySQL) Password() string {
	return m.password
}

func (m *MySQL) Database() string {
	return m.database
}

func (m *MySQL) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", m.Username(), m.Password(), m.Host(), m.Port(), m.Database())
}

func (m *MySQL) Open() (*sql.DB, error) {
	return sql.Open("mysql", m.DSN())
}
