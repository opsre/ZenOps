package server

import (
	"net/http"
	"strconv"

	"github.com/eryajf/zenops/internal/knowledge"
	"github.com/gin-gonic/gin"
)

// KnowledgeHandler 知识库 API Handler
type KnowledgeHandler struct {
	retriever *knowledge.Retriever
}

// NewKnowledgeHandler 创建知识库 Handler
func NewKnowledgeHandler(retriever *knowledge.Retriever) *KnowledgeHandler {
	return &KnowledgeHandler{retriever: retriever}
}

// RegisterRoutes 注册路由
func (h *KnowledgeHandler) RegisterRoutes(r *gin.RouterGroup) {
	kg := r.Group("/knowledge")
	{
		kg.GET("/documents", h.ListDocuments)
		kg.GET("/documents/:id", h.GetDocument)
		kg.POST("/documents", h.CreateDocument)
		kg.PUT("/documents/:id", h.UpdateDocument)
		kg.DELETE("/documents/:id", h.DeleteDocument)
		kg.PATCH("/documents/:id/toggle", h.ToggleDocument)

		kg.GET("/stats", h.GetStats)
		kg.GET("/categories", h.GetCategories)
		kg.GET("/tags", h.GetTags)
		kg.POST("/search", h.SearchDocuments)
	}
}

// ListDocuments 获取文档列表
func (h *KnowledgeHandler) ListDocuments(c *gin.Context) {
	category := c.Query("category")
	enabledStr := c.Query("enabled")

	var enabled *bool
	if enabledStr != "" {
		e := enabledStr == "true"
		enabled = &e
	}

	docs, err := h.retriever.ListDocuments(category, enabled)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": docs,
	})
}

// GetDocument 获取单个文档
func (h *KnowledgeHandler) GetDocument(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	doc, err := h.retriever.GetDocumentByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": doc,
	})
}

// CreateDocument 创建文档
func (h *KnowledgeHandler) CreateDocument(c *gin.Context) {
	var req knowledge.AddDocumentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "error": err.Error()})
		return
	}

	docID, err := h.retriever.AddDocument(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{"id": docID},
	})
}

// UpdateDocument 更新文档
func (h *KnowledgeHandler) UpdateDocument(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var req knowledge.AddDocumentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "error": err.Error()})
		return
	}

	if err := h.retriever.UpdateDocument(uint(id), &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "updated"})
}

// DeleteDocument 删除文档
func (h *KnowledgeHandler) DeleteDocument(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	if err := h.retriever.DeleteDocument(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "deleted"})
}

// ToggleDocument 启用/禁用文档
func (h *KnowledgeHandler) ToggleDocument(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "error": err.Error()})
		return
	}

	if err := h.retriever.ToggleDocument(uint(id), req.Enabled); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "toggled"})
}

// GetStats 获取统计信息
func (h *KnowledgeHandler) GetStats(c *gin.Context) {
	stats, err := h.retriever.GetStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": stats,
	})
}

// GetCategories 获取所有分类
func (h *KnowledgeHandler) GetCategories(c *gin.Context) {
	stats, err := h.retriever.GetStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": stats["categories"],
	})
}

// GetTags 获取所有标签
func (h *KnowledgeHandler) GetTags(c *gin.Context) {
	// TODO: 实现从所有文档中提取标签统计
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": []string{},
	})
}

// SearchDocuments 搜索文档
func (h *KnowledgeHandler) SearchDocuments(c *gin.Context) {
	var req struct {
		Query    string `json:"query" binding:"required"`
		Category string `json:"category"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "error": err.Error()})
		return
	}

	docs, err := h.retriever.Retrieve(c.Request.Context(), req.Query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "error": err.Error()})
		return
	}

	// 如果指定了分类，过滤结果
	if req.Category != "" {
		var filtered []*knowledge.Document
		for _, doc := range docs {
			if doc.Category == req.Category {
				filtered = append(filtered, doc)
			}
		}
		docs = filtered
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"documents": docs,
			"query":     req.Query,
			"total":     len(docs),
		},
	})
}
