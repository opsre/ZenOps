package server

import (
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
