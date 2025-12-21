package server

import (
	"fmt"
	"net/http"
	"sync"

	"cnb.cool/zhiqiangwang/pkg/logx"
	"github.com/eryajf/zenops/internal/config"
	"github.com/gin-gonic/gin"
)

var configMutex sync.Mutex

// ==================== MCP Server 管理 API ====================

// handleMCPServerList 获取所有 MCP Server 列表
func (s *HTTPGinServer) handleMCPServerList(c *gin.Context) {
	if s.config.MCPServersConfig == "" {
		s.success(c, []any{})
		return
	}

	configMutex.Lock()
	defer configMutex.Unlock()

	mcpConfig, err := config.LoadMCPServersConfig(s.config.MCPServersConfig)
	if err != nil {
		// 如果文件不存在或为空，返回空列表
		s.success(c, []any{})
		return
	}

	// 转换为列表返回
	var servers []*config.MCPServerConfig
	for _, server := range mcpConfig.MCPServers {
		servers = append(servers, server)
	}

	s.success(c, gin.H{
		"total":   len(servers),
		"servers": servers,
	})
}

// handleMCPServerGet 获取单个 MCP Server 详情
func (s *HTTPGinServer) handleMCPServerGet(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		s.error(c, http.StatusBadRequest, "name is required")
		return
	}

	if s.config.MCPServersConfig == "" {
		s.error(c, http.StatusNotFound, "MCP config not configured")
		return
	}

	configMutex.Lock()
	defer configMutex.Unlock()

	mcpConfig, err := config.LoadMCPServersConfig(s.config.MCPServersConfig)
	if err != nil {
		s.error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to load config: %v", err))
		return
	}

	server, ok := mcpConfig.MCPServers[name]
	if !ok {
		s.error(c, http.StatusNotFound, "MCP server not found")
		return
	}

	s.success(c, gin.H{
		"server": server,
	})
}

// handleMCPServerAdd 添加新的 MCP Server
func (s *HTTPGinServer) handleMCPServerAdd(c *gin.Context) {
	var req config.MCPServerConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		s.error(c, http.StatusBadRequest, fmt.Sprintf("Invalid request: %v", err))
		return
	}

	if req.Name == "" {
		s.error(c, http.StatusBadRequest, "name is required")
		return
	}

	if s.config.MCPServersConfig == "" {
		s.error(c, http.StatusBadRequest, "MCP config file path not configured in main config")
		return
	}

	configMutex.Lock()
	defer configMutex.Unlock()

	// 加载现有配置
	mcpConfig, err := config.LoadMCPServersConfig(s.config.MCPServersConfig)
	if err != nil {
		// 如果加载失败，尝试创建一个新的配置
		mcpConfig = &config.MCPServersConfig{
			MCPServers: make(map[string]*config.MCPServerConfig),
		}
	}

	if _, exists := mcpConfig.MCPServers[req.Name]; exists {
		s.error(c, http.StatusConflict, "MCP server already exists")
		return
	}

	// 设置默认值
	if req.Timeout == 0 {
		req.Timeout = 300
	}
	if req.ToolPrefix == "" {
		req.ToolPrefix = req.Name + "_"
	}

	// 保存到配置
	mcpConfig.MCPServers[req.Name] = &req
	if err := config.SaveMCPServersConfig(s.config.MCPServersConfig, mcpConfig); err != nil {
		s.error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to save config: %v", err))
		return
	}

	// 如果启用，注册到 Manager
	if req.IsActive && s.mcpClientManager != nil {
		if err := s.mcpClientManager.Register(req.Name, &req); err != nil {
			logx.Error("Failed to register MCP server %s: %v", req.Name, err)
			// 不返回错误，因为配置已经保存成功，只是启动失败
		}
	}

	s.success(c, gin.H{
		"server": req,
	})
}

// handleMCPServerUpdate 更新 MCP Server 配置
func (s *HTTPGinServer) handleMCPServerUpdate(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		s.error(c, http.StatusBadRequest, "name is required")
		return
	}

	var req config.MCPServerConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		s.error(c, http.StatusBadRequest, fmt.Sprintf("Invalid request: %v", err))
		return
	}

	// 确保 name 一致
	req.Name = name

	configMutex.Lock()
	defer configMutex.Unlock()

	mcpConfig, err := config.LoadMCPServersConfig(s.config.MCPServersConfig)
	if err != nil {
		s.error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to load config: %v", err))
		return
	}

	if _, exists := mcpConfig.MCPServers[name]; !exists {
		s.error(c, http.StatusNotFound, "MCP server not found")
		return
	}

	// 更新配置
	mcpConfig.MCPServers[name] = &req
	if err := config.SaveMCPServersConfig(s.config.MCPServersConfig, mcpConfig); err != nil {
		s.error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to save config: %v", err))
		return
	}

	// 更新 Manager
	if s.mcpClientManager != nil {
		// 先关闭旧的
		_ = s.mcpClientManager.Close(name)

		// 如果启用，重新注册
		if req.IsActive {
			if err := s.mcpClientManager.Register(name, &req); err != nil {
				logx.Error("Failed to register MCP server %s: %v", name, err)
			}
		}
	}

	s.success(c, gin.H{
		"server": req,
	})
}

