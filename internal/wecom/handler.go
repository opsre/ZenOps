package wecom

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"cnb.cool/zhiqiangwang/pkg/logx"
	"github.com/eryajf/zenops/internal/config"
	"github.com/eryajf/zenops/internal/imcp"
	"github.com/eryajf/zenops/internal/llm"
	"github.com/eryajf/zenops/internal/model"
	"github.com/eryajf/zenops/internal/service"
	"github.com/google/uuid"
)

// ConversationState 对话状态
type ConversationState struct {
	Question         string
	Buffer           strings.Builder
	NotificationChan chan string
	IsVisited        bool
	IsDone           bool
	Mutex            sync.Mutex
}

// MessageHandler 企业微信消息处理器
type MessageHandler struct {
	config              *config.Config
	Client              *AIBotClient // 导出以便外部访问
	mcpServer           *imcp.MCPServer
	llmClient           *llm.Client
	chatLogService      *service.ChatLogService
	conversationManager sync.Map // 存储对话状态
	msgIDCache          sync.Map // 消息ID缓存,用于去重
}

// NewMessageHandler 创建消息处理器
func NewMessageHandler(cfg *config.Config, mcpServer *imcp.MCPServer) (*MessageHandler, error) {
	client, err := NewAIBotClient(context.Background(), cfg.Wecom.Token, cfg.Wecom.EncodingAESKey)
	if err != nil {
		return nil, err
	}

	// 初始化 LLM 客户端
	var llmClient *llm.Client
	if cfg.LLM.Enabled {
		llmConfig := &llm.Config{
			Model:   cfg.LLM.Model,
			APIKey:  cfg.LLM.APIKey,
			BaseURL: cfg.LLM.BaseURL,
		}
		llmClient = llm.NewClient(llmConfig, mcpServer)
		logx.Info("LLM client initialized for Wecom, model %s", cfg.LLM.Model)
	}

	handler := &MessageHandler{
		config:         cfg,
		Client:         client,
		mcpServer:      mcpServer,
		llmClient:      llmClient,
		chatLogService: service.NewChatLogService(),
	}

	// 启动消息缓存清理协程
	go handler.startMessageCleanup()

	return handler, nil
}

// HandleTextMessage 处理文本消息
func (h *MessageHandler) HandleTextMessage(ctx context.Context, req *UserReq) (string, error) {
	// 生成对话ID
	conversationID := uuid.New().String()

	// 缓存消息ID映射
	h.msgIDCache.Store(req.Msgid, conversationID)

	// 创建对话状态
	state := &ConversationState{
		Question:         req.Text.Content,
		NotificationChan: make(chan string),
		IsVisited:        false,
		IsDone:           false,
	}
	h.conversationManager.Store(conversationID, state)

	// 异步处理消息 - 使用独立的 background context,避免请求 context 取消
	go h.processMessage(context.Background(), req, conversationID, state)

	// 立即返回初始响应
	return h.Client.MakeStreamResp("", req.Msgid, "<think>正在思考您的问题,请稍候...</think>", false)
}

// HandleStreamRequest 处理流式轮询请求
func (h *MessageHandler) HandleStreamRequest(ctx context.Context, req *UserReq) (string, error) {
	// 从缓存中获取对话ID
	conversationIDVal, ok := h.msgIDCache.Load(req.Stream.Id)
	if !ok {
		logx.Warn("Conversation not found for stream ID: %s", req.Stream.Id)
		return h.Client.MakeStreamResp("", req.Stream.Id, "服务内部异常，请稍后重试", true)
	}

	conversationID := conversationIDVal.(string)
	val, ok := h.conversationManager.Load(conversationID)
	if !ok {
		return h.Client.MakeStreamResp("", req.Stream.Id, "服务暂时不可用，请稍后重试", true)
	}

	state := val.(*ConversationState)
	state.Mutex.Lock()
	content := state.Buffer.String()
	isDone := state.IsDone
	state.Mutex.Unlock()

	if content == "" {
		content = "<think>正在思考您的问题,请稍候...</think>"
	}

	// 如果完成,添加尾注并清理状态
	if isDone {
		h.conversationManager.Delete(conversationID)
		h.msgIDCache.Delete(req.Stream.Id)
		content += "\n\n---  \n\n✅ 回答完成 | 由 ZenOps 智能机器人提供"
	}

	return h.Client.MakeStreamResp("", req.Stream.Id, content, isDone)
}

