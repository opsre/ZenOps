package imcp

import (
	"context"
	"fmt"

	"cnb.cool/zhiqiangwang/pkg/logx"
	"github.com/eryajf/zenops/internal/config"
	"github.com/eryajf/zenops/internal/service"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// MCPServer 基于 mcp-go 库的 MCP 服务器
type MCPServer struct {
	config    *config.Config
	mcpServer *server.MCPServer
	sseServer *server.SSEServer
}

// NewMCPServer 创建基于 mcp-go 库的 MCP 服务器
func NewMCPServer(cfg *config.Config) *MCPServer {
	// 创建 MCP 服务器
	mcpServer := server.NewMCPServer(
		"zenops",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	s := &MCPServer{
		config:    cfg,
		mcpServer: mcpServer,
	}

	// 注册工具
	s.registerTools()

	return s
}

// registerTools 注册所有 MCP 工具
func (s *MCPServer) registerTools() {
	// 1. search_ecs_by_ip - 根据 IP 搜索 ECS 实例
	s.mcpServer.AddTool(
		mcp.NewTool("search_ecs_by_ip",
			mcp.WithDescription("根据 IP 地址精确搜索阿里云 ECS 实例(支持私网 IP、公网 IP 和弹性 IP)"),
			mcp.WithString("ip",
				mcp.Required(),
				mcp.Description("要搜索的 IP 地址"),
			),
			mcp.WithString("account",
				mcp.Description("阿里云账号名称(可选,默认使用第一个可用账号)"),
			),
			mcp.WithString("region",
				mcp.Description("区域(可选,默认使用配置的第一个区域)"),
			),
			mcp.WithString("ip_type",
				mcp.Description("IP 类型: private(内网 IP), public(公网 IP), eip(弹性 IP), 不指定则自动尝试所有类型"),
			),
		),
		s.handleSearchECSByIP,
	)

	// 2. search_ecs_by_name - 根据名称搜索 ECS 实例
	s.mcpServer.AddTool(
		mcp.NewTool("search_ecs_by_name",
			mcp.WithDescription("根据实例名称精确搜索阿里云 ECS 实例(支持精确匹配)"),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("实例名称(精确匹配)"),
			),
			mcp.WithString("account",
				mcp.Description("阿里云账号名称(可选)"),
			),
			mcp.WithString("region",
				mcp.Description("区域(可选)"),
			),
		),
		s.handleSearchECSByName,
	)

	// 3. list_ecs - 列出所有 ECS 实例
	s.mcpServer.AddTool(
		mcp.NewTool("list_ecs",
			mcp.WithDescription("列出阿里云 ECS 实例,支持按状态和计费方式筛选"),
			mcp.WithString("account",
				mcp.Description("阿里云账号名称(可选)"),
			),
			mcp.WithString("region",
				mcp.Description("区域(可选)"),
			),
			mcp.WithString("status",
				mcp.Description("实例状态(可选): Pending, Running, Starting, Stopping, Stopped"),
			),
			mcp.WithString("instance_charge_type",
				mcp.Description("计费方式(可选): PostPaid(按量付费), PrePaid(包年包月)"),
			),
		),
		s.handleListECS,
	)

	// 4. get_ecs - 获取 ECS 实例详情
	s.mcpServer.AddTool(
		mcp.NewTool("get_ecs",
			mcp.WithDescription("获取指定 ECS 实例的详细信息"),
			mcp.WithString("instance_id",
				mcp.Required(),
				mcp.Description("实例 ID"),
			),
			mcp.WithString("account",
				mcp.Description("阿里云账号名称(可选)"),
			),
		),
		s.handleGetECS,
	)

	// 5. list_rds - 列出所有 RDS 实例
	s.mcpServer.AddTool(
		mcp.NewTool("list_rds",
			mcp.WithDescription("列出所有阿里云 RDS 数据库实例"),
			mcp.WithString("account",
				mcp.Description("阿里云账号名称(可选)"),
			),
			mcp.WithString("region",
				mcp.Description("区域(可选)"),
			),
		),
		s.handleListRDS,
	)

	// 6. search_rds_by_name - 根据名称搜索 RDS 实例
	s.mcpServer.AddTool(
		mcp.NewTool("search_rds_by_name",
			mcp.WithDescription("根据名称搜索阿里云 RDS 数据库实例"),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("RDS 实例名称"),
			),
			mcp.WithString("account",
				mcp.Description("阿里云账号名称(可选)"),
			),
		),
		s.handleSearchRDSByName,
	)

	// 7. list_oss - 列出所有 OSS 存储桶
	s.mcpServer.AddTool(
		mcp.NewTool("list_oss",
			mcp.WithDescription("列出所有阿里云 OSS 存储桶"),
			mcp.WithString("account",
				mcp.Description("阿里云账号名称(可选)"),
			),
		),
		s.handleListOSS,
	)

	// 8. get_oss - 获取 OSS 存储桶详情
	s.mcpServer.AddTool(
		mcp.NewTool("get_oss",
			mcp.WithDescription("获取指定 OSS 存储桶的详细信息"),
			mcp.WithString("bucket_name",
				mcp.Required(),
				mcp.Description("存储桶名称"),
			),
			mcp.WithString("account",
				mcp.Description("阿里云账号名称(可选)"),
			),
		),
		s.handleGetOSS,
	)

	// ==================== 腾讯云 CVM 工具 ====================

	// 7. search_cvm_by_ip - 根据 IP 搜索腾讯云 CVM
	s.mcpServer.AddTool(
		mcp.NewTool("search_cvm_by_ip",
			mcp.WithDescription("根据 IP 地址搜索腾讯云 CVM 实例(支持私网 IP 和公网 IP)"),
			mcp.WithString("ip",
				mcp.Required(),
				mcp.Description("要搜索的 IP 地址"),
			),
			mcp.WithString("account",
				mcp.Description("腾讯云账号名称(可选,默认使用第一个启用的账号)"),
			),
		),
		s.handleSearchCVMByIP,
	)

	// 8. search_cvm_by_name - 根据名称搜索腾讯云 CVM
	s.mcpServer.AddTool(
		mcp.NewTool("search_cvm_by_name",
			mcp.WithDescription("根据实例名称搜索腾讯云 CVM 实例"),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("实例名称"),
			),
			mcp.WithString("account",
				mcp.Description("腾讯云账号名称(可选)"),
			),
		),
		s.handleSearchCVMByName,
	)

	// 9. list_cvm - 列出腾讯云 CVM 实例
	s.mcpServer.AddTool(
		mcp.NewTool("list_cvm",
			mcp.WithDescription("列出所有腾讯云 CVM 实例"),
			mcp.WithString("account",
				mcp.Description("腾讯云账号名称(可选)"),
			),
			mcp.WithString("region",
				mcp.Description("区域(可选)"),
			),
		),
		s.handleListCVM,
	)

	// 10. get_cvm - 获取腾讯云 CVM 实例详情
	s.mcpServer.AddTool(
		mcp.NewTool("get_cvm",
			mcp.WithDescription("获取指定腾讯云 CVM 实例的详细信息"),
			mcp.WithString("instance_id",
				mcp.Required(),
				mcp.Description("实例 ID"),
			),
			mcp.WithString("account",
				mcp.Description("腾讯云账号名称(可选)"),
			),
		),
		s.handleGetCVM,
	)

	// ==================== 腾讯云 CDB 工具 ====================

	// 11. list_cdb - 列出腾讯云 CDB 实例
	s.mcpServer.AddTool(
		mcp.NewTool("list_cdb",
			mcp.WithDescription("列出所有腾讯云 CDB 数据库实例"),
			mcp.WithString("account",
				mcp.Description("腾讯云账号名称(可选)"),
			),
			mcp.WithString("region",
				mcp.Description("区域(可选)"),
			),
		),
		s.handleListCDB,
	)

	// 12. search_cdb_by_name - 根据名称搜索腾讯云 CDB
	s.mcpServer.AddTool(
		mcp.NewTool("search_cdb_by_name",
			mcp.WithDescription("根据名称搜索腾讯云 CDB 数据库实例"),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("CDB 实例名称"),
			),
			mcp.WithString("account",
				mcp.Description("腾讯云账号名称(可选)"),
			),
		),
		s.handleSearchCDBByName,
	)

	// ==================== 腾讯云 COS 工具 ====================

	// 13. list_cos - 列出腾讯云 COS 存储桶
	s.mcpServer.AddTool(
		mcp.NewTool("list_cos",
			mcp.WithDescription("列出所有腾讯云 COS 存储桶"),
			mcp.WithString("account",
				mcp.Description("腾讯云账号名称(可选)"),
			),
		),
		s.handleListCOS,
	)

	// 14. get_cos - 获取腾讯云 COS 存储桶详情
	s.mcpServer.AddTool(
		mcp.NewTool("get_cos",
			mcp.WithDescription("获取指定腾讯云 COS 存储桶的详细信息"),
			mcp.WithString("bucket_name",
				mcp.Required(),
				mcp.Description("存储桶名称"),
			),
			mcp.WithString("account",
				mcp.Description("腾讯云账号名称(可选)"),
			),
		),
		s.handleGetCOS,
	)

	// ==================== Jenkins 工具 ====================

	// 13. list_jenkins_jobs - 列出 Jenkins Jobs
	s.mcpServer.AddTool(
		mcp.NewTool("list_jenkins_jobs",
			mcp.WithDescription("列出所有 Jenkins Job"),
		),
		s.handleListJenkinsJobs,
	)

	// 14. get_jenkins_job - 获取 Jenkins Job 详情
	s.mcpServer.AddTool(
		mcp.NewTool("get_jenkins_job",
			mcp.WithDescription("获取指定 Jenkins Job 的详细信息"),
			mcp.WithString("job_name",
				mcp.Required(),
				mcp.Description("Job 名称"),
			),
		),
		s.handleGetJenkinsJob,
	)

	// 15. list_jenkins_builds - 列出 Jenkins 构建历史
	s.mcpServer.AddTool(
		mcp.NewTool("list_jenkins_builds",
			mcp.WithDescription("列出指定 Jenkins Job 的构建历史"),
			mcp.WithString("job_name",
				mcp.Required(),
				mcp.Description("Job 名称"),
			),
			mcp.WithNumber("limit",
				mcp.Description("限制返回的构建数量(默认 20)"),
			),
		),
		s.handleListJenkinsBuilds,
	)
}

