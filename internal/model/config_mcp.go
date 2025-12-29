package model

import (
	"time"
)

// MCPServer MCP服务器配置模型
type MCPServer struct {
	ID            uint        `gorm:"primaryKey" json:"id"`
	Name          string      `gorm:"size:100;not null;uniqueIndex" json:"name"`
	IsActive      bool        `gorm:"default:false" json:"is_active"`
	Type          string      `gorm:"size:50;not null" json:"type"` // stdio, sse, streamableHttp
	Description   string      `gorm:"type:text" json:"description"`
	BaseURL       string      `gorm:"type:text" json:"base_url"`
	Command       string      `gorm:"type:text" json:"command"`
	Args          StringArray `gorm:"type:text" json:"args"`
	Env           JSONMap     `gorm:"type:text" json:"env"`
	Headers       JSONMap     `gorm:"type:text" json:"headers"`
	LongRunning   bool        `gorm:"default:true" json:"long_running"`
	Timeout       int         `gorm:"default:300" json:"timeout"`
	InstallSource string      `gorm:"size:50" json:"install_source"`
	ToolPrefix    string      `gorm:"size:50" json:"tool_prefix"`
	AutoRegister  bool        `gorm:"default:true" json:"auto_register"`
	Provider      string      `gorm:"size:100" json:"provider"`
	ProviderURL   string      `gorm:"type:text" json:"provider_url"`
	LogoURL       string      `gorm:"type:text" json:"logo_url"`
	Tags          StringArray `gorm:"type:text" json:"tags"`
	Tools         []MCPTool   `gorm:"foreignKey:ServerID" json:"tools"` // 关联的工具列表
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
}

// TableName 指定表名
func (MCPServer) TableName() string {
	return "mcp_servers"
}
