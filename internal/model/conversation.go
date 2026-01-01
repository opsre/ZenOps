package model

import "time"

// Conversation 会话模型
type Conversation struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	DeletedAt     *time.Time `json:"deleted_at" gorm:"index"`
	Username      string     `json:"username" gorm:"index;size:100"` // 所属用户
	Title         string     `json:"title" gorm:"size:255"`          // 会话标题
	LastMessageAt time.Time  `json:"last_message_at" gorm:"index"`   // 最后消息时间，用于排序
}

// TableName 指定表名
func (Conversation) TableName() string {
	return "conversations"
}
