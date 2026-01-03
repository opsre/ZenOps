package memory

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"cnb.cool/zhiqiangwang/pkg/logx"
	"github.com/cloudwego/eino-ext/components/embedding/openai"
	"github.com/cloudwego/eino/components/embedding"
)

// EmbeddingService 向量嵌入服务
type EmbeddingService struct {
	embedder embedding.Embedder
	model    string       // 当前使用的模型标识
	cache    *RedisCache  // 可选，缓存 embedding 结果
}

// EmbeddingConfig Embedding 配置
type EmbeddingConfig struct {
	APIKey  string
	BaseURL string
	Model   string // 如 "text-embedding-ada-002"
}

// NewEmbeddingService 创建 Embedding 服务（复用 Eino）
func NewEmbeddingService(cfg *EmbeddingConfig, redis *RedisCache) (*EmbeddingService, error) {
	embedder, err := openai.NewEmbedder(context.Background(), &openai.EmbeddingConfig{
		APIKey:  cfg.APIKey,
		BaseURL: cfg.BaseURL,
		Model:   cfg.Model,
		Timeout: 30 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create embedder: %w", err)
	}

	return &EmbeddingService{
		embedder: embedder,
		model:    cfg.Model,
		cache:    redis,
	}, nil
}

// Embed 获取文本的向量表示
func (s *EmbeddingService) Embed(ctx context.Context, text string) ([]float64, error) {
	// 1. 先检查 Redis 缓存
	if s.cache != nil {
		cacheKey := s.calculateCacheKey(text)
		cached, err := s.cache.GetEmbedding(cacheKey)
		if err == nil && cached != nil {
			logx.Debug("Embedding cache hit: key=%s", cacheKey[:16])
			return cached, nil
		}
	}

	// 2. 调用 Eino Embedder
	vectors, err := s.embedder.EmbedStrings(ctx, []string{text})
	if err != nil {
		return nil, fmt.Errorf("embedding failed: %w", err)
	}

	if len(vectors) == 0 || len(vectors[0]) == 0 {
		return nil, fmt.Errorf("empty embedding result")
	}

	result := vectors[0]

	// 3. 缓存结果
	if s.cache != nil {
		cacheKey := s.calculateCacheKey(text)
		if err := s.cache.SetEmbedding(cacheKey, result); err != nil {
			logx.Warn("Failed to cache embedding: %v", err)
		}
	}

	return result, nil
}

// EmbedBatch 批量获取文本的向量表示
func (s *EmbeddingService) EmbedBatch(ctx context.Context, texts []string) ([][]float64, error) {
	vectors, err := s.embedder.EmbedStrings(ctx, texts)
	if err != nil {
		return nil, fmt.Errorf("batch embedding failed: %w", err)
	}
	return vectors, nil
}

// GetModel 获取当前模型标识
func (s *EmbeddingService) GetModel() string {
	return s.model
}

// calculateCacheKey 计算缓存键
func (s *EmbeddingService) calculateCacheKey(text string) string {
	hash := sha256.Sum256([]byte(s.model + ":" + text))
	return fmt.Sprintf("emb:%x", hash[:16])
}

// VectorToJSON 将向量转换为 JSON 字符串
func VectorToJSON(vector []float64) (string, error) {
	data, err := json.Marshal(vector)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// JSONToVector 将 JSON 字符串转换为向量
func JSONToVector(jsonStr string) ([]float64, error) {
	if jsonStr == "" {
		return nil, nil
	}
	var vector []float64
	err := json.Unmarshal([]byte(jsonStr), &vector)
	if err != nil {
		return nil, err
	}
	return vector, nil
}
