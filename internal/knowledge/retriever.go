package knowledge

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"cnb.cool/zhiqiangwang/pkg/logx"
	"github.com/eryajf/zenops/internal/model"
	"gorm.io/gorm"
)

// Retriever 知识检索器
type Retriever struct {
	db               *gorm.DB
	useVector        bool // 是否启用向量检索
	maxResults       int  // 最大返回结果数
	embeddingService EmbeddingService
}

// EmbeddingService 简化接口（避免循环依赖）
type EmbeddingService interface {
	Embed(ctx context.Context, text string) ([]float64, error)
	GetModel() string
}

// NewRetriever 创建知识检索器
func NewRetriever(db *gorm.DB, useVector bool, maxResults int) *Retriever {
	if maxResults <= 0 {
		maxResults = 3 // 默认返回 3 条
	}

	return &Retriever{
		db:         db,
		useVector:  useVector,
		maxResults: maxResults,
	}
}

// SetEmbeddingService 设置 Embedding 服务（用于向量检索）
func (r *Retriever) SetEmbeddingService(service EmbeddingService) {
	r.embeddingService = service
	if service != nil {
		r.useVector = true
		logx.Info("✅ Knowledge Retriever: Vector search enabled with model %s", service.GetModel())
	}
}

// Retrieve 检索相关文档（实现 Eino Retriever 接口）
func (r *Retriever) Retrieve(ctx context.Context, query string) ([]*Document, error) {
	// 根据配置选择检索策略
	if r.useVector && r.embeddingService != nil {
		// 使用混合检索（FTS5 + 向量）
		return r.retrieveHybrid(ctx, query)
	}

	// 仅使用 FTS5 全文检索
	return r.retrieveByFTS5(query)
}

// retrieveByFTS5 使用 FTS5 全文检索
func (r *Retriever) retrieveByFTS5(query string) ([]*Document, error) {
	// 清理查询文本，移除 FTS5 特殊字符
	cleanedQuery := sanitizeFTS5Query(query)
	if cleanedQuery == "" {
		logx.Warn("FTS5 query is empty after sanitization, original: %s", query)
		return []*Document{}, nil
	}

	// FTS5 查询语法
	sql := `
		SELECT
			d.id,
			d.title,
			d.content,
			d.doc_type,
			d.category,
			d.tags,
			d.metadata,
			rank AS score
		FROM knowledge_documents d
		JOIN knowledge_fts f ON d.id = f.rowid
		WHERE knowledge_fts MATCH ?
		AND d.enabled = 1
		ORDER BY rank
		LIMIT ?
	`

	var results []struct {
		ID       uint    `gorm:"column:id"`
		Title    string  `gorm:"column:title"`
		Content  string  `gorm:"column:content"`
		DocType  string  `gorm:"column:doc_type"`
		Category string  `gorm:"column:category"`
		Tags     string  `gorm:"column:tags"`
		Metadata string  `gorm:"column:metadata"`
		Score    float64 `gorm:"column:score"`
	}

	if err := r.db.Raw(sql, cleanedQuery, r.maxResults).Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("FTS5 search failed: %w", err)
	}

	// 转换为 Document 结构
	var documents []*Document
	for _, res := range results {
		doc := &Document{
			ID:       res.ID,
			Title:    res.Title,
			Content:  res.Content,
			DocType:  res.DocType,
			Category: res.Category,
			Score:    res.Score,
			Metadata: make(map[string]string),
			Tags:     []string{},
		}

		// 解析 JSON metadata
		if res.Metadata != "" {
			if err := json.Unmarshal([]byte(res.Metadata), &doc.Metadata); err != nil {
				logx.Warn("Failed to parse metadata for doc %d: %v", res.ID, err)
			}
		}

		// 解析 JSON tags
		if res.Tags != "" {
			if err := json.Unmarshal([]byte(res.Tags), &doc.Tags); err != nil {
				logx.Warn("Failed to parse tags for doc %d: %v", res.ID, err)
			}
		}

		documents = append(documents, doc)
	}

	logx.Info("FTS5 search found %d documents for query: %s", len(documents), query)

	// 如果 FTS5 没有结果，降级使用 LIKE 搜索（对中文更友好）
	if len(documents) == 0 {
		logx.Warn("FTS5 returned 0 results, falling back to LIKE search")
		return r.retrieveByLike(query)
	}

	return documents, nil
}

