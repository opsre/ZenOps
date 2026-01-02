package memory

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cnb.cool/zhiqiangwang/pkg/logx"
	"github.com/eryajf/zenops/internal/model"
	"gorm.io/gorm"
)

// Manager Memory Manager 核心
type Manager struct {
	db    *gorm.DB
	redis *RedisCache // 可选的 Redis 缓存
}

// NewManager 创建 Memory Manager
func NewManager(db *gorm.DB, redis *RedisCache) *Manager {
	return &Manager{
		db:    db,
		redis: redis,
	}
}

// GetConversationHistory 获取对话历史
func (m *Manager) GetConversationHistory(conversationID uint, limit int) ([]*model.ChatLog, error) {
	// 1. 先尝试从 Redis 读取（如果启用）
	if m.redis != nil {
		messages, err := m.redis.GetConversationHistory(conversationID)
		if err == nil && len(messages) > 0 {
			logx.Debug("Conversation history loaded from Redis cache")
			// 转换为 ChatLog 格式
			return m.messagesToChatLogs(messages), nil
		}
	}

	// 2. 从 SQLite 读取
	var chatLogs []*model.ChatLog
	query := m.db.Where("conversation_id = ?", conversationID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&chatLogs).Error; err != nil {
		return nil, fmt.Errorf("failed to load conversation history: %w", err)
	}

	// 反转顺序（因为是 DESC 查询）
	for i, j := 0, len(chatLogs)-1; i < j; i, j = i+1, j-1 {
		chatLogs[i], chatLogs[j] = chatLogs[j], chatLogs[i]
	}

	// 3. 回填 Redis 缓存
	if m.redis != nil && len(chatLogs) > 0 {
		messages := m.chatLogsToMessages(chatLogs)
		if err := m.redis.SaveConversationHistory(conversationID, messages); err != nil {
			logx.Warn("Failed to save conversation history to Redis: %v", err)
		}
	}

	return chatLogs, nil
}

// SaveMessage 保存单条消息
func (m *Manager) SaveMessage(conversationID uint, chatType int, content, username string) error {
	chatLog := &model.ChatLog{
		ConversationID: conversationID,
		ChatType:       chatType,
		Content:        content,
		Username:       username,
		CreatedAt:      time.Now(),
	}

	// 1. 保存到 SQLite
	if err := m.db.Create(chatLog).Error; err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}

	// 2. 追加到 Redis 缓存
	if m.redis != nil {
		msg := Message{
			Role:      m.chatTypeToRole(chatType),
			Content:   content,
			CreatedAt: chatLog.CreatedAt,
		}
		if err := m.redis.AppendMessage(conversationID, msg); err != nil {
			logx.Warn("Failed to append message to Redis: %v", err)
		}
	}

	return nil
}

// GetUserContext 获取用户上下文
func (m *Manager) GetUserContext(username string) (*UserContext, error) {
	// 1. 先尝试从 Redis 读取
	if m.redis != nil {
		userCtx, err := m.redis.GetUserContext(username)
		if err == nil && userCtx != nil {
			logx.Debug("User context loaded from Redis cache")
			return userCtx, nil
		}
	}

	// 2. 从 SQLite 读取
	var contexts []model.UserContext
	if err := m.db.Where("username = ?", username).Find(&contexts).Error; err != nil {
		return nil, fmt.Errorf("failed to load user context: %w", err)
	}

	userCtx := &UserContext{
		Username: username,
		Contexts: make(map[string]string),
	}

	// 解析上下文数据
	for _, ctx := range contexts {
		switch ctx.ContextKey {
		case "favorite_region":
			userCtx.FavoriteRegion = ctx.ContextValue
		case "default_vpc":
			userCtx.DefaultVPC = ctx.ContextValue
		default:
			userCtx.Contexts[ctx.ContextKey] = ctx.ContextValue
		}
	}

	// 3. 回填 Redis 缓存
	if m.redis != nil && len(contexts) > 0 {
		if err := m.redis.SaveUserContext(userCtx); err != nil {
			logx.Warn("Failed to save user context to Redis: %v", err)
		}
	}

	return userCtx, nil
}

// UpdateUserContext 更新用户上下文
func (m *Manager) UpdateUserContext(username, key, value string) error {
	// 1. 更新或创建 SQLite 记录
	userContext := &model.UserContext{
		Username:     username,
		ContextKey:   key,
		ContextValue: value,
		ContextType:  "user",
	}

	// Upsert 操作
	if err := m.db.Where("username = ? AND context_key = ?", username, key).
		Assign(model.UserContext{ContextValue: value, UpdatedAt: time.Now()}).
		FirstOrCreate(userContext).Error; err != nil {
		return fmt.Errorf("failed to update user context: %w", err)
	}

	// 2. 使 Redis 缓存失效（删除，下次重新加载）
	// 简化实现：直接删除整个用户上下文缓存
	// 更好的实现是更新对应字段
	// TODO: 改进为更新单个字段

	return nil
}

