package bkconfig

import (
	"testing"
)

func TestConfigRW(t *testing.T) {
	modelBackend := GenerateTestData()
	WriteModelsToFile(modelBackend, "./abc.json")
	ReadModelsFromFile("./abc.json")
}
