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
	config              *config.Config
	chatLogService      *service.ChatLogService
	conversationService *service.ConversationService
	llmClient           *llm.Client
	mcpServer           *imcp.MCPServer
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
		config:              cfg,
		chatLogService:      service.NewChatLogService(),
		conversationService: service.NewConversationService(),
		llmClient:           llmClient,
		mcpServer:           mcpServer,
	}
}

// ChatMessage 对话消息
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest 对话请求
type ChatRequest struct {
	Messages       []ChatMessage `json:"messages"`
	Model          string        `json:"model,omitempty"`
	Stream         bool          `json:"stream"`
	Temperature    float64       `json:"temperature,omitempty"`
	MaxTokens      int           `json:"max_tokens,omitempty"`
	ConversationID uint          `json:"conversation_id,omitempty"` // 所属会话ID
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
		userLog, err = h.chatLogService.CreateUserMessageWithConversation(username, "API", userMessage, req.ConversationID)
		if err != nil {
			logx.Error("Failed to save user message: %v", err)
			// 不阻断请求，继续处理
		}
		// 如果有会话ID，更新会话的最后消息时间
		if req.ConversationID > 0 && err == nil {
			if err := h.conversationService.UpdateLastMessageAt(req.ConversationID); err != nil {
				logx.Error("Failed to update conversation last message time: %v", err)
			}
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

	// 将前端传来的消息转换为 LLM 消息格式
	llmMessages := make([]llm.Message, 0, len(req.Messages))
	for _, msg := range req.Messages {
		llmMessages = append(llmMessages, llm.Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// 调用 LLM 流式对话（传递完整的消息历史，会自动使用已启用的 MCP 工具）
	var responseCh <-chan string
	var err error
	if len(llmMessages) > 0 {
		// 使用新方法传递完整的消息历史
		responseCh, err = h.llmClient.ChatWithToolsAndStreamMessages(ctx, llmMessages)
	} else {
		// 降级：如果没有消息历史，使用旧方法
		responseCh, err = h.llmClient.ChatWithToolsAndStream(ctx, userMessage)
	}

	if err != nil {
		logx.Error("Failed to call LLM: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: fmt.Sprintf("LLM调用失败: %v", err),
		})
		return
	}

	logx.Info("Calling LLM with %d messages in history", len(llmMessages))

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
			_, err := h.chatLogService.CreateAIMessageWithConversation(username, "API", aiResponse.String(), userLog.ID, req.ConversationID)
			if err != nil {
				logx.Error("Failed to save AI response: %v", err)
			}
			// 如果有会话ID，更新会话的最后消息时间
			if req.ConversationID > 0 && err == nil {
				if err := h.conversationService.UpdateLastMessageAt(req.ConversationID); err != nil {
					logx.Error("Failed to update conversation last message time: %v", err)
				}

				// 检查是否需要生成标题
				shouldGenerate, err := h.conversationService.ShouldGenerateTitle(req.ConversationID)
				if err == nil && shouldGenerate && userMessage != "" {
					// 异步生成标题，避免阻塞响应
					go func() {
						title := h.generateConversationTitle(context.Background(), userMessage)
						if err := h.conversationService.UpdateConversation(req.ConversationID, title); err != nil {
							logx.Error("Failed to update conversation title: %v", err)
						} else {
							logx.Info("Generated conversation title: %s", title)
						}
					}()
				}
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
			_, err := h.chatLogService.CreateAIMessageWithConversation(username, "API", aiMessage, userLog.ID, req.ConversationID)
			if err != nil {
				logx.Error("Failed to save AI response: %v", err)
			}
			// 如果有会话ID，更新会话的最后消息时间
			if req.ConversationID > 0 && err == nil {
				if err := h.conversationService.UpdateLastMessageAt(req.ConversationID); err != nil {
					logx.Error("Failed to update conversation last message time: %v", err)
				}

				// 检查是否需要生成标题
				shouldGenerate, err := h.conversationService.ShouldGenerateTitle(req.ConversationID)
				if err == nil && shouldGenerate && userMessage != "" {
					// 异步生成标题，避免阻塞响应
					go func() {
						title := h.generateConversationTitle(context.Background(), userMessage)
						if err := h.conversationService.UpdateConversation(req.ConversationID, title); err != nil {
							logx.Error("Failed to update conversation title: %v", err)
						} else {
							logx.Info("Generated conversation title: %s", title)
						}
					}()
				}
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

// generateConversationTitle 生成会话标题
func (h *ChatHandler) generateConversationTitle(ctx context.Context, userMessage string) string {
	// 构建生成标题的提示词
	titlePrompt := fmt.Sprintf(`请根据下面的用户问题，生成一个简短的会话标题（5-15个字）。
只返回标题文本，不要包含任何其他内容、标点符号或解释。

用户问题：%s

会话标题：`, userMessage)

	// 调用 LLM 生成标题
	responseCh, err := h.llmClient.ChatWithToolsAndStream(ctx, titlePrompt)
	if err != nil {
		logx.Error("Failed to generate conversation title: %v", err)
		// 如果生成失败，使用用户消息的前10个字符作为标题
		if len(userMessage) > 10 {
			return userMessage[:10] + "..."
		}
		return userMessage
	}

	// 收集完整响应
	var title strings.Builder
	for content := range responseCh {
		title.WriteString(content)
	}

	generatedTitle := strings.TrimSpace(title.String())

	// 清理标题（移除引号、换行等）
	generatedTitle = strings.Trim(generatedTitle, `"'`)
	generatedTitle = strings.ReplaceAll(generatedTitle, "\n", " ")
	generatedTitle = strings.TrimSpace(generatedTitle)

	// 限制标题长度
	if len([]rune(generatedTitle)) > 20 {
		runes := []rune(generatedTitle)
		generatedTitle = string(runes[:20]) + "..."
	}

	// 如果生成的标题为空，使用默认值
	if generatedTitle == "" {
		if len(userMessage) > 10 {
			return userMessage[:10] + "..."
		}
		return userMessage
	}

	return generatedTitle
}

// GetModels 获取可用的模型列表
func (h *ChatHandler) GetModels(c *gin.Context) {
	// 从数据库读取 LLM 配置
	configService := service.NewConfigService()
	llmConfigs, err := configService.ListLLMConfigs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: fmt.Sprintf("Failed to load LLM configs: %v", err),
		})
		return
	}

	// 转换为前端期望的格式，只返回启用的模型
	models := []map[string]interface{}{}
	for _, cfg := range llmConfigs {
		if !cfg.Enabled {
			continue
		}

		model := map[string]interface{}{
			"id":       cfg.Model,
			"name":     cfg.Name,
			"provider": cfg.Provider,
		}

		// 标记默认模型（与当前配置的模型匹配）
		if h.config.LLM.Enabled && cfg.Model == h.config.LLM.Model {
			model["default"] = true
		}

		models = append(models, model)
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
