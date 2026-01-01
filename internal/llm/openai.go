package llm

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"time"

	"cnb.cool/zhiqiangwang/pkg/logx"
	openai "github.com/sashabaranov/go-openai"
)

// OpenAIClient OpenAI 兼容的客户端
type OpenAIClient struct {
	config *Config
	client *openai.Client
}

// NewOpenAIClient 创建新的 OpenAI 客户端
func NewOpenAIClient(config *Config) *OpenAIClient {
	clientConfig := openai.DefaultConfig(config.APIKey)

	// 配置 BaseURL
	if config.BaseURL != "" {
		// 直接使用配置的 BaseURL,不自动添加 /v1
		// 因为不同的 API 提供商可能有不同的路径格式
		// 例如:OpenAI 使用 /v1,智普 AI 使用 /api/paas/v4
		clientConfig.BaseURL = config.BaseURL
		logx.Debug("OpenAI client BaseURL: %s", config.BaseURL)
	}

	// 配置 HTTP 客户端 - 参考 chatgpt-dingtalk 的实现
	// 关键:禁用 HTTP/2,强制使用 HTTP/1.1 以避免 INTERNAL_ERROR
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		// 禁用 HTTP/2 - 设置空的 TLSNextProto map 会阻止 HTTP/2
		TLSNextProto: make(map[string]func(authority string, c *tls.Conn) http.RoundTripper),
	}

	clientConfig.HTTPClient = &http.Client{
		Transport: transport,
		Timeout:   600 * time.Second,
	}

	client := openai.NewClientWithConfig(clientConfig)

	logx.Info("OpenAI client initialized, model %s", config.Model)

	return &OpenAIClient{
		config: config,
		client: client,
	}
}

// convertContent 转换 any 内容为字符串
func convertContent(content any) string {
	if content == nil {
		return ""
	}
	if s, ok := content.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", content)
}

// ChatStream 流式对话
func (c *OpenAIClient) ChatStream(ctx context.Context, req *ChatRequest) (<-chan string, <-chan error, error) {
	messages := make([]openai.ChatCompletionMessage, 0, len(req.Messages))

	// 转换消息格式
	for _, msg := range req.Messages {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: convertContent(msg.Content),
		})
	}

	// 构建请求
	openaiReq := openai.ChatCompletionRequest{
		Model:       c.config.Model,
		Messages:    messages,
		Temperature: 0.7,
		Stream:      true,
	}

	// 添加工具定义
	if len(req.Tools) > 0 {
		tools := make([]openai.Tool, 0, len(req.Tools))
		for _, tool := range req.Tools {
			tools = append(tools, openai.Tool{
				Type: openai.ToolTypeFunction,
				Function: &openai.FunctionDefinition{
					Name:        tool.Function.Name,
					Description: tool.Function.Description,
					Parameters:  tool.Function.Parameters,
				},
			})
		}
		openaiReq.Tools = tools
		// 设置工具调用策略为 auto,让 AI 根据需要决定是否调用工具
		openaiReq.ToolChoice = "auto"
	}

	contentCh := make(chan string, 10)
	errCh := make(chan error, 1)

	// 异步处理流式响应
	go func() {
		defer close(contentCh)
		defer close(errCh)

		logx.Debug("Creating chat completion stream")
		stream, err := c.client.CreateChatCompletionStream(ctx, openaiReq)
		if err != nil {
			logx.Error("Failed to create chat completion stream %v", err)
			errCh <- err
			return
		}
		defer func() { _ = stream.Close() }()

		for {
			response, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				logx.Debug("Stream completed successfully")
				break
			}
			if err != nil {
				logx.Error("Stream error %v", err)
				errCh <- err
				return
			}

			// 处理流式内容
			if len(response.Choices) > 0 {
				delta := response.Choices[0].Delta.Content
				if delta != "" {
					contentCh <- delta
				}

				// 处理工具调用
				if response.Choices[0].Delta.ToolCalls != nil {
					// 流式模式下工具调用比较复杂,暂不处理
					logx.Debug("Tool call detected in stream (not implemented in stream mode)")
				}
			}
		}
	}()

	return contentCh, errCh, nil
}

