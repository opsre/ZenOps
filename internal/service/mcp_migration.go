package service

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/eryajf/zenops/internal/model"
)

// MCPServersConfig MCP Servers 配置文件结构(兼容 Claude Desktop 格式)
type MCPServersConfig struct {
	MCPServers map[string]MCPServerDef `json:"mcpServers"`
}

// MCPServerDef MCP Server 定义
type MCPServerDef struct {
	IsActive      bool              `json:"isActive"`
	Name          string            `json:"name"`
	Type          string            `json:"type"`
	Description   string            `json:"description"`
	BaseURL       string            `json:"baseUrl"`
	URL           string            `json:"url"` // sse 类型使用
	Command       string            `json:"command"`
	Args          []string          `json:"args"`
	Env           map[string]string `json:"env"`
	Headers       map[string]string `json:"headers"`
	LongRunning   bool              `json:"longRunning"`
	Timeout       int               `json:"timeout"`
	InstallSource string            `json:"installSource"`
	ToolPrefix    string            `json:"toolPrefix"`
	AutoRegister  bool              `json:"autoRegister"`
	Provider      string            `json:"provider"`
	ProviderURL   string            `json:"providerUrl"`
	LogoURL       string            `json:"logoUrl"`
	Tags          []string          `json:"tags"`
}

// MigrateMCPServersFromJSON 从 JSON 文件迁移 MCP Servers 配置
func (s *ConfigService) MigrateMCPServersFromJSON(mcpConfigPath string) error {
	// 检查文件是否存在
	if _, err := os.Stat(mcpConfigPath); os.IsNotExist(err) {
		log.Printf("MCP servers config file not found: %s, skipping migration", mcpConfigPath)
		return nil
	}

	log.Printf("Migrating MCP servers from: %s", mcpConfigPath)

	// 读取文件
	data, err := os.ReadFile(mcpConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read MCP servers config: %w", err)
	}

	// 解析 JSON
	var mcpConfig MCPServersConfig
	if err := json.Unmarshal(data, &mcpConfig); err != nil {
		return fmt.Errorf("failed to parse MCP servers config: %w", err)
	}

	// 迁移每个 MCP Server
	for name, def := range mcpConfig.MCPServers {
		// 检查是否已存在
		existing, err := s.GetMCPServerByName(name)
		if err != nil {
			return err
		}
		if existing != nil {
			log.Printf("MCP server already exists: %s, skipping", name)
			continue
		}

		// 转换环境变量
		envMap := make(model.JSONMap)
		for k, v := range def.Env {
			envMap[k] = v
		}

		// 转换 headers
		headersMap := make(model.JSONMap)
		for k, v := range def.Headers {
			headersMap[k] = v
		}

		// 处理 baseUrl (兼容 sse 类型的 url 字段)
		baseURL := def.BaseURL
		if baseURL == "" && def.URL != "" {
			baseURL = def.URL
		}

		// 创建 MCP Server 记录
		server := &model.MCPServer{
			Name:          name,
			IsActive:      def.IsActive,
			Type:          def.Type,
			Description:   def.Description,
			BaseURL:       baseURL,
			Command:       def.Command,
			Args:          def.Args,
			Env:           envMap,
			Headers:       headersMap,
			LongRunning:   def.LongRunning,
			Timeout:       def.Timeout,
			InstallSource: def.InstallSource,
			ToolPrefix:    def.ToolPrefix,
			AutoRegister:  def.AutoRegister,
			Provider:      def.Provider,
			ProviderURL:   def.ProviderURL,
			LogoURL:       def.LogoURL,
			Tags:          def.Tags,
		}

		if err := s.CreateMCPServer(server); err != nil {
			return fmt.Errorf("failed to create MCP server %s: %w", name, err)
		}
		log.Printf("Migrated MCP server: %s", name)
	}

	log.Println("MCP servers migration completed")
	return nil
}

// ExportMCPServersToJSON 导出 MCP Servers 配置到 JSON 文件
func (s *ConfigService) ExportMCPServersToJSON(outputPath string) error {
	servers, err := s.ListMCPServers()
	if err != nil {
		return fmt.Errorf("failed to list MCP servers: %w", err)
	}

	mcpServers := make(map[string]MCPServerDef)
	for _, server := range servers {
		// 转换环境变量
		env := make(map[string]string)
		for k, v := range server.Env {
			if str, ok := v.(string); ok {
				env[k] = str
			}
		}

		// 转换 headers
		headers := make(map[string]string)
		for k, v := range server.Headers {
			if str, ok := v.(string); ok {
				headers[k] = str
			}
		}

		def := MCPServerDef{
			IsActive:      server.IsActive,
			Name:          server.Name,
			Type:          server.Type,
			Description:   server.Description,
			BaseURL:       server.BaseURL,
			Command:       server.Command,
			Args:          server.Args,
			Env:           env,
			Headers:       headers,
			LongRunning:   server.LongRunning,
			Timeout:       server.Timeout,
			InstallSource: server.InstallSource,
			ToolPrefix:    server.ToolPrefix,
			AutoRegister:  server.AutoRegister,
			Provider:      server.Provider,
			ProviderURL:   server.ProviderURL,
			LogoURL:       server.LogoURL,
			Tags:          server.Tags,
		}

		// sse 类型同时设置 url 字段
		if server.Type == "sse" {
			def.URL = server.BaseURL
		}

		mcpServers[server.Name] = def
	}

	config := MCPServersConfig{
		MCPServers: mcpServers,
	}

	// 写入文件
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal MCP servers config: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write MCP servers config: %w", err)
	}

	log.Printf("MCP servers config exported to: %s", outputPath)
	return nil
}

// LoadMCPServersFromDB 从数据库加载活动的 MCP Servers 配置
func (s *ConfigService) LoadMCPServersFromDB() (map[string]interface{}, error) {
	servers, err := s.ListMCPServers()
	if err != nil {
		return nil, err
	}

	result := make(map[string]interface{})
	for _, server := range servers {
		if !server.IsActive {
			continue
		}

		// 转换为通用格式
		serverMap := map[string]interface{}{
			"type":          server.Type,
			"command":       server.Command,
			"args":          server.Args,
			"env":           server.Env,
			"url":           server.BaseURL,
			"headers":       server.Headers,
			"longRunning":   server.LongRunning,
			"timeout":       server.Timeout,
			"toolPrefix":    server.ToolPrefix,
			"autoRegister":  server.AutoRegister,
		}

		result[server.Name] = serverMap
	}

	return result, nil
}