// processMessage 处理用户消息
func (h *MessageHandler) processMessage(ctx context.Context, req *UserReq, conversationID string, state *ConversationState) {
	userMessage := strings.TrimSpace(req.Text.Content)
	if userMessage == "" {
		state.Mutex.Lock()
		state.IsDone = true
		state.Mutex.Unlock()
		return
	}

	logx.Info("Processing message from Wecom: user %s, message %s", req.From.Userid, userMessage)

	// 确定消息来源（私聊/群聊）
	source := "私聊"
	if req.Chattype != "" && req.Chattype != "single" {
		source = "群聊"
	}

	// 保存用户消息到数据库
	userLog, err := h.chatLogService.CreateUserMessage(req.From.Userid, source, userMessage)
	if err != nil {
		logx.Error("Failed to save user message to database: %v", err)
	}

	// 特殊命令处理
	if strings.Contains(userMessage, "帮助") || strings.Contains(userMessage, "help") {
		h.sendHelpMessage(state, req.From.Userid, source, userLog)
		return
	}

	// 如果启用了 LLM,使用 LLM 处理
	if h.config.LLM.Enabled && h.llmClient != nil {
		h.processLLMMessage(ctx, userMessage, state, req.From.Userid, source, userLog)
		return
	}

	// 否则返回默认消息
	state.Mutex.Lock()
	state.Buffer.WriteString("ZenOps 企业微信机器人已收到您的消息。\n\n")
	state.Buffer.WriteString("当前未启用 LLM 对话功能,请联系管理员配置。")
	state.IsDone = true
	state.Mutex.Unlock()
}

// processLLMMessage 使用 LLM 处理消息
func (h *MessageHandler) processLLMMessage(ctx context.Context, userMessage string, state *ConversationState, username, source string, userLog *model.ChatLog) {
	// 调用 LLM 流式对话
	responseCh, err := h.llmClient.ChatWithToolsAndStream(ctx, userMessage)
	if err != nil {
		logx.Error("Failed to call LLM: %v", err)
		state.Mutex.Lock()
		state.Buffer.WriteString(fmt.Sprintf("❌ LLM 调用失败: %v", err))
		state.IsDone = true
		state.Mutex.Unlock()
		return
	}

	// 用于收集完整的AI响应
	var aiResponse strings.Builder

	// 流式接收并缓存响应
	for event := range responseCh {
		state.Mutex.Lock()
		state.Buffer.WriteString(event)
		aiResponse.WriteString(event)
		if state.IsVisited {
			select {
			case state.NotificationChan <- event:
			default:
			}
		}
		state.Mutex.Unlock()
	}

	// 保存AI响应到数据库
	if userLog != nil && aiResponse.Len() > 0 {
		var parentID uint
		parentID = userLog.ID
		_, err := h.chatLogService.CreateAIMessage(username, source, aiResponse.String(), parentID)
		if err != nil {
			logx.Error("Failed to save AI response to database: %v", err)
		}
	}

	// 标记完成
	state.Mutex.Lock()
	state.IsDone = true
	state.Mutex.Unlock()
	close(state.NotificationChan)

	logx.Info("LLM conversation completed for Wecom")
}

// sendHelpMessage 发送帮助消息
func (h *MessageHandler) sendHelpMessage(state *ConversationState, username, source string, userLog *model.ChatLog) {
	helpText := GetHelpMessage()
	state.Mutex.Lock()
	state.Buffer.WriteString(helpText)
	state.IsDone = true
	state.Mutex.Unlock()

	// 保存帮助消息到数据库
	if userLog != nil {
		_, err := h.chatLogService.CreateAIMessage(username, source, helpText, userLog.ID)
		if err != nil {
			logx.Error("Failed to save help message to database: %v", err)
		}
	}
}

// startMessageCleanup 启动消息缓存清理协程
func (h *MessageHandler) startMessageCleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		// 清理超过30分钟的对话状态
		now := time.Now()
		h.conversationManager.Range(func(key, value interface{}) bool {
			state := value.(*ConversationState)
			state.Mutex.Lock()
			isDone := state.IsDone
			state.Mutex.Unlock()

			// 如果已完成超过30分钟,删除
			if isDone {
				h.conversationManager.Delete(key)
			}
			return true
		})

		logx.Debug("Wecom message cleanup completed at %s", now.Format("2006-01-02 15:04:05"))
	}
}

// GetHelpMessage 获取帮助信息
func GetHelpMessage() string {
	return `# ZenOps 企业微信机器人使用指南

## 功能说明
ZenOps 是一个运维工具集成平台,支持通过企业微信智能机器人与云平台交互。

## 支持的功能

### 1. LLM 智能对话
直接发送问题,机器人会通过 AI 大模型为您解答。

**示例:**
• 帮我查询阿里云 ECS 列表
• 列出腾讯云的 CVM 实例
• 查看 Jenkins 最近的构建任务

### 2. 云平台查询
支持查询以下云平台资源:
• **阿里云**: ECS、RDS 等
• **腾讯云**: CVM、CDB 等
• **Jenkins**: 构建任务、Job 状态等

## 使用提示
• 发送 "帮助" 或 "help" 查看此帮助信息
• 私聊机器人即可使用

## 技术支持
如有问题,请联系运维团队。
`
}
