package bkconfig

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"modhub/common"
	"os"
	"path/filepath"
)

// 生成测试数据
func GenerateTestData() []common.ModelBackend {
	// Ollama 后端
	ollamaParams := map[string]string{
		"temperature": "0.7",
		"max_tokens":  "1000",
	}
	ollamaData := common.ModelBackendNodeOllamaOrOpenAI{
		Endpoint:   "https://xxxxxxx",
		Token:      "ollama-token-123",
		Parameters: ollamaParams,
	}
	// OpenAI 后端
	openaiParams := map[string]string{
		"temperature": "0.5",
		"max_tokens":  "2048",
	}
	openaiData := common.ModelBackendNodeOllamaOrOpenAI{
		Endpoint:   "https://api.xxxx.net",
		Token:      "sk-xxxx",
		Parameters: openaiParams,
	}
	// Dify 后端
	difyParams := map[string]string{
		"top_p": "0.9",
	}
	difyDataLua := common.ModelBackendNodeDify{
		DifyType:   "dify_chat",
		Endpoint:   "http://localhost",
		Token:      "app-L0d1pXYEaaRjomsk7LJMaN6T",
		Parameters: difyParams,
	}
	// 创建 ModelBackend 数组
	models := []common.ModelBackend{
		{
			ID:        "model-1",
			Name:      "山大DeepSeek",
			ModelID:   "ollama-sdu-0001",
			ModelName: "deepseek-v3:671b(sdu)",
			Type:      "Ollama",
			ModelData: ollamaData,
		},
		{
			ID:        "model-2",
			Name:      "OpenAI GPT-4",
			ModelID:   "openai-0001",
			ModelName: "gpt-4o",
			Type:      "OpenAI",
			ModelData: openaiData,
		},
		{
			ID:        "model-3",
			Name:      "房地产LUA",
			ModelID:   "dify-chat-1",
			ModelName: "Dify Chat Model",
			Type:      "Dify",
			ModelData: difyDataLua,
		},
	}
	return models
}

func GetModelByModelID(modelName string) common.ModelBackend {
	//modelBackends := GenerateTestData()
	modelBackends, _ := ReadModelsFromFile("./bkconfig.json")
	for _, modelBackend := range modelBackends {
		if modelBackend.Name == modelName {
			return modelBackend
		}
	}
	return common.ModelBackend{}
}

func GenerateModelList() []common.Model {
	var modelList []common.Model
	//modelBackends := GenerateTestData()
	modelBackends, _ := ReadModelsFromFile("./bkconfig.json")

	for _, model := range modelBackends {
		modelListNode := common.Model{
			Name:  model.Name,
			Model: model.ModelID,
		}
		modelList = append(modelList, modelListNode)
	}
	return modelList
}

// 将 ModelBackend 数组写入文件
func WriteModelsToFile(models []common.ModelBackend, filename string) error {
	jsonData, err := json.MarshalIndent(models, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling models to JSON: %v", err)
	}
	err = ioutil.WriteFile(filename, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("error writing to file: %v", err)
	}
	return nil
}

func GetBaseDir(fn string) string {
	exePath, err := os.Executable()
	if err != nil {
		fmt.Println("Error getting executable path:", err)
		return "./"
	}

	// 获取可执行文件所在的目录
	exeDir := filepath.Dir(exePath)
	fullPath := filepath.Join(exeDir, fn)
	return fullPath
}

// 从文件读取 ModelBackend 数组
func ReadModelsFromFile(filename string) ([]common.ModelBackend, error) {
	filename = GetBaseDir(filename)
	fileData, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %v", err)
	}
	var models []common.ModelBackend
	err = json.Unmarshal(fileData, &models)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling JSON: %v", err)
	}
	// 根据 Type 字段解析 ModelData
	for i := range models {
		switch models[i].Type {
		case "Ollama", "OpenAI":
			var data common.ModelBackendNodeOllamaOrOpenAI
			jsonData, _ := json.Marshal(models[i].ModelData)
			json.Unmarshal(jsonData, &data)
			models[i].ModelData = data
		case "Dify":
			var data common.ModelBackendNodeDify
			jsonData, _ := json.Marshal(models[i].ModelData)
			json.Unmarshal(jsonData, &data)
			models[i].ModelData = data
		}
	}
	return models, nil
}
