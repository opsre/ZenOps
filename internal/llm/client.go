package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"cnb.cool/zhiqiangwang/pkg/logx"
	"github.com/eryajf/zenops/internal/service"
	"github.com/mark3labs/mcp-go/mcp"
)

// MCPServer MCP服务器接口(避免循环导入)
type MCPServer interface {
	ListTools(ctx context.Context) (*mcp.ListToolsResult, error)
	ListEnabledTools(ctx context.Context) (*mcp.ListToolsResult, error)
	CallTool(ctx context.Context, name string, arguments map[string]any) (*mcp.CallToolResult, error)
}

// Client LLM 客户端
type Client struct {
	config    *Config
	mcpServer MCPServer
}

// Config LLM 配置
type Config struct {
	Model   string `mapstructure:"model"`
	APIKey  string `mapstructure:"api_key"`
	BaseURL string `mapstructure:"base_url"`
}

// NewClient 创建 LLM 客户端
func NewClient(config *Config, mcpServer MCPServer) *Client {
	return &Client{
		config:    config,
		mcpServer: mcpServer,
	}
}

// Message 消息结构
type Message struct {
	Role       string     `json:"role"`
	Content    any        `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`   // 用于 assistant 角色的工具调用
	ToolCallID string     `json:"tool_call_id,omitempty"` // 用于 tool 角色的响应
	Name       string     `json:"name,omitempty"`         // 用于 tool 角色的函数名
}

// ToolCall 工具调用
type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// ChatRequest 聊天请求
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Stream      bool      `json:"stream"`
	Temperature float64   `json:"temperature,omitempty"`
	Tools       []Tool    `json:"tools,omitempty"`
}

// Tool 工具定义
type Tool struct {
	Type     string   `json:"type"`
	Function Function `json:"function"`
}