// ChatWithTools 支持工具调用的对话(非流式)
func (c *OpenAIClient) ChatWithTools(ctx context.Context, messages []Message, tools []Tool) (*ChatResponse, error) {
	openaiMessages := make([]openai.ChatCompletionMessage, 0, len(messages))
	for _, msg := range messages {
		content := convertContent(msg.Content)
		openaiMsg := openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: content,
		}

		// 处理 assistant 的工具调用
		if len(msg.ToolCalls) > 0 {
			toolCalls := make([]openai.ToolCall, 0, len(msg.ToolCalls))
			for _, tc := range msg.ToolCalls {
				toolCalls = append(toolCalls, openai.ToolCall{
					ID:   tc.ID,
					Type: openai.ToolType(tc.Type),
					Function: openai.FunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				})
			}
			openaiMsg.ToolCalls = toolCalls
		}

		// 处理 tool 角色的响应
		if msg.ToolCallID != "" {
			openaiMsg.ToolCallID = msg.ToolCallID
		}
		if msg.Name != "" {
			openaiMsg.Name = msg.Name
		}

		openaiMessages = append(openaiMessages, openaiMsg)
	}

	// 构建请求
	req := openai.ChatCompletionRequest{
		Model:       c.config.Model,
		Messages:    openaiMessages,
		Temperature: 0.7,
		Stream:      false, // 工具调用时不使用流式
	}

	// 添加工具定义
	if len(tools) > 0 {
		openaiTools := make([]openai.Tool, 0, len(tools))
		for _, tool := range tools {
			openaiTools = append(openaiTools, openai.Tool{
				Type: openai.ToolTypeFunction,
				Function: &openai.FunctionDefinition{
					Name:        tool.Function.Name,
					Description: tool.Function.Description,
					Parameters:  tool.Function.Parameters,
				},
			})
		}
		req.Tools = openaiTools
		// 设置工具调用策略为 auto,让 AI 根据需要决定是否调用工具
		req.ToolChoice = "auto"
	}

	// 调用 API
	logx.Debug("Calling OpenAI API for tool execution")
	resp, err := c.client.CreateChatCompletion(ctx, req)
	if err != nil {
		logx.Error("Failed to create chat completion %v", err)
		return nil, err
	}

	if len(resp.Choices) == 0 {
		return nil, errors.New("no response choices")
	}

	choice := resp.Choices[0]

	// 转换响应
	response := &ChatResponse{
		Choices: []struct {
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
		}{
			{
				Index: choice.Index,
				Message: struct {
					Role      string     `json:"role"`
					Content   string     `json:"content"`
					ToolCalls []ToolCall `json:"tool_calls,omitempty"`
				}{
					Role:    choice.Message.Role,
					Content: choice.Message.Content,
				},
				FinishReason: string(choice.FinishReason),
			},
		},
	}

	// 转换工具调用
	if len(choice.Message.ToolCalls) > 0 {
		toolCalls := make([]ToolCall, 0, len(choice.Message.ToolCalls))
		for _, tc := range choice.Message.ToolCalls {
			toolCalls = append(toolCalls, ToolCall{
				ID:   tc.ID,
				Type: string(tc.Type),
				Function: struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				}{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			})
		}
		response.Choices[0].Message.ToolCalls = toolCalls
	}

	return response, nil
}

// Chat 普通对话(非流式)
func (c *OpenAIClient) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	messages := make([]openai.ChatCompletionMessage, 0, len(req.Messages))

	// 转换消息格式
	for _, msg := range req.Messages {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: convertContent(msg.Content),
		})
	}

	// 构建请求
	openaiReq := openai.ChatCompletionRequest{
		Model:       c.config.Model,
		Messages:    messages,
		Temperature: 0.7,
		Stream:      false,
	}

	// 调用 API
	resp, err := c.client.CreateChatCompletion(ctx, openaiReq)
	if err != nil {
		return nil, err
	}

	if len(resp.Choices) == 0 {
		return nil, errors.New("no response choices")
	}

	return &ChatResponse{
		Choices: []struct {
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
		}{
			{
				Index: resp.Choices[0].Index,
				Message: struct {
					Role      string     `json:"role"`
					Content   string     `json:"content"`
					ToolCalls []ToolCall `json:"tool_calls,omitempty"`
				}{
					Role:    resp.Choices[0].Message.Role,
					Content: resp.Choices[0].Message.Content,
				},
				FinishReason: string(resp.Choices[0].FinishReason),
			},
		},
	}, nil
}

