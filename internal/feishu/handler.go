package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cnb.cool/zhiqiangwang/pkg/logx"
	"github.com/eryajf/zenops/internal/config"
	"github.com/eryajf/zenops/internal/imcp"
	"github.com/eryajf/zenops/internal/llm"
	"github.com/eryajf/zenops/internal/model"
	"github.com/eryajf/zenops/internal/service"
	larkcontact "github.com/larksuite/oapi-sdk-go/v3/service/contact/v3"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
)

// MessageHandler 飞书消息处理器
type MessageHandler struct {
	client         *Client
	config         *config.Config
	mcpServer      *imcp.MCPServer
	llmClient      *llm.Client
	chatLogService *service.ChatLogService
}

// NewMessageHandler 创建消息处理器
func NewMessageHandler(cfg *config.Config, mcpServer *imcp.MCPServer) (*MessageHandler, error) {
	client := NewClient(cfg.Feishu.AppID, cfg.Feishu.AppSecret)

	// 初始化 LLM 客户端
	// 优先从数据库读取 LLM 配置，如果数据库没有配置则使用 config.yaml
	var llmClient *llm.Client
	configService := service.NewConfigService()
	dbLLMConfig, err := configService.GetDefaultLLMConfig()

	var llmEnabled bool
	var llmConfig *llm.Config

	if err == nil && dbLLMConfig != nil && dbLLMConfig.Enabled {
		// 使用数据库配置
		llmEnabled = true
		llmConfig = &llm.Config{
			Model:   dbLLMConfig.Model,
			APIKey:  dbLLMConfig.APIKey,
			BaseURL: dbLLMConfig.BaseURL,
		}
		logx.Info("⚗️ Using LLM Config from Database: %s (Model: %s)", dbLLMConfig.Name, dbLLMConfig.Model)
	} else if cfg.LLM.Enabled {
		// 降级使用 config.yaml 配置
		llmEnabled = true
		llmConfig = &llm.Config{
			Model:   cfg.LLM.Model,
			APIKey:  cfg.LLM.APIKey,
			BaseURL: cfg.LLM.BaseURL,
		}
		logx.Info("⚗️ Using LLM Config from config.yaml (Model: %s)", cfg.LLM.Model)
	}

	if llmEnabled {
		llmClient = llm.NewClient(llmConfig, mcpServer)
		logx.Info("⚗️ LLM Client Initialized For Feishu")
	}

	return &MessageHandler{
		client:         client,
		config:         cfg,
		mcpServer:      mcpServer,
		llmClient:      llmClient,
		chatLogService: service.NewChatLogService(),
	}, nil
}

// MessageContent 消息内容
type MessageContent struct {
	Text string `json:"text"`
}

// HandleTextMessage 处理文本消息
func (h *MessageHandler) HandleTextMessage(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
	// 解析消息内容
	var content MessageContent
	if err := json.Unmarshal([]byte(*event.Event.Message.Content), &content); err != nil {
		logx.Error("Failed to unmarshal message content: %v", err)
		return err
	}

	userMessage := strings.TrimSpace(content.Text)
	if userMessage == "" {
		return nil
	}

	logx.Info("Received message from Feishu: user %s, message %s",
		*event.Event.Sender.SenderId.OpenId,
		userMessage)

	// 确定消息来源（私聊/群聊）
	source := "私聊"
	if *event.Event.Message.ChatType == "group" {
		source = "群聊"
	}

	// 保存用户消息到数据库
	username := *event.Event.Sender.SenderId.OpenId
	userLog, err := h.chatLogService.CreateUserMessage(username, source, userMessage)
	if err != nil {
		logx.Error("Failed to save user message to database: %v", err)
	}

	// 特殊命令处理
	if strings.Contains(userMessage, "帮助") || strings.Contains(userMessage, "help") {
		return h.sendHelpMessage(ctx, event, username, source, userLog)
	}

	// 如果启用了 LLM,使用 LLM 处理
	if h.config.LLM.Enabled && h.llmClient != nil {
		return h.processLLMMessage(ctx, event, userMessage, username, source, userLog)
	}

	// 否则返回默认消息
	receiveIDType := "open_id"
	receiveID := *event.Event.Sender.SenderId.OpenId
	if *event.Event.Message.ChatType == "group" {
		receiveIDType = "chat_id"
		receiveID = *event.Event.Message.ChatId
	}

	return h.client.SendTextMessage(ctx, receiveIDType, receiveID,
		"ZenOps 飞书机器人已收到您的消息。当前未启用 LLM 对话功能,请联系管理员配置。")
}

