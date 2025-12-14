package model

// OSSBucket 统一的 OSS Bucket 模型 (跨云平台)
type OSSBucket struct {
	Name         string         `json:"name"`
	Provider     string         `json:"provider"`      // 提供商: aliyun, tencent
	Region       string         `json:"region"`        // 区域
	StorageClass string         `json:"storage_class"` // 存储类型
	ACL          string         `json:"acl"`           // 访问控制
	CreatedAt    string         `json:"created_at"`
	Metadata     map[string]any `json:"metadata"`    // 扩展字段
	ConsoleURL   string         `json:"console_url"` // 控制台跳转地址
}

// OSSBucketList Bucket 列表
type OSSBucketList struct {
	Items    []*OSSBucket `json:"items"`
	PageInfo *PageInfo    `json:"page_info,omitempty"`
}
