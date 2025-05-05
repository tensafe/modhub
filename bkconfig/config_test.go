package bkconfig

import (
	"fmt"
	"testing"
)

func TestConfigRW(t *testing.T) {
	modelBackend := GenerateTestData()
	WriteModelsToFile(modelBackend, "./abc.json")
	ReadModelsFromFile("./abc.json")
}

func TestSyncData(t *testing.T) {

	SyncDataToJSON()
}

func TestBuildModelMapCacheInfo(t *testing.T) {
	abc, err := BuildModelMapCacheInfo()
	fmt.Println(abc, err)
}