// Function 函数定义
type Function struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// ChatResponse 聊天响应
type ChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role      string     `json:"role"`
			Content   string     `json:"content"`
			ToolCalls []ToolCall `json:"tool_calls,omitempty"`
		} `json:"message"`
		Delta struct {
			Role      string     `json:"role,omitempty"`
			Content   string     `json:"content,omitempty"`
			ToolCalls []ToolCall `json:"tool_calls,omitempty"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

// Chat 与 LLM 对话 (非流式)
func (c *Client) Chat(ctx context.Context, messages []Message) (string, error) {
	// 获取 MCP 工具列表
	tools, err := c.getMCPTools(ctx)
	if err != nil {
		logx.Warn("Failed to get MCP tools: %v", err)
		tools = nil // 即使获取工具失败,也继续进行对话
	}

	req := &ChatRequest{
		Model:    c.config.Model,
		Messages: messages,
		Stream:   false,
		Tools:    tools,
	}

	// TODO: 这里需要根据不同的 provider 调用不同的 API
	// 当前是一个简化的示例实现
	logx.Debug("Chat request %v", req)

	return "暂未实现完整的 LLM 调用,请配置实际的 API 调用逻辑", nil
}

// ChatStream 与 LLM 流式对话
func (c *Client) ChatStream(ctx context.Context, messages []Message) (<-chan string, error) {
	// 获取 MCP 工具列表
	tools, err := c.getMCPTools(ctx)
	if err != nil {
		logx.Warn("Failed to get MCP tools: %v", err)
		tools = nil
	}

	req := &ChatRequest{
		Model:    c.config.Model,
		Messages: messages,
		Stream:   true,
		Tools:    tools,
	}

	logx.Debug("Chat stream request %v", req)

	// 创建输出通道
	responseCh := make(chan string, 100)

	// TODO: 这里需要根据不同的 provider 调用不同的 API
	// 当前是一个简化的示例实现
	go func() {
		defer close(responseCh)
		// 模拟流式响应
		responseCh <- "暂未实现完整的 LLM 流式调用,请配置实际的 API 调用逻辑"
	}()

	return responseCh, nil
}

// ChatWithMCPTools 使用 MCP 工具与 LLM 对话
func (c *Client) ChatWithMCPTools(ctx context.Context, userMessage string) (<-chan string, error) {
	responseCh := make(chan string, 100)

	go func() {
		defer close(responseCh)

		// 初始化消息历史
		messages := []Message{
			{
				Role:    "system",
				Content: c.buildSystemPrompt(),
			},
			{
				Role:    "user",
				Content: userMessage,
			},
		}

		maxIterations := 10 // 最大工具调用迭代次数
		for i := 0; i < maxIterations; i++ {
			// 调用 LLM
			resp, err := c.callLLMWithTools(ctx, messages)
			if err != nil {
				responseCh <- fmt.Sprintf("❌ LLM 调用失败: %v", err)
				return
			}

			// 检查是否有工具调用
			if len(resp.ToolCalls) > 0 {
				// 处理工具调用
				for _, toolCall := range resp.ToolCalls {
					responseCh <- fmt.Sprintf("> 🔧 调用工具: %s\n", toolCall.Function.Name)

					result, err := c.executeToolCall(ctx, toolCall)
					if err != nil {
						responseCh <- fmt.Sprintf("❌ 工具调用失败: %v\n", err)
						continue
					}

					// 添加工具调用结果到消息历史
					messages = append(messages, Message{
						Role:    "tool",
						Content: result,
					})
				}
				// 继续循环,让 LLM 处理工具结果
				continue
			}

			// 没有工具调用,返回最终响应
			if resp.Content != "" {
				responseCh <- resp.Content
			}
			return
		}

		responseCh <- "\n\n⚠️ 达到最大工具调用次数限制"
	}()

	return responseCh, nil
}

// LLMResponse LLM 响应结构
type LLMResponse struct {
	Content   string
	ToolCalls []ToolCall
}

// callLLMWithTools 调用 LLM (支持工具)
func (c *Client) callLLMWithTools(ctx context.Context, messages []Message) (*LLMResponse, error) {
	// TODO: 实现实际的 LLM API 调用
	// 这里是一个简化的示例实现
	return &LLMResponse{
		Content:   "暂未实现完整的 LLM 调用",
		ToolCalls: nil,
	}, nil
}

// executeToolCall 执行工具调用
func (c *Client) executeToolCall(ctx context.Context, toolCall ToolCall) (string, error) {
	// 解析参数
	var params map[string]any
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &params); err != nil {
		return "", fmt.Errorf("failed to parse tool arguments: %w", err)
	}

	logx.Debug("Executing tool call, tool %s, params %v",
		toolCall.Function.Name,
		params)

	// 记录调用开始时间
	startTime := time.Now()

	// 调用 MCP 工具
	result, err := c.mcpServer.CallTool(ctx, toolCall.Function.Name, params)
	latency := time.Since(startTime).Milliseconds()

	// 解析 server_name 和 tool_name
	// 外部 MCP 工具格式: "prefix_toolname"，例如 "aliyun-ack_list_clusters"
	// 内置工具没有前缀，例如 "search_ecs_by_ip"
	serverName := "zenops" // 默认为内置工具
	toolName := toolCall.Function.Name

	// 尝试从工具名中提取前缀（外部 MCP 工具）
	if idx := strings.Index(toolCall.Function.Name, "_"); idx > 0 {
		// 可能是外部工具，检查前缀是否包含连字符（如 "aliyun-ack"）
		prefix := toolCall.Function.Name[:idx]
		if strings.Contains(prefix, "-") {
			serverName = prefix
			toolName = toolCall.Function.Name[idx+1:]
		}
	}

	// 记录 MCP 调用日志
	mcpLogService := service.NewMCPLogService()
	logParams := &service.MCPLogParams{
		ServerName: serverName,
		ToolName:   toolName,
		Username:   "llm", // LLM 自动调用，用户信息需要从上下文传递
		Source:     "llm",
		Request:    params,
		Response:   result,
		Latency:    latency,
		Success:    err == nil,
	}
	if err != nil {
		logParams.ErrorMessage = err.Error()
	}
	if _, logErr := mcpLogService.CreateMCPLog(logParams); logErr != nil {
		logx.Warn("Failed to save MCP log: %v", logErr)
	}

	if err != nil {
		return "", fmt.Errorf("failed to call MCP tool: %w", err)
	}

	// 提取文本结果
	if len(result.Content) > 0 {
		if textContent, ok := result.Content[0].(mcp.TextContent); ok {
			return textContent.Text, nil
		}
	}

	return "工具执行完成,但未返回结果", nil
}

// getMCPTools 获取 MCP 工具列表（只返回启用的工具）
func (c *Client) getMCPTools(ctx context.Context) ([]Tool, error) {
	if c.mcpServer == nil {
		return nil, fmt.Errorf("MCP server not initialized")
	}

	// 获取启用的工具列表（会从数据库过滤被禁用的工具）
	toolList, err := c.mcpServer.ListEnabledTools(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list enabled MCP tools: %w", err)
	}

	var tools []Tool
	for _, tool := range toolList.Tools {
		// 转换 MCP 工具定义为 OpenAI 工具格式
		tools = append(tools, Tool{
			Type: "function",
			Function: Function{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  c.convertMCPSchemaToOpenAI(tool.InputSchema),
			},
		})
	}

	logx.Info("Loaded %d enabled MCP tools for LLM", len(tools))
	return tools, nil
}

// convertMCPSchemaToOpenAI 转换 MCP Schema 为 OpenAI 格式
func (c *Client) convertMCPSchemaToOpenAI(schema any) map[string]any {
	// 如果已经是 map 格式,直接返回
	if m, ok := schema.(map[string]any); ok {
		return m
	}

	// 如果是其他格式,尝试序列化再反序列化
	data, err := json.Marshal(schema)
	if err != nil {
		return map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		}
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		}
	}

	return result
}

// buildSystemPrompt 构建系统提示词
func (c *Client) buildSystemPrompt() string {
	var builder strings.Builder

	builder.WriteString("你是一个智能运维助手,可以帮助用户查询和管理云资源、CI/CD 任务等。\n\n")
	builder.WriteString("你可以使用以下工具来获取信息:\n")

	// 获取可用的工具列表
	if c.mcpServer != nil {
		tools, err := c.mcpServer.ListTools(context.Background())
		if err == nil {
			for _, tool := range tools.Tools {
				builder.WriteString(fmt.Sprintf("- %s: %s\n", tool.Name, tool.Description))
			}
		}
	}

	builder.WriteString("\n当用户询问相关信息时,请主动调用相应的工具来获取准确的数据。")
	builder.WriteString("回复时请简洁明了,使用 Markdown 格式化输出。")

	return builder.String()
}

// ParseSSEResponse 解析 SSE 响应流
func ParseSSEResponse(reader io.Reader, responseCh chan<- string) error {
	decoder := json.NewDecoder(reader)
	var buffer strings.Builder

	for {
		var resp ChatResponse
		if err := decoder.Decode(&resp); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		if len(resp.Choices) > 0 {
			delta := resp.Choices[0].Delta
			if delta.Content != "" {
				buffer.WriteString(delta.Content)
				responseCh <- delta.Content
			}

			// 处理工具调用
			// 当前版本暂不处理流式工具调用
			// if len(delta.ToolCalls) > 0 {
			// TODO: 处理流式工具调用
			// }
		}
	}

	return nil
}
