package model

import "time"

// Instance 统一的实例模型 (跨云平台)
type Instance struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Provider     string            `json:"provider"`      // 提供商: aliyun, tencent
	Region       string            `json:"region"`        // 区域
	Zone         string            `json:"zone"`          // 可用区
	InstanceType string            `json:"instance_type"` // 实例规格
	Status       string            `json:"status"`        // 状态
	PrivateIP    []string          `json:"private_ip"`
	PublicIP     []string          `json:"public_ip"`
	CPU          int               `json:"cpu"`
	Memory       int               `json:"memory"` // MB
	OSType       string            `json:"os_type"`
	OSName       string            `json:"os_name"`
	CreatedAt    time.Time         `json:"created_at"`
	ExpiredAt    *time.Time        `json:"expired_at,omitempty"`
	Tags         map[string]string `json:"tags"`
	Metadata     map[string]any    `json:"metadata"`    // 扩展字段
	ConsoleURL   string            `json:"console_url"` // 控制台跳转地址
}

// InstanceList 实例列表
type InstanceList struct {
	Items    []*Instance `json:"items"`
	PageInfo *PageInfo   `json:"page_info,omitempty"`
}
