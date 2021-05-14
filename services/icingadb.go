package services

import (
	_ "embed"
	"github.com/go-sql-driver/mysql"
	"io"
	"regexp"
	"strings"
	"text/template"
)

type IcingaDb interface {
	Instance(redis RedisServer, mysql MysqlDatabase) IcingaDbInstance
	Cleanup()
}

type IcingaDbInstance interface {
	Redis() RedisServer
	Mysql() MysqlDatabase
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

// TODO(jb): properly import schema
//go:embed icingadb_mysql_schema.sql
var icingadbMysqlSchema string

func IcingaDbInstanceImportSchema(i IcingaDbInstance) {
	db, err := MysqlDatabaseOpen(i.Mysql())
	if err != nil {
		panic(err)
	}
	// duplicated from https://github.com/Icinga/docker-icingadb/blob/master/entrypoint/main.go
	sqlComment := regexp.MustCompile(`(?m)^--.*`)
	sqlStmtSep := regexp.MustCompile(`(?m);$`)
	for _, ddl := range sqlStmtSep.Split(string(sqlComment.ReplaceAllString(icingadbMysqlSchema, "")), -1) {
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
