package services

type MysqlServer interface {
	Database() MysqlDatabase
	Cleanup()
}