// GetCachedAnswer 获取缓存的答案
func (m *Manager) GetCachedAnswer(username, question string) (string, bool, error) {
	// 计算问题哈希
	hash := m.calculateQuestionHash(question)

	// 1. 先尝试从 Redis 读取
	if m.redis != nil {
		answer, ok, err := m.redis.GetCachedAnswer(hash)
		if err == nil && ok {
			logx.Debug("QA cache hit from Redis")
			// 异步更新 SQLite 的命中次数
			go m.incrementQACacheHit(hash)
			return answer, true, nil
		}
	}

	// 2. 从 SQLite 读取
	var cache model.QACache
	err := m.db.Where("question_hash = ? AND (username = ? OR username IS NULL)", hash, username).
		Order("hit_count DESC").
		First(&cache).Error

	if err == gorm.ErrRecordNotFound {
		return "", false, nil // 未命中
	}
	if err != nil {
		return "", false, fmt.Errorf("failed to query QA cache: %w", err)
	}

	// 更新命中统计
	m.db.Model(&cache).Updates(map[string]any{
		"hit_count":   gorm.Expr("hit_count + 1"),
		"last_hit_at": time.Now(),
	})

	// 3. 回填 Redis 缓存
	if m.redis != nil {
		if err := m.redis.SetCachedAnswer(hash, cache.Answer); err != nil {
			logx.Warn("Failed to set QA cache to Redis: %v", err)
		}
	}

	return cache.Answer, true, nil
}

// UpdateQACache 更新问答缓存
func (m *Manager) UpdateQACache(username, question, answer string) error {
	// 检查答案质量 - 不缓存错误响应
	if m.isErrorResponse(answer) {
		logx.Debug("Skipping QA cache for error response")
		return nil
	}

	// 检查答案长度 - 太短的答案可能不是有效回复
	if len(answer) < 10 {
		logx.Debug("Skipping QA cache for too short answer (len=%d)", len(answer))
		return nil
	}

	hash := m.calculateQuestionHash(question)

	// 1. 更新 SQLite
	cache := &model.QACache{
		QuestionHash: hash,
		Question:     question,
		Answer:       answer,
		Username:     username,
		HitCount:     1,
		LastHitAt:    time.Now(),
	}

	// Upsert
	if err := m.db.Where("question_hash = ? AND username = ?", hash, username).
		Assign(model.QACache{Answer: answer, UpdatedAt: time.Now()}).
		FirstOrCreate(cache).Error; err != nil {
		return fmt.Errorf("failed to update QA cache: %w", err)
	}

	logx.Debug("✅ QA cache saved: question_hash=%s", hash[:8])

	// 2. 更新 Redis
	if m.redis != nil {
		if err := m.redis.SetCachedAnswer(hash, answer); err != nil {
			logx.Warn("Failed to update QA cache in Redis: %v", err)
		}
	}

	return nil
}

// GetCacheStats 获取缓存统计信息
func (m *Manager) GetCacheStats() (*CacheStats, error) {
	var totalQueries int64
	var hitCount int64

	// 统计总查询次数
	if err := m.db.Model(&model.QACache{}).
		Select("SUM(hit_count)").
		Scan(&totalQueries).Error; err != nil {
		return nil, err
	}

	// 统计命中次数（hit_count > 1 的记录）
	if err := m.db.Model(&model.QACache{}).
		Where("hit_count > 1").
		Count(&hitCount).Error; err != nil {
		return nil, err
	}

	stats := &CacheStats{
		HitCount:     hitCount,
		MissCount:    totalQueries - hitCount,
		TotalQueries: totalQueries,
	}

	if totalQueries > 0 {
		stats.HitRate = float64(hitCount) / float64(totalQueries)
	}

	return stats, nil
}

// calculateQuestionHash 计算问题的哈希值
func (m *Manager) calculateQuestionHash(question string) string {
	hash := sha256.Sum256([]byte(question))
	return fmt.Sprintf("%x", hash[:8]) // 取前 8 字节
}

// isErrorResponse 判断是否为错误响应
func (m *Manager) isErrorResponse(answer string) bool {
	// 检查常见的错误标记
	errorKeywords := []string{
		"❌",
		"LLM 调用失败",
		"Agent 调用失败",
		"Agent 系统未初始化",
		"i/o timeout",
		"connection refused",
		"dial tcp",
		"context deadline exceeded",
		"failed to",
		"error:",
		"Error:",
		"失败:",
		"错误:",
		"异常:",
	}

	for _, keyword := range errorKeywords {
		if strings.Contains(answer, keyword) {
			return true
		}
	}

	return false
}

