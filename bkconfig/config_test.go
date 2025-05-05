package bkconfig

import (
	"testing"
)

func TestConfigRW(t *testing.T) {
	modelBackend := GenerateTestData()
	WriteModelsToFile(modelBackend, "./abc.json")
	ReadModelsFromFile("./abc.json")
}

func TestSyncData(t *testing.T) {
	SetConfigValue("db_address", "140.143.208.64:3306")
	SetConfigValue("db_username", "root")
	SetConfigValue("db_password", "Tzwy@1234")
	SetConfigValue("db_dbname", "knowledge")
	SyncDataToJSON()
}
