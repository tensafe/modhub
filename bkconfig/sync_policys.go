package bkconfig

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	_ "github.com/glebarez/sqlite"
	_ "github.com/go-sql-driver/mysql" // MySQL 驱动
	"log"
	"modhub/common"
	"strings"
	"sync"
)

var (
	local_sqlite_name    = "./local_database.db"
	local_model_cache    = map[string]interface{}{}
	local_model_cache_mu sync.RWMutex // 读写锁
)

// 设置配置值的函数
func SetConfigValue(key string, value string) error {
	db, err := sql.Open("sqlite", local_sqlite_name)
	if err != nil {
		log.Printf("无法连接到 SQLite 数据库: %v", err)
		return err
	}
	defer db.Close()

	createTableSQL := `
	CREATE TABLE IF NOT EXISTS kvconfig (
		key   TEXT UNIQUE NOT NULL,  -- key 为唯一约束
		value TEXT
	);`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Printf("创建 SQLite 表失败: %v", err)
		return err
	}

	query := `INSERT INTO kvconfig (key, value) VALUES (?, ?)
	          ON CONFLICT(key) DO UPDATE SET value = excluded.value`
	_, err = db.Exec(query, key, value)
	if err != nil {
		return fmt.Errorf("设置配置项失败: %v", err)
	}
	return nil
}

// 获取配置值的函数
func GetConfigValue(key string) (string, error) {
	db, err := sql.Open("sqlite", local_sqlite_name)
	if err != nil {
		log.Printf("无法连接到 SQLite 数据库: %v", err)
		return "", err
	}
	defer db.Close()

	query := `SELECT value FROM kvconfig WHERE key = ?`
	var value string
	err = db.QueryRow(query, key).Scan(&value)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("未找到配置项: %s", key)
		}
		return "", fmt.Errorf("查询配置项失败: %v", err)
	}
	return value, nil
}

func GetBackendDBInfo() (address string, username string, password string, dbname string, err error) {
	// 获取地址、用户名、密码配置
	address, err = GetConfigValue("db_address")
	if err != nil {
		log.Printf("获取地址失败: %v", err)
		return "", "", "", "", err
	}

	username, err = GetConfigValue("db_username")
	if err != nil {
		log.Printf("获取用户名失败: %v", err)
		return "", "", "", "", err
	}

	password, err = GetConfigValue("db_password")
	if err != nil {
		log.Printf("获取密码失败: %v", err)
		return "", "", "", "", err
	}
	dbname, err = GetConfigValue("db_dbname")
	if err != nil {
		log.Printf("获取数据库名失败: %v", err)
		return "", "", "", "", err
	}
	return address, username, password, dbname, nil
}

// 主同步函数：从 MySQL 获取数据并保存到 SQLite 的 JSON 字段
func SyncDataToJSON() error {
	// 连接 MySQL
	sqliteDB, err := sql.Open("sqlite", local_sqlite_name)
	if err != nil {
		log.Printf("无法连接到 SQLite 数据库: %v", err)
		return err
	}
	defer sqliteDB.Close()

	address, username, password, dbname, err := GetBackendDBInfo()
	mysqlDSN := fmt.Sprintf("%s:%s@tcp(%s)/%s", username, password, address, dbname)

	mysqlDB, err := sql.Open("mysql", mysqlDSN)
	if err != nil {
		log.Printf("无法连接到 MySQL: %v", err)
		return err
	}
	defer mysqlDB.Close()

	// 创建 SQLite 表 (如果不存在)
	createTableSQL := `
		CREATE TABLE IF NOT EXISTS backend_models (
		model_id VARCHAR(255) NOT NULL UNIQUE,
		model_info TEXT
	);`
	_, err = sqliteDB.Exec(createTableSQL)
	if err != nil {
		log.Printf("创建 SQLite 表失败: %v", err)
		return err
	}

	// 从 MySQL 读取数据
	rows, err := mysqlDB.Query("SELECT * FROM sys_model_information where is_enable = 1")
	if err != nil {
		log.Printf("查询 MySQL 数据失败: %v", err)
		return err
	}
	defer rows.Close()

	insertOrReplaceQuery := `
	INSERT OR REPLACE INTO backend_models (
		model_id,
		model_info
	) values (?, ?);`

	// 遍历 MySQL 数据
	for rows.Next() {
		columns, err := rows.Columns()
		if err != nil {
			log.Printf("Failed to get columns: %v", err)
			continue
		}

		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		err = rows.Scan(valuePtrs...)
		if err != nil {
			log.Printf("Failed to scan row: %v", err)
			continue
		}

		rowData := make(map[string]interface{})
		for i, colName := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				rowData[colName] = string(b)
			} else {
				rowData[colName] = val
			}
		}

		backend_type := ""
		if value, exists := rowData["type"]; exists && value != nil {
			backend_type = fmt.Sprintf("%v", value)
		}
		if strings.EqualFold(backend_type, "dify") {
			model_id, model_json, err := buildDifyModelJson(rowData)
			if err == nil {
				//fmt.Println(model_id, model_json)
				_, err = sqliteDB.Exec(insertOrReplaceQuery, model_id, model_json)
				if err != nil {
					log.Printf("Failed to insert or replace data: %v", err)
				}
			} else {
				log.Printf("<UNK>: %v", err)
			}
		} else if strings.EqualFold(backend_type, "openai") || strings.EqualFold(backend_type, "ollama") {
			model_id, model_json, err := buildOllamaOrOpenAIModelJson(rowData)
			if err == nil {
				//fmt.Println(model_id, model_json)
				_, err = sqliteDB.Exec(insertOrReplaceQuery, model_id, model_json)
				if err != nil {
					log.Printf("Failed to insert or replace data: %v", err)
				}
			} else {
				log.Printf("<UNK>: %v", err)
			}
		}
	}

	log.Println("数据同步完成：从 MySQL 到 SQLite 的 JSON 字段")
	return nil
}

