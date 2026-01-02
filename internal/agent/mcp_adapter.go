package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cnb.cool/zhiqiangwang/pkg/logx"
	"github.com/cloudwego/eino/schema"
	"github.com/eryajf/zenops/internal/imcp"
	"github.com/eryajf/zenops/internal/service"
	"github.com/mark3labs/mcp-go/mcp"
)

// MCPToolAdapter 将 MCP Tool 适配为 Eino Tool
type MCPToolAdapter struct {
	name      string
	desc      string
	schema    any
	mcpServer *imcp.Server
	username  string // 调用用户（用于日志记录）
}

// NewMCPToolAdapter 创建 MCP Tool 适配器
func NewMCPToolAdapter(name, desc string, schema any, mcpServer *imcp.Server, username string) *MCPToolAdapter {
	return &MCPToolAdapter{
		name:      name,
		desc:      desc,
		schema:    schema,
		mcpServer: mcpServer,
		username:  username,
	}
}

// Info 返回工具信息（实现 Eino Tool 接口）
func (t *MCPToolAdapter) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name:        t.name,
		Desc:        t.desc,
		ParamsOneOf: t.schema,
	}, nil
}

// InvokableRun 执行工具（实现 Eino Tool 接口）
func (t *MCPToolAdapter) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...schema.OptionItem[schema.RunOption]) (string, error) {
	logx.Debug("🔧 MCP Tool invoked: %s, args: %s", t.name, argumentsInJSON)

	// 解析参数
	var params map[string]any
	if argumentsInJSON != "" {
		if err := json.Unmarshal([]byte(argumentsInJSON), &params); err != nil {
			return "", fmt.Errorf("failed to parse tool arguments: %w", err)
		}
	} else {
		params = make(map[string]any)
	}

	// 记录开始时间
	startTime := time.Now()

	// 调用 MCP Server
	result, err := t.mcpServer.CallTool(ctx, t.name, params)
	latency := time.Since(startTime).Milliseconds()

	// 记录 MCP 调用日志
	t.logMCPCall(t.name, params, result, latency, err)

	if err != nil {
		errMsg := fmt.Sprintf("MCP tool call failed: %v", err)
		logx.Error(errMsg)
		return "", fmt.Errorf(errMsg)
	}

	// 提取文本结果
	textResult := t.extractTextResult(result)
	logx.Debug("✅ MCP Tool completed: %s, result length: %d", t.name, len(textResult))

	return textResult, nil
}

// extractTextResult 从 MCP CallToolResult 中提取文本结果
func (t *MCPToolAdapter) extractTextResult(result *mcp.CallToolResult) string {
	if result == nil || len(result.Content) == 0 {
		return "工具执行完成，但未返回结果"
	}

	var textResults []string
	for _, content := range result.Content {
		if textContent, ok := content.(mcp.TextContent); ok {
			textResults = append(textResults, textContent.Text)
		}
	}

	if len(textResults) == 0 {
		return "工具执行完成，但未返回文本结果"
	}

	// 合并所有文本结果
	var combined string
	for _, text := range textResults {
		combined += text + "\n"
	}

	return combined
}

// logMCPCall 记录 MCP 调用日志
func (t *MCPToolAdapter) logMCPCall(toolName string, params map[string]any, result *mcp.CallToolResult, latency int64, err error) {
	// 解析 server_name 和 tool_name
	// 外部 MCP 工具格式: "prefix_toolname"，例如 "aliyun-ack_list_clusters"
	// 内置工具没有前缀，例如 "search_ecs_by_ip"
	serverName := "zenops" // 默认为内置工具
	actualToolName := toolName

	// 尝试从工具名中提取前缀（外部 MCP 工具）
	// TODO: 改进前缀检测逻辑
	// if idx := strings.Index(toolName, "_"); idx > 0 {
	// 	prefix := toolName[:idx]
	// 	if strings.Contains(prefix, "-") {
	// 		serverName = prefix
	// 		actualToolName = toolName[idx+1:]
	// 	}
	// }

	mcpLogService := service.NewMCPLogService()
	logParams := &service.MCPLogParams{
		ServerName: serverName,
		ToolName:   actualToolName,
		Username:   t.username,
		Source:     "agent", // 来自 Eino Agent
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
}

// BuildMCPTools 从 MCP Server 构建 Eino Tools
func BuildMCPTools(mcpServer *imcp.Server, username string) ([]schema.ToolInfo, error) {
	// 获取启用的 MCP 工具列表
	toolList, err := mcpServer.ListEnabledTools(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to list enabled MCP tools: %w", err)
	}

	var tools []schema.ToolInfo
	for _, tool := range toolList.Tools {
		adapter := NewMCPToolAdapter(tool.Name, tool.Description, tool.InputSchema, mcpServer, username)

		// 构建 ToolInfo
		info := schema.ToolInfo{
			Name:        tool.Name,
			Desc:        tool.Description,
			ParamsOneOf: tool.InputSchema,
		}

		tools = append(tools, info)

		logx.Debug("📦 Loaded MCP tool: %s", tool.Name)
	}

	logx.Info("✅ Loaded %d enabled MCP tools for Eino Agent", len(tools))
	return tools, nil
}
