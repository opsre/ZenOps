package memory

import (
	"context"
	"math"

	"cnb.cool/zhiqiangwang/pkg/logx"
	"github.com/eryajf/zenops/internal/model"
)

// SemanticCacheConfig 语义缓存配置
type SemanticCacheConfig struct {
	Enabled             bool    // 是否启用语义缓存
	SimilarityThreshold float64 // 相似度阈值，默认 0.85
	MaxCandidates       int     // 最大候选数量，默认 100
}

// QACacheWithScore 带相似度分数的缓存
type QACacheWithScore struct {
	*model.QACache
	Score float64
}

// GetSemanticCachedAnswer 语义缓存查询
func (m *Manager) GetSemanticCachedAnswer(ctx context.Context, username, question string) (string, bool, error) {
	// 检查语义缓存是否启用
	if m.semanticConfig == nil || !m.semanticConfig.Enabled || m.embeddingService == nil {
		return "", false, nil
	}

	// 1. 生成问题向量
	questionVec, err := m.embeddingService.Embed(ctx, question)
	if err != nil {
		logx.Warn("Embedding failed, skip semantic cache: %v", err)
		return "", false, nil // 不返回错误，让调用方继续尝试精确匹配
	}

	// 2. 从 SQLite 加载候选缓存（带向量）
	candidates, err := m.loadCacheCandidates(username, m.semanticConfig.MaxCandidates)
	if err != nil {
		logx.Warn("Failed to load cache candidates: %v", err)
		return "", false, nil
	}

	if len(candidates) == 0 {
		return "", false, nil
	}

	// 3. 计算相似度，找最佳匹配
	var bestMatch *QACacheWithScore
	for _, candidate := range candidates {
		// 解析缓存的向量
		cachedVec, err := JSONToVector(candidate.Embedding)
		if err != nil || cachedVec == nil {
			continue
		}

		// 检查向量维度是否匹配
		if len(cachedVec) != len(questionVec) {
			continue
		}

		// 计算余弦相似度
		similarity := cosineSimilarity(questionVec, cachedVec)
		if similarity >= m.semanticConfig.SimilarityThreshold {
			if bestMatch == nil || similarity > bestMatch.Score {
				bestMatch = &QACacheWithScore{
					QACache: candidate,
					Score:   similarity,
				}
			}
		}
	}

	// 4. 返回结果
	if bestMatch != nil {
		// 截取问题前20个字符用于日志显示
		displayQuestion := bestMatch.Question
		if len(displayQuestion) > 20 {
			displayQuestion = displayQuestion[:20] + "..."
		}
		logx.Info("✅ Semantic cache hit: similarity=%.3f, cached_question=%s",
			bestMatch.Score, displayQuestion)

		// 异步更新命中统计
		go m.incrementQACacheHit(bestMatch.QuestionHash)

		return bestMatch.Answer, true, nil
	}

	return "", false, nil
}

// loadCacheCandidates 加载候选缓存（只加载有 embedding 的记录）
func (m *Manager) loadCacheCandidates(username string, limit int) ([]*model.QACache, error) {
	var caches []*model.QACache

	// 查询有 embedding 的缓存记录
	// 优先查询用户自己的缓存，然后是公共缓存
	query := m.db.Where("embedding IS NOT NULL AND embedding != ''").
		Where("username = ? OR username = '' OR username IS NULL", username).
		Order("hit_count DESC, updated_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&caches).Error; err != nil {
		return nil, err
	}

	return caches, nil
}

// cosineSimilarity 计算余弦相似度
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}
