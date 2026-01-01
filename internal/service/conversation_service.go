package service

import (
	"time"

	"gorm.io/gorm"

	"github.com/eryajf/zenops/internal/database"
	"github.com/eryajf/zenops/internal/model"
)

// ConversationService 会话服务
type ConversationService struct {
	db *gorm.DB
}

// NewConversationService 创建会话服务实例
func NewConversationService() *ConversationService {
	return &ConversationService{
		db: database.GetDB(),
	}
}

// CreateConversation 创建会话
func (s *ConversationService) CreateConversation(username, title string) (*model.Conversation, error) {
	conversation := &model.Conversation{
		Username:      username,
		Title:         title,
		LastMessageAt: time.Now(),
	}
	if err := s.db.Create(conversation).Error; err != nil {
		return nil, err
	}
	return conversation, nil
}

// GetConversation 获取会话
func (s *ConversationService) GetConversation(id uint) (*model.Conversation, error) {
	var conversation model.Conversation
	if err := s.db.First(&conversation, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &conversation, nil
}

// ListConversations 列出用户的会话列表
func (s *ConversationService) ListConversations(username string) ([]model.Conversation, error) {
	var conversations []model.Conversation
	err := s.db.Where("username = ?", username).
		Order("last_message_at DESC").
		Find(&conversations).Error
	return conversations, err
}

// UpdateConversation 更新会话
func (s *ConversationService) UpdateConversation(id uint, title string) error {
	return s.db.Model(&model.Conversation{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"title":      title,
			"updated_at": time.Now(),
		}).Error
}

// UpdateLastMessageAt 更新会话最后消息时间
func (s *ConversationService) UpdateLastMessageAt(id uint) error {
	return s.db.Model(&model.Conversation{}).
		Where("id = ?", id).
		Update("last_message_at", time.Now()).Error
}

// DeleteConversation 删除会话（软删除）
func (s *ConversationService) DeleteConversation(id uint) error {
	// 删除会话下的所有消息
	if err := s.db.Where("conversation_id = ?", id).Delete(&model.ChatLog{}).Error; err != nil {
		return err
	}
	// 删除会话
	return s.db.Delete(&model.Conversation{}, id).Error
}

// GetConversationMessages 获取会话的所有消息
func (s *ConversationService) GetConversationMessages(conversationID uint) ([]model.ChatLog, error) {
	var messages []model.ChatLog
	err := s.db.Where("conversation_id = ?", conversationID).
		Order("created_at ASC").
		Find(&messages).Error
	return messages, err
}

// ShouldGenerateTitle 检查是否需要生成标题（标题为默认值且有消息）
func (s *ConversationService) ShouldGenerateTitle(conversationID uint) (bool, error) {
	conversation, err := s.GetConversation(conversationID)
	if err != nil || conversation == nil {
		return false, err
	}
	// 如果标题是"新会话"，说明需要生成标题
	return conversation.Title == "新会话", nil
}