// handleMCPServerDelete 删除 MCP Server
func (s *HTTPGinServer) handleMCPServerDelete(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		s.error(c, http.StatusBadRequest, "name is required")
		return
	}

	configMutex.Lock()
	defer configMutex.Unlock()

	mcpConfig, err := config.LoadMCPServersConfig(s.config.MCPServersConfig)
	if err != nil {
		s.error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to load config: %v", err))
		return
	}

	if _, exists := mcpConfig.MCPServers[name]; !exists {
		s.error(c, http.StatusNotFound, "MCP server not found")
		return
	}

	// 从配置中删除
	delete(mcpConfig.MCPServers, name)
	if err := config.SaveMCPServersConfig(s.config.MCPServersConfig, mcpConfig); err != nil {
		s.error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to save config: %v", err))
		return
	}

	// 从 Manager 中移除
	if s.mcpClientManager != nil {
		_ = s.mcpClientManager.Close(name)
	}

	s.success(c, gin.H{
		"message": "MCP server deleted",
	})
}

// handleMCPServerToggle 启用/禁用 MCP Server
func (s *HTTPGinServer) handleMCPServerToggle(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		s.error(c, http.StatusBadRequest, "name is required")
		return
	}

	var req struct {
		IsActive bool `json:"isActive"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		s.error(c, http.StatusBadRequest, fmt.Sprintf("Invalid request: %v", err))
		return
	}

	configMutex.Lock()
	defer configMutex.Unlock()

	mcpConfig, err := config.LoadMCPServersConfig(s.config.MCPServersConfig)
	if err != nil {
		s.error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to load config: %v", err))
		return
	}

	server, exists := mcpConfig.MCPServers[name]
	if !exists {
		s.error(c, http.StatusNotFound, "MCP server not found")
		return
	}

	// 更新状态
	server.IsActive = req.IsActive
	if err := config.SaveMCPServersConfig(s.config.MCPServersConfig, mcpConfig); err != nil {
		s.error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to save config: %v", err))
		return
	}

	// 更新 Manager
	if s.mcpClientManager != nil {
		if req.IsActive {
			// 启用：注册
			// 先尝试关闭以防万一
			_ = s.mcpClientManager.Close(name)
			if err := s.mcpClientManager.Register(name, server); err != nil {
				logx.Error("Failed to register MCP server %s: %v", name, err)
				s.error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to start server: %v", err))
				return
			}
		} else {
			// 禁用：关闭
			if err := s.mcpClientManager.Close(name); err != nil {
				logx.Warn("Failed to close MCP server %s: %v", name, err)
			}
		}
	}

	s.success(c, gin.H{
		"server": server,
	})
}

// ==================== MCP 工具管理 API ====================

// handleMCPToolList 获取 MCP 的工具列表
func (s *HTTPGinServer) handleMCPToolList(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		s.error(c, http.StatusBadRequest, "name is required")
		return
	}

	if s.mcpClientManager == nil {
		s.error(c, http.StatusServiceUnavailable, "MCP manager not initialized")
		return
	}

	client, err := s.mcpClientManager.Get(name)
	if err != nil {
		s.error(c, http.StatusNotFound, fmt.Sprintf("MCP server not found or not active: %v", err))
		return
	}

	s.success(c, gin.H{
		"tools": client.Tools,
	})
}

// handleMCPToolTest 调试工具
func (s *HTTPGinServer) handleMCPToolTest(c *gin.Context) {
	serverName := c.Param("name")
	toolName := c.Param("toolName")

	if serverName == "" || toolName == "" {
		s.error(c, http.StatusBadRequest, "server name and tool name are required")
		return
	}

	var args map[string]any
	if err := c.ShouldBindJSON(&args); err != nil {
		// 允许空参数
		args = make(map[string]any)
	}

	if s.mcpClientManager == nil {
		s.error(c, http.StatusServiceUnavailable, "MCP manager not initialized")
		return
	}

	result, err := s.mcpClientManager.CallTool(c.Request.Context(), serverName, toolName, args)
	if err != nil {
		s.error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to call tool: %v", err))
		return
	}

	s.success(c, gin.H{
		"result": result,
	})
}
