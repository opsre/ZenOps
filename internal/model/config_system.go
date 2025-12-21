package model

import (
	"time"
)

// SystemConfig 系统配置模型
type SystemConfig struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	ConfigKey   string    `gorm:"size:100;not null;uniqueIndex" json:"config_key"`
	ConfigValue string    `gorm:"type:text" json:"config_value"`
	Description string    `gorm:"type:text" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName 指定表名
func (SystemConfig) TableName() string {
	return "system_config"
}

// 系统配置键常量
const (
	ConfigKeyServerHTTPEnabled                     = "server.http.enabled"
	ConfigKeyServerHTTPPort                        = "server.http.port"
	ConfigKeyServerMCPEnabled                      = "server.mcp.enabled"
	ConfigKeyServerMCPPort                         = "server.mcp.port"
	ConfigKeyServerMCPAutoRegisterExternalTools    = "server.mcp.auto_register_external_tools"
	ConfigKeyServerMCPToolNameFormat               = "server.mcp.tool_name_format"
	ConfigKeyAuthEnabled                           = "auth.enabled"
	ConfigKeyAuthType                              = "auth.type"
	ConfigKeyAuthTokens                            = "auth.tokens"
	ConfigKeyCacheEnabled                          = "cache.enabled"
	ConfigKeyCacheType                             = "cache.type"
	ConfigKeyCacheTTL                              = "cache.ttl"
)