// retrieveByLike 使用 LIKE 搜索（FTS5 失败时的降级方案，对中文友好）
func (r *Retriever) retrieveByLike(query string) ([]*Document, error) {
	// 构建 LIKE 查询
	likePattern := "%" + query + "%"

	sql := `
		SELECT
			id,
			title,
			content,
			doc_type,
			category,
			tags,
			metadata,
			1.0 AS score
		FROM knowledge_documents
		WHERE (title LIKE ? OR content LIKE ? OR tags LIKE ?)
		AND enabled = 1
		ORDER BY
			CASE
				WHEN title LIKE ? THEN 1
				WHEN content LIKE ? THEN 2
				ELSE 3
			END
		LIMIT ?
	`

	var results []struct {
		ID       uint    `gorm:"column:id"`
		Title    string  `gorm:"column:title"`
		Content  string  `gorm:"column:content"`
		DocType  string  `gorm:"column:doc_type"`
		Category string  `gorm:"column:category"`
		Tags     string  `gorm:"column:tags"`
		Metadata string  `gorm:"column:metadata"`
		Score    float64 `gorm:"column:score"`
	}

	if err := r.db.Raw(sql,
		likePattern, likePattern, likePattern, // WHERE 子句
		likePattern, likePattern, // ORDER BY 子句
		r.maxResults,
	).Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("LIKE search failed: %w", err)
	}

	// 转换为 Document 结构
	var documents []*Document
	for _, res := range results {
		doc := &Document{
			ID:       res.ID,
			Title:    res.Title,
			Content:  res.Content,
			DocType:  res.DocType,
			Category: res.Category,
			Score:    res.Score,
			Metadata: make(map[string]string),
			Tags:     []string{},
		}

		// 解析 JSON metadata
		if res.Metadata != "" {
			if err := json.Unmarshal([]byte(res.Metadata), &doc.Metadata); err != nil {
				logx.Warn("Failed to parse metadata for doc %d: %v", res.ID, err)
			}
		}

		// 解析 JSON tags
		if res.Tags != "" {
			if err := json.Unmarshal([]byte(res.Tags), &doc.Tags); err != nil {
				logx.Warn("Failed to parse tags for doc %d: %v", res.ID, err)
			}
		}

		documents = append(documents, doc)
	}

	logx.Info("LIKE search found %d documents for query: %s", len(documents), query)
	return documents, nil
}

// AddDocument 添加文档到知识库
func (r *Retriever) AddDocument(req *AddDocumentRequest) (uint, error) {
	return r.AddDocumentWithContext(context.Background(), req)
}

// AddDocumentWithContext 添加文档到知识库（支持 context）
func (r *Retriever) AddDocumentWithContext(ctx context.Context, req *AddDocumentRequest) (uint, error) {
	// 序列化 metadata
	metadataJSON, err := json.Marshal(req.Metadata)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// 序列化 tags
	tagsJSON, err := json.Marshal(req.Tags)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal tags: %w", err)
	}

	doc := &model.KnowledgeDocument{
		Title:    req.Title,
		Content:  req.Content,
		DocType:  req.DocType,
		Category: req.Category,
		Tags:     string(tagsJSON),
		Metadata: string(metadataJSON),
		Enabled:  true,
	}

	// 如果启用向量检索，生成 embedding
	if r.useVector && r.embeddingService != nil {
		// 合并标题和内容生成向量
		text := req.Title + "\n\n" + req.Content
		embedding, err := r.embeddingService.Embed(ctx, text)
		if err != nil {
			logx.Warn("Failed to generate embedding for document: %v", err)
		} else {
			embBytes, _ := json.Marshal(embedding)
			doc.Embedding = string(embBytes)
			doc.EmbeddingModel = r.embeddingService.GetModel()
			logx.Debug("Generated embedding for document: model=%s, dim=%d", doc.EmbeddingModel, len(embedding))
		}
	}

	if err := r.db.Create(doc).Error; err != nil {
		return 0, fmt.Errorf("failed to create document: %w", err)
	}

	logx.Info("✅ Added document to knowledge base: %s (ID: %d, has_embedding=%v)",
		doc.Title, doc.ID, doc.Embedding != "")
	return doc.ID, nil
}

