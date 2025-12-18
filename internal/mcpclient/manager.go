package mcpclient

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cnb.cool/zhiqiangwang/pkg/logx"
	"github.com/eryajf/zenops/internal/config"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

// Manager MCP 客户端管理器
type Manager struct {
	clients map[string]*MCPClient
	mu      sync.RWMutex
}

// MCPClient MCP 客户端封装
type MCPClient struct {
	Config *config.MCPServerConfig
	Client *client.Client
	Tools  []mcp.Tool
}

// NewManager 创建管理器
func NewManager() *Manager {
	return &Manager{
		clients: make(map[string]*MCPClient),
	}
}

// LoadFromConfig 从配置加载所有 MCP 客户端
func (m *Manager) LoadFromConfig(cfg *config.MCPServersConfig) error {
	if cfg == nil || cfg.MCPServers == nil {
		logx.Info("No MCP servers configured")
		return nil
	}

	for name, serverCfg := range cfg.MCPServers {
		if !serverCfg.IsActive {
			logx.Info("⏭️  Skip inactive MCP server: %s", name)
			continue
		}

		if err := m.Register(name, serverCfg); err != nil {
			logx.Error("❌ Failed to register MCP server %s: %v", name, err)
			continue
		}
	}
	return nil
}

// Register 注册一个 MCP 客户端
func (m *Manager) Register(name string, cfg *config.MCPServerConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查是否已存在
	if _, exists := m.clients[name]; exists {
		return fmt.Errorf("MCP client %s already registered", name)
	}

	// 创建客户端
	c, err := m.createClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// 初始化客户端
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Timeout)*time.Second)
	defer cancel()

	if err := m.initializeClient(ctx, c); err != nil {
		c.Close()
		return fmt.Errorf("failed to initialize client: %w", err)
	}

	// 获取工具列表
	tools, err := m.listTools(ctx, c)
	if err != nil {
		c.Close()
		return fmt.Errorf("failed to list tools: %w", err)
	}

	// 保存客户端
	m.clients[name] = &MCPClient{
		Config: cfg,
		Client: c,
		Tools:  tools,
	}

	logx.Info("✅ Registered MCP server: %s (%s) with %d tools",
		name, cfg.Type, len(tools))

	return nil
}

// createClient 根据配置创建客户端
func (m *Manager) createClient(cfg *config.MCPServerConfig) (*client.Client, error) {
	switch cfg.Type {
	case "stdio":
		return m.createStdioClient(cfg)
	case "sse":
		return m.createSSEClient(cfg)
	case "streamableHttp", "streamable-http":
		return m.createStreamableHttpClient(cfg)
	default:
		return nil, fmt.Errorf("unsupported MCP type: %s", cfg.Type)
	}
}

// createStdioClient 创建 Stdio 客户端
func (m *Manager) createStdioClient(cfg *config.MCPServerConfig) (*client.Client, error) {
	// 转换环境变量
	env := make([]string, 0, len(cfg.Env))
	for k, v := range cfg.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	logx.Debug("Creating Stdio MCP client: command=%s args=%v", cfg.Command, cfg.Args)

	// 创建 Stdio 客户端
	c, err := client.NewStdioMCPClient(
		cfg.Command,
		env,
		cfg.Args...,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create stdio client: %w", err)
	}

	return c, nil
}

// createSSEClient 创建 SSE 客户端
func (m *Manager) createSSEClient(cfg *config.MCPServerConfig) (*client.Client, error) {
	// 构建选项
	opts := []transport.ClientOption{}

	// 添加 Headers
	if len(cfg.Headers) > 0 {
		opts = append(opts, transport.WithHeaders(cfg.Headers))
	}

	logx.Debug("Creating SSE MCP client: baseURL=%s", cfg.BaseURL)

	// 创建 SSE 客户端
	c, err := client.NewSSEMCPClient(cfg.BaseURL, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create sse client: %w", err)
	}

	return c, nil
}

// createStreamableHttpClient 创建 Streamable HTTP 客户端
func (m *Manager) createStreamableHttpClient(cfg *config.MCPServerConfig) (*client.Client, error) {
	// 构建选项
	opts := []transport.StreamableHTTPCOption{}

	// 添加 Headers (注意: streamableHttp 使用 WithHTTPHeaders)
	if len(cfg.Headers) > 0 {
		opts = append(opts, transport.WithHTTPHeaders(cfg.Headers))
	}

	logx.Debug("Creating Streamable HTTP MCP client: baseURL=%s", cfg.BaseURL)

	// 创建 Streamable HTTP 客户端
	c, err := client.NewStreamableHttpClient(cfg.BaseURL, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create streamable http client: %w", err)
	}

	return c, nil
}

// initializeClient 初始化客户端
func (m *Manager) initializeClient(ctx context.Context, c *client.Client) error {
	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{
		Name:    "zenops",
		Version: "1.0.0",
	}
	initReq.Params.Capabilities = mcp.ClientCapabilities{}

	_, err := c.Initialize(ctx, initReq)
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}

	return nil
}

// listTools 获取工具列表
func (m *Manager) listTools(ctx context.Context, c *client.Client) ([]mcp.Tool, error) {
	toolsReq := mcp.ListToolsRequest{}
	result, err := c.ListTools(ctx, toolsReq)
	if err != nil {
		return nil, fmt.Errorf("failed to list tools: %w", err)
	}
	return result.Tools, nil
}

// Get 获取客户端
func (m *Manager) Get(name string) (*MCPClient, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	c, ok := m.clients[name]
	if !ok {
		return nil, fmt.Errorf("MCP client %s not found", name)
	}
	return c, nil
}

// List 列出所有客户端
func (m *Manager) List() []*MCPClient {
	m.mu.RLock()
	defer m.mu.RUnlock()

	clients := make([]*MCPClient, 0, len(m.clients))
	for _, c := range m.clients {
		clients = append(clients, c)
	}
	return clients
}

// CallTool 调用工具
func (m *Manager) CallTool(ctx context.Context, serverName, toolName string, args map[string]any) (*mcp.CallToolResult, error) {
	mcpClient, err := m.Get(serverName)
	if err != nil {
		return nil, err
	}

	callReq := mcp.CallToolRequest{}
	callReq.Params.Name = toolName
	callReq.Params.Arguments = args

	result, err := mcpClient.Client.CallTool(ctx, callReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call tool %s on %s: %w", toolName, serverName, err)
	}

	return result, nil
}

// Close 关闭客户端
func (m *Manager) Close(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	c, ok := m.clients[name]
	if !ok {
		return fmt.Errorf("client %s not found", name)
	}

	c.Client.Close()
	delete(m.clients, name)

	logx.Info("Closed MCP client: %s", name)
	return nil
}

// CloseAll 关闭所有客户端
func (m *Manager) CloseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, c := range m.clients {
		c.Client.Close()
		logx.Info("Closed MCP client: %s", name)
	}
	m.clients = make(map[string]*MCPClient)
}
