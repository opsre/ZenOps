package model

import "time"

// Job Jenkins 任务模型
type Job struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	URL         string `json:"url"`
	Description string `json:"description"`
	Buildable   bool   `json:"buildable"`
	LastBuild   *Build `json:"last_build,omitempty"`
}

// Build 构建模型
type Build struct {
	Number    int       `json:"number"`
	Status    string    `json:"status"`
	Result    string    `json:"result"`
	Timestamp time.Time `json:"timestamp"`
	Duration  int64     `json:"duration"` // 毫秒
	URL       string    `json:"url"`
}

// JobList 任务列表
type JobList struct {
	Items    []*Job    `json:"items"`
	PageInfo *PageInfo `json:"page_info,omitempty"`
}
