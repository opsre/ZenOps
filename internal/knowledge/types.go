package knowledge

// Document 检索结果文档
type Document struct {
	ID       uint              `json:"id"`
	Title    string            `json:"title"`
	Content  string            `json:"content"`
	DocType  string            `json:"doc_type"`
	Category string            `json:"category"`
	Score    float64           `json:"score"`     // 相关性评分
	Metadata map[string]string `json:"metadata"`
}

// SearchResult 搜索结果
type SearchResult struct {
	Documents  []*Document `json:"documents"`
	TotalCount int         `json:"total_count"`
	Query      string      `json:"query"`
}

// AddDocumentRequest 添加文档请求
type AddDocumentRequest struct {
	Title    string            `json:"title" binding:"required"`
	Content  string            `json:"content" binding:"required"`
	DocType  string            `json:"doc_type"`  // markdown, pdf, url, manual
	Category string            `json:"category"`
	Metadata map[string]string `json:"metadata"`
}
