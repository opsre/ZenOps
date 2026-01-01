package model

import "time"

// ChatLog 对话记录模型
type ChatLog struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	DeletedAt      *time.Time `json:"deleted_at" gorm:"index"`
	Username       string     `json:"username" gorm:"index;size:100"`
	Source         string     `json:"source" gorm:"size:50"`      // "私聊" | "群聊"
	ChatType       int        `json:"chat_type" gorm:"index"`     // 1=用户提问, 2=AI回答
	ParentContent  uint       `json:"parent_content"`             // 父消息ID
	ConversationID uint       `json:"conversation_id" gorm:"index"` // 所属会话ID
	Content        string     `json:"content" gorm:"type:text"`
}

// TableName 指定表名
func (ChatLog) TableName() string {
	return "chat_logs"
}
