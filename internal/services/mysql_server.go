package services

import "github.com/icinga/icinga-testing/services"

type MysqlServer interface {
	Database() services.MysqlDatabaseBase
	Cleanup()
}
