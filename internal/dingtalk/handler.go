package dingtalk

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cnb.cool/zhiqiangwang/pkg/logx"
	"github.com/eryajf/zenops/internal/config"
	"github.com/eryajf/zenops/internal/imcp"
	"github.com/eryajf/zenops/internal/llm"
	"github.com/eryajf/zenops/internal/service"
	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
)

// MessageHandler 消息处理器
type MessageHandler struct {
	client         *Client
	parser         *IntentParser
	mcpServer      *imcp.MCPServer
	config         *config.Config
	streamMgr      *StreamManager
	llmClient      *llm.Client
	chatLogService *service.ChatLogService
}

// // NewMessageHandler 创建消息处理器
// func NewMessageHandler(cfg *config.Config, mcpServer *imcp.MCPServer) (*MessageHandler, error) {
// 	client := NewClient(
// 		cfg.DingTalk.AppKey,
// 		cfg.DingTalk.AppSecret,
// 		cfg.DingTalk.AgentID,
// 	)

// 	crypto, err := NewCallbackCrypto(
// 		cfg.DingTalk.Callback.Token,
// 		cfg.DingTalk.Callback.AESKey,
// 		cfg.DingTalk.AppKey,
// 	)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to create callback crypto: %w", err)
// 	}

// 	// 初始化 LLM 客户端
// 	var llmClient *llm.Client
// 	if cfg.LLM.Enabled && cfg.DingTalk.EnableLLMConversation {
// 		llmConfig := &llm.Config{
// 			Provider: cfg.LLM.Provider,
// 			Model:    cfg.LLM.Model,
// 			APIKey:   cfg.LLM.APIKey,
// 			BaseURL:  cfg.LLM.BaseURL,
// 		}
// 		llmClient = llm.NewClient(llmConfig, mcpServer)
// 		logx.Info("LLM client initialized provider %s, model %s",
// 			cfg.LLM.Provider,
// 			cfg.LLM.Model)
// 	}

// 	return &MessageHandler{
// 		client:    client,
// 		crypto:    crypto,
// 		parser:    NewIntentParser(),
// 		mcpServer: mcpServer,
// 		config:    cfg,
// 		streamMgr: NewStreamManager(client),
// 		llmClient: llmClient,
// 	}, nil
// }

// HandleMessage 处理消息
func (h *MessageHandler) HandleMessage(ctx context.Context, msg *CallbackMessage) (*CallbackResponse, error) {
	logx.Info("Handling message: sender %s, msg_id %s, conversation_id %s",
		msg.SenderNick,
		msg.MsgID,
		msg.ConversationID)

	// 提取用户消息(去除 @机器人)
	userMessage := ExtractUserMessage(msg)
	if userMessage == "" {
		return CreateTextResponse("请输入您的查询内容"), nil
	}

	logx.Debug("User message %s", userMessage)

	// 特殊命令处理
	if strings.Contains(userMessage, "帮助") || strings.Contains(userMessage, "help") {
		return CreateMarkdownResponse("使用帮助", GetHelpMessage()), nil
	}

	// 如果启用了 LLM,使用 LLM 处理
	if h.config.LLM.Enabled && h.llmClient != nil {
		// 如果启用了流式卡片,使用卡片流式交互
		if h.config.DingTalk.CardTemplateID != "" {
			go h.processLLMWithStreamCard(ctx, msg, userMessage)
			return CreateTextResponse("🤖 正在思考中,请稍候..."), nil
		}
		// 否则使用普通流式消息
		go h.processLLMWithStream(ctx, msg, userMessage)
		return CreateTextResponse("🤖 正在思考中,请稍候..."), nil
	}

	// 传统的意图解析模式
	intent, err := h.parser.Parse(userMessage)
	if err != nil {
		logx.Warn("Failed to parse intent: %v", err)
		return CreateTextResponse(fmt.Sprintf("抱歉,%s\n\n发送\"帮助\"查看使用说明", err.Error())), nil
	}

	// 立即返回确认消息
	go h.processQueryAsync(ctx, msg, intent)

	return CreateTextResponse(fmt.Sprintf("🔍 正在查询 %s %s,请稍候...", h.getProviderName(intent.Provider), h.getResourceName(intent.Resource))), nil
}

