package model

import "time"

// MCPLog MCP 调用日志
type MCPLog struct {
	ID           uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Timestamp    time.Time `json:"timestamp" gorm:"index"`
	ServerName   string    `json:"server_name" gorm:"index;size:100"`
	ToolName     string    `json:"tool_name" gorm:"index;size:100"`
	Status       string    `json:"status" gorm:"size:20;index"` // "success" | "error"
	Latency      int64     `json:"latency"` // 延迟(毫秒)
	Username     string    `json:"username" gorm:"index;size:100"` // 调用用户
	Source       string    `json:"source" gorm:"size:50"` // 调用来源: "admin_test", "dingtalk", "feishu", "wecom", "llm"
	ChatLogID    uint      `json:"chat_log_id" gorm:"index"` // 关联的对话记录ID（如果是从LLM调用）
	Request      string    `json:"request" gorm:"type:text"` // 请求参数 JSON
	Response     string    `json:"response" gorm:"type:text"` // 响应结果 JSON
	ErrorMessage string    `json:"error_message" gorm:"type:text"` // 错误信息（如果失败）
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// TableName 指定表名
func (MCPLog) TableName() string {
	return "mcp_logs"
}
