package model

import "time"

// KnowledgeDocument 知识库文档模型
type KnowledgeDocument struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	DocType        string    `json:"doc_type" gorm:"size:50"`           // 'markdown', 'pdf', 'url', 'manual'
	Title          string    `json:"title" gorm:"size:255"`
	Content        string    `json:"content" gorm:"type:text"`
	Metadata       string    `json:"metadata" gorm:"type:json"`         // 存储来源、作者等元信息
	Enabled        bool      `json:"enabled" gorm:"default:true;index"`
	Category       string    `json:"category" gorm:"size:100;index"`    // 分类：运维文档、API文档等
	Tags           string    `json:"tags" gorm:"type:text"`             // JSON 数组 ["tag1", "tag2"]
	Embedding      string    `json:"embedding" gorm:"type:text"`        // JSON 格式的向量 (用于语义搜索)
	EmbeddingModel string    `json:"embedding_model" gorm:"size:64"`    // Embedding 模型标识
}

// TableName 指定表名
func (KnowledgeDocument) TableName() string {
	return "knowledge_documents"
}
