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

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
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

// Response 通用响应结构
type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// PageInfo 分页信息
type PageInfo struct {
	PageNum   int `json:"page_num"`
	PageSize  int `json:"page_size"`
	Total     int `json:"total"`
	TotalPage int `json:"total_page"`
}

// ListResponse 列表响应
type ListResponse struct {
	Items    any       `json:"items"`
	PageInfo *PageInfo `json:"page_info,omitempty"`
}

// TimeRange 时间范围
type TimeRange struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}
