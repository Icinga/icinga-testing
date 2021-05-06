package services

import (
	_ "embed"
	"io"
	"io/ioutil"
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

func IcingaDbInstanceImportSchema(i IcingaDbInstance) {
	// TODO: properly import schema
	schema, err := ioutil.ReadFile("/testing/icingadb.mysql.schema.sql")
	if err != nil {
		panic(err)
	}
	db, err := MysqlDatabaseOpen(i.Mysql())
	if err != nil {
		panic(err)
	}
	// duplicated from https://github.com/Icinga/docker-icingadb/blob/master/entrypoint/main.go
	sqlComment := regexp.MustCompile(`(?m)^--.*`)
	sqlStmtSep := regexp.MustCompile(`(?m);$`)
	for _, ddl := range sqlStmtSep.Split(string(sqlComment.ReplaceAll(schema, nil)), -1) {
		if ddl = strings.TrimSpace(ddl); ddl != "" {
			if _, err := db.Exec(ddl); err != nil {
				panic(err)
			}
		}
	}
}
