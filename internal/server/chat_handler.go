package server

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"cnb.cool/zhiqiangwang/pkg/logx"
	"github.com/eryajf/zenops/internal/config"
	"github.com/gin-gonic/gin"
)

// ChatHandler 处理 AI 对话请求
type ChatHandler struct {
	config *config.Config
}

// NewChatHandler 创建 ChatHandler
func NewChatHandler(cfg *config.Config) *ChatHandler {
	return &ChatHandler{config: cfg}
}

// ChatMessage 对话消息
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest 对话请求
type ChatRequest struct {
	Messages    []ChatMessage `json:"messages"`
	Model       string        `json:"model,omitempty"`
	Stream      bool          `json:"stream"`
	Temperature float64       `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
}

// ChatResponse 非流式对话响应
type ChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// StreamChunk 流式响��块
type StreamChunk struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Role    string `json:"role,omitempty"`
			Content string `json:"content,omitempty"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

// Completions 处理对话请求 (支持流式和非流式)
func (h *ChatHandler) Completions(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	// 检查 LLM 配置
	if !h.config.LLM.Enabled {
		c.JSON(http.StatusServiceUnavailable, Response{
			Code:    503,
			Message: "LLM service is not enabled",
		})
		return
	}

	// 获取配置
	apiKey := h.config.LLM.APIKey
	baseURL := h.config.LLM.BaseURL
	model := h.config.LLM.Model

	if apiKey == "" {
		c.JSON(http.StatusServiceUnavailable, Response{
			Code:    503,
			Message: "LLM API key is not configured",
		})
		return
	}

	// 如果没有指定 base_url，使用 OpenAI 默认
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	// 使用请求中的模型或配置中的模型
	if req.Model == "" {
		req.Model = model
	}
	if req.Model == "" {
		req.Model = "gpt-4o"
	}

	// 构建请求体
	requestBody := map[string]interface{}{
		"model":    req.Model,
		"messages": req.Messages,
		"stream":   req.Stream,
	}
	if req.Temperature > 0 {
		requestBody["temperature"] = req.Temperature
	}
	if req.MaxTokens > 0 {
		requestBody["max_tokens"] = req.MaxTokens
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "Failed to marshal request",
		})
		return
	}

	// 创建 HTTP 请求
	httpReq, err := http.NewRequest("POST", baseURL+"/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "Failed to create request",
		})
		return
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	// 发送请求
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		logx.Error("LLM request failed: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "LLM request failed: " + err.Error(),
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		logx.Error("LLM API error: status=%d, body=%s", resp.StatusCode, string(body))
		c.JSON(resp.StatusCode, Response{
			Code:    resp.StatusCode,
			Message: "LLM API error: " + string(body),
		})
		return
	}

	// 处理响应
	if req.Stream {
		// 流式响应
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("Transfer-Encoding", "chunked")

		flusher, ok := c.Writer.(http.Flusher)
		if !ok {
			c.JSON(http.StatusInternalServerError, Response{
				Code:    500,
				Message: "Streaming not supported",
			})
			return
		}

		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					break
				}
				logx.Error("Error reading stream: %v", err)
				break
			}

			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			// 直接转发 SSE 数据
			fmt.Fprintf(c.Writer, "%s\n\n", line)
			flusher.Flush()

			// 检查是否是结束标记
			if line == "data: [DONE]" {
				break
			}
		}
	} else {
		// 非流式响应
		var chatResp ChatResponse
		if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
			c.JSON(http.StatusInternalServerError, Response{
				Code:    500,
				Message: "Failed to decode response",
			})
			return
		}

		c.JSON(http.StatusOK, Response{
			Code:    200,
			Message: "success",
			Data:    chatResp,
		})
	}
}

// GetModels 获取可用的模型列表
func (h *ChatHandler) GetModels(c *gin.Context) {
	// 返回一些常用的模型
	models := []map[string]interface{}{
		{"id": "gpt-4o", "name": "GPT-4o", "provider": "openai"},
		{"id": "gpt-4o-mini", "name": "GPT-4o Mini", "provider": "openai"},
		{"id": "gpt-4-turbo", "name": "GPT-4 Turbo", "provider": "openai"},
		{"id": "gpt-3.5-turbo", "name": "GPT-3.5 Turbo", "provider": "openai"},
		{"id": "deepseek-chat", "name": "DeepSeek Chat", "provider": "deepseek"},
		{"id": "deepseek-coder", "name": "DeepSeek Coder", "provider": "deepseek"},
		{"id": "claude-3-5-sonnet-20241022", "name": "Claude 3.5 Sonnet", "provider": "anthropic"},
	}

	// 如果配置了默认模型，标记它
	configuredModel := h.config.LLM.Model
	for i, m := range models {
		if m["id"] == configuredModel {
			models[i]["default"] = true
		}
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "success",
		Data: gin.H{
			"models":      models,
			"llm_enabled": h.config.LLM.Enabled,
		},
	})
}
