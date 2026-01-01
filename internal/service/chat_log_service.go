package service

import (
	"gorm.io/gorm"

	"github.com/eryajf/zenops/internal/database"
	"github.com/eryajf/zenops/internal/model"
)

// ChatLogService 对话日志服务
type ChatLogService struct {
	db *gorm.DB
}

// NewChatLogService 创建对话日志服务实例
func NewChatLogService() *ChatLogService {
	return &ChatLogService{
		db: database.GetDB(),
	}
}

// GetDB 获取数据库连接
func (s *ChatLogService) GetDB() *gorm.DB {
	return s.db
}

// CreateChatLog 创建对话日志
func (s *ChatLogService) CreateChatLog(log *model.ChatLog) error {
	return s.db.Create(log).Error
}

// CreateUserMessage 创建用户消息日志
func (s *ChatLogService) CreateUserMessage(username, source, content string) (*model.ChatLog, error) {
	return s.CreateUserMessageWithConversation(username, source, content, 0)
}

// CreateUserMessageWithConversation 创建用户消息日志（带会话ID）
func (s *ChatLogService) CreateUserMessageWithConversation(username, source, content string, conversationID uint) (*model.ChatLog, error) {
	log := &model.ChatLog{
		Username:       username,
		Source:         source,
		ChatType:       1, // 1=用户提问
		ParentContent:  0,
		ConversationID: conversationID,
		Content:        content,
	}
	err := s.CreateChatLog(log)
	if err != nil {
		return nil, err
	}
	return log, nil
}

// CreateAIMessage 创建AI回复日志
func (s *ChatLogService) CreateAIMessage(username, source, content string, parentID uint) (*model.ChatLog, error) {
	return s.CreateAIMessageWithConversation(username, source, content, parentID, 0)
}

// CreateAIMessageWithConversation 创建AI回复日志（带会话ID）
func (s *ChatLogService) CreateAIMessageWithConversation(username, source, content string, parentID, conversationID uint) (*model.ChatLog, error) {
	log := &model.ChatLog{
		Username:       username,
		Source:         source,
		ChatType:       2, // 2=AI回答
		ParentContent:  parentID,
		ConversationID: conversationID,
		Content:        content,
	}
	err := s.CreateChatLog(log)
	if err != nil {
		return nil, err
	}
	return log, nil
}

// GetChatLogByID 根据ID获取对话日志
func (s *ChatLogService) GetChatLogByID(id uint) (*model.ChatLog, error) {
	var log model.ChatLog
	err := s.db.First(&log, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &log, nil
}

// ListChatLogs 列出对话日志
func (s *ChatLogService) ListChatLogs(username, source string, chatType int, limit, offset int) ([]model.ChatLog, int64, error) {
	query := s.db.Model(&model.ChatLog{})

	// 条件过滤
	if username != "" {
		query = query.Where("username = ?", username)
	}
	if source != "" {
		query = query.Where("source = ?", source)
	}
	if chatType > 0 {
		query = query.Where("chat_type = ?", chatType)
	}

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	var logs []model.ChatLog
	err := query.Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&logs).Error

	return logs, total, err
}

// GetConversationContext 获取对话上下文（包括父消息）
func (s *ChatLogService) GetConversationContext(messageID uint) ([]model.ChatLog, error) {
	var messages []model.ChatLog

	// 获取当前消息
	currentMsg, err := s.GetChatLogByID(messageID)
	if err != nil || currentMsg == nil {
		return nil, err
	}

	messages = append(messages, *currentMsg)

	// 递归获取父消息
	for currentMsg.ParentContent > 0 {
		parentMsg, err := s.GetChatLogByID(currentMsg.ParentContent)
		if err != nil || parentMsg == nil {
			break
		}
		messages = append([]model.ChatLog{*parentMsg}, messages...)
		currentMsg = parentMsg
	}

	return messages, nil
}

// DeleteChatLog 删除对话日志（软删除）
func (s *ChatLogService) DeleteChatLog(id uint) error {
	return s.db.Delete(&model.ChatLog{}, id).Error
}
