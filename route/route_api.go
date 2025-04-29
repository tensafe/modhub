package route

import (
	"github.com/gin-gonic/gin"
	"log"
	"modhub/common"
	"modhub/modproxy"
	"modhub/openai"
	"net/http"
	"strings"
	"time"
)

var mockModelsList = []common.Model{
	{
		Name:  "deepseek-r1:671b(sdu)",
		Model: "deepseek-r1:671b(sdu)",
	},
	{
		Name:  "deepseek-v3:671b(sdu)",
		Model: "deepseek-v3:671b(sdu)",
	},
	{
		Name:  "deepseek-r1-o:671b(sdu)",
		Model: "deepseek-r1-o:671b(sdu)",
	},
	{
		Name:  "deepseek-v3-o:671b(sdu)",
		Model: "deepseek-v3-o:671b(sdu)",
	},
	{
		Name:  "qwq-o:32b(sdu)",
		Model: "qwq-o:32b(sdu)",
	},
	{
		Name:  "qwq:32b(sdu)",
		Model: "qwq:32b(sdu)",
	},
}

//var baseUrl = "https://aiassist.sdu.edu.cn"

func RouterApi() {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	// 全局 CORS 中间件
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*") // 允许所有域名（生产环境建议指定具体域名）
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// 直接放行 OPTIONS 请求
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204) // 204 No Content
			return
		}

		c.Next() // 继续处理其他请求
	})
	// Models list endpoint (HEAD)
	router.HEAD("/api/tags", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "success",
		})
	})
	router.OPTIONS("/api/tags", func(c *gin.Context) {
		// 设置 CORS 响应头
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Status(http.StatusOK) // 返回 200 状态码
	})
	// Models list endpoint (GET)
	router.GET("/api/tags", ListHandler)
	// Generate response endpoint
	router.POST("/api/generate", func(c *gin.Context) {
		var req common.GenerateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		mockGenerateResponse := gin.H{
			"status": "success",
			"data": gin.H{
				"model":     req.Model,
				"prompt":    req.Prompt,
				"response":  "The sky is blue because of the way the Earth's atmosphere scatters sunlight.",
				"timestamp": time.Now().Format(time.RFC3339),
			},
		}

		c.JSON(http.StatusOK, mockGenerateResponse)
	})
	router.OPTIONS("/api/chat", func(c *gin.Context) {
		// 设置 CORS 响应头
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Status(http.StatusOK) // 返回 200 状态码
	})
	// Chat endpoint
	router.POST("/api/chat", ChatHandler)
	router.POST("/v1/chat/completions", openai.ChatMiddleware(), ChatHandler)
	//r.POST("/v1/completions", openai.CompletionsMiddleware(), GenerateHandler)
	//r.POST("/v1/embeddings", openai.EmbeddingsMiddleware(), EmbedHandler)
	router.GET("/v1/models", openai.ListMiddleware(), ListHandler)

	router.Run("127.0.0.1:11436")
}

func ListHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"models": mockModelsList,
	})
}

