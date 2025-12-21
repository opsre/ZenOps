package model

import (
	"time"
)

// CICDConfig CICD配置模型
type CICDConfig struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Platform  string    `gorm:"size:50;not null;uniqueIndex" json:"platform"` // jenkins
	Enabled   bool      `gorm:"default:true" json:"enabled"`
	URL       string    `gorm:"type:text;not null" json:"url"`
	Username  string    `gorm:"size:100" json:"username"`
	Token     string    `gorm:"type:text" json:"token"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (CICDConfig) TableName() string {
	return "cicd_config"
}
