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

	if strings.HasPrefix(req.Model, "openai-") {
		log.Printf("检测到模型 ID 以 'openai_' 开头，开始流式转发请求: %s", req.Model)

		// 构造后端 Ollama 请求 URL
		openaiUrl := "https://ecs.tensafe.com:6014/v1/chat/completions" // 替换为实际服务地址
		//req.Model = "gpt-4"
		if err := modproxy.ForwardToOpenAIStream(openaiUrl, "sk-xxxxx", req, c); err != nil {
			log.Printf("转发到 Ollama 服务时出错: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "转发到 Ollama 服务失败"})
		}
		return
	}

	if strings.HasPrefix(req.Model, "dify_comp-") {
		log.Printf("检测到模型 ID 以 'dify_' 开头，开始流式转发请求: %s", req.Model)

		// 构造后端 Ollama 请求 URL
		//difyUrl := "http://localhost/v1/chat-messages" // 替换为实际服务地址
		difyUrl := "http://localhost/v1/completion-messages" // 替换为实际服务地址
		//req.Model = "gpt-4"
		if err := modproxy.ForwardToDifyCompletionStream(difyUrl, "app-smfYlwzlSCXfOPpdN7xaNYLq", req, c); err != nil {
			log.Printf("转发到 Ollama 服务时出错: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "转发到 Ollama 服务失败"})
		}
		return
	}
	if strings.HasPrefix(req.Model, "deepseek-") {
		log.Printf("检测到模型 ID 以 'dify_' 开头，开始流式转发请求: %s", req.Model)

		// 构造后端 Ollama 请求 URL
		//difyUrl := "http://localhost/v1/chat-messages" // 替换为实际服务地址
		difyUrl := "http://localhost/v1/workflows/run" // 替换为实际服务地址
		//req.Model = "gpt-4"
		if err := modproxy.ForwardToDifyWorkFlowStream(difyUrl, "app-MIf6XcOBcSfh6UgJJHidumr1", req, c); err != nil {
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
