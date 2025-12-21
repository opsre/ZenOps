package model

import (
	"time"
)

// LLMConfig LLM配置模型
type LLMConfig struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Enabled   bool      `gorm:"default:true" json:"enabled"`
	Model     string    `gorm:"size:100;not null" json:"model"`
	APIKey    string    `gorm:"type:text;not null" json:"api_key"`
	BaseURL   string    `gorm:"type:text" json:"base_url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (LLMConfig) TableName() string {
	return "llm_config"
}
