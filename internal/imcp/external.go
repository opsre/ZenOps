package imcp

import (
	"context"
	"fmt"

	"cnb.cool/zhiqiangwang/pkg/logx"
	"github.com/eryajf/zenops/internal/mcpclient"
	"github.com/mark3labs/mcp-go/mcp"
)

// RegisterExternalMCPTools 将外部 MCP 的工具注册到 ZenOps MCP Server
func (s *MCPServer) RegisterExternalMCPTools(ctx context.Context, manager *mcpclient.Manager) error {
	if manager == nil {
		return fmt.Errorf("MCP client manager is nil")
	}

	totalTools := 0
	totalServers := 0

	// 遍历所有外部 MCP 客户端
	for _, mcpClient := range manager.List() {
		if !mcpClient.Config.AutoRegister {
			logx.Info("⏭️  Skip auto-register for MCP: %s", mcpClient.Config.Name)
			continue
		}

		registeredCount := 0
		failedCount := 0

		// 为每个工具创建代理
		for _, tool := range mcpClient.Tools {
			if err := s.registerProxyTool(ctx, mcpClient, tool); err != nil {
				logx.Error("❌ Failed to register tool %s from %s: %v",
					tool.Name, mcpClient.Config.Name, err)
				failedCount++
				continue
			}
			registeredCount++
		}

		if registeredCount > 0 {
			logx.Info("✅ Registered %d tools from MCP: %s (failed: %d)",
				registeredCount, mcpClient.Config.Name, failedCount)
			totalTools += registeredCount
			totalServers++
		}
	}

	if totalServers > 0 {
		logx.Info("🎉 Successfully registered %d tools from %d external MCP servers",
			totalTools, totalServers)
	} else {
		logx.Info("ℹ️  No external MCP tools registered")
	}

	return nil
}

// registerProxyTool 注册单个代理工具
func (s *MCPServer) registerProxyTool(ctx context.Context, mcpClient *mcpclient.MCPClient, tool mcp.Tool) error {
	// 构建工具名称 (带前缀)
	toolName := mcpClient.Config.ToolPrefix + tool.Name

	// 检查工具名称是否冲突
	existingTools := s.mcpServer.ListTools()
	if _, exists := existingTools[toolName]; exists {
		return fmt.Errorf("tool name conflict: %s already exists", toolName)
	}

	// 复制工具定义
	proxyTool := mcp.NewTool(
		toolName,
		mcp.WithDescription(fmt.Sprintf("[%s] %s", mcpClient.Config.Name, tool.Description)),
	)

	// 复制参数定义
	proxyTool.InputSchema = tool.InputSchema

	// 捕获当前迭代的值
	clientName := mcpClient.Config.Name
	originalToolName := tool.Name

	// 创建代理处理函数
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// 转发请求到外部 MCP
		proxyReq := mcp.CallToolRequest{}
		proxyReq.Params.Name = originalToolName // 使用原始工具名
		proxyReq.Params.Arguments = request.Params.Arguments

		logx.Debug("🔄 Proxy call: %s -> %s.%s",
			toolName, clientName, originalToolName)

		// 调用外部 MCP
		result, err := mcpClient.Client.CallTool(ctx, proxyReq)
		if err != nil {
			logx.Error("❌ Proxy call failed: %s -> %s.%s: %v",
				toolName, clientName, originalToolName, err)
			return mcp.NewToolResultError(fmt.Sprintf("Failed to call external MCP: %v", err)), nil
		}

		logx.Debug("✅ Proxy call success: %s -> %s.%s",
			toolName, clientName, originalToolName)

		return result, nil
	}

	// 注册到本地 MCP Server
	s.mcpServer.AddTool(proxyTool, handler)

	logx.Debug("📝 Registered proxy tool: %s -> %s.%s",
		toolName, clientName, originalToolName)

	return nil
}
