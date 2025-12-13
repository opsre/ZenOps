package model

import "time"

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
