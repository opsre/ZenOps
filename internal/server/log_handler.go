package server

import (
	"net/http"
	"strconv"

	"github.com/eryajf/zenops/internal/model"
	"github.com/eryajf/zenops/internal/service"
	"github.com/gin-gonic/gin"
)

// LogHandler 日志处理器
type LogHandler struct {
	configService *service.ConfigService
	mcpLogService *service.MCPLogService
}

// NewLogHandler 创建日志处理器
func NewLogHandler() *LogHandler {
	return &LogHandler{
		configService: service.NewConfigService(),
		mcpLogService: service.NewMCPLogService(),
	}
}

// GetMCPLogs 获取 MCP 调用日志
func (h *LogHandler) GetMCPLogs(c *gin.Context) {
	// 分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	search := c.Query("search")
	status := c.Query("status")
	serverName := c.Query("server_name")
	toolName := c.Query("tool_name")
	source := c.Query("source")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 构建查询
	db := h.configService.GetDB()
	query := db.Model(&model.MCPLog{})

	// 搜索过滤（搜索服务器名、工具名、用户名）
	if search != "" {
		query = query.Where("tool_name LIKE ? OR server_name LIKE ? OR username LIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	// 状态过滤
	if status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}

	// 服务器名过滤
	if serverName != "" {
		query = query.Where("server_name = ?", serverName)
	}

	// 工具名过滤
	if toolName != "" {
		query = query.Where("tool_name = ?", toolName)
	}

	// 来源过滤
	if source != "" && source != "all" {
		query = query.Where("source = ?", source)
	}

	// 计算总数
	var total int64
	query.Count(&total)

	// 分页查询
	var logs []model.MCPLog
	offset := (page - 1) * pageSize
	query.Order("timestamp DESC").Offset(offset).Limit(pageSize).Find(&logs)

	// 返回完整字段
	items := make([]gin.H, len(logs))
	for i, log := range logs {
		items[i] = gin.H{
			"id":            log.ID,
			"timestamp":     log.Timestamp.Format("2006-01-02 15:04:05"),
			"server_name":   log.ServerName,
			"tool_name":     log.ToolName,
			"status":        log.Status,
			"latency":       log.Latency,
			"username":      log.Username,
			"source":        log.Source,
			"chat_log_id":   log.ChatLogID,
			"request":       log.Request,
			"response":      log.Response,
			"error_message": log.ErrorMessage,
			"created_at":    log.CreatedAt.Format("2006-01-02 15:04:05"),
			"updated_at":    log.UpdatedAt.Format("2006-01-02 15:04:05"),
		}
	}

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data: gin.H{
			"total":    total,
			"page":     page,
			"pageSize": pageSize,
			"items":    items,
		},
	})
}

// GetMCPLogStats 获取 MCP 调用统计信息
func (h *LogHandler) GetMCPLogStats(c *gin.Context) {
	stats, err := h.mcpLogService.GetMCPLogStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    stats,
	})
}
