package model

import (
	"time"
)

// LLMProviderInstance LLM 提供商实例
type LLMProviderInstance struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Enabled  bool   `json:"enabled"`
	Provider string `json:"provider"` // "openai" | "anthropic" | "deepseek" | etc.
	Model    string `json:"model"`
	APIKey   string `json:"api_key"`
	BaseURL  string `json:"base_url"`
}

// LLMConfig LLM配置模型
type LLMConfig struct {
	ID        uint                  `gorm:"primaryKey" json:"id"`
	Providers []LLMProviderInstance `gorm:"serializer:json;type:text" json:"providers"`
	CreatedAt time.Time             `json:"created_at"`
	UpdatedAt time.Time             `json:"updated_at"`
}

// TableName 指定表名
func (LLMConfig) TableName() string {
	return "llm_config"
}
