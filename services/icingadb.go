package services

import (
	_ "embed"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"io"
	"os"
	"regexp"
	"strings"
	"text/template"
)

type IcingaDb interface {
	Instance(redis RedisServer, mysql MysqlDatabase) IcingaDbInstance
	Cleanup()
}

type IcingaDbInstance interface {
	// Redis returns the instance information of the Redis server this instance is using.
	Redis() RedisServer

	// Mysql returns the instance information of the MySQL database this instance is using.
	Mysql() MysqlDatabase

	// Cleanup stops the instance and removes everything that was created to start it.
	Cleanup()
}

type icingaDbInstanceInfo struct {
	redis RedisServer
	mysql MysqlDatabase
}

func (i *icingaDbInstanceInfo) Redis() RedisServer {
	return i.redis
}

func (i *icingaDbInstanceInfo) Mysql() MysqlDatabase {
	return i.mysql
}

//go:embed icingadb.yml
var icingadbYmlRawTemplate string
var icingadbYmlTemplate = template.Must(template.New("icingadb.yml").Parse(icingadbYmlRawTemplate))

func IcingaDbInstanceWriteConfig(i IcingaDbInstance, w io.Writer) error {
	return icingadbYmlTemplate.Execute(w, i)
}

func IcingaDbInstanceImportSchema(m MysqlDatabase) {
	key := "ICINGA_TESTING_ICINGADB_SCHEMA"
	schemaFile, ok := os.LookupEnv(key)
	if !ok {
		panic(fmt.Errorf("environment variable %s must be set", key))
	}

	schema, err := os.ReadFile(schemaFile)
	if err != nil {
		panic(fmt.Errorf("failed to read icingadb schema file %q: %w", schemaFile, err))
	}

	db, err := MysqlDatabaseOpen(m)
	if err != nil {
		panic(err)
	}
	// duplicated from https://github.com/Icinga/docker-icingadb/blob/master/entrypoint/main.go
	sqlComment := regexp.MustCompile(`(?m)^--.*`)
	sqlStmtSep := regexp.MustCompile(`(?m);$`)
	for _, ddl := range sqlStmtSep.Split(string(sqlComment.ReplaceAll(schema, nil)), -1) {
		if ddl = strings.TrimSpace(ddl); ddl != "" {
			if _, err := db.Exec(ddl); err != nil {
				if mysqlErr, ok := err.(*mysql.MySQLError); ok && mysqlErr.Number == 1050 {
					// ignore 'table already exists' errors for now, probably the schema was already imported
					// TODO(jb): find a proper solution for this
					return
				} else {
					panic(err)
				}
			}
		}
	}
}
