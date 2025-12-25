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
}

// NewLogHandler 创建日志处理器
func NewLogHandler() *LogHandler {
	return &LogHandler{
		configService: service.NewConfigService(),
	}
}

// GetMCPLogs 获取 MCP 调用日志
func (h *LogHandler) GetMCPLogs(c *gin.Context) {
	// 分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	search := c.Query("search")
	status := c.Query("status")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 构建查询
	db := h.configService.GetDB()
	query := db.Model(&model.MCPLog{})

	// 搜索过滤
	if search != "" {
		query = query.Where("tool_name LIKE ? OR server_name LIKE ? OR request LIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	// 状态过滤
	if status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}

	// 计算总数
	var total int64
	query.Count(&total)

	// 分页查询
	var logs []model.MCPLog
	offset := (page - 1) * pageSize
	query.Order("timestamp DESC").Offset(offset).Limit(pageSize).Find(&logs)

	// 格式化时间戳
	items := make([]gin.H, len(logs))
	for i, log := range logs {
		items[i] = gin.H{
			"id":         log.ID,
			"timestamp":  log.Timestamp.Format("2006-01-02 15:04:05"),
			"serverName": log.ServerName,
			"toolName":   log.ToolName,
			"status":     log.Status,
			"latency":    log.Latency,
			"request":    log.Request,
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