// Start 启动 MCP 服务器 (stdio 模式)
func (s *MCPServer) Start() error {
	logx.Info("Starting MCP server in stdio mode (using mcp-go library)")
	return server.ServeStdio(s.mcpServer)
}

// StartSSE 启动 MCP 服务器 (SSE 模式)
func (s *MCPServer) StartSSE() error {
	addr := fmt.Sprintf("0.0.0.0:%d", s.config.Server.MCP.Port)

	// 记录工具数量
	tools := s.mcpServer.ListTools()
	logx.Info("🧰 Starting MCP Server In SSE Mode, Listening On %s (Total tools: %d)", addr, len(tools))

	// 创建 SSE 服务器
	s.sseServer = server.NewSSEServer(
		s.mcpServer,
		server.WithSSEEndpoint("/sse"),
		server.WithMessageEndpoint("/message"),
	)

	// 启动服务器
	return s.sseServer.Start(addr)
}

// StopSSE 停止 SSE 服务器
func (s *MCPServer) StopSSE(ctx context.Context) error {
	if s.sseServer != nil {
		return s.sseServer.Shutdown(ctx)
	}
	return nil
}

// CallTool 调用 MCP 工具(公开方法,供其他包使用)
func (s *MCPServer) CallTool(ctx context.Context, toolName string, arguments map[string]any) (*mcp.CallToolResult, error) {
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      toolName,
			Arguments: arguments,
		},
	}

	// 根据工具名称调用对应的处理函数
	switch toolName {
	// 阿里云 ECS
	case "search_ecs_by_ip":
		return s.handleSearchECSByIP(ctx, request)
	case "search_ecs_by_name":
		return s.handleSearchECSByName(ctx, request)
	case "list_ecs":
		return s.handleListECS(ctx, request)
	case "get_ecs":
		return s.handleGetECS(ctx, request)

	// 阿里云 RDS
	case "list_rds":
		return s.handleListRDS(ctx, request)
	case "search_rds_by_name":
		return s.handleSearchRDSByName(ctx, request)

	// 阿里云 OSS
	case "list_oss":
		return s.handleListOSS(ctx, request)
	case "get_oss":
		return s.handleGetOSS(ctx, request)

	// 腾讯云 CVM
	case "search_cvm_by_ip":
		return s.handleSearchCVMByIP(ctx, request)
	case "search_cvm_by_name":
		return s.handleSearchCVMByName(ctx, request)
	case "list_cvm":
		return s.handleListCVM(ctx, request)
	case "get_cvm":
		return s.handleGetCVM(ctx, request)

	// 腾讯云 CDB
	case "list_cdb":
		return s.handleListCDB(ctx, request)
	case "search_cdb_by_name":
		return s.handleSearchCDBByName(ctx, request)

	// 腾讯云 COS
	case "list_cos":
		return s.handleListCOS(ctx, request)
	case "get_cos":
		return s.handleGetCOS(ctx, request)

	// Jenkins
	case "list_jenkins_jobs":
		return s.handleListJenkinsJobs(ctx, request)
	case "get_jenkins_job":
		return s.handleGetJenkinsJob(ctx, request)
	case "list_jenkins_builds":
		return s.handleListJenkinsBuilds(ctx, request)

	default:
		// 尝试从底层 MCP Server 调用工具(用于外部 MCP 工具,如 CNB)
		logx.Debug("Tool not in built-in list, trying to call from registered handlers: %s", toolName)

		// 获取工具定义和处理器
		serverTool := s.mcpServer.GetTool(toolName)
		if serverTool == nil {
			logx.Error("Tool not found in MCP server: %s", toolName)
			return mcp.NewToolResultError(fmt.Sprintf("unsupported tool: %s", toolName)), nil
		}

		// 调用工具处理器
		result, err := serverTool.Handler(ctx, request)
		if err != nil {
			logx.Error("Failed to call tool handler: %s, error: %v", toolName, err)
			return mcp.NewToolResultError(fmt.Sprintf("failed to call tool: %v", err)), nil
		}

		return result, nil
	}
}

