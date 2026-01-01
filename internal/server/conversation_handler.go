package server

import (
	"net/http"
	"strconv"

	"github.com/eryajf/zenops/internal/service"
	"github.com/gin-gonic/gin"
)

// ConversationHandler 会话处理器
type ConversationHandler struct {
	conversationService *service.ConversationService
	chatLogService      *service.ChatLogService
}

// NewConversationHandler 创建会话处理器
func NewConversationHandler() *ConversationHandler {
	return &ConversationHandler{
		conversationService: service.NewConversationService(),
		chatLogService:      service.NewChatLogService(),
	}
}

// CreateConversationRequest 创建会话请求
type CreateConversationRequest struct {
	Title string `json:"title" binding:"required"`
}

// UpdateConversationRequest 更新会话请求
type UpdateConversationRequest struct {
	Title string `json:"title" binding:"required"`
}

// ConversationWithMessages 会话及其消息
type ConversationWithMessages struct {
	ID            uint      `json:"id"`
	Title         string    `json:"title"`
	Username      string    `json:"username"`
	CreatedAt     string    `json:"created_at"`
	UpdatedAt     string    `json:"updated_at"`
	LastMessageAt string    `json:"last_message_at"`
	Messages      []Message `json:"messages"`
}

// Message 消息
type Message struct {
	ID        uint   `json:"id"`
	Role      string `json:"role"` // "user" or "model"
	Content   string `json:"content"`
	Timestamp string `json:"timestamp"`
}

// CreateConversation 创建会话
func (h *ConversationHandler) CreateConversation(c *gin.Context) {
	var req CreateConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	// 获取用户名（从请求头或使用默认值）
	username := c.GetHeader("X-Username")
	if username == "" {
		username = "api_user"
	}

	conversation, err := h.conversationService.CreateConversation(username, req.Title)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "Failed to create conversation: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "success",
		Data:    conversation,
	})
}

// ListConversations 列出用户的会话列表
func (h *ConversationHandler) ListConversations(c *gin.Context) {
	// 获取用户名（从请求头或使用默认值）
	username := c.GetHeader("X-Username")
	if username == "" {
		username = "api_user"
	}

	conversations, err := h.conversationService.ListConversations(username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "Failed to list conversations: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "success",
		Data:    conversations,
	})
}

// GetConversation 获取会话详情（包含所有消息）
func (h *ConversationHandler) GetConversation(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "Invalid conversation ID",
		})
		return
	}

	conversation, err := h.conversationService.GetConversation(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "Failed to get conversation: " + err.Error(),
		})
		return
	}

	if conversation == nil {
		c.JSON(http.StatusNotFound, Response{
			Code:    404,
			Message: "Conversation not found",
		})
		return
	}

	// 获取会话的所有消息
	chatLogs, err := h.conversationService.GetConversationMessages(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "Failed to get messages: " + err.Error(),
		})
		return
	}

	// 转换消息格式
	messages := make([]Message, 0, len(chatLogs))
	for _, log := range chatLogs {
		role := "user"
		if log.ChatType == 2 {
			role = "model"
		}
		messages = append(messages, Message{
			ID:        log.ID,
			Role:      role,
			Content:   log.Content,
			Timestamp: log.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	result := ConversationWithMessages{
		ID:            conversation.ID,
		Title:         conversation.Title,
		Username:      conversation.Username,
		CreatedAt:     conversation.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:     conversation.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		LastMessageAt: conversation.LastMessageAt.Format("2006-01-02T15:04:05Z07:00"),
		Messages:      messages,
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "success",
		Data:    result,
	})
}

// UpdateConversation 更新会话
func (h *ConversationHandler) UpdateConversation(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "Invalid conversation ID",
		})
		return
	}

	var req UpdateConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	if err := h.conversationService.UpdateConversation(uint(id), req.Title); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "Failed to update conversation: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "success",
	})
}

// DeleteConversation 删除会话
func (h *ConversationHandler) DeleteConversation(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "Invalid conversation ID",
		})
		return
	}

	if err := h.conversationService.DeleteConversation(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "Failed to delete conversation: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "success",
	})
}
