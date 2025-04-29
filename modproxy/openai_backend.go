package modproxy

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"log"
	"modhub/common"
	"net/http"
	"strings"
	"time"
)

type OpenAIStreamResponse struct {
	ID                string `json:"id"`
	Object            string `json:"object"`
	Created           int64  `json:"created"`
	Model             string `json:"model"`
	SystemFingerprint string `json:"system_fingerprint,omitempty"` // 可选字段
	Choices           []struct {
		Index int `json:"index"`
		Delta struct {
			Content string `json:"content"`
			// 如果有其他 delta 字段（如 role, function_call 等），可以在这里添加
		} `json:"delta"`
		Logprobs     interface{} `json:"logprobs"`      // 可能是 null 或复杂结构
		FinishReason string      `json:"finish_reason"` // 可能是 null 或 "stop"/"length"/"function_call" 等
	} `json:"choices"`
	Usage *struct { // 可选字段（流式响应中通常为 null）
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage,omitempty"`
}

func ForwardToOpenAIStream(openaiURL string, apiKey string, req common.ChatRequest, c *gin.Context) error {
	// 将请求对象序列化为 JSON
	reqBody, err := json.Marshal(req)
	if err != nil {
		return err
	}

	// 构造 HTTP 请求
	client := &http.Client{}
	httpReq, err := http.NewRequest("POST", openaiURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey) // 设置 OpenAI 的 API 密钥

	// 发起 HTTP 请求
	resp, err := client.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 检查后端服务的响应状态
	if resp.StatusCode != http.StatusOK {
		var errorResponse map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&errorResponse) // 尝试解析错误信息
		log.Printf("OpenAI 服务响应状态码错误: %d, 错误信息: %v", resp.StatusCode, errorResponse)
		return fmt.Errorf("OpenAI 服务响应状态码错误: %d", resp.StatusCode)
	}

	// 设置响应头，支持流式传输
	c.Header("Content-Type", resp.Header.Get("Content-Type")) // 继承后端服务响应的内容类型，通常是 "application/json"
	c.Header("Transfer-Encoding", "chunked")
	c.Status(http.StatusOK)

	// 使用流式处理逐块读取响应体并处理
	reader := bufio.NewReader(resp.Body)

	isStream := true
	if req.Stream != nil && !*req.Stream {
		isStream = false
	}
	var sb strings.Builder

	for {
		// 每次读取一个数据块
		chunk, err := reader.ReadBytes('\n') // OpenAI 的流式 API 通常以换行符分割
		if len(chunk) > 0 {
			processedChunk := ConvertToOllama(string(chunk), req.Model, &sb)
			if len(processedChunk) == 0 {
				continue
			}
			if isStream {
				// 将处理后的数据写入客户端响应
				if _, writeErr := c.Writer.Write([]byte(processedChunk)); writeErr != nil {
					log.Println("写入客户端响应时出错:", writeErr)
					c.JSON(http.StatusInternalServerError, gin.H{"error": "写入客户端响应时出错"})
					return writeErr
				}

				// 刷新缓冲区，保证客户端实时接收到数据
				c.Writer.Flush()
			} else {
				// 非流式模式下，累积数据到缓冲区
				sb.Write([]byte(processedChunk))
			}
		}

		// 检查是否遇到 EOF 或其他错误
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Println("读取响应数据时出错:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "读取响应数据时出错"})
			return err
		}
	}

	if !isStream {
		// 非流式模式下，组装完整的响应
		output := common.OutputData{
			Model:     req.Model,
			CreatedAt: time.Now().Format(time.RFC3339Nano), // 当前时间
			Message: common.Message{
				Role:    "assistant",
				Content: sb.String(),
			},
			Done: true, // 假设响应是完整的，直接标记 done 为 true
		}
		c.Header("Content-Type", "application/json")
		c.JSON(http.StatusOK, output)
	}

	return nil
}

//	func ConvertToOllama(input string, model string, sb *strings.Builder) (result string) {
//		defer func() {
//			if r := recover(); r != nil {
//				result = ""
//			}
//		}()
//		// 解析输入数据
//		cleaned := strings.Trim(strings.TrimPrefix(input, "data:"), "\n")
//		log.Println(cleaned)
//
//		if len(cleaned) == 0 {
//			return ""
//		}
//
//		var inputData map[string]interface{}
//		err := json.Unmarshal([]byte(cleaned), &inputData)
//		if err != nil {
//			return ""
//		}
//		//delete inputData["choices"][0]["delta"]["role"]
//
//		return cleaned
//	}
//
// ConvertToOllama converts OpenAI streaming response chunk to Ollama format
// Parameters:
// - chunk: Raw JSON string from OpenAI stream
// - model: Model name to include in response
// - sb: Optional string builder for accumulating context (used for non-streaming mode)
// Returns:
// - Processed JSON string in Ollama format
// - Empty string if chunk should be skipped
func ConvertToOllama(chunk string, model string, sb *strings.Builder) string {
	// Handle OpenAI's [DONE] event
	chunk = strings.Trim(strings.TrimPrefix(chunk, "data:"), "\n")
	var output common.OutputData

	if len(chunk) == 0 {
		return ""
	}

	if chunk == "[DONE]" {
		// 创建输出数据
		output = common.OutputData{
			Model:     model,
			CreatedAt: time.Now().Format(time.RFC3339Nano), // 当前时间
			Message: common.Message{
				Role:    "assistant",
				Content: "",
			},
			Done: true, // 假设 Answer 是完整的，直接标记 done 为 true
		}
	} else {
		var inputData OpenAIStreamResponse
		err := json.Unmarshal([]byte(chunk), &inputData)
		if err != nil {
			return ""
		}
		if len(inputData.Choices) == 0 {
			return ""
		}

		output = common.OutputData{
			Model:     model,
			CreatedAt: time.Now().Format(time.RFC3339Nano), // 当前时间
			Message: common.Message{
				Role:    "assistant",
				Content: inputData.Choices[0].Delta.Content,
			},
			Done: true, // 假设 Answer 是完整的，直接标记 done 为 true
		}
	}

	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false) // 禁用 HTML 转义
	err := encoder.Encode(output)
	if err != nil {
		log.Println("Error encoding JSON:", err)
		return ""
	}
	return string(buffer.String())
}

/*
{
    "id": "chatcmpl-140",
    "object": "chat.completion.chunk",
    "created": 1745906094,
    "model": "deepseek-v3:671b(sdu)",
    "system_fingerprint": "fp_ollama",
    "choices": [
        {
            "index": 0,
            "delta": {
                "role": "assistant",
                "content": " 😊"
            },
            "finish_reason": null
        }
    ]
}

{
    "id": "chatcmpl-BRY8hW9BP3CkzRqqJbMmyzmnHRSR8",
    "object": "chat.completion.chunk",
    "created": 1745905787,
    "model": "gpt-4o-2024-11-20",
    "system_fingerprint": "fp_ee1d74bde0",
    "choices": [
        {
            "index": 0,
            "delta": {
                "content": "http"
            },
            "logprobs": null,
            "finish_reason": null
        }
    ],
    "usage": null
}

{
    "id": "chatcmpl-BRY8hW9BP3CkzRqqJbMmyzmnHRSR8",
    "object": "chat.completion.chunk",
    "created": 1745905787,
    "model": "gpt-4o-2024-11-20",
    "system_fingerprint": "fp_ee1d74bde0",
    "choices": [
        {
            "index": 0,
            "delta": {
                "content": "domain"
            },
            "logprobs": null,
            "finish_reason": null
        }
    ],
    "usage": null
}
*/
