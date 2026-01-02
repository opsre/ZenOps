package agent

import (
	"context"
	"fmt"
	"strings"

	"cnb.cool/zhiqiangwang/pkg/logx"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/eryajf/zenops/internal/knowledge"
	"github.com/eryajf/zenops/internal/memory"
)

// StreamHandler 流式对话处理器
type StreamHandler struct {
	orchestrator *Orchestrator
	chatModel    model.ChatModel
	tools        []schema.ToolInfo
}

// NewStreamHandler 创建流式处理器
func NewStreamHandler(orchestrator *Orchestrator, modelConfig ModelConfig) (*StreamHandler, error) {
	// 创建 Eino ChatModel (OpenAI 兼容)
	chatModel, err := openai.NewChatModel(context.Background(), &openai.ChatModelConfig{
		Model:   modelConfig.Model,
		APIKey:  modelConfig.APIKey,
		BaseURL: modelConfig.BaseURL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create chat model: %w", err)
	}

	return &StreamHandler{
		orchestrator: orchestrator,
		chatModel:    chatModel,
	}, nil
}

// ChatStream 流式对话（兼容现有接口）
func (s *StreamHandler) ChatStream(ctx context.Context, req *ChatRequest) (<-chan string, error) {
	responseCh := make(chan string, 100)

	go func() {
		defer close(responseCh)

		// 1. 检查 QA 缓存
		cachedAnswer, hit, err := s.orchestrator.memoryMgr.GetCachedAnswer(req.Username, req.Message)
		if err == nil && hit {
			logx.Info("✅ QA cache hit, returning cached answer")
			responseCh <- cachedAnswer
			return
		}

		// 2. 加载对话历史
		chatLogs, err := s.orchestrator.memoryMgr.GetConversationHistory(req.ConversationID, 10)
		if err != nil {
			logx.Warn("Failed to load conversation history: %v", err)
		}

		// 转换为 memory.Message 格式
		var history []memory.Message
		for _, log := range chatLogs {
			history = append(history, memory.Message{
				Role:      s.chatTypeToRole(log.ChatType),
				Content:   log.Content,
				CreatedAt: log.CreatedAt,
			})
		}

		// 3. 加载用户上下文
		userCtx, err := s.orchestrator.memoryMgr.GetUserContext(req.Username)
		if err != nil {
			logx.Warn("Failed to load user context: %v", err)
		}

		// 4. 检索知识库
		var knowledgeDocs []*knowledge.Document
		if s.orchestrator.knowledgeRet != nil {
			knowledgeDocs, err = s.orchestrator.knowledgeRet.Retrieve(ctx, req.Message)
			if err != nil {
				logx.Warn("Failed to retrieve knowledge: %v", err)
			}
		}

		// 5. 构建 MCP 工具列表
		tools, err := s.buildMCPToolInfos(req.Username)
		if err != nil {
			logx.Warn("Failed to build MCP tools: %v", err)
			tools = nil
		}
		s.tools = tools

		// 6. 构建消息
		messages := s.buildMessages(history, userCtx, knowledgeDocs, req.Message)

		// 7. 执行推理循环（支持多轮工具调用）
		fullResponse := s.executeLLMWithTools(ctx, messages, req.Username, responseCh)

		// 8. 保存消息到历史
		if err := s.orchestrator.memoryMgr.SaveMessage(req.ConversationID, 1, req.Message, req.Username); err != nil {
			logx.Warn("Failed to save user message: %v", err)
		}
		if err := s.orchestrator.memoryMgr.SaveMessage(req.ConversationID, 2, fullResponse, req.Username); err != nil {
			logx.Warn("Failed to save assistant message: %v", err)
		}

		// 9. 更新 QA 缓存
		if err := s.orchestrator.memoryMgr.UpdateQACache(req.Username, req.Message, fullResponse); err != nil {
			logx.Warn("Failed to update QA cache: %v", err)
		}
	}()

	return responseCh, nil
}

// executeLLMWithTools 执行 LLM 推理（支持工具调用）
func (s *StreamHandler) executeLLMWithTools(
	ctx context.Context,
	messages []*schema.Message,
	username string,
	responseCh chan<- string,
) string {
	var fullResponse strings.Builder
	maxIterations := s.orchestrator.maxIterations

	for i := 0; i < maxIterations; i++ {
		logx.Debug("🔄 Iteration %d/%d", i+1, maxIterations)

		// 构建请求选项
		opts := []model.Option{
			model.WithTemperature(0.7),
		}

		// 添加工具（如果有）
		if len(s.tools) > 0 {
			// 转换为 []*schema.ToolInfo
			var toolPtrs []*schema.ToolInfo
			for i := range s.tools {
				toolPtrs = append(toolPtrs, &s.tools[i])
			}
			opts = append(opts, model.WithTools(toolPtrs))
		}

		// 调用 ChatModel (流式)
		streamReader, err := s.chatModel.Stream(ctx, messages, opts...)
		if err != nil {
			errMsg := fmt.Sprintf("❌ LLM 调用失败: %v", err)
			responseCh <- errMsg
			logx.Error(errMsg)
			return errMsg
		}

		// 处理流式响应
		var currentContent strings.Builder
		var toolCalls []schema.ToolCall

		for {
			chunk, err := streamReader.Recv()
			if err != nil {
				break // 流结束
			}

			// 流式输出内容
			if chunk.Content != "" {
				currentContent.WriteString(chunk.Content)
				fullResponse.WriteString(chunk.Content)
				responseCh <- chunk.Content
			}

			// 收集工具调用
			if len(chunk.ToolCalls) > 0 {
				toolCalls = append(toolCalls, chunk.ToolCalls...)
			}
		}

		// 检查是否有工具调用
		if len(toolCalls) == 0 {
			// 没有工具调用，对话结束
			logx.Info("✅ LLM response completed without tool calls")
			break
		}

		// 处理工具调用
		logx.Info("🔧 Executing %d tool calls...", len(toolCalls))
		responseCh <- "\n\n"

		// 添加 assistant 消息到历史
		messages = append(messages, &schema.Message{
			Role:      schema.Assistant,
			Content:   currentContent.String(),
			ToolCalls: toolCalls,
		})

		// 执行所有工具调用
		for _, toolCall := range toolCalls {
			responseCh <- fmt.Sprintf("> 🔧 调用工具: **%s**\n", toolCall.Function.Name)

			toolResult, err := s.executeToolCall(ctx, &toolCall, username)
			if err != nil {
				errMsg := fmt.Sprintf("❌ 工具调用失败: %v\n\n", err)
				responseCh <- errMsg
				toolResult = errMsg
			} else {
				responseCh <- "✅ 工具执行完成\n\n"
			}

			// 添加工具结果到消息历史
			messages = append(messages, &schema.Message{
				Role:       schema.Tool,
				Content:    toolResult,
				ToolCallID: toolCall.ID,
				Name:       toolCall.Function.Name,
			})
		}
	}

	if len(fullResponse.String()) == 0 {
		return "⚠️ 达到最大工具调用次数限制"
	}

	return fullResponse.String()
}

// executeToolCall 执行工具调用
func (s *StreamHandler) executeToolCall(ctx context.Context, toolCall *schema.ToolCall, username string) (string, error) {
	// 查找对应的 MCP Tool Adapter
	adapter := NewMCPToolAdapter(
		toolCall.Function.Name,
		"",
		nil,
		s.orchestrator.mcpServer,
		username,
	)

	// 执行工具
	result, err := adapter.InvokableRun(ctx, toolCall.Function.Arguments)
	if err != nil {
		return "", fmt.Errorf("tool execution failed: %w", err)
	}

	return result, nil
}

// buildMessages 构建消息列表
func (s *StreamHandler) buildMessages(
	history []memory.Message,
	userCtx *memory.UserContext,
	knowledgeDocs []*knowledge.Document,
	userMessage string,
) []*schema.Message {
	var messages []*schema.Message

	// System prompt
	systemPrompt := s.orchestrator.buildSystemPrompt(userCtx, knowledgeDocs)
	messages = append(messages, &schema.Message{
		Role:    schema.System,
		Content: systemPrompt,
	})

	// 历史消息
	for _, msg := range history {
		messages = append(messages, &schema.Message{
			Role:    s.roleStringToEnum(msg.Role),
			Content: msg.Content,
		})
	}

	// 用户消息
	messages = append(messages, &schema.Message{
		Role:    schema.User,
		Content: userMessage,
	})

	return messages
}

// roleStringToEnum 将字符串 role 转换为 Eino schema.RoleType
func (s *StreamHandler) roleStringToEnum(role string) schema.RoleType {
	switch role {
	case "user":
		return schema.User
	case "assistant":
		return schema.Assistant
	case "system":
		return schema.System
	case "tool":
		return schema.Tool
	default:
		return schema.User
	}
}

// buildMCPToolInfos 构建 MCP 工具信息列表
func (s *StreamHandler) buildMCPToolInfos(username string) ([]schema.ToolInfo, error) {
	// 获取启用的 MCP 工具列表
	toolList, err := s.orchestrator.mcpServer.ListEnabledTools(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to list enabled MCP tools: %w", err)
	}

	var tools []schema.ToolInfo
	for _, tool := range toolList.Tools {
		// 构建 ToolInfo（暂时不设置 ParamsOneOf，因为类型不匹配）
		// TODO: 实现 MCP InputSchema 到 Eino ParamsOneOf 的转换
		info := schema.ToolInfo{
			Name: tool.Name,
			Desc: tool.Description,
			// ParamsOneOf: 需要类型转换
		}

		tools = append(tools, info)
		logx.Debug("📦 Loaded MCP tool: %s", tool.Name)
	}

	logx.Info("✅ Loaded %d enabled MCP tools for stream handler", len(tools))
	return tools, nil
}

// chatTypeToRole 将 ChatType 转换为 Role 字符串
func (s *StreamHandler) chatTypeToRole(chatType int) string {
	switch chatType {
	case 1:
		return "user"
	case 2:
		return "assistant"
	default:
		return "system"
	}
}

// ModelConfig LLM 模型配置
type ModelConfig struct {
	Model   string
	APIKey  string
	BaseURL string
}
