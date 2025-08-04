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
	Model    string        `json:"model,omitempty"`
	Messages []ChatMessage `json:"messages,omitempty"`
	// Stream enables streaming of returned responses; true by default.
	Stream *bool `json:"stream,omitempty"`

	Inputs map[string]interface{} `json:"inputs,omitempty"`

	// Format is the format to return the response in (e.g. "json").
	Format json.RawMessage `json:"format,omitempty"`

	// KeepAlive controls how long the model will stay loaded into memory
	// following the request.
	KeepAlive *Duration `json:"keep_alive,omitempty"`

	// Tools is an optional list of tools the model has access to.
	Tools `json:"tools,omitempty"`

	// Options lists model-specific options.
	Options map[string]interface{} `json:"options,omitempty"`
	KbIds   string                 `json:"kbIds,omitempty"`
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

type ModelBackend struct {
	ID        string      `json:"id"`             // 唯一标识，区分不同模型实例
	Name      string      `json:"name"`           // 模型名称
	ModelID   string      `json:"model_id"`       // 模型唯一ID
	ModelName string      `json:"model_name"`     // 模型名称
	Type      string      `json:"type"`           // 后端类型，例如 "Ollama", "OpenAI", "Dify", "LangChat" ...
	ModelData interface{} `json:"data,omitempty"` // 事件中的具体数据，动态解析为对应结构体
}

type ModelBackendNodeOllamaOrOpenAI struct {
	Endpoint   string            `json:"endpoint"`   // 后端服务的地址（URL）
	Token      string            `json:"token"`      // 认证 token（用于 JWT 或 API Key）
	Parameters map[string]string `json:"parameters"` // 模型调用附加参数，例如温度、最大 token 数
}

type ModelBackendNodeDify struct {
	DifyType   string            `json:"dify_type"`  // dify_chat, dify_comp, dify_agent, dify_chat_flow, dify_work_flow
	Endpoint   string            `json:"endpoint"`   // 后端服务的地址（URL）
	Token      string            `json:"token"`      // 认证 token（用于 JWT 或 API Key）
	Parameters map[string]string `json:"parameters"` // 模型调用附加参数
}

type DifyRequest struct {
	Query          string                 `json:"query"`
	Inputs         map[string]interface{} `json:"inputs"`
	ResponseMode   string                 `json:"response_mode"`
	User           string                 `json:"user"`
	ConversationID string                 `json:"conversation_id,omitempty"`
}

type DifyEvent struct {
	Event          string                 `json:"event"`
	TaskID         string                 `json:"task_id,omitempty"`
	MessageID      string                 `json:"message_id,omitempty"`
	Answer         string                 `json:"answer,omitempty"`
	CreatedAt      int64                  `json:"created_at,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	ConversationID string                 `json:"conversation_id,omitempty"`
}

type DifyWorkFlowEvent struct {
	Event          string      `json:"event"`                     // 事件类型（如 workflow_started, node_started 等）
	TaskID         string      `json:"task_id,omitempty"`         // 任务 ID
	WorkflowRunID  string      `json:"workflow_run_id,omitempty"` // 工作流运行 ID（仅适用于 workflow 相关事件）
	ConversationID string      `json:"conversation_id,omitempty"` // 对话 ID（仅适用于 tts_message 相关事件）
	MessageID      string      `json:"message_id,omitempty"`      // 消息 ID（仅适用于 tts_message 相关事件）
	CreatedAt      int64       `json:"created_at,omitempty"`      // 创建时间戳
	Audio          string      `json:"audio,omitempty,omitempty"` // 音频数据（仅适用于 tts_message 相关事件）
	Data           interface{} `json:"data,omitempty"`            // 事件中的具体数据，动态解析为对应结构体
}

type DifyWorkflowData struct {
	ID             string                 `json:"id"`                        // 工作流运行 ID
	WorkflowID     string                 `json:"workflow_id,omitempty"`     // 工作流 ID
	SequenceNumber int                    `json:"sequence_number,omitempty"` // 步骤序号
	CreatedAt      int64                  `json:"created_at,omitempty"`      // 创建时间戳
	Outputs        map[string]interface{} `json:"outputs,omitempty"`         // 输出数据（仅适用于 workflow_finished）
	Status         string                 `json:"status,omitempty"`          // 状态（succeeded 等，仅适用于 workflow_finished）
	ElapsedTime    float64                `json:"elapsed_time,omitempty"`    // 总耗时（仅适用于 workflow_finished）
	TotalTokens    int64                  `json:"total_tokens,omitempty"`    // 总 Token 数量（仅适用于 workflow_finished）
	TotalSteps     string                 `json:"total_steps,omitempty"`     // 总步骤数（仅适用于 workflow_finished）
	FinishedAt     int64                  `json:"finished_at,omitempty"`     // 完成时间戳（仅适用于 workflow_finished）
}

type DifyWorkFlowNodeData struct {
	ID                string                 `json:"id"`                            // 节点运行 ID
	NodeID            string                 `json:"node_id,omitempty"`             // 节点 ID
	NodeType          string                 `json:"node_type,omitempty"`           // 节点类型（start 等）
	Title             string                 `json:"title,omitempty"`               // 节点标题
	Index             int                    `json:"index,omitempty"`               // 节点索引
	PredecessorNodeID string                 `json:"predecessor_node_id,omitempty"` // 前置节点 ID
	Inputs            map[string]interface{} `json:"inputs,omitempty"`              // 输入数据（仅适用于 node_started）
	Outputs           map[string]interface{} `json:"outputs,omitempty"`             // 输出数据（仅适用于 node_finished）
	Status            string                 `json:"status,omitempty"`              // 节点状态（succeeded 等，仅适用于 node_finished）
	ElapsedTime       float64                `json:"elapsed_time,omitempty"`        // 节点耗时（仅适用于 node_finished）
	ExecutionMetadata map[string]interface{} `json:"execution_metadata,omitempty"`  // 执行元数据（如 total_tokens 等，仅适用于 node_finished）
	CreatedAt         int64                  `json:"created_at,omitempty"`          // 创建时间戳
}

type DifyWorkFlowTTSMessageData struct {
	ConversationID string `json:"conversation_id,omitempty"` // 对话 ID
	MessageID      string `json:"message_id,omitempty"`      // 消息 ID
	CreatedAt      int64  `json:"created_at,omitempty"`      // 创建时间戳
	TaskID         string `json:"task_id,omitempty"`         // 任务 ID
	Audio          string `json:"audio,omitempty"`           // 音频数据
}