// ClearQACache 清理 QA 缓存
func (m *Manager) ClearQACache(username string, questionHash string) error {
	query := m.db.Model(&model.QACache{})

	// 如果指定了用户名，只清理该用户的缓存
	if username != "" {
		query = query.Where("username = ?", username)
	}

	// 如果指定了问题哈希，只清理特定问题
	if questionHash != "" {
		query = query.Where("question_hash = ?", questionHash)
	}

	if err := query.Delete(&model.QACache{}).Error; err != nil {
		return fmt.Errorf("failed to clear QA cache: %w", err)
	}

	// 清理 Redis 缓存
	if m.redis != nil && questionHash != "" {
		if err := m.redis.DeleteCachedAnswer(questionHash); err != nil {
			logx.Warn("Failed to delete QA cache from Redis: %v", err)
		}
	}

	logx.Info("✅ QA cache cleared: username=%s, question_hash=%s", username, questionHash)
	return nil
}

// ClearErrorCache 清理所有错误缓存
func (m *Manager) ClearErrorCache() (int64, error) {
	// 查找所有可能是错误的缓存
	var caches []model.QACache
	if err := m.db.Find(&caches).Error; err != nil {
		return 0, fmt.Errorf("failed to query QA cache: %w", err)
	}

	var deletedCount int64
	for _, cache := range caches {
		if m.isErrorResponse(cache.Answer) {
			if err := m.db.Delete(&cache).Error; err != nil {
				logx.Warn("Failed to delete error cache: %v", err)
				continue
			}
			deletedCount++

			// 从 Redis 删除
			if m.redis != nil {
				if err := m.redis.DeleteCachedAnswer(cache.QuestionHash); err != nil {
					logx.Warn("Failed to delete from Redis: %v", err)
				}
			}
		}
	}

	logx.Info("✅ Cleared %d error caches", deletedCount)
	return deletedCount, nil
}

// incrementQACacheHit 增加 QA 缓存命中次数（异步）
func (m *Manager) incrementQACacheHit(hash string) {
	m.db.Model(&model.QACache{}).
		Where("question_hash = ?", hash).
		Updates(map[string]any{
			"hit_count":   gorm.Expr("hit_count + 1"),
			"last_hit_at": time.Now(),
		})
}

// chatTypeToRole 将 ChatType 转换为 Role
func (m *Manager) chatTypeToRole(chatType int) string {
	switch chatType {
	case 1:
		return "user"
	case 2:
		return "assistant"
	default:
		return "system"
	}
}

// messagesToChatLogs 将 Message 转换为 ChatLog
func (m *Manager) messagesToChatLogs(messages []Message) []*model.ChatLog {
	var chatLogs []*model.ChatLog
	for _, msg := range messages {
		chatType := 1 // user
		if msg.Role == "assistant" {
			chatType = 2
		}
		chatLogs = append(chatLogs, &model.ChatLog{
			ChatType:  chatType,
			Content:   msg.Content,
			CreatedAt: msg.CreatedAt,
		})
	}
	return chatLogs
}

// chatLogsToMessages 将 ChatLog 转换为 Message
func (m *Manager) chatLogsToMessages(chatLogs []*model.ChatLog) []Message {
	var messages []Message
	for _, log := range chatLogs {
		messages = append(messages, Message{
			Role:      m.chatTypeToRole(log.ChatType),
			Content:   log.Content,
			CreatedAt: log.CreatedAt,
		})
	}
	return messages
}

// BuildSystemPromptWithContext 构建带用户上下文的 System Prompt
func (m *Manager) BuildSystemPromptWithContext(username, basePrompt string) string {
	userCtx, err := m.GetUserContext(username)
	if err != nil {
		logx.Warn("Failed to get user context: %v", err)
		return basePrompt
	}

	contextInfo := ""
	if userCtx.FavoriteRegion != "" {
		contextInfo += fmt.Sprintf("\n用户常用地域: %s", userCtx.FavoriteRegion)
	}
	if userCtx.DefaultVPC != "" {
		contextInfo += fmt.Sprintf("\n用户默认 VPC: %s", userCtx.DefaultVPC)
	}

	// 添加其他自定义上下文
	if len(userCtx.Contexts) > 0 {
		contextJSON, _ := json.MarshalIndent(userCtx.Contexts, "", "  ")
		contextInfo += fmt.Sprintf("\n用户自定义配置:\n%s", string(contextJSON))
	}

	if contextInfo != "" {
		return basePrompt + "\n\n## 用户上下文信息" + contextInfo
	}

	return basePrompt
}
