package model

import (
	"time"
)

// LLMConfig LLM配置模型 - 每条记录代表一个LLM实例
type LLMConfig struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:100;not null" json:"name"`
	Enabled   bool      `gorm:"default:true" json:"enabled"`
	Provider  string    `gorm:"size:50;not null" json:"provider"` // "openai" | "anthropic" | "deepseek" | etc.
	Model     string    `gorm:"size:100;not null" json:"model"`
	APIKey    string    `gorm:"size:500" json:"api_key"`
	BaseURL   string    `gorm:"size:500" json:"base_url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (LLMConfig) TableName() string {
	return "llm_config"
}
