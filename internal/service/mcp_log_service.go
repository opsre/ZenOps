package service

import (
	"encoding/json"
	"time"

	"github.com/eryajf/zenops/internal/database"
	"github.com/eryajf/zenops/internal/model"
	"gorm.io/gorm"
)

// MCPLogService MCP 日志服务
type MCPLogService struct {
	db *gorm.DB
}

// NewMCPLogService 创建 MCP 日志服务
func NewMCPLogService() *MCPLogService {
	return &MCPLogService{
		db: database.GetDB(),
	}
}

// MCPLogParams MCP 日志参数
type MCPLogParams struct {
	ServerName   string
	ToolName     string
	Username     string
	Source       string // "admin_test", "dingtalk", "feishu", "wecom", "llm"
	ChatLogID    uint   // 关联的对话记录ID（可选）
	Request      map[string]interface{}
	Response     interface{}
	ErrorMessage string
	Latency      int64 // 毫秒
	Success      bool
}

// CreateMCPLog 创建 MCP 调用日志
func (s *MCPLogService) CreateMCPLog(params *MCPLogParams) (*model.MCPLog, error) {
	// 序列化请求参数
	requestJSON, err := json.Marshal(params.Request)
	if err != nil {
		requestJSON = []byte("{}")
	}

	// 序列化响应结果
	responseJSON, err := json.Marshal(params.Response)
	if err != nil {
		responseJSON = []byte("{}")
	}

	status := "success"
	if !params.Success {
		status = "error"
	}

	log := &model.MCPLog{
		Timestamp:    time.Now(),
		ServerName:   params.ServerName,
		ToolName:     params.ToolName,
		Status:       status,
		Latency:      params.Latency,
		Username:     params.Username,
		Source:       params.Source,
		ChatLogID:    params.ChatLogID,
		Request:      string(requestJSON),
		Response:     string(responseJSON),
		ErrorMessage: params.ErrorMessage,
	}

	if err := s.db.Create(log).Error; err != nil {
		return nil, err
	}

	return log, nil
}

// ListMCPLogs 查询 MCP 日志列表
func (s *MCPLogService) ListMCPLogs(serverName, toolName, status string, limit, offset int) ([]model.MCPLog, int64, error) {
	var logs []model.MCPLog
	var total int64

	query := s.db.Model(&model.MCPLog{})

	// 过滤条件
	if serverName != "" {
		query = query.Where("server_name = ?", serverName)
	}
	if toolName != "" {
		query = query.Where("tool_name = ?", toolName)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	if err := query.Order("timestamp DESC").
		Limit(limit).
		Offset(offset).
		Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// GetMCPLogByID 根据ID获取日志
func (s *MCPLogService) GetMCPLogByID(id uint) (*model.MCPLog, error) {
	var log model.MCPLog
	if err := s.db.First(&log, id).Error; err != nil {
		return nil, err
	}
	return &log, nil
}

// DeleteMCPLog 删除日志
func (s *MCPLogService) DeleteMCPLog(id uint) error {
	return s.db.Delete(&model.MCPLog{}, id).Error
}

// GetMCPLogStats 获取统计信息
func (s *MCPLogService) GetMCPLogStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 总调用次数
	var total int64
	if err := s.db.Model(&model.MCPLog{}).Count(&total).Error; err != nil {
		return nil, err
	}
	stats["total"] = total

	// 成功次数
	var successCount int64
	if err := s.db.Model(&model.MCPLog{}).Where("status = ?", "success").Count(&successCount).Error; err != nil {
		return nil, err
	}
	stats["success_count"] = successCount

	// 失败次数
	var errorCount int64
	if err := s.db.Model(&model.MCPLog{}).Where("status = ?", "error").Count(&errorCount).Error; err != nil {
		return nil, err
	}
	stats["error_count"] = errorCount

	// 平均延迟
	var avgLatency float64
	if err := s.db.Model(&model.MCPLog{}).Select("COALESCE(AVG(latency), 0)").Scan(&avgLatency).Error; err != nil {
		return nil, err
	}
	stats["avg_latency"] = avgLatency

	return stats, nil
}
