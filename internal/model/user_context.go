package model

import "time"

// UserContext 用户上下文模型
type UserContext struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	Username     string     `json:"username" gorm:"index;size:100;not null"`
	ContextKey   string     `json:"context_key" gorm:"size:100;not null"`   // 如: "favorite_region", "default_vpc"
	ContextValue string     `json:"context_value" gorm:"type:text"`          // JSON 格式存储值
	ContextType  string     `json:"context_type" gorm:"size:20;default:user"` // user/system/auto_learned
}

// TableName 指定表名
func (UserContext) TableName() string {
	return "user_contexts"
}