func buildDifyModelJson(rowData map[string]interface{}) (string, string, error) {
	//retJson := ""
	id := ""
	name := ""
	model_id := ""
	model_name := ""
	dify_type := ""
	end_point := ""
	token := ""

	if value, exists := rowData["id"]; exists && value != nil {
		id = fmt.Sprintf("%v", value)
	} else {
		return "", "", errors.New("id not exists")
	}

	if value, exists := rowData["name"]; exists && value != nil {
		name = fmt.Sprintf("%v", value)
	} else {
		return "", "", errors.New("name not exists")
	}

	if value, exists := rowData["model_id"]; exists && value != nil {
		model_id = fmt.Sprintf("%v", value)
	} else {
		return "", "", errors.New("model_id not exists")
	}

	if value, exists := rowData["model_name"]; exists && value != nil {
		model_name = fmt.Sprintf("%v", value)
	} else {
		return "", "", errors.New("model_name not exists")
	}

	if value, exists := rowData["dify_type"]; exists && value != nil {
		dify_type = fmt.Sprintf("%v", value)
	} else {
		return "", "", errors.New("dify_type not exists")
	}

	if value, exists := rowData["endpoint"]; exists && value != nil {
		end_point = fmt.Sprintf("%v", value)
	} else {
		return "", "", errors.New("endpoint not exists")
	}

	if value, exists := rowData["token"]; exists && value != nil {
		token = fmt.Sprintf("%v", value)
	} else {
		return "", "", errors.New("token not exists")
	}

	dify_backend := common.ModelBackend{}
	dify_backend.ID = id
	dify_backend.Name = name
	dify_backend.ModelID = model_id
	dify_backend.ModelName = model_name
	dify_backend.Type = "Dify"

	dify_params := common.ModelBackendNodeDify{}
	dify_params.DifyType = dify_type
	dify_params.Endpoint = end_point
	dify_params.Token = token
	dify_params.Parameters = nil

	dify_backend.ModelData = dify_params

	jsonData, err := json.Marshal(dify_backend)
	if err != nil {
		return "", "", err
	}
	return id, string(jsonData), nil
}

