package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// JSONMap JSON对象类型
type JSONMap map[string]interface{}

// Scan 实现 sql.Scanner 接口
func (jm *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*jm = make(map[string]interface{})
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal JSONMap value: %v", value)
	}

	// 初始化 map
	*jm = make(map[string]interface{})

	// 如果是空字符串或空JSON对象，直接返回
	if len(bytes) == 0 || string(bytes) == "{}" || string(bytes) == "null" {
		return nil
	}

	// 尝试反序列化
	if err := json.Unmarshal(bytes, jm); err != nil {
		return fmt.Errorf("failed to unmarshal JSONMap: %w", err)
	}

	return nil
}

// Value 实现 driver.Valuer 接口
func (jm JSONMap) Value() (driver.Value, error) {
	if len(jm) == 0 {
		return "{}", nil
	}
	return json.Marshal(jm)
}

// IMConfig IM配置模型 (钉钉/飞书/企微)
type IMConfig struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Platform   string    `gorm:"size:50;not null;uniqueIndex" json:"platform"` // dingtalk, feishu, wecom
	Enabled    bool      `gorm:"default:true" json:"enabled"`
	ConfigData JSONMap   `gorm:"type:text;not null" json:"config_data"` // 存储各平台的具体配置
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// TableName 指定表名
func (IMConfig) TableName() string {
	return "im_config"
}

// DingTalkConfig 钉钉配置结构
type DingTalkConfig struct {
	AppKey         string `json:"app_key"`
	AppSecret      string `json:"app_secret"`
	AgentID        string `json:"agent_id"`
	CardTemplateID string `json:"card_template_id"`
}

// FeishuConfig 飞书配置结构
type FeishuConfig struct {
	AppID     string `json:"app_id"`
	AppSecret string `json:"app_secret"`
}

// WecomConfig 企微配置结构
type WecomConfig struct {
	Token          string `json:"token"`
	EncodingAESKey string `json:"encoding_aes_key"`
}
