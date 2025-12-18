package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// MCPServerConfig 标准 MCP Server 配置 (兼容 Claude Desktop 格式)
type MCPServerConfig struct {
	IsActive      bool              `yaml:"is_active" json:"isActive"`
	Name          string            `yaml:"name" json:"name"`
	Type          string            `yaml:"type" json:"type"` // "stdio" | "sse"
	Description   string            `yaml:"description" json:"description"`
	BaseURL       string            `yaml:"base_url" json:"baseUrl"`
	Command       string            `yaml:"command" json:"command"`
	Args          []string          `yaml:"args" json:"args"`
	Env           map[string]string `yaml:"env" json:"env"`
	Headers       map[string]string `yaml:"headers" json:"headers"` // 用于 SSE/HTTP
	Provider      string            `yaml:"provider" json:"provider"`
	ProviderURL   string            `yaml:"provider_url" json:"providerUrl"`
	LogoURL       string            `yaml:"logo_url" json:"logoUrl"`
	Tags          []string          `yaml:"tags" json:"tags"`
	LongRunning   bool              `yaml:"long_running" json:"longRunning"`
	Timeout       int               `yaml:"timeout" json:"timeout"`
	InstallSource string            `yaml:"install_source" json:"installSource"`

	// ZenOps 扩展字段
	ToolPrefix   string `yaml:"tool_prefix" json:"toolPrefix"`     // 工具名前缀
	AutoRegister bool   `yaml:"auto_register" json:"autoRegister"` // 是否自动注册
}

// MCPServersConfig MCP Servers 配置集合
type MCPServersConfig struct {
	MCPServers map[string]*MCPServerConfig `yaml:"mcp_servers" json:"mcpServers"`
}

// LoadMCPServersConfig 加载 MCP Servers 配置
func LoadMCPServersConfig(configPath string) (*MCPServersConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config MCPServersConfig

	// 根据文件扩展名判断格式
	if isJSON(configPath) {
		err = json.Unmarshal(data, &config)
	} else {
		err = yaml.Unmarshal(data, &config)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// 校验配置
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &config, nil
}

// Validate 校验配置
func (c *MCPServersConfig) Validate() error {
	if c.MCPServers == nil {
		return fmt.Errorf("mcpServers is required")
	}

	for name, server := range c.MCPServers {
		if server.Name == "" {
			server.Name = name // 如果没有设置 name,使用 key 作为 name
		}

		if server.Type != "stdio" && server.Type != "sse" && server.Type != "streamableHttp" && server.Type != "streamable-http" {
			return fmt.Errorf("server %s: type must be 'stdio', 'sse', or 'streamableHttp'", name)
		}

		if server.Type == "stdio" && server.Command == "" {
			return fmt.Errorf("server %s: command is required for stdio type", name)
		}

		if (server.Type == "sse" || server.Type == "streamableHttp" || server.Type == "streamable-http") && server.BaseURL == "" {
			return fmt.Errorf("server %s: baseUrl is required for sse/streamableHttp type", name)
		}

		// 设置默认超时时间
		if server.Timeout == 0 {
			server.Timeout = 300 // 默认 5 分钟
		}

		// 设置默认工具前缀
		if server.ToolPrefix == "" {
			server.ToolPrefix = name + "_"
		}
	}

	return nil
}

// isJSON 判断文件是否为 JSON 格式
func isJSON(filename string) bool {
	return strings.HasSuffix(filename, ".json")
}
