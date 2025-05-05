package route

import (
	"encoding/json"
	"fmt"
	"github.com/corazawaf/coraza/v3"
	txhttp "github.com/corazawaf/coraza/v3/http"
	"github.com/corazawaf/coraza/v3/types"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"modhub/bkconfig"
	"modhub/common"
	"modhub/metrics"
	"modhub/modproxy"
	"modhub/openai"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Gin ResponseWriter wrapper
// ginWAFResponseRecorder is a custom ResponseWriter for capturing responses
type ginWAFResponseRecorder struct {
	gin.ResponseWriter
	StatusCode int
	Body       []byte
}

func (r *ginWAFResponseRecorder) WriteHeader(statusCode int) {
	r.StatusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *ginWAFResponseRecorder) Write(data []byte) (int, error) {
	r.Body = append(r.Body, data...)
	return r.ResponseWriter.Write(data)
}

// Gin handler wrapped with Coraza WAF
func corazaMiddleware(waf coraza.WAF) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Wrap the Gin request with Coraza WAF
		recorder := &ginWAFResponseRecorder{ResponseWriter: c.Writer, Body: []byte{}}
		txhttp.WrapHandler(waf, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c.Request = r
			c.Writer = recorder
			c.Next()
		})).ServeHTTP(recorder, c.Request)

		// Write the recorded response back to the client
		for k, v := range recorder.Header() {
			for _, vv := range v {
				c.Writer.Header().Add(k, vv)
			}
		}
		c.Writer.WriteHeader(recorder.StatusCode)
		c.Abort()
	}
}

func createWAF() coraza.WAF {
	waf, err := coraza.NewWAF(
		coraza.NewWAFConfig().
			WithErrorCallback(logError).
			WithDirectivesFromFile("./tswaf_coreruleset/coraza.conf").
			WithDirectivesFromFile("./tswaf_coreruleset/crs-setup.conf.example").
			WithDirectivesFromFile("./tswaf_coreruleset/rules/*.conf"))
	if err != nil {
		log.Printf("error creating waf: %v", err)
		return nil
	}
	return waf
}

func logError(error types.MatchedRule) {
	msg := error.ErrorLog()
	fmt.Printf("[logError][%s] %s\n", error.Rule().Severity(), msg)
}

func ModelMetricsHandler(w http.ResponseWriter, r *http.Request) {
	// 自定义逻辑：例如打印日志或其他操作
	// 调用标准的 Prometheus 处理器
	promhttp.HandlerFor(metrics.ModelMetricRegistry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	}).ServeHTTP(w, r)
}

func RouterApi() {
	waf := createWAF()
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	if waf != nil {
		router.Use(corazaMiddleware(waf))
	}

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
	router.GET("/v1/models", openai.ListMiddleware(), ListHandler)

	//router.POST("/v1/completions", openai.CompletionsMiddleware(), GenerateHandler)
	//router.POST("/v1/embeddings", openai.EmbeddingsMiddleware(), EmbedHandler)

	//router.HandleFunc("/tsdata/metrics", TsDataMetricsHandler)

	log.Printf("服务监听地址：127.0.0.1:11436")
	router.Run("0.0.0.0:11436")
}

func ListHandler(c *gin.Context) {
	modelList := bkconfig.GenerateModelList()
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"models": modelList,
	})
}

func ChatHandler(c *gin.Context) {
	var req common.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	modelBackend := bkconfig.GetModelByModelID(req.Model)
	if len(modelBackend.ModelID) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "model not found"})
		return
	}

	// 如果模型 ID 以 "ollama_" 开头，则转发到 ollama 后端服务
	if strings.EqualFold(modelBackend.Type, "Ollama") {
		log.Printf("检测到Ollama模型，开始流式转发请求: %s", req.Model)
		// 构造后端 Ollama 请求 URL
		var ollamaData common.ModelBackendNodeOllamaOrOpenAI
		parseBackendData(modelBackend.ModelData, &ollamaData)
		req.Model = modelBackend.ModelName
		ollamaURL, _ := url.JoinPath(ollamaData.Endpoint, "/api/chat")
		if err := modproxy.ForwardToOllamaStream(ollamaURL, req, c); err != nil {
			log.Printf("转发到 Ollama 服务时出错: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "转发到 Ollama 服务失败"})
		}
		return
	}

	if strings.EqualFold(modelBackend.Type, "OpenAI") {
		log.Printf("检测到OpenAI模型，开始流式转发请求: %s", req.Model)
		// 构造后端 OpenAI 请求 URL
		var openAIData common.ModelBackendNodeOllamaOrOpenAI
		parseBackendData(modelBackend.ModelData, &openAIData)
		req.Model = modelBackend.ModelName
		openAIURL, _ := url.JoinPath(openAIData.Endpoint, "/v1/chat/completions")
		if err := modproxy.ForwardToOpenAIStream(openAIURL, openAIData.Token, req, c); err != nil {
			log.Printf("转发到 OpenAI 服务时出错: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "转发到 OpenAI 服务失败"})
		}
		return
	}

	if strings.EqualFold(modelBackend.Type, "Dify") {
		log.Printf("检测到Dify模型，开始流式转发请求: %s", req.Model)

		var difyData common.ModelBackendNodeDify
		parseBackendData(modelBackend.ModelData, &difyData)
		//dify_chat, dify_comp, dify_agent, dify_chat_flow, dify_work_flow
		switch difyData.DifyType {
		case "dify_chat":
			fallthrough
		case "dify_agent":
			fallthrough
		case "dify_chat_flow":
			difyUrl, _ := url.JoinPath(difyData.Endpoint, "v1/chat-messages")
			if err := modproxy.ForwardToDifyChatStream(difyUrl, difyData.Token, req, c); err != nil {
				log.Printf("转发到 Dify 服务时出错: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "转发到 Dify 服务失败"})
			}
		case "dify_comp":
			difyUrl, _ := url.JoinPath(difyData.Endpoint, "/v1/completion-messages")
			if err := modproxy.ForwardToDifyCompletionStream(difyUrl, difyData.Token, req, c); err != nil {
				log.Printf("转发到 Dify 服务时出错: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "转发到 Dify 服务失败"})
			}
		case "dify_work_flow":
			difyUrl, _ := url.JoinPath(difyData.Endpoint, "/v1/workflows/run")
			if err := modproxy.ForwardToDifyCompletionStream(difyUrl, difyData.Token, req, c); err != nil {
				log.Printf("转发到 Dify 服务时出错: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "转发到 Dify 服务失败"})
			}
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "未匹配合适Dify服务"})
		}
	}
}

// 动态解析 data 字段
func parseBackendData(data interface{}, out interface{}) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, out)
}