// processLLMMessage 使用 LLM 处理消息(流式卡片更新)
func (h *MessageHandler) processLLMMessage(ctx context.Context, event *larkim.P2MessageReceiveV1, userMessage, username, source string, userLog *model.ChatLog) error {
	receiveIDType := "open_id"
	receiveID := *event.Event.Sender.SenderId.OpenId
	if *event.Event.Message.ChatType == "group" {
		receiveIDType = "chat_id"
		receiveID = *event.Event.Message.ChatId
	}

	// 调用 LLM 流式对话
	responseCh, err := h.llmClient.ChatWithToolsAndStream(ctx, userMessage)
	if err != nil {
		logx.Error("Failed to call LLM: %v", err)
		return h.client.SendTextMessage(ctx, receiveIDType, receiveID,
			fmt.Sprintf("LLM 调用失败: %v", err))
	}

	// 创建流式卡片
	// 标题显示问题
	cardTitle := fmt.Sprintf("问题: %s", userMessage)
	// 内容从"回答:"开始
	answerHeader := "**回答:**\n\n"
	initialContent := answerHeader + "正在思考中..."

	// 添加时间戳到 context
	ctxWithTimestamp := context.WithValue(ctx, "timestamp", time.Now().UnixNano())

	cardID, err := h.client.CreateStreamingCard(ctxWithTimestamp, cardTitle, initialContent)
	if err != nil {
		logx.Error("Failed to create streaming card: %v", err)
		return h.client.SendTextMessage(ctx, receiveIDType, receiveID,
			fmt.Sprintf("创建卡片失败: %v", err))
	}

	// 发送卡片消息
	_, err = h.client.SendCardMessage(ctx, receiveIDType, receiveID, cardID)
	if err != nil {
		logx.Error("Failed to send card message: %v", err)
		return err
	}

	// 流式接收并更新卡片
	var fullResponse strings.Builder
	fullResponse.WriteString(answerHeader)

	// 用于收集AI响应（不包含header）
	var aiResponse strings.Builder

	updateTicker := time.NewTicker(300 * time.Millisecond) // 每 300ms 更新一次
	defer updateTicker.Stop()

	sequence := 0
	lastUpdate := ""

	for {
		select {
		case content, ok := <-responseCh:
			if !ok {
				// 流结束,发送最终更新
				finalContent := fullResponse.String()
				finalContent += fmt.Sprintf("\n\n---\n⏰ *%s*", time.Now().Format("2006-01-02 15:04:05"))
				sequence++
				if err := h.client.UpdateCardElement(ctxWithTimestamp, cardID, "markdown_content", finalContent, sequence); err != nil {
					logx.Error("Failed to send final update: %v", err)
				}

				// 保存AI响应到数据库
				if userLog != nil && aiResponse.Len() > 0 {
					_, err := h.chatLogService.CreateAIMessage(username, source, aiResponse.String(), userLog.ID)
					if err != nil {
						logx.Error("Failed to save AI response to database: %v", err)
					}
				}

				return nil
			}
			fullResponse.WriteString(content)
			aiResponse.WriteString(content)

		case <-updateTicker.C:
			// 定时更新卡片
			currentContent := fullResponse.String()
			if currentContent != lastUpdate && len(currentContent) > len(answerHeader) {
				sequence++
				if err := h.client.UpdateCardElement(ctxWithTimestamp, cardID, "markdown_content", currentContent, sequence); err != nil {
					logx.Warn("Failed to update card element: %v", err)
				} else {
					lastUpdate = currentContent
				}
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// sendHelpMessage 发送帮助消息
func (h *MessageHandler) sendHelpMessage(ctx context.Context, event *larkim.P2MessageReceiveV1, username, source string, userLog *model.ChatLog) error {
	receiveIDType := "open_id"
	receiveID := *event.Event.Sender.SenderId.OpenId
	if *event.Event.Message.ChatType == "group" {
		receiveIDType = "chat_id"
		receiveID = *event.Event.Message.ChatId
	}

	helpText := GetHelpMessage()
	_, err := h.client.SendMarkdownMessage(ctx, receiveIDType, receiveID, "使用帮助", helpText)

	// 保存帮助消息到数据库
	if userLog != nil {
		_, saveErr := h.chatLogService.CreateAIMessage(username, source, helpText, userLog.ID)
		if saveErr != nil {
			logx.Error("Failed to save help message to database: %v", saveErr)
		}
	}

	return err
}

// GetUserInfo 获取用户信息
func (h *MessageHandler) GetUserInfo(ctx context.Context, userOpenID string) (*larkcontact.User, error) {
	req := larkcontact.NewGetUserReqBuilder().
		UserId(userOpenID).
		UserIdType("open_id").
		Build()

	resp, err := h.client.client.Contact.User.Get(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	if !resp.Success() {
		return nil, fmt.Errorf("failed to get user info: code=%d, msg=%s", resp.Code, resp.Msg)
	}

	return resp.Data.User, nil
}

// GetHelpMessage 获取帮助信息
func GetHelpMessage() string {
	return `# ZenOps 飞书机器人使用指南

## 功能说明
ZenOps 是一个运维工具集成平台,支持通过飞书机器人与云平台交互。

## 支持的功能

### 1. LLM 智能对话
直接发送问题,机器人会通过 AI 大模型为您解答。

示例:
- "帮我查询阿里云 ECS 列表"
- "列出腾讯云的 CVM 实例"
- "查看 Jenkins 最近的构建任务"

### 2. 云平台查询
支持查询以下云平台资源:
- 阿里云: ECS、RDS 等
- 腾讯云: CVM、CDB 等
- Jenkins: 构建任务、Job 状态等

## 使用提示
- 发送 "帮助" 或 "help" 查看此帮助信息
- 私聊或在群里 @机器人 都可以使用

## 技术支持
如有问题,请联系运维团队。
`
}
