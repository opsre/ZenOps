package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cnb.cool/zhiqiangwang/pkg/logx"
	"github.com/eryajf/zenops/internal/agent"
	"github.com/eryajf/zenops/internal/config"
	"github.com/eryajf/zenops/internal/imcp"
	larkcontact "github.com/larksuite/oapi-sdk-go/v3/service/contact/v3"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
)

// MessageHandler 飞书消息处理器
type MessageHandler struct {
	client    *Client
	config    *config.Config
	mcpServer *imcp.MCPServer
}

// NewMessageHandler 创建消息处理器
func NewMessageHandler(cfg *config.Config, mcpServer *imcp.MCPServer) (*MessageHandler, error) {
	client := NewClient(cfg.Feishu.AppID, cfg.Feishu.AppSecret)

	return &MessageHandler{
		client:    client,
		config:    cfg,
		mcpServer: mcpServer,
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

	username := *event.Event.Sender.SenderId.OpenId

	// 特殊命令处理
	if strings.Contains(userMessage, "帮助") || strings.Contains(userMessage, "help") {
		return h.sendHelpMessage(ctx, event, username, source)
	}

	// 使用新的 Agent 系统处理消息
	agentSystem := agent.GetGlobalAgent()
	if agentSystem != nil && agentSystem.StreamHandler != nil {
		return h.processAgentMessage(ctx, event, userMessage, username, source)
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

// processAgentMessage 使用 Agent 系统处理消息(流式卡片更新)
func (h *MessageHandler) processAgentMessage(ctx context.Context, event *larkim.P2MessageReceiveV1, userMessage, username, source string) error {
	receiveIDType := "open_id"
	receiveID := *event.Event.Sender.SenderId.OpenId
	if *event.Event.Message.ChatType == "group" {
		receiveIDType = "chat_id"
		receiveID = *event.Event.Message.ChatId
	}

	// 获取全局 Agent 系统
	agentSystem := agent.GetGlobalAgent()
	if agentSystem == nil || agentSystem.StreamHandler == nil {
		return h.client.SendTextMessage(ctx, receiveIDType, receiveID,
			"❌ Agent 系统未初始化")
	}

	// 构建 Agent 请求
	agentReq := &agent.ChatRequest{
		Username:       username,
		Message:        userMessage,
		ConversationID: 0, // Feishu 不使用数据库会话管理
		Source:         source,
	}

	// 调用 Agent 流式对话
	responseCh, err := agentSystem.StreamHandler.ChatStream(ctx, agentReq)
	if err != nil {
		logx.Error("Failed to call Agent: %v", err)
		return h.client.SendTextMessage(ctx, receiveIDType, receiveID,
			fmt.Sprintf("❌ Agent 调用失败: %v", err))
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

				// Agent already handles message persistence
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
func (h *MessageHandler) sendHelpMessage(ctx context.Context, event *larkim.P2MessageReceiveV1, username, source string) error {
	receiveIDType := "open_id"
	receiveID := *event.Event.Sender.SenderId.OpenId
	if *event.Event.Message.ChatType == "group" {
		receiveIDType = "chat_id"
		receiveID = *event.Event.Message.ChatId
	}

	helpText := GetHelpMessage()
	_, err := h.client.SendMarkdownMessage(ctx, receiveIDType, receiveID, "使用帮助", helpText)
	// Agent already handles message persistence
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
