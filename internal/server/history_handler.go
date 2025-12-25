package server

import (
	"net/http"
	"strconv"

	"github.com/eryajf/zenops/internal/model"
	"github.com/eryajf/zenops/internal/service"
	"github.com/gin-gonic/gin"
)

// HistoryHandler 对话历史处理器
type HistoryHandler struct {
	configService *service.ConfigService
}

// NewHistoryHandler 创建对话历史处理器
func NewHistoryHandler() *HistoryHandler {
	return &HistoryHandler{
		configService: service.NewConfigService(),
	}
}

// GetChatLogs 获取对话记录
func (h *HistoryHandler) GetChatLogs(c *gin.Context) {
	// 分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	search := c.Query("search")
	source := c.Query("source")
	chatTypeStr := c.Query("chat_type")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 构建查询
	db := h.configService.GetDB()
	query := db.Model(&model.ChatLog{}).Where("deleted_at IS NULL")

	// 搜索过滤
	if search != "" {
		query = query.Where("content LIKE ? OR username LIKE ?",
			"%"+search+"%", "%"+search+"%")
	}

	// 来源过滤
	if source != "" && source != "all" {
		query = query.Where("source = ?", source)
	}

	// 类型过滤
	if chatTypeStr != "" && chatTypeStr != "all" {
		chatType, _ := strconv.Atoi(chatTypeStr)
		query = query.Where("chat_type = ?", chatType)
	}

	// 计算总数
	var total int64
	query.Count(&total)

	// 分页查询
	var logs []model.ChatLog
	offset := (page - 1) * pageSize
	query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&logs)

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data: gin.H{
			"total":    total,
			"page":     page,
			"pageSize": pageSize,
			"items":    logs,
		},
	})
}

// GetChatContext 获取消息上下文
func (h *HistoryHandler) GetChatContext(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "invalid id",
		})
		return
	}

	// 获取当前消息
	db := h.configService.GetDB()
	var current model.ChatLog
	if err := db.First(&current, id).Error; err != nil {
		c.JSON(http.StatusNotFound, Response{
			Code:    404,
			Message: "message not found",
		})
		return
	}

	// 获取父消息
	var parent *model.ChatLog
	if current.ParentContent > 0 {
		var p model.ChatLog
		if err := db.First(&p, current.ParentContent).Error; err == nil {
			parent = &p
		}
	}

	// 获取子消息
	var children []model.ChatLog
	db.Where("parent_content = ?", current.ID).Find(&children)

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data: gin.H{
			"current":  current,
			"parent":   parent,
			"children": children,
		},
	})
}
