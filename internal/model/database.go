package model

import "time"

// Database 数据库模型
type Database struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Provider      string            `json:"provider"`
	Region        string            `json:"region"`
	Engine        string            `json:"engine"`         // mysql, postgresql, redis
	EngineVersion string            `json:"engine_version"`
	Status        string            `json:"status"`
	Endpoint      string            `json:"endpoint"`
	Port          int               `json:"port"`
	CreatedAt     time.Time         `json:"created_at"`
	Tags          map[string]string `json:"tags"`
	ConsoleURL    string            `json:"console_url"` // 控制台跳转地址
}

// DatabaseList 数据库列表
type DatabaseList struct {
	Items    []*Database `json:"items"`
	PageInfo *PageInfo   `json:"page_info,omitempty"`
}
