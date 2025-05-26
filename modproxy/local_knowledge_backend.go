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

// 转发请求到 Ollama 服务
func ForwardToLocalKnowledgeOllamaStream(ollamaURL string, req common.ChatRequest, c *gin.Context) error {
	// 将请求对象序列化为 JSON
	reqBody, err := json.Marshal(req)
	if err != nil {
		return err
	}

	// 构造 HTTP 请求
	client := &http.Client{}
	httpReq, err := http.NewRequest("POST", ollamaURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	serJwtToken := c.GetHeader("Authorization")
	if len(serJwtToken) > 0 {
		httpReq.Header.Set("Authorization", serJwtToken)
	} else {
		log.Println("No Authorization header")
	}
	//check jwt是否包含
	// 增加model_id变量

	// 发起 HTTP 请求
	resp, err := client.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 检查后端服务的响应状态
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Ollama 服务响应状态码错误: %d", resp.StatusCode)
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
		chunk, err := reader.ReadBytes('\n') // 或者使用 reader.Read() 按字节读取
		if len(chunk) > 0 {
			if isStream {
				// 将处理后的数据写入客户端响应
				if _, writeErr := c.Writer.Write([]byte(chunk)); writeErr != nil {
					log.Println("Error writing processed chunk:", writeErr)
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Error writing processed chunk"})
					return writeErr
				}

				// 刷新缓冲区，保证客户端实时接收到数据
				c.Writer.Flush()
			}
		}

		// 检查是否遇到 EOF 或其他错误
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Println("Error reading response chunk:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error reading response chunk"})
			return err
		}
	}

	if !isStream {
		output := common.OutputData{
			Model:     req.Model,
			CreatedAt: time.Now().Format(time.RFC3339Nano), // 当前时间
			Message: common.Message{
				Role:    "assistant",
				Content: sb.String(),
			},
			Done: true, // 假设 Answer 是完整的，直接标记 done 为 true
		}
		c.Header("Content-Type", "application/json")
		c.JSON(http.StatusOK, output)
	}

	return nil
}
