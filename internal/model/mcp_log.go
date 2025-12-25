package model

import "time"

// MCPLog MCP 调用日志
type MCPLog struct {
	ID         string    `gorm:"primaryKey;size:50" json:"id"`
	Timestamp  time.Time `json:"timestamp" gorm:"index"`
	ServerName string    `json:"serverName" gorm:"index;size:100"`
	ToolName   string    `json:"toolName" gorm:"index;size:100"`
	Status     string    `json:"status" gorm:"size:20"` // "success" | "error" | "warning"
	Latency    int       `json:"latency"` // 延迟(毫秒)
	Request    string    `json:"request" gorm:"type:text"`
	Response   string    `json:"response" gorm:"type:text"`
	CreatedAt  time.Time `json:"created_at"`
}

// TableName 指定表名
func (MCPLog) TableName() string {
	return "mcp_logs"
}