// ListTools 列出所有可用的工具
func (s *MCPServer) ListTools(ctx context.Context) (*mcp.ListToolsResult, error) {
	// 通过 MCP 服务器获取工具列表
	toolsMap := s.mcpServer.ListTools()

	// 转换为 ListToolsResult 格式
	var tools []mcp.Tool
	for _, serverTool := range toolsMap {
		tools = append(tools, serverTool.Tool)
	}

	return &mcp.ListToolsResult{
		Tools: tools,
	}, nil
}

// ListEnabledTools 列出所有启用的工具（过滤数据库中被禁用的工具）
func (s *MCPServer) ListEnabledTools(ctx context.Context) (*mcp.ListToolsResult, error) {
	// 1. 获取所有已注册的工具
	toolsMap := s.mcpServer.ListTools()

	// 2. 从数据库获取所有 MCP 服务器及其工具的启用状态
	configService := service.NewConfigService()
	servers, err := configService.ListMCPServers()
	if err != nil {
		logx.Warn("Failed to load MCP servers from database, returning all tools: %v", err)
		// 如果数据库查询失败，返回所有工具（保持向后兼容）
		var tools []mcp.Tool
		for _, serverTool := range toolsMap {
			tools = append(tools, serverTool.Tool)
		}
		return &mcp.ListToolsResult{Tools: tools}, nil
	}

	// 3. 构建工具启用状态映射：map[toolName]isEnabled
	// 只有当服务器启用 AND 工具启用时，工具才真正可用
	toolEnabledMap := make(map[string]bool)

	for _, server := range servers {
		// 如果服务器未启用，该服务器的所有工具都不可用
		if !server.IsActive {
			for _, tool := range server.Tools {
				toolName := server.ToolPrefix + tool.Name
				toolEnabledMap[toolName] = false
			}
			continue
		}

		// 服务器已启用，检查每个工具的状态
		for _, tool := range server.Tools {
			toolName := server.ToolPrefix + tool.Name
			toolEnabledMap[toolName] = tool.IsEnabled
		}
	}

	// 4. 过滤只返回启用的工具
	var enabledTools []mcp.Tool
	for _, serverTool := range toolsMap {
		toolName := serverTool.Tool.Name

		// 如果工具在数据库中有记录且被禁用，则跳过
		if enabled, exists := toolEnabledMap[toolName]; exists && !enabled {
			logx.Debug("Tool %s is disabled in database, skipping", toolName)
			continue
		}

		// 工具启用或不在数据库中（内置工具默认启用）
		enabledTools = append(enabledTools, serverTool.Tool)
	}

	logx.Debug("Filtered %d enabled tools out of %d total tools", len(enabledTools), len(toolsMap))

	return &mcp.ListToolsResult{
		Tools: enabledTools,
	}, nil
}