// DeleteDocument 删除文档
func (r *Retriever) DeleteDocument(docID uint) error {
	result := r.db.Delete(&model.KnowledgeDocument{}, docID)
	if result.Error != nil {
		return fmt.Errorf("failed to delete document: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("document not found: %d", docID)
	}

	logx.Info("✅ Deleted document from knowledge base: ID %d", docID)
	return nil
}

// UpdateDocument 更新文档
func (r *Retriever) UpdateDocument(docID uint, req *AddDocumentRequest) error {
	metadataJSON, err := json.Marshal(req.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	tagsJSON, err := json.Marshal(req.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	updates := map[string]any{
		"title":    req.Title,
		"content":  req.Content,
		"doc_type": req.DocType,
		"category": req.Category,
		"tags":     string(tagsJSON),
		"metadata": string(metadataJSON),
	}

	result := r.db.Model(&model.KnowledgeDocument{}).Where("id = ?", docID).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to update document: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("document not found: %d", docID)
	}

	logx.Info("✅ Updated document in knowledge base: ID %d", docID)
	return nil
}

// ListDocuments 列出文档
func (r *Retriever) ListDocuments(category string, enabled *bool) ([]*Document, error) {
	query := r.db.Model(&model.KnowledgeDocument{})

	if category != "" {
		query = query.Where("category = ?", category)
	}

	if enabled != nil {
		query = query.Where("enabled = ?", *enabled)
	}

	var docs []model.KnowledgeDocument
	if err := query.Order("created_at DESC").Find(&docs).Error; err != nil {
		return nil, fmt.Errorf("failed to list documents: %w", err)
	}

	// 转换为 Document 结构
	var documents []*Document
	for _, doc := range docs {
		d := &Document{
			ID:        doc.ID,
			Title:     doc.Title,
			Content:   doc.Content,
			DocType:   doc.DocType,
			Category:  doc.Category,
			Enabled:   doc.Enabled,
			CreatedAt: doc.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt: doc.UpdatedAt.Format("2006-01-02 15:04:05"),
			Metadata:  make(map[string]string),
			Tags:      []string{},
		}

		if doc.Metadata != "" {
			if err := json.Unmarshal([]byte(doc.Metadata), &d.Metadata); err != nil {
				logx.Warn("Failed to parse metadata for doc %d: %v", doc.ID, err)
			}
		}

		if doc.Tags != "" {
			if err := json.Unmarshal([]byte(doc.Tags), &d.Tags); err != nil {
				logx.Warn("Failed to parse tags for doc %d: %v", doc.ID, err)
			}
		}

		documents = append(documents, d)
	}

	return documents, nil
}

// GetDocumentByID 根据 ID 获取文档
func (r *Retriever) GetDocumentByID(docID uint) (*Document, error) {
	var doc model.KnowledgeDocument
	if err := r.db.First(&doc, docID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("document not found: %d", docID)
		}
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	d := &Document{
		ID:        doc.ID,
		Title:     doc.Title,
		Content:   doc.Content,
		DocType:   doc.DocType,
		Category:  doc.Category,
		Enabled:   doc.Enabled,
		CreatedAt: doc.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt: doc.UpdatedAt.Format("2006-01-02 15:04:05"),
		Metadata:  make(map[string]string),
		Tags:      []string{},
	}

	if doc.Metadata != "" {
		if err := json.Unmarshal([]byte(doc.Metadata), &d.Metadata); err != nil {
			return nil, fmt.Errorf("failed to parse metadata: %w", err)
		}
	}

	if doc.Tags != "" {
		if err := json.Unmarshal([]byte(doc.Tags), &d.Tags); err != nil {
			return nil, fmt.Errorf("failed to parse tags: %w", err)
		}
	}

	return d, nil
}

// ToggleDocument 启用/禁用文档
func (r *Retriever) ToggleDocument(docID uint, enabled bool) error {
	result := r.db.Model(&model.KnowledgeDocument{}).
		Where("id = ?", docID).
		Update("enabled", enabled)

	if result.Error != nil {
		return fmt.Errorf("failed to toggle document: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("document not found: %d", docID)
	}

	status := "disabled"
	if enabled {
		status = "enabled"
	}

	logx.Info("✅ Document %d %s", docID, status)
	return nil
}

// GetStats 获取知识库统计信息
func (r *Retriever) GetStats() (map[string]any, error) {
	var totalCount int64
	var enabledCount int64
	var categories []string

	// 总文档数
	if err := r.db.Model(&model.KnowledgeDocument{}).Count(&totalCount).Error; err != nil {
		return nil, err
	}

	// 启用的文档数
	if err := r.db.Model(&model.KnowledgeDocument{}).
		Where("enabled = ?", true).
		Count(&enabledCount).Error; err != nil {
		return nil, err
	}

	// 分类列表
	if err := r.db.Model(&model.KnowledgeDocument{}).
		Distinct("category").
		Pluck("category", &categories).Error; err != nil {
		return nil, err
	}

	return map[string]any{
		"total_count":   totalCount,
		"enabled_count": enabledCount,
		"categories":    categories,
	}, nil
}

// sanitizeFTS5Query 清理 FTS5 查询文本，移除特殊字符
func sanitizeFTS5Query(query string) string {
	// FTS5 特殊字符: " * : ( ) AND OR NOT
	// 简单策略：只保留字母、数字、中文、空格
	var result []rune
	for _, r := range query {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || // 字母
			(r >= '0' && r <= '9') || // 数字
			(r >= 0x4e00 && r <= 0x9fa5) || // 中文
			r == ' ' { // 空格
			result = append(result, r)
		}
	}

	// 去除首尾空格，压缩多个空格为一个
	cleaned := strings.TrimSpace(string(result))
	cleaned = strings.Join(strings.Fields(cleaned), " ")

	return cleaned
}
