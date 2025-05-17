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

// Dify 请求响应结构体

func ForwardToDifyChatStream(difyURL, apiKey string, req common.ChatRequest, c *gin.Context) error {
	log.Println("recv ForwardToDifyChatStream..")
	difyReq, err := convertToDifyChatRequest(req)
	if err != nil {
		return err
	}

	difyReqBody, err := json.Marshal(difyReq)
	if err != nil {
		return err
	}
	// 构造 HTTP 请求
	client := &http.Client{}
	httpReq, err := http.NewRequest("POST", difyURL, bytes.NewBuffer(difyReqBody))
	if err != nil {
		log.Printf("构造 HTTP 请求时出错: %v", err)
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	// 发起 HTTP 请求
	resp, err := client.Do(httpReq)
	if err != nil {
		log.Printf("请求 Dify 服务失败: %v", err)
		return err
	}
	defer resp.Body.Close()

	// 检查 HTTP 响应状态码
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("Dify 响应状态码错误: %d, 响应内容: %s", resp.StatusCode, string(body))
		return fmt.Errorf("Dify 服务响应状态码错误: %d", resp.StatusCode)
	}

	// 使用流式处理逐块读取响应体并处理
	reader := bufio.NewReader(resp.Body)

	isStream := true
	if req.Stream != nil && !*req.Stream {
		isStream = false
	}
	var sb strings.Builder

	log.Println("开始读取数据....")
	for {
		// 每次读取一个数据块
		chunk, err := reader.ReadBytes('\n') // Dify 的流式 API 通常以换行符分割
		if len(chunk) > 0 {
			processedChunk := convertDifyChatToOllama(string(chunk), req.Model, &sb, isStream)
			if len(processedChunk) == 0 {
				fmt.Println("processedChunk is empty", string(chunk))
				continue
			}
			if isStream {
				// 将处理后的数据写入客户端响应
				if _, writeErr := c.Writer.Write([]byte(processedChunk)); writeErr != nil {
					log.Println("写入客户端响应时出错:", writeErr)
					c.JSON(http.StatusInternalServerError, gin.H{"error": "写入客户端响应时出错"})
					return writeErr
				}
				log.Println("write data.....", len(processedChunk))
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
		// 非流式模式下，组装完整的响应,使用convertDifyChatToOllama转换之后的函数
		//output := common.OutputData{
		//	Model:     req.Model,
		//	CreatedAt: time.Now().Format(time.RFC3339Nano), // 当前时间
		//	Message: common.Message{
		//		Role:    "assistant",
		//		Content: sb.String(),
		//	},
		//	Done: true, // 假设响应是完整的，直接标记 done 为 true
		//}
		c.Header("Content-Type", "application/json")
		c.String(http.StatusOK, sb.String())
	}

	return nil
}

func convertDifyChatToOllama(chunk string, model string, sb *strings.Builder, isStream bool) string {
	chunk = strings.Trim(strings.TrimPrefix(chunk, "data:"), "\n")
	var output common.OutputData

	if len(chunk) == 0 {
		return ""
	}

	var inputData common.DifyEvent
	err := json.Unmarshal([]byte(chunk), &inputData)
	if err != nil {
		return ""
	}

	switch inputData.Event {
	case "message":
		output = common.OutputData{
			Model:     model,
			CreatedAt: time.Now().Format(time.RFC3339Nano), // 当前时间
			Message: common.Message{
				Role:    "assistant",
				Content: inputData.Answer,
			},
			Done: false, // 假设 Answer 是完整的，直接标记 done 为 true
		}
	case "message_end":
		output = common.OutputData{
			Model:     model,
			CreatedAt: time.Now().Format(time.RFC3339Nano), // 当前时间
			Message: common.Message{
				Role:    "assistant",
				Content: "",
			},
			Done: true, // 假设 Answer 是完整的，直接标记 done 为 true
		}
	default:
		output = common.OutputData{}
	}

	if !isStream {
		output.Done = true
	}

	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false) // 禁用 HTML 转义
	err = encoder.Encode(output)
	if err != nil {
		log.Println("Error encoding JSON:", err)
		return ""
	}
	return string(buffer.String())
}

func convertToDifyChatRequest(req common.ChatRequest) (*common.DifyRequest, error) {
	var query string
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			query = req.Messages[i].Content
			break
		}
	}
	if query == "" {
		return nil, fmt.Errorf("no user message found")
	}

	mode := "blocking"
	if *req.Stream {
		mode = "streaming"
	}

	return &common.DifyRequest{
		Query:        query,
		Inputs:       make(map[string]interface{}),
		ResponseMode: mode,
		User:         req.Model,
	}, nil
}
