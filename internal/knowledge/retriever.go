package knowledge

import (
	"context"
	"encoding/json"
	"fmt"

	"cnb.cool/zhiqiangwang/pkg/logx"
	"github.com/eryajf/zenops/internal/model"
	"gorm.io/gorm"
)

// Retriever 知识检索器
type Retriever struct {
	db         *gorm.DB
	useVector  bool // 是否启用向量检索（暂未实现）
	maxResults int  // 最大返回结果数
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

// Retrieve 检索相关文档（实现 Eino Retriever 接口）
func (r *Retriever) Retrieve(ctx context.Context, query string) ([]*Document, error) {
	// 目前只实现 FTS5 全文检索
	// TODO: 未来可以添加向量检索
	return r.retrieveByFTS5(query)
}

// retrieveByFTS5 使用 FTS5 全文检索
func (r *Retriever) retrieveByFTS5(query string) ([]*Document, error) {
	// FTS5 查询语法
	sql := `
		SELECT
			d.id,
			d.title,
			d.content,
			d.doc_type,
			d.category,
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
		Metadata string  `gorm:"column:metadata"`
		Score    float64 `gorm:"column:score"`
	}

	if err := r.db.Raw(sql, query, r.maxResults).Scan(&results).Error; err != nil {
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
		}

		// 解析 JSON metadata
		if res.Metadata != "" {
			if err := json.Unmarshal([]byte(res.Metadata), &doc.Metadata); err != nil {
				logx.Warn("Failed to parse metadata for doc %d: %v", res.ID, err)
			}
		}

		documents = append(documents, doc)
	}

	logx.Info("FTS5 search found %d documents for query: %s", len(documents), query)
	return documents, nil
}

// AddDocument 添加文档到知识库
func (r *Retriever) AddDocument(req *AddDocumentRequest) (uint, error) {
	// 序列化 metadata
	metadataJSON, err := json.Marshal(req.Metadata)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	doc := &model.KnowledgeDocument{
		Title:    req.Title,
		Content:  req.Content,
		DocType:  req.DocType,
		Category: req.Category,
		Metadata: string(metadataJSON),
		Enabled:  true,
	}

	if err := r.db.Create(doc).Error; err != nil {
		return 0, fmt.Errorf("failed to create document: %w", err)
	}

	logx.Info("✅ Added document to knowledge base: %s (ID: %d)", doc.Title, doc.ID)
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

	updates := map[string]any{
		"title":    req.Title,
		"content":  req.Content,
		"doc_type": req.DocType,
		"category": req.Category,
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
			ID:       doc.ID,
			Title:    doc.Title,
			Content:  doc.Content,
			DocType:  doc.DocType,
			Category: doc.Category,
			Metadata: make(map[string]string),
		}

		if doc.Metadata != "" {
			if err := json.Unmarshal([]byte(doc.Metadata), &d.Metadata); err != nil {
				logx.Warn("Failed to parse metadata for doc %d: %v", doc.ID, err)
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
		ID:       doc.ID,
		Title:    doc.Title,
		Content:  doc.Content,
		DocType:  doc.DocType,
		Category: doc.Category,
		Metadata: make(map[string]string),
	}

	if doc.Metadata != "" {
		if err := json.Unmarshal([]byte(doc.Metadata), &d.Metadata); err != nil {
			return nil, fmt.Errorf("failed to parse metadata: %w", err)
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
