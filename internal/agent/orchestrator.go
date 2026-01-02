package agent

import (
	"context"
	"fmt"

	"cnb.cool/zhiqiangwang/pkg/logx"
	"github.com/eryajf/zenops/internal/imcp"
	"github.com/eryajf/zenops/internal/knowledge"
	"github.com/eryajf/zenops/internal/memory"
)

// Orchestrator Agent 编排器（简化版）
// TODO: 未来将使用 Eino Graph 实现完整的编排能力
type Orchestrator struct {
	memoryMgr     *memory.Manager
	knowledgeRet  *knowledge.Retriever
	mcpServer     *imcp.MCPServer
	maxIterations int
}

// NewOrchestrator 创建 Agent 编排器
func NewOrchestrator(
	memoryMgr *memory.Manager,
	knowledgeRet *knowledge.Retriever,
	mcpServer *imcp.MCPServer,
) *Orchestrator {
	return &Orchestrator{
		memoryMgr:    memoryMgr,
		knowledgeRet: knowledgeRet,
		mcpServer:    mcpServer,
		maxIterations: 10, // 最大迭代次数
	}
}

// Execute 执行对话（简化版，未使用 Eino Graph）
func (o *Orchestrator) Execute(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	logx.Info("🚀 Agent executing request from user: %s", req.Username)

	// 1. 检查 QA 缓存
	cachedAnswer, hit, err := o.memoryMgr.GetCachedAnswer(req.Username, req.Message)
	if err == nil && hit {
		logx.Info("✅ QA cache hit, returning cached answer")
		return &ChatResponse{Content: cachedAnswer}, nil
	}

	// 2. 加载对话历史
	chatLogs, err := o.memoryMgr.GetConversationHistory(req.ConversationID, 10)
	if err != nil {
		logx.Warn("Failed to load conversation history: %v", err)
	}

	// 转换为 memory.Message 格式
	var history []memory.Message
	for _, log := range chatLogs {
		history = append(history, memory.Message{
			Role:      o.chatTypeToRole(log.ChatType),
			Content:   log.Content,
			CreatedAt: log.CreatedAt,
		})
	}
	logx.Debug("Loaded %d messages from conversation history", len(history))

	// 3. 加载用户上下文
	userCtx, err := o.memoryMgr.GetUserContext(req.Username)
	if err != nil {
		logx.Warn("Failed to load user context: %v", err)
	}

	// 4. 检索知识库
	var knowledgeDocs []*knowledge.Document
	if o.knowledgeRet != nil {
		knowledgeDocs, err = o.knowledgeRet.Retrieve(ctx, req.Message)
		if err != nil {
			logx.Warn("Failed to retrieve knowledge: %v", err)
		} else {
			logx.Debug("Retrieved %d knowledge documents", len(knowledgeDocs))
		}
	}

	// 5. 构建消息（暂时保留，但不使用 - 用于未来的完整 Eino Graph 实现）
	_ = o.buildMessages(history, userCtx, knowledgeDocs, req.Message)

	// 6. 执行推理循环（简化版）
	// TODO: 替换为 Eino Graph 实现
	response := &ChatResponse{
		Content: "（简化版 Agent）您的消息已收到，完整的 Eino 集成正在开发中...",
	}

	// 7. 保存消息到历史
	if err := o.memoryMgr.SaveMessage(req.ConversationID, 1, req.Message, req.Username); err != nil {
		logx.Warn("Failed to save user message: %v", err)
	}
	if err := o.memoryMgr.SaveMessage(req.ConversationID, 2, response.Content, req.Username); err != nil {
		logx.Warn("Failed to save assistant message: %v", err)
	}

	// 8. 更新 QA 缓存
	if err := o.memoryMgr.UpdateQACache(req.Username, req.Message, response.Content); err != nil {
		logx.Warn("Failed to update QA cache: %v", err)
	}

	logx.Info("✅ Agent execution completed")
	return response, nil
}

// buildMessages 构建 LLM 消息（包含历史、上下文、知识库）
func (o *Orchestrator) buildMessages(
	history []memory.Message,
	userCtx *memory.UserContext,
	knowledgeDocs []*knowledge.Document,
	userMessage string,
) []Message {
	var messages []Message

	// System prompt
	systemPrompt := o.buildSystemPrompt(userCtx, knowledgeDocs)
	messages = append(messages, Message{
		Role:    "system",
		Content: systemPrompt,
	})

	// 历史消息
	for _, msg := range history {
		messages = append(messages, Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// 用户消息
	messages = append(messages, Message{
		Role:    "user",
		Content: userMessage,
	})

	return messages
}

// buildSystemPrompt 构建 System Prompt
func (o *Orchestrator) buildSystemPrompt(userCtx *memory.UserContext, knowledgeDocs []*knowledge.Document) string {
	prompt := "你是一个智能运维助手，可以帮助用户查询和管理云资源、CI/CD 任务等。\n\n"

	// 用户上下文
	if userCtx != nil {
		if userCtx.FavoriteRegion != "" {
			prompt += fmt.Sprintf("用户常用地域: %s\n", userCtx.FavoriteRegion)
		}
		if userCtx.DefaultVPC != "" {
			prompt += fmt.Sprintf("用户默认 VPC: %s\n", userCtx.DefaultVPC)
		}
	}

	// 知识库内容
	if len(knowledgeDocs) > 0 {
		prompt += "\n参考资料:\n"
		for _, doc := range knowledgeDocs {
			prompt += fmt.Sprintf("- %s: %s\n", doc.Title, doc.Content[:min(200, len(doc.Content))])
		}
	}

	prompt += "\n当用户询问相关信息时，请主动调用相应的工具来获取准确的数据。"
	prompt += "回复时请简洁明了，使用 Markdown 格式化输出。"

	return prompt
}

// min 返回两个整数的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// chatTypeToRole 将 ChatType 转换为 Role 字符串
func (o *Orchestrator) chatTypeToRole(chatType int) string {
	switch chatType {
	case 1:
		return "user"
	case 2:
		return "assistant"
	default:
		return "system"
	}
}
