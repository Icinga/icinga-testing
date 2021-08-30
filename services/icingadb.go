package services

import (
	_ "embed"
	"io"
	"text/template"
)

type IcingaDbBase interface {
	// Redis returns the instance information of the Redis server this instance is using.
	Redis() RedisServerBase

	// Mysql returns the instance information of the MySQL database this instance is using.
	Mysql() MysqlDatabaseBase

	// Cleanup stops the instance and removes everything that was created to start it.
	Cleanup()
}

// IcingaDb wraps the IcingaDbBase interface and adds some more helper functions.
type IcingaDb struct {
	IcingaDbBase
}

//go:embed icingadb.yml
var icingadbYmlRawTemplate string
var icingadbYmlTemplate = template.Must(template.New("icingadb.yml").Parse(icingadbYmlRawTemplate))

func (i IcingaDb) WriteConfig(w io.Writer) error {
	return icingadbYmlTemplate.Execute(w, i)
}
