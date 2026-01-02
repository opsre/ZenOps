package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCache Redis 缓存层
type RedisCache struct {
	client *redis.Client
	ttl    time.Duration
}

// NewRedisCache 创建 Redis 缓存
func NewRedisCache(addr, password string, db int, ttl time.Duration) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &RedisCache{
		client: client,
		ttl:    ttl,
	}, nil
}

// GetConversationHistory 获取对话历史（Redis）
func (r *RedisCache) GetConversationHistory(conversationID uint) ([]Message, error) {
	key := fmt.Sprintf("conv:%d:history", conversationID)
	ctx := context.Background()

	// 从 Redis List 中获取最近的消息
	result, err := r.client.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		return nil, err
	}

	var messages []Message
	for _, item := range result {
		var msg Message
		if err := json.Unmarshal([]byte(item), &msg); err != nil {
			continue
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// SaveConversationHistory 保存对话历史（Redis）
func (r *RedisCache) SaveConversationHistory(conversationID uint, messages []Message) error {
	key := fmt.Sprintf("conv:%d:history", conversationID)
	ctx := context.Background()

	// 清空旧数据
	if err := r.client.Del(ctx, key).Err(); err != nil {
		return err
	}

	// 逐个插入消息
	for _, msg := range messages {
		data, err := json.Marshal(msg)
		if err != nil {
			continue
		}
		if err := r.client.RPush(ctx, key, data).Err(); err != nil {
			return err
		}
	}

	// 设置过期时间
	return r.client.Expire(ctx, key, r.ttl).Err()
}

// AppendMessage 追加单条消息到历史
func (r *RedisCache) AppendMessage(conversationID uint, msg Message) error {
	key := fmt.Sprintf("conv:%d:history", conversationID)
	ctx := context.Background()

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	// 追加到列表
	if err := r.client.RPush(ctx, key, data).Err(); err != nil {
		return err
	}

	// 更新过期时间
	return r.client.Expire(ctx, key, r.ttl).Err()
}

// GetUserContext 获取用户上下文（Redis）
func (r *RedisCache) GetUserContext(username string) (*UserContext, error) {
	key := fmt.Sprintf("user:%s:context", username)
	ctx := context.Background()

	data, err := r.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return nil, nil // 缓存未命中
	}

	userCtx := &UserContext{
		Username: username,
		Contexts: make(map[string]string),
	}

	// 解析数据
	for k, v := range data {
		switch k {
		case "favorite_region":
			userCtx.FavoriteRegion = v
		case "default_vpc":
			userCtx.DefaultVPC = v
		default:
			userCtx.Contexts[k] = v
		}
	}

	return userCtx, nil
}

// SaveUserContext 保存用户上下文（Redis）
func (r *RedisCache) SaveUserContext(userCtx *UserContext) error {
	key := fmt.Sprintf("user:%s:context", userCtx.Username)
	ctx := context.Background()

	data := make(map[string]any)
	if userCtx.FavoriteRegion != "" {
		data["favorite_region"] = userCtx.FavoriteRegion
	}
	if userCtx.DefaultVPC != "" {
		data["default_vpc"] = userCtx.DefaultVPC
	}
	for k, v := range userCtx.Contexts {
		data[k] = v
	}

	if len(data) > 0 {
		return r.client.HSet(ctx, key, data).Err()
	}

	return nil
}

// GetCachedAnswer 获取缓存的答案（Redis）
func (r *RedisCache) GetCachedAnswer(questionHash string) (string, bool, error) {
	key := fmt.Sprintf("qa:%s", questionHash)
	ctx := context.Background()

	answer, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", false, nil // 缓存未命中
	}
	if err != nil {
		return "", false, err
	}

	return answer, true, nil
}

// SetCachedAnswer 设置缓存的答案（Redis）
func (r *RedisCache) SetCachedAnswer(questionHash, answer string) error {
	key := fmt.Sprintf("qa:%s", questionHash)
	ctx := context.Background()

	return r.client.Set(ctx, key, answer, r.ttl).Err()
}

// DeleteCachedAnswer 删除缓存的答案（Redis）
func (r *RedisCache) DeleteCachedAnswer(questionHash string) error {
	key := fmt.Sprintf("qa:%s", questionHash)
	ctx := context.Background()

	return r.client.Del(ctx, key).Err()
}

// GetActiveSession 获取用户当前活跃会话 ID
func (r *RedisCache) GetActiveSession(username string) (uint, error) {
	key := fmt.Sprintf("session:%s:active", username)
	ctx := context.Background()

	val, err := r.client.Get(ctx, key).Uint64()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	return uint(val), nil
}

// SetActiveSession 设置用户当前活跃会话 ID
func (r *RedisCache) SetActiveSession(username string, conversationID uint) error {
	key := fmt.Sprintf("session:%s:active", username)
	ctx := context.Background()

	return r.client.Set(ctx, key, conversationID, 24*time.Hour).Err()
}

// Close 关闭 Redis 连接
func (r *RedisCache) Close() error {
	return r.client.Close()
}