// processQueryAsync 异步处理查询
func (h *MessageHandler) processQueryAsync(ctx context.Context, msg *CallbackMessage, intent *Intent) {
	logx.Info("Processing query asynchronously mcp_tool %s, params %v",
		intent.MCPTool,
		intent.Params)

	// 创建流式推送
	streamID := fmt.Sprintf("stream_%s_%d", msg.MsgID, time.Now().Unix())

	// 发送进度消息
	_ = h.streamMgr.Send(ctx, msg.ConversationID, streamID, "⏳ 正在连接服务...\n\n", false)

	// 调用 MCP 工具
	result, err := h.callMCPTool(ctx, intent)
	if err != nil {
		logx.Error("Failed to call MCP tool: %v", err)
		_ = h.streamMgr.Send(ctx, msg.ConversationID, streamID,
			fmt.Sprintf("❌ 查询失败: %v", err), true)
		return
	}

	// 格式化结果
	formatted := h.formatResult(intent, result)

	// 流式发送结果
	_ = h.streamMgr.SendInChunks(ctx, msg.ConversationID, streamID, formatted)
}

// callMCPTool 调用 MCP 工具
func (h *MessageHandler) callMCPTool(ctx context.Context, intent *Intent) (string, error) {
	logx.Debug("Calling MCP tool: tool %s, params %v",
		intent.MCPTool,
		intent.Params)

	// 使用 MCP Server 的 CallTool 方法
	result, err := h.mcpServer.CallTool(ctx, intent.MCPTool, h.convertParams(intent.Params))
	if err != nil {
		return "", fmt.Errorf("failed to call MCP tool: %w", err)
	}

	// 提取文本结果
	if len(result.Content) > 0 {
		if textContent, ok := result.Content[0].(mcp.TextContent); ok {
			return textContent.Text, nil
		}
	}

	return "查询完成,但未返回结果", nil
}

// convertParams 转换参数为 map[string]any
func (h *MessageHandler) convertParams(params map[string]string) map[string]any {
	result := make(map[string]any)
	for k, v := range params {
		result[k] = v
	}
	return result
}

// formatResult 格式化查询结果
func (h *MessageHandler) formatResult(intent *Intent, result string) string {
	var builder strings.Builder

	// 添加头部
	builder.WriteString(fmt.Sprintf("✅ **%s %s 查询完成**\n\n",
		h.getProviderName(intent.Provider),
		h.getResourceName(intent.Resource)))

	// 添加结果内容
	builder.WriteString(result)

	// 添加时间戳
	builder.WriteString(fmt.Sprintf("\n\n---\n⏰ 查询时间: %s",
		time.Now().Format("2006-01-02 15:04:05")))

	return builder.String()
}

// getProviderName 获取云平台名称
func (h *MessageHandler) getProviderName(provider string) string {
	names := map[string]string{
		"aliyun":  "阿里云",
		"tencent": "腾讯云",
		"jenkins": "Jenkins",
	}
	if name, ok := names[provider]; ok {
		return name
	}
	return provider
}

// getResourceName 获取资源名称
func (h *MessageHandler) getResourceName(resource string) string {
	names := map[string]string{
		"ecs":   "ECS",
		"rds":   "RDS",
		"cvm":   "CVM",
		"cdb":   "CDB",
		"job":   "Job",
		"build": "Build",
	}
	if name, ok := names[resource]; ok {
		return name
	}
	return resource
}

// processLLMWithStream 使用普通流式消息处理 LLM 对话
func (h *MessageHandler) processLLMWithStream(ctx context.Context, msg *CallbackMessage, userMessage string) {
	logx.Info("Processing LLM with stream: user %s, message %s",
		msg.SenderNick,
		userMessage)

	// 创建流式推送
	streamID := fmt.Sprintf("llm_stream_%s_%d", msg.MsgID, time.Now().Unix())

	// 发送初始消息
	_ = h.streamMgr.Send(ctx, msg.ConversationID, streamID, "🤖 正在思考...\n\n", false)

	// 调用 LLM 流式对话
	responseCh, err := h.llmClient.ChatWithToolsAndStream(ctx, userMessage)
	if err != nil {
		logx.Error("Failed to call LLM: %v", err)
		_ = h.streamMgr.Send(ctx, msg.ConversationID, streamID,
			fmt.Sprintf("❌ LLM 调用失败: %v", err), true)
		return
	}

	// 累积响应内容
	var fullResponse strings.Builder
	fullResponse.WriteString(fmt.Sprintf("**问题:** %s\n\n", userMessage))
	fullResponse.WriteString("**回答:**\n\n")

	headerLen := fullResponse.Len()

	// 流式接收并发送
	for content := range responseCh {
		fullResponse.WriteString(content)
		// 每接收一定量内容就发送一次更新
		if fullResponse.Len()-headerLen > 500 {
			_ = h.streamMgr.Send(ctx, msg.ConversationID, streamID, fullResponse.String(), false)
		}
	}

	// 发送最终内容
	fullResponse.WriteString(fmt.Sprintf("\n\n---\n⏰ %s", time.Now().Format("2006-01-02 15:04:05")))
	_ = h.streamMgr.Send(ctx, msg.ConversationID, streamID, fullResponse.String(), true)

	logx.Info("LLM conversation completed user %s", msg.SenderNick)
}

