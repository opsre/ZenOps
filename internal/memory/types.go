package memory

import "time"

// UserContext 用户上下文（内存结构）
type UserContext struct {
	Username       string            `json:"username"`
	Contexts       map[string]string `json:"contexts"`        // key-value 上下文
	FavoriteRegion string            `json:"favorite_region"` // 常用地域
	DefaultVPC     string            `json:"default_vpc"`     // 默认 VPC
	CustomFields   map[string]any    `json:"custom_fields"`   // 自定义字段
}

// Message 消息结构（兼容 LLM）
type Message struct {
	Role      string    `json:"role"`       // user/assistant/tool/system
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// CacheStats 缓存统计
type CacheStats struct {
	HitCount     int64   `json:"hit_count"`
	MissCount    int64   `json:"miss_count"`
	HitRate      float64 `json:"hit_rate"`
	TotalQueries int64   `json:"total_queries"`
}