func ChatHandler(c *gin.Context) {
	var req common.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 如果模型 ID 以 "ollama_" 开头，则转发到 ollama 后端服务
	//if strings.HasPrefix(req.Model, "ollama_") {
	if strings.HasPrefix(req.Model, "ollama_") {
		log.Printf("检测到模型 ID 以 'ollama_' 开头，开始流式转发请求: %s", req.Model)

		// 构造后端 Ollama 请求 URL
		ollamaURL := "https://ecs.tensafe.com:6014/api/chat" // 替换为实际服务地址
		if err := modproxy.ForwardToOllamaStream(ollamaURL, req, c); err != nil {
			log.Printf("转发到 Ollama 服务时出错: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "转发到 Ollama 服务失败"})
		}
		return
	}

	if strings.HasPrefix(req.Model, "deepseek-") {
		log.Printf("检测到模型 ID 以 'openai_' 开头，开始流式转发请求: %s", req.Model)

		// 构造后端 Ollama 请求 URL
		ollamaURL := "https://ecs.tensafe.com:6014/v1/chat/completions" // 替换为实际服务地址
		//ollamaURL := "https://api.gptsapi.net/v1/chat/completions" // 替换为实际服务地址
		//req.Model = "gpt-4"
		if err := modproxy.ForwardToOpenAIStream(ollamaURL, "sk-xxxxx", req, c); err != nil {
			log.Printf("转发到 Ollama 服务时出错: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "转发到 Ollama 服务失败"})
		}
		return
	}

	//err := DeepSeek_Sdu_Compose_Chat(&req, c)
	var err error
	if err != nil {
		log.Println(err)
	}
}

//func FastCheckLoginState(_login_cookie string) bool {
//	url := baseUrl + "/site/ai/compose_chat"
//
//	var buffer bytes.Buffer
//	writer := multipart.NewWriter(&buffer)
//
//	_ = writer.WriteField("compose_id", "73")     // 固定值
//	_ = writer.WriteField("deep_search", "2")     // 推理 1 ;
//	_ = writer.WriteField("internet_search", "2") // 联网思考 1；不联网思考：2
//
//	_ = writer.WriteField("content", "who are you?")
//
//	err := writer.Close()
//	if err != nil {
//		log.Println(err)
//		return false
//	}
//
//	client := &http.Client{}
//	req_info, err := http.NewRequest("POST", url, &buffer)
//	if err != nil {
//		log.Println("Error creating forward request:", err)
//		return false
//	}
//
//	req_info.Header.Add("Cookie", _login_cookie)
//
//	req_info.Header.Set("Content-Type", writer.FormDataContentType())
//	// 发送转发请求
//	forwardRes, err := client.Do(req_info)
//	if err != nil {
//		log.Println("Error sending forward request:", err)
//		return false
//	}
//	defer forwardRes.Body.Close()
//
//	// 使用流式处理逐块读取响应体并处理
//	reader := bufio.NewReader(forwardRes.Body)
//	buffer = bytes.Buffer{}
//
//	bret := true
//
//	for {
//		// 每次读取一个数据块
//		chunk, _err := reader.ReadBytes('\n') // 或者使用 reader.Read() 按字节读取
//		if _err != nil {
//			if err == io.EOF {
//				break
//			}
//		}
//
//		chunk_string := string(chunk)
//		if strings.HasPrefix(chunk_string, "data:") {
//			if strings.Contains(chunk_string, "请登录") || strings.Contains(chunk_string, "10042") {
//				// cookie过期，需要重新登录
//				bret = false
//			}
//			break
//		}
//
//	}
//
//	return bret
//}
//func DeepSeek_Sdu_Compose_Chat(req *ChatRequest, c *gin.Context) error {
//	url := baseUrl + "/site/ai/compose_chat"
//
//	var buffer bytes.Buffer
//	writer := multipart.NewWriter(&buffer)
//
//	deep_search := "1"
//	internet_search := "2"
//	model_id := "73" //默认deepseek
//	switch req.Model {
//	case "deepseek-r1:671b(sdu)":
//		deep_search = "1"
//		internet_search = "2"
//		model_id = "73"
//	case "deepseek-v3:671b(sdu)":
//		deep_search = "2"
//		internet_search = "2"
//		model_id = "73"
//	case "deepseek-r1-o:671b(sdu)":
//		deep_search = "1"
//		internet_search = "1"
//		model_id = "73"
//	case "deepseek-v3-o:671b(sdu)":
//		deep_search = "2"
//		internet_search = "1"
//		model_id = "73"
//	case "qwq:32b(sdu)":
//		deep_search = "1"
//		internet_search = "2"
//		model_id = "72"
//	case "qwq-o:32b(sdu)":
//		deep_search = "1"
//		internet_search = "1"
//		model_id = "72"
//	default:
//		deep_search = "2"
//		internet_search = "2"
//	}
//
//	_ = writer.WriteField("compose_id", model_id)             // 模型id
//	_ = writer.WriteField("deep_search", deep_search)         // 推理 1 ;
//	_ = writer.WriteField("internet_search", internet_search) // 联网思考 1；不联网思考：2
//
//	for index, msg := range req.Messages {
//		if index == len(req.Messages)-1 {
//			break
//		}
//		_ = writer.WriteField(fmt.Sprintf("history[%d][role]", index), msg.Role)
//		_ = writer.WriteField(fmt.Sprintf("history[%d][content]", index), msg.Content)
//	}
//
//	if len(req.Messages) > 0 {
//		lastElement := req.Messages[len(req.Messages)-1]
//		_ = writer.WriteField("content", lastElement.Content)
//	}
//
//	err := writer.Close()
//	if err != nil {
//		log.Println(err)
//		return err
//	}
//
//	client := &http.Client{}
//	req_info, err := http.NewRequest("POST", url, &buffer)
//	if err != nil {
//		log.Println("Error creating forward request:", err)
//		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating forward request"})
//		return err
//	}
//
//	//req_info.Header.Add("Cookie", login_cookie_kvs)
//
//	req_info.Header.Set("Content-Type", writer.FormDataContentType())
//	// 发送转发请求
//	forwardRes, err := client.Do(req_info)
//	if err != nil {
//		log.Println("Error sending forward request:", err)
//		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error sending forward request"})
//		return err
//	}
//	defer forwardRes.Body.Close()
//
//	// 设置响应头
//	c.Writer.Header().Set("Content-Type", forwardRes.Header.Get("Content-Type"))
//	c.Writer.WriteHeader(forwardRes.StatusCode)
//
//	// 使用流式处理逐块读取响应体并处理
//	reader := bufio.NewReader(forwardRes.Body)
//	buffer = bytes.Buffer{}
//
//	isStream := true
//	if req.Stream != nil && !*req.Stream {
//		isStream = false
//	}
//	var sb strings.Builder
//
//	for {
//		// 每次读取一个数据块
//		chunk, err := reader.ReadBytes('\n') // 或者使用 reader.Read() 按字节读取
//		if len(chunk) > 0 {
//			// 处理数据（例如将其转换为大写）
//			processedChunk := ConvertToOllama(string(chunk), req.Model, &sb)
//			if len(processedChunk) == 0 {
//				continue
//			}
//
//			if isStream {
//				// 将处理后的数据写入客户端响应
//				if _, writeErr := c.Writer.Write([]byte(processedChunk)); writeErr != nil {
//					log.Println("Error writing processed chunk:", writeErr)
//					c.JSON(http.StatusInternalServerError, gin.H{"error": "Error writing processed chunk"})
//					return writeErr
//				}
//
//				// 刷新缓冲区，保证客户端实时接收到数据
//				c.Writer.Flush()
//			}
//		}
//
//		// 检查是否遇到 EOF 或其他错误
//		if err != nil {
//			if err == io.EOF {
//				break
//			}
//			log.Println("Error reading response chunk:", err)
//			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error reading response chunk"})
//			return err
//		}
//	}
//
//	if !isStream {
//		output := OutputData{
//			Model:     req.Model,
//			CreatedAt: time.Now().Format(time.RFC3339Nano), // 当前时间
//			Message: Message{
//				Role:    "assistant",
//				Content: sb.String(),
//			},
//			Done: true, // 假设 Answer 是完整的，直接标记 done 为 true
//		}
//		c.Header("Content-Type", "application/json")
//		c.JSON(http.StatusOK, output)
//	}
//
//	return nil
//}

//func ConvertToOllama(input string, model string, sb *strings.Builder) (result string) {
//	// 输入 JSON 字符串
//	//inputJSON := `{"e":0,"m":"操作成功","d":{"type":"0","answer":"<think>好的","url":"","message_id":"","id":"","recommend_data":[],"source":[],"ext":[]}}`
//	defer func() {
//		if r := recover(); r != nil {
//			// 如果发生panic，返回错误信息
//			result = ""
//		}
//	}()
//	// 解析输入数据
//	var inputData InputData
//	cleaned := strings.Trim(strings.TrimPrefix(input, "data:"), "\n")
//	//log.Println(cleaned)
//
//	//{
//	//	"e": 10042,
//	//	"d": [],
//	//"m": "请登录"
//	//}
//
//	if len(cleaned) == 0 {
//		return ""
//	}
//	err := json.Unmarshal([]byte(cleaned), &inputData)
//	if err != nil {
//		if strings.Contains(cleaned, "请登录") || strings.Contains(cleaned, "10042") {
//			// cookie过期，需要重新登录
//			LoginWeb()
//		}
//		return ""
//	}
//
//	outStr := inputData.D.Answer
//
//	if len(outStr) == 0 {
//		if inputData.E != 1 {
//			return ""
//		}
//	}
//
//	var markdown strings.Builder
//	if inputData.D.Ext != nil {
//		if extMap, ok := inputData.D.Ext.(map[string]interface{}); ok {
//			// 检查是否包含键 "site_search"
//			if _, exists := extMap["site_search"]; exists {
//				site_search_array := extMap["site_search"].([]interface{})
//				for _, siteItem := range site_search_array {
//					if siteNode, ok := siteItem.(map[string]interface{}); ok {
//						if title, ok := siteNode["title"].(string); ok {
//							markdown.WriteString(fmt.Sprintf("### %s\n\n", title)) // 二级标题
//						}
//						if content, ok := siteNode["content"].(string); ok {
//							markdown.WriteString(fmt.Sprintf("%s\n\n", content)) // 内容段落
//						}
//						if url, ok := siteNode["url"].(string); ok {
//							markdown.WriteString(fmt.Sprintf("[Read more](%s)\n\n", url)) // Markdown 链接
//						}
//					}
//				}
//			} else {
//				//fmt.Println("Ext 是 map[string]interface{} 但不包含键 'site_search'")
//			}
//		}
//	}
//
//	markstr := markdown.String()
//
//	if len(markstr) > 0 {
//		outStr = markstr + outStr
//	}
//
//	if sb != nil {
//		sb.WriteString(outStr)
//	}
//	// 创建输出数据
//	output := OutputData{
//		Model:     model,
//		CreatedAt: time.Now().Format(time.RFC3339Nano), // 当前时间
//		Message: Message{
//			Role:    "assistant",
//			Content: outStr,
//		},
//		Done: inputData.E == 1, // 假设 Answer 是完整的，直接标记 done 为 true
//	}
//
//	var buffer bytes.Buffer
//	encoder := json.NewEncoder(&buffer)
//	encoder.SetEscapeHTML(false) // 禁用 HTML 转义
//	err = encoder.Encode(output)
//	if err != nil {
//		log.Println("Error encoding JSON:", err)
//		return ""
//	}
//	return string(buffer.String())
//}