// processLLMWithStreamCard 使用流式卡片处理 LLM 对话
func (h *MessageHandler) processLLMWithStreamCard(ctx context.Context, msg *CallbackMessage, userMessage string) {
	logx.Info("Processing LLM with stream card: user %s, message %s",
		msg.SenderNick,
		userMessage)

	// 生成唯一追踪ID
	trackID := uuid.New().String()

	// 获取访问令牌
	accessToken, err := h.client.GetAccessToken(ctx)
	if err != nil {
		logx.Error("Failed to get access token: %v", err)
		// 降级为普通流式消息
		h.processLLMWithStream(ctx, msg, userMessage)
		return
	}

	// 创建流式卡片客户端
	cardClient, err := NewStreamCardClient()
	if err != nil {
		logx.Error("Failed to create stream card client: %v", err)
		h.processLLMWithStream(ctx, msg, userMessage)
		return
	}

	// 构建 OpenSpaceID
	var openSpaceID string
	conversationType := msg.ConversationType
	if conversationType == "" {
		conversationType = "2" // 默认群聊
	}

	if conversationType == "2" {
		openSpaceID = fmt.Sprintf("dtv1.card//IM_GROUP.%s", msg.ConversationID)
	} else {
		openSpaceID = fmt.Sprintf("dtv1.card//IM_ROBOT.%s", msg.SenderStaffID)
	}

	logx.Debug("Creating stream card with track_id %s, open_space_id %s, conversation_type %s",
		trackID,
		openSpaceID,
		conversationType)

	// 创建并投放卡片
	createReq := &CreateAndDeliverCardRequest{
		CardTemplateID:   h.config.DingTalk.CardTemplateID,
		OutTrackID:       trackID,
		ConversationID:   msg.ConversationID,
		SenderStaffID:    msg.SenderStaffID,
		RobotCode:        msg.RobotCode,
		OpenSpaceID:      openSpaceID,
		ConversationType: conversationType,
		CardData: map[string]string{
			"content": "",
		},
	}

	if err := cardClient.CreateAndDeliverCard(accessToken, createReq); err != nil {
		logx.Error("Failed to create card: %v", err)
		// 降级为普通流式消息
		h.processLLMWithStream(ctx, msg, userMessage)
		return
	}

	// 发送初始状态
	initialContent := fmt.Sprintf("**%s**\n\n正在思考中...", userMessage)
	if err := h.client.UpdateAIStreamCard(trackID, initialContent, false); err != nil {
		logx.Warn("Failed to update initial card: %v", err)
	}

	// 调用 LLM 流式对话
	responseCh, err := h.llmClient.ChatWithToolsAndStream(ctx, userMessage)
	if err != nil {
		logx.Error("Failed to call LLM %v", err)
		errorMsg := fmt.Sprintf("**%s**\n\n❌ 调用失败: %v", userMessage, err)
		_ = h.client.UpdateAIStreamCardWithError(trackID, errorMsg)
		return
	}

	// 构建响应内容
	questionHeader := fmt.Sprintf("**%s**\n\n", userMessage)
	fullContent := questionHeader

	// 改进的缓冲机制
	updateBuffer := ""
	minUpdateInterval := 200 * time.Millisecond // 减少到200ms,提升响应速度
	minBufferSize := 5                          // 至少累积5个字符再更新

	ticker := time.NewTicker(minUpdateInterval)
	defer ticker.Stop()

	for {
		select {
		case content, ok := <-responseCh:
			if !ok {
				// 流结束,发送最终更新
				if updateBuffer != "" {
					fullContent += updateBuffer
				}
				fullContent += fmt.Sprintf("\n\n---\n⏰ %s", time.Now().Format("2006-01-02 15:04:05"))
				if err := h.client.UpdateAIStreamCard(trackID, fullContent, true); err != nil {
					logx.Error("Failed to finalize card: %v", err)
				}
				return
			}

			// 累积到缓冲区
			updateBuffer += content

		case <-ticker.C:
			// 定时检查是否需要更新
			if updateBuffer != "" && len(updateBuffer) >= minBufferSize {
				fullContent += updateBuffer
				updateBuffer = ""

				// 更新卡片
				if err := h.client.UpdateAIStreamCard(trackID, fullContent, false); err != nil {
					logx.Warn("Failed to update card: %v", err)
				}
			}
		}
	}

}
