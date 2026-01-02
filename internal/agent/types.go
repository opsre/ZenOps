package agent

import "time"

// ChatRequest 对话请求
type ChatRequest struct {
	Username       string `json:"username"`
	Message        string `json:"message"`
	ConversationID uint   `json:"conversation_id"`
	Source         string `json:"source"` // web/dingtalk/feishu/wecom
}

// ChatResponse 对话响应
type ChatResponse struct {
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// ToolCall 工具调用
type ToolCall struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// Message LLM 消息
type Message struct {
	Role       string     `json:"role"` // system/user/assistant/tool
	Content    any        `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	Name       string     `json:"name,omitempty"`
}

// AgentState Agent 执行状态
type AgentState struct {
	Username       string                 `json:"username"`
	ConversationID uint                   `json:"conversation_id"`
	UserMessage    string                 `json:"user_message"`
	Messages       []Message              `json:"messages"`
	History        []Message              `json:"history"`
	UserContext    map[string]string      `json:"user_context"`
	KnowledgeDocs  []map[string]any       `json:"knowledge_docs"`
	LLMResponse    *ChatResponse          `json:"llm_response"`
	ToolResults    map[string]string      `json:"tool_results"`
	Iteration      int                    `json:"iteration"`
	Extra          map[string]any         `json:"extra"`
}

// StreamCallbacks 流式回调接口
type StreamCallbacks interface {
	OnChatModelStream(content string)
	OnToolStart(toolName string)
	OnToolEnd(toolName string, result string)
	OnError(err error)
}

// Stats Agent 统计信息
type Stats struct {
	TotalQueries     int64         `json:"total_queries"`
	AvgLatency       time.Duration `json:"avg_latency"`
	ToolCallCount    int64         `json:"tool_call_count"`
	CacheHitRate     float64       `json:"cache_hit_rate"`
	AvgIterations    float64       `json:"avg_iterations"`
}
