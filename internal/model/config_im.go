package model

import (
	"time"
)

// IMConfig IM配置模型 (钉钉/飞书/企微)
type IMConfig struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Platform   string    `gorm:"size:50;not null;uniqueIndex" json:"platform"` // dingtalk, feishu, wecom
	Enabled    bool      `gorm:"default:false" json:"enabled"`
	AppID      string    `gorm:"column:app_id;size:200" json:"app_id"`           // 应用ID
	AppKey     string    `gorm:"column:app_key;size:200" json:"app_key"`         // 应用Key/Secret
	AgentID    string    `gorm:"column:agent_id;size:200" json:"agent_id"`       // Agent ID
	TemplateID string    `gorm:"column:template_id;size:200" json:"template_id"` // 模板ID
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// TableName 指定表名
func (IMConfig) TableName() string {
	return "im_config"
}
