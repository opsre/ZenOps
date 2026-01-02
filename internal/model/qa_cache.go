package model

import "time"

// QACache 问答缓存模型
type QACache struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	QuestionHash string     `json:"question_hash" gorm:"size:64;not null;index"` // 问题的语义哈希
	Question     string     `json:"question" gorm:"type:text;not null"`
	Answer       string     `json:"answer" gorm:"type:text"`
	Username     string     `json:"username" gorm:"size:100;index"` // 可选：用户级别缓存
	HitCount     int        `json:"hit_count" gorm:"default:1;index"`
	LastHitAt    time.Time  `json:"last_hit_at"`
}

// TableName 指定表名
func (QACache) TableName() string {
	return "qa_cache"
}