// ChatWithToolsAndStream 支持工具调用的流式对话(Client 方法)
func (c *Client) ChatWithToolsAndStream(ctx context.Context, userMessage string) (<-chan string, error) {
	// 为了向后兼容，将单个消息转换为消息列表
	messages := []Message{
		{
			Role:    "user",
			Content: userMessage,
		},
	}
	return c.ChatWithToolsAndStreamMessages(ctx, messages)
}

// ChatWithToolsAndStreamMessages 使用完整的消息历史与 LLM 对话
func (c *Client) ChatWithToolsAndStreamMessages(ctx context.Context, historyMessages []Message) (<-chan string, error) {
	responseCh := make(chan string, 100)

	go func() {
		defer close(responseCh)

		// 构建完整的消息历史，在最前面添加系统提示
		messages := []Message{
			{
				Role:    "system",
				Content: c.buildSystemPrompt(),
			},
		}
		// 添加历史消息
		messages = append(messages, historyMessages...)

		// 创建 OpenAI 客户端
		openaiClient := NewOpenAIClient(c.config)

		// 获取工具列表
		tools, err := c.getMCPTools(ctx)
		if err != nil {
			logx.Warn("Failed to get MCP tools, proceeding without tools: %v", err)
			tools = nil
		}

		maxIterations := 10
		for i := 0; i < maxIterations; i++ {
			// 使用流式 API (支持工具调用)
			result, hasToolCalls, err := c.streamChatWithTools(ctx, openaiClient, messages, tools, responseCh)
			if err != nil {
				responseCh <- fmt.Sprintf("❌ LLM 调用失败: %v", err)
				return
			}

			// 如果没有工具调用,说明对话结束
			if !hasToolCalls {
				return
			}

			// 有工具调用,添加 assistant 消息到历史
			messages = append(messages, Message{
				Role:      "assistant",
				Content:   result.Content,
				ToolCalls: result.ToolCalls,
			})

			// 执行所有工具调用
			for _, toolCall := range result.ToolCalls {
				responseCh <- fmt.Sprintf("\n> 🔧 调用工具: **%s**\n", toolCall.Function.Name)

				// 执行工具调用
				toolResult, err := c.executeToolCall(ctx, toolCall)
				if err != nil {
					responseCh <- fmt.Sprintf("❌ 工具调用失败: %v\n\n", err)
					toolResult = fmt.Sprintf("Error: %v", err)
				}

				// 添加工具结果到历史
				messages = append(messages, Message{
					Role:       "tool",
					Content:    toolResult,
					ToolCallID: toolCall.ID,
					Name:       toolCall.Function.Name,
				})

				responseCh <- "✅ 工具执行完成\n\n"
			}
			// 继续循环,让 LLM 处理工具结果
		}

		responseCh <- "\n\n⚠️ 达到最大工具调用次数限制"
	}()

	return responseCh, nil
}

