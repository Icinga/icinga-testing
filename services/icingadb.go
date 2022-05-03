package services

import (
	_ "embed"
	"io"
	"text/template"
)

type IcingaDbBase interface {
	// Redis returns the instance information of the Redis server this instance is using.
	Redis() RedisServerBase

	// RelationalDatabase returns the instance information of the relational database this instance is using.
	RelationalDatabase() RelationalDatabase

	// Cleanup stops the instance and removes everything that was created to start it.
	Cleanup()
}

// IcingaDb wraps the IcingaDbBase interface and adds some more helper functions.
type IcingaDb struct {
	IcingaDbBase
	config string
}

//go:embed icingadb.yml
var icingadbYmlRawTemplate string
var icingadbYmlTemplate = template.Must(template.New("icingadb.yml").Parse(icingadbYmlRawTemplate))

func (i IcingaDb) WriteConfig(w io.Writer) error {
	return icingadbYmlTemplate.Execute(w, i)
}

// Config returns additional raw YAML configuration, if any.
func (i IcingaDb) Config() string {
	return i.config
}

// IcingaDbOption configures IcingaDb.
type IcingaDbOption func(*IcingaDb)

// WithIcingaDbConfig sets additional raw YAML configuration.
func WithIcingaDbConfig(config string) func(*IcingaDb) {
	return func(db *IcingaDb) {
		db.config = config
	}
}
