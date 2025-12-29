package model

import "time"

// MCPTool MCP 工具模型
type MCPTool struct {
	ID          uint                   `gorm:"primaryKey" json:"id"`
	ServerID    uint                   `json:"server_id" gorm:"index;uniqueIndex:idx_server_tool"`
	Name        string                 `json:"name" gorm:"not null;size:100;uniqueIndex:idx_server_tool"`
	Description string                 `json:"description" gorm:"type:text"`
	IsEnabled   bool                   `json:"isEnabled" gorm:"default:true"`
	InputSchema map[string]interface{} `json:"inputSchema" gorm:"serializer:json;type:text"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// TableName 指定表名
func (MCPTool) TableName() string {
	return "mcp_tools"
}
