package knowledge

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"

	"cnb.cool/zhiqiangwang/pkg/logx"
	"github.com/eryajf/zenops/internal/model"
)

// retrieveByVector 使用向量检索
func (r *Retriever) retrieveByVector(ctx context.Context, query string) ([]*Document, error) {
	// 1. 生成查询向量
	queryVector, err := r.embeddingService.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// 2. 从数据库加载所有有embedding的文档
	var docs []model.KnowledgeDocument
	if err := r.db.Where("enabled = ? AND embedding != ''", true).
		Find(&docs).Error; err != nil {
		return nil, fmt.Errorf("failed to load documents: %w", err)
	}

	if len(docs) == 0 {
		logx.Warn("No documents with embeddings found")
		return []*Document{}, nil
	}

	// 3. 计算相似度并排序
	type scoredDoc struct {
		doc   *model.KnowledgeDocument
		score float64
	}

	var scoredDocs []scoredDoc
	for i := range docs {
		// 解析 embedding
		var docVector []float64
		if err := json.Unmarshal([]byte(docs[i].Embedding), &docVector); err != nil {
			logx.Warn("Failed to parse embedding for doc %d: %v", docs[i].ID, err)
			continue
		}

		// 计算余弦相似度
		similarity := cosineSimilarity(queryVector, docVector)
		scoredDocs = append(scoredDocs, scoredDoc{
			doc:   &docs[i],
			score: similarity,
		})
	}

	// 4. 按相似度降序排序
	sort.Slice(scoredDocs, func(i, j int) bool {
		return scoredDocs[i].score > scoredDocs[j].score
	})

	// 5. 取前 maxResults 个
	limit := r.maxResults
	if len(scoredDocs) < limit {
		limit = len(scoredDocs)
	}

	// 6. 转换为 Document
	var documents []*Document
	for i := 0; i < limit; i++ {
		doc := scoredDocs[i].doc
		d := &Document{
			ID:       doc.ID,
			Title:    doc.Title,
			Content:  doc.Content,
			DocType:  doc.DocType,
			Category: doc.Category,
			Score:    scoredDocs[i].score,
			Metadata: make(map[string]string),
		}

		if doc.Metadata != "" {
			if err := json.Unmarshal([]byte(doc.Metadata), &d.Metadata); err != nil {
				logx.Warn("Failed to parse metadata for doc %d: %v", doc.ID, err)
			}
		}

		documents = append(documents, d)
	}

	logx.Info("Vector search found %d documents (query embedding dim=%d)", len(documents), len(queryVector))
	return documents, nil
}

// retrieveHybrid 混合检索（FTS5 + 向量）
func (r *Retriever) retrieveHybrid(ctx context.Context, query string) ([]*Document, error) {
	// 1. FTS5 检索
	fts5Docs, err := r.retrieveByFTS5(query)
	if err != nil {
		logx.Warn("FTS5 search failed, falling back to vector-only: %v", err)
		return r.retrieveByVector(ctx, query)
	}

	// 2. 向量检索
	vectorDocs, err := r.retrieveByVector(ctx, query)
	if err != nil {
		logx.Warn("Vector search failed, using FTS5 results only: %v", err)
		return fts5Docs, nil
	}

	// 3. 合并结果（RRF - Reciprocal Rank Fusion）
	merged := mergeResults(fts5Docs, vectorDocs, r.maxResults)

	logx.Info("Hybrid search completed: FTS5=%d, Vector=%d, Merged=%d",
		len(fts5Docs), len(vectorDocs), len(merged))

	return merged, nil
}

// mergeResults 使用 RRF (Reciprocal Rank Fusion) 合并两个结果集
func mergeResults(fts5Docs, vectorDocs []*Document, maxResults int) []*Document {
	const k = 60.0 // RRF 常数

	// 计算每个文档的 RRF 分数
	scoreMap := make(map[uint]float64)
	docMap := make(map[uint]*Document)

	// FTS5 结果
	for rank, doc := range fts5Docs {
		rrf := 1.0 / (float64(rank+1) + k)
		scoreMap[doc.ID] = rrf
		docMap[doc.ID] = doc
	}

	// 向量结果
	for rank, doc := range vectorDocs {
		rrf := 1.0 / (float64(rank+1) + k)
		scoreMap[doc.ID] += rrf // 累加分数
		if _, exists := docMap[doc.ID]; !exists {
			docMap[doc.ID] = doc
		}
	}

	// 按 RRF 分数排序
	type scoredDoc struct {
		doc   *Document
		score float64
	}

	var scored []scoredDoc
	for id, score := range scoreMap {
		doc := docMap[id]
		doc.Score = score // 更新分数为 RRF 分数
		scored = append(scored, scoredDoc{
			doc:   doc,
			score: score,
		})
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// 取前 maxResults 个
	limit := maxResults
	if len(scored) < limit {
		limit = len(scored)
	}

	var merged []*Document
	for i := 0; i < limit; i++ {
		merged = append(merged, scored[i].doc)
	}

	return merged
}

// cosineSimilarity 计算两个向量的余弦相似度
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		logx.Warn("Vector dimension mismatch: %d vs %d", len(a), len(b))
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
