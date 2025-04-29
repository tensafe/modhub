package common

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"time"
)

type GenerateRequest struct {
	Model  string `json:"model" binding:"required"`
	Prompt string `json:"prompt" binding:"required"`
}
type ChatMessage struct {
	Role    string `json:"role" binding:"required"`
	Content string `json:"content" binding:"required"`
}

type Duration struct {
	time.Duration
}

func (d Duration) MarshalJSON() ([]byte, error) {
	if d.Duration < 0 {
		return []byte("-1"), nil
	}
	return []byte("\"" + d.Duration.String() + "\""), nil
}

func (d *Duration) UnmarshalJSON(b []byte) (err error) {
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}

	d.Duration = 5 * time.Minute

	switch t := v.(type) {
	case float64:
		if t < 0 {
			d.Duration = time.Duration(math.MaxInt64)
		} else {
			d.Duration = time.Duration(int(t) * int(time.Second))
		}
	case string:
		d.Duration, err = time.ParseDuration(t)
		if err != nil {
			return err
		}
		if d.Duration < 0 {
			d.Duration = time.Duration(math.MaxInt64)
		}
	default:
		return fmt.Errorf("Unsupported type: '%s'", reflect.TypeOf(v))
	}

	return nil
}

type Tools []Tool

func (t Tools) String() string {
	bts, _ := json.Marshal(t)
	return string(bts)
}

func (t Tool) String() string {
	bts, _ := json.Marshal(t)
	return string(bts)
}

type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  struct {
		Type       string   `json:"type"`
		Required   []string `json:"required"`
		Properties map[string]struct {
			Type        string   `json:"type"`
			Description string   `json:"description"`
			Enum        []string `json:"enum,omitempty"`
		} `json:"properties"`
	} `json:"parameters"`
}

func (t *ToolFunction) String() string {
	bts, _ := json.Marshal(t)
	return string(bts)
}

type ChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages" binding:"required"`
	// Stream enables streaming of returned responses; true by default.
	Stream *bool `json:"stream,omitempty"`

	// Format is the format to return the response in (e.g. "json").
	Format json.RawMessage `json:"format,omitempty"`

	// KeepAlive controls how long the model will stay loaded into memory
	// following the request.
	KeepAlive *Duration `json:"keep_alive,omitempty"`

	// Tools is an optional list of tools the model has access to.
	Tools `json:"tools,omitempty"`

	// Options lists model-specific options.
	Options map[string]interface{} `json:"options"`
}

// 定义输入数据结构
type InputData struct {
	E int    `json:"e"`
	M string `json:"m"`
	D struct {
		Type          string        `json:"type"`
		Answer        string        `json:"answer"`
		Url           string        `json:"url"`
		MessageID     string        `json:"message_id"`
		ID            string        `json:"id"`
		RecommendData []interface{} `json:"recommend_data"`
		Source        []interface{} `json:"source"`
		Ext           interface{}   `json:"ext"`
	} `json:"d"`
}

// 定义输出数据结构
type OutputData struct {
	Model         string  `json:"model"`
	CreatedAt     string  `json:"created_at"`
	Message       Message `json:"message"`
	Done          bool    `json:"done"`
	DoneReason    string  `json:"done_reason,omitempty"`
	TotalDuration int64   `json:"total_duration,omitempty"`
	LoadDuration  int64   `json:"load_duration,omitempty"`
}
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type ModelDetails struct {
	ParentModel       string   `json:"parent_model"`
	Format            string   `json:"format"`
	Family            string   `json:"family"`
	Families          []string `json:"families"`
	ParameterSize     string   `json:"parameter_size"`
	QuantizationLevel string   `json:"quantization_level"`
}
type Model struct {
	Name       string       `json:"name"`
	Model      string       `json:"model"`
	ModifiedAt time.Time    `json:"modified_at"`
	Size       int64        `json:"size"`
	Digest     string       `json:"digest"`
	Details    ModelDetails `json:"details"`
}

type ModNode struct {
	ID       string `json:"id"`       // 唯一标识，区分不同模型实例
	Name     string `json:"name"`     // 模型名称
	Type     string `json:"type"`     // 后端类型，例如 "Ollama", "OpenAI", "Dify", "LangChat" ...
	Endpoint string `json:"endpoint"` // 后端服务的地址（URL）
	Token    string `json:"token"`    // 认证 token（用于 JWT 或 API Key）
	ModelID  string `json:"model_id"` // 模型 ID（针对需要明确模型的接口，例如 OpenAI 的 fine-tuned 模型）

	Parameters  map[string]string `json:"parameters"`   // 模型调用的参数，例如温度、最大 token 数
	ProxyConfig map[string]string `json:"proxy_config"` // 代理相关配置
}

//ModNode{
//ID:         "ollama_model_1",
//Name:       "Ollama GPT",
//Type:       "Ollama",
//Endpoint:   "http://localhost:8000/api/v1/ollama",
//Token:      "your-ollama-token",
//ModelID:    "ollama-gpt",
//Parameters: map[string]string{
//"temperature": "0.7",
//"max_tokens": "1000",
//},
//}

//ModNode{
//ID:         "openai_model_1",
//Name:       "OpenAI GPT-4",
//Type:       "OpenAI",
//Endpoint:   "https://api.openai.com/v1/completions",
//Token:      "your-openai-api-key",
//ModelID:    "gpt-4",
//Parameters: map[string]string{
//"temperature": "0.9",
//"max_tokens": "1500",
//},
//}
