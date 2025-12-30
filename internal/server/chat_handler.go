package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cnb.cool/zhiqiangwang/pkg/logx"
	"github.com/eryajf/zenops/internal/config"
	"github.com/eryajf/zenops/internal/imcp"
	"github.com/eryajf/zenops/internal/llm"
	"github.com/eryajf/zenops/internal/model"
	"github.com/eryajf/zenops/internal/service"
	"github.com/gin-gonic/gin"
)

// ChatHandler 处理 AI 对话请求
type ChatHandler struct {
	config         *config.Config
	chatLogService *service.ChatLogService
	llmClient      *llm.Client
	mcpServer      *imcp.MCPServer
}

// NewChatHandler 创建 ChatHandler
func NewChatHandler(cfg *config.Config, mcpServer *imcp.MCPServer) *ChatHandler {
	// 创建 LLM 客户端配置
	var llmClient *llm.Client
	if cfg.LLM.Enabled {
		llmConfig := &llm.Config{
			Model:   cfg.LLM.Model,
			APIKey:  cfg.LLM.APIKey,
			BaseURL: cfg.LLM.BaseURL,
		}
		llmClient = llm.NewClient(llmConfig, mcpServer)
	}

	return &ChatHandler{
		config:         cfg,
		chatLogService: service.NewChatLogService(),
		llmClient:      llmClient,
		mcpServer:      mcpServer,
	}
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

// Completions 处理对话请求 (支持流式和非流式，集成 MCP 工具)
func (h *ChatHandler) Completions(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	// 获取用户名（从请求头或使用默认值）
	username := c.GetHeader("X-Username")
	if username == "" {
		username = "api_user"
	}

	// 提取用户最后一条消息
	var userMessage string
	if len(req.Messages) > 0 {
		for i := len(req.Messages) - 1; i >= 0; i-- {
			if req.Messages[i].Role == "user" {
				userMessage = req.Messages[i].Content
				break
			}
		}
	}

	// 保存用户消息到数据库
	var userLog *model.ChatLog
	if userMessage != "" {
		var err error
		userLog, err = h.chatLogService.CreateUserMessage(username, "API", userMessage)
		if err != nil {
			logx.Error("Failed to save user message: %v", err)
			// 不阻断请求，继续处理
		}
	}

	// 检查 LLM 配置
	if !h.config.LLM.Enabled || h.llmClient == nil {
		c.JSON(http.StatusServiceUnavailable, Response{
			Code:    503,
			Message: "LLM service is not enabled",
		})
		return
	}

	// 使用 llm.Client 调用 LLM（支持 MCP 工具）
	ctx := context.Background()

	// 调用 LLM 流式对话（会自动使用已启用的 MCP 工具）
	responseCh, err := h.llmClient.ChatWithToolsAndStream(ctx, userMessage)
	if err != nil {
		logx.Error("Failed to call LLM: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: fmt.Sprintf("LLM调用失败: %v", err),
		})
		return
	}

	// 处理流式响应
	if req.Stream {
		// 设置 SSE 响应头
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

		// 用于收集 AI 的完整响应
		var aiResponse strings.Builder
		responseCounter := 0

		// 从响应通道读取流式数据
		for content := range responseCh {
			if content == "" {
				continue
			}

			aiResponse.WriteString(content)

			// 构建 SSE 格式的响应（OpenAI 格式）
			chunk := StreamChunk{
				ID:      fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano()),
				Object:  "chat.completion.chunk",
				Created: time.Now().Unix(),
				Model:   req.Model,
				Choices: []struct {
					Index int `json:"index"`
					Delta struct {
						Role    string `json:"role,omitempty"`
						Content string `json:"content,omitempty"`
					} `json:"delta"`
					FinishReason *string `json:"finish_reason"`
				}{
					{
						Index: 0,
						Delta: struct {
							Role    string `json:"role,omitempty"`
							Content string `json:"content,omitempty"`
						}{
							Content: content,
						},
						FinishReason: nil,
					},
				},
			}

			// 序列化为 JSON
			chunkJSON, err := json.Marshal(chunk)
			if err != nil {
				logx.Error("Failed to marshal chunk: %v", err)
				continue
			}

			// 发送 SSE 数据
			fmt.Fprintf(c.Writer, "data: %s\n\n", string(chunkJSON))
			flusher.Flush()
			responseCounter++
		}

		// 发送结束标记
		fmt.Fprintf(c.Writer, "data: [DONE]\n\n")
		flusher.Flush()

		logx.Info("Stream completed: sent %d chunks", responseCounter)

		// 保存 AI 响应到数据库
		if userLog != nil && aiResponse.Len() > 0 {
			_, err := h.chatLogService.CreateAIMessage(username, "API", aiResponse.String(), userLog.ID)
			if err != nil {
				logx.Error("Failed to save AI response: %v", err)
			}
		}
	} else {
		// 非流式响应：收集所有响应内容
		var fullResponse strings.Builder
		for content := range responseCh {
			fullResponse.WriteString(content)
		}

		aiMessage := fullResponse.String()

		// 保存 AI 响应到数据库
		if userLog != nil && aiMessage != "" {
			_, err := h.chatLogService.CreateAIMessage(username, "API", aiMessage, userLog.ID)
			if err != nil {
				logx.Error("Failed to save AI response: %v", err)
			}
		}

		// 构建非流式响应
		chatResp := ChatResponse{
			ID:      fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano()),
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   req.Model,
			Choices: []struct {
				Index   int `json:"index"`
				Message struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			}{
				{
					Index: 0,
					Message: struct {
						Role    string `json:"role"`
						Content string `json:"content"`
					}{
						Role:    "assistant",
						Content: aiMessage,
					},
					FinishReason: "stop",
				},
			},
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