// streamChatWithTools 使用流式 API 进行对话(支持工具调用)
// 返回: (累积的消息内容, 是否有工具调用, 错误)
func (c *Client) streamChatWithTools(
	ctx context.Context,
	openaiClient *OpenAIClient,
	messages []Message,
	tools []Tool,
	responseCh chan<- string,
) (*StreamResult, bool, error) {
	// 构建 OpenAI 请求
	openaiMessages := make([]openai.ChatCompletionMessage, 0, len(messages))
	for _, msg := range messages {
		content := convertContent(msg.Content)
		openaiMsg := openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: content,
		}

		// 处理 assistant 的工具调用
		if len(msg.ToolCalls) > 0 {
			toolCalls := make([]openai.ToolCall, 0, len(msg.ToolCalls))
			for _, tc := range msg.ToolCalls {
				toolCalls = append(toolCalls, openai.ToolCall{
					ID:   tc.ID,
					Type: openai.ToolType(tc.Type),
					Function: openai.FunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				})
			}
			openaiMsg.ToolCalls = toolCalls
		}

		// 处理 tool 角色的响应
		if msg.ToolCallID != "" {
			openaiMsg.ToolCallID = msg.ToolCallID
		}
		if msg.Name != "" {
			openaiMsg.Name = msg.Name
		}

		openaiMessages = append(openaiMessages, openaiMsg)
	}

	// 构建工具定义
	var openaiTools []openai.Tool
	if len(tools) > 0 {
		for _, tool := range tools {
			openaiTools = append(openaiTools, openai.Tool{
				Type: openai.ToolTypeFunction,
				Function: &openai.FunctionDefinition{
					Name:        tool.Function.Name,
					Description: tool.Function.Description,
					Parameters:  tool.Function.Parameters,
				},
			})
		}
	}

	// 创建流式请求
	openaiReq := openai.ChatCompletionRequest{
		Model:    c.config.Model,
		Messages: openaiMessages,
		Stream:   true,
	}

	if len(openaiTools) > 0 {
		openaiReq.Tools = openaiTools
		// 设置工具调用策略为 auto,让 AI 根据需要决定是否调用工具
		// 如果想强制调用工具,可以改为 "required"
		openaiReq.ToolChoice = "auto"
	}

	logx.Debug("Creating streaming chat completion with tools")
	stream, err := openaiClient.client.CreateChatCompletionStream(ctx, openaiReq)
	if err != nil {
		return nil, false, fmt.Errorf("failed to create stream: %w", err)
	}
	defer func() { _ = stream.Close() }()

	// 累积结果
	result := &StreamResult{
		Content:   "",
		ToolCalls: []ToolCall{},
	}

	// 工具调用累积器 (key: index, value: 累积的工具调用)
	toolCallsAccumulator := make(map[int]*ToolCall)

	// 处理流式响应
	for {
		response, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			logx.Debug("Stream completed successfully")
			break
		}
		if err != nil {
			return nil, false, fmt.Errorf("stream error: %w", err)
		}

		if len(response.Choices) == 0 {
			continue
		}

		delta := response.Choices[0].Delta

		// 处理内容流
		if delta.Content != "" {
			result.Content += delta.Content
			responseCh <- delta.Content // 实时推送内容
		}

		// 处理工具调用流 (逐步累积)
		if len(delta.ToolCalls) > 0 {
			for _, tc := range delta.ToolCalls {
				index := tc.Index
				if index == nil {
					logx.Warn("Tool call index is nil, skipping")
					continue
				}

				// 获取或创建工具调用
				if _, exists := toolCallsAccumulator[*index]; !exists {
					newToolCall := &ToolCall{
						ID:   tc.ID,
						Type: string(tc.Type),
					}
					newToolCall.Function.Name = tc.Function.Name
					newToolCall.Function.Arguments = ""
					toolCallsAccumulator[*index] = newToolCall
				}

				// 累积参数
				if tc.Function.Arguments != "" {
					toolCallsAccumulator[*index].Function.Arguments += tc.Function.Arguments
				}

				// 更新 ID (如果有)
				if tc.ID != "" {
					toolCallsAccumulator[*index].ID = tc.ID
				}

				// 更新函数名 (如果有)
				if tc.Function.Name != "" {
					toolCallsAccumulator[*index].Function.Name = tc.Function.Name
				}
			}
		}

		// 检查是否结束
		if response.Choices[0].FinishReason != "" {
			logx.Debug("Stream finished, reason: %s", response.Choices[0].FinishReason)
			break
		}
	}

	// 将累积的工具调用转换为有序列表
	if len(toolCallsAccumulator) > 0 {
		// 按索引排序
		indices := make([]int, 0, len(toolCallsAccumulator))
		for idx := range toolCallsAccumulator {
			indices = append(indices, idx)
		}
		sort.Ints(indices)

		// 构建工具调用列表
		for _, idx := range indices {
			result.ToolCalls = append(result.ToolCalls, *toolCallsAccumulator[idx])
		}

		logx.Info("Accumulated %d tool calls", len(result.ToolCalls))
		return result, true, nil
	}

	// 没有工具调用,对话结束
	return result, false, nil
}

// StreamResult 流式响应的累积结果
type StreamResult struct {
	Content   string
	ToolCalls []ToolCall
}

// SetProxy 设置代理
func SetProxy(proxyURL string) error {
	if proxyURL == "" {
		return nil
	}

	_, err := url.Parse(proxyURL)
	return err
}