func buildOllamaOrOpenAIModelJson(rowData map[string]interface{}) (string, string, error) {
	//retJson := ""
	id := ""
	name := ""
	model_id := ""
	model_name := ""
	end_point := ""
	token := ""
	backend_type := ""

	if value, exists := rowData["id"]; exists && value != nil {
		id = fmt.Sprintf("%v", value)
	} else {
		return "", "", errors.New("id not exists")
	}

	if value, exists := rowData["name"]; exists && value != nil {
		name = fmt.Sprintf("%v", value)
	} else {
		return "", "", errors.New("name not exists")
	}

	if value, exists := rowData["model_id"]; exists && value != nil {
		model_id = fmt.Sprintf("%v", value)
	} else {
		return "", "", errors.New("model_id not exists")
	}

	if value, exists := rowData["model_name"]; exists && value != nil {
		model_name = fmt.Sprintf("%v", value)
	} else {
		return "", "", errors.New("model_name not exists")
	}

	if value, exists := rowData["type"]; exists && value != nil {
		backend_type = fmt.Sprintf("%v", value)
	} else {
		return "", "", errors.New("type not exists")
	}

	if value, exists := rowData["endpoint"]; exists && value != nil {
		end_point = fmt.Sprintf("%v", value)
	} else {
		return "", "", errors.New("endpoint not exists")
	}

	if value, exists := rowData["token"]; exists && value != nil {
		token = fmt.Sprintf("%v", value)
	} else {
		return "", "", errors.New("token not exists")
	}

	openai_backend := common.ModelBackend{}
	openai_backend.ID = id
	openai_backend.Name = name
	openai_backend.ModelID = model_id
	openai_backend.ModelName = model_name
	openai_backend.Type = backend_type

	openai_params := common.ModelBackendNodeOllamaOrOpenAI{}
	openai_params.Endpoint = end_point
	openai_params.Token = token

	openai_backend.ModelData = openai_params

	jsonData, err := json.Marshal(openai_backend)
	if err != nil {
		return "", "", err
	}
	return id, string(jsonData), nil
}

func BuildModelMapCacheInfo() (map[string]interface{}, error) {
	sqliteDB, err := sql.Open("sqlite", local_sqlite_name)
	if err != nil {
		log.Printf("无法连接到 SQLite 数据库: %v", err)
		return nil, err
	}

	defer sqliteDB.Close()
	// 查询 backend_models 表数据
	query := `SELECT model_id, model_info FROM backend_models`

	// 执行查询
	rows, err := sqliteDB.Query(query)
	if err != nil {
		log.Printf("查询 backend_models 数据失败: %v", err)
		return nil, err
	}
	defer rows.Close()
	// 用于存储查询结果的 map
	modelMapCache := make(map[string]interface{})

	// 遍历查询结果，将数据存入 map
	for rows.Next() {
		var modelID string
		var modelJSON string

		// 扫描每行数据到变量
		err := rows.Scan(&modelID, &modelJSON)
		if err != nil {
			log.Printf("扫描数据行失败: %v", err)
			return nil, err
		}
		// 将结果存入 map，键为 model_id，值为 model_json
		var modelNode common.ModelBackend
		err = json.Unmarshal([]byte(modelJSON), &modelNode)
		if err != nil {
			continue
		}
		// 根据 Type 字段解析 ModelData
		switch modelNode.Type {
		case "ollama", "openai":
			var data common.ModelBackendNodeOllamaOrOpenAI
			jsonData, _ := json.Marshal(modelNode.ModelData)
			json.Unmarshal(jsonData, &data)
			modelNode.ModelData = data
		case "dify":
			var data common.ModelBackendNodeDify
			jsonData, _ := json.Marshal(modelNode.ModelData)
			json.Unmarshal(jsonData, &data)
			modelNode.ModelData = data
		}

		modelMapCache[modelID] = modelNode
	}
	return modelMapCache, nil
}

func SyncBackendData(async bool) map[string]interface{} {
	if async {
		SyncDataToJSON()
	}
	modelCache, err := BuildModelMapCacheInfo()
	if err != nil {
		return nil
	}

	local_model_cache_mu.Lock()         // 写锁
	defer local_model_cache_mu.Unlock() // 解锁
	local_model_cache = modelCache
	return modelCache
}

func GetModelByModelID(modelName string) common.ModelBackend {
	local_model_cache_mu.RLock()         // 写锁
	defer local_model_cache_mu.RUnlock() // 解锁k
	for _, modelBackend := range local_model_cache {
		modelBackendNode := modelBackend.(common.ModelBackend)
		if modelBackendNode.Name == modelName {
			return modelBackendNode
		}
	}
	return common.ModelBackend{}
}

func GenerateModelList() []common.Model {
	var modelList []common.Model
	for _, model := range local_model_cache {
		modelBackendNode := model.(common.ModelBackend)
		modelListNode := common.Model{
			Name:  modelBackendNode.Name,
			Model: modelBackendNode.ModelID,
		}
		modelList = append(modelList, modelListNode)
	}
	return modelList
}
