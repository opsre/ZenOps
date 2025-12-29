package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// StringArray 字符串数组类型,用于存储 regions
type StringArray []string

// Scan 实现 sql.Scanner 接口
func (sa *StringArray) Scan(value interface{}) error {
	if value == nil {
		*sa = []string{}
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("failed to unmarshal StringArray value: %v", value)
	}

	// 处理空字符串
	if len(bytes) == 0 {
		*sa = []string{}
		return nil
	}

	return json.Unmarshal(bytes, sa)
}

// Value 实现 driver.Valuer 接口
func (sa StringArray) Value() (driver.Value, error) {
	if len(sa) == 0 {
		return "[]", nil
	}
	return json.Marshal(sa)
}

// ProviderAccount 云厂商账号配置模型
type ProviderAccount struct {
	ID        uint        `gorm:"primaryKey" json:"id"`
	Provider  string      `gorm:"size:50;not null;index:idx_provider_name" json:"provider"` // aliyun, tencent
	Name      string      `gorm:"size:100;not null;index:idx_provider_name" json:"name"`
	Enabled   bool        `gorm:"default:true" json:"enabled"`
	AccessKey string      `gorm:"type:text;not null" json:"access_key"`
	SecretKey string      `gorm:"type:text;not null" json:"secret_key"`
	Regions   StringArray `gorm:"type:text" json:"regions"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

// TableName 指定表名
func (ProviderAccount) TableName() string {
	return "provider_accounts"
}
