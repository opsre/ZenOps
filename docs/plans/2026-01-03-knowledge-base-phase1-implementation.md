# 知识库功能 Phase 1 实施计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 实现知识库管理的核心功能，包括文档 CRUD、分类管理、标签管理、Markdown 编辑器和 AI 对话集成

**Architecture:**
- 后端：扩展 KnowledgeDocument 模型添加 Tags 字段，创建 KnowledgeHandler 提供 REST API
- 前端：React 组件树（KnowledgeView → DocumentList/Editor/CategoryTree），使用 react-markdown-editor-lite
- 集成：知识库自动集成到 AI 对话（已有机制，无需修改）

**Tech Stack:**
- 后端：Go, Gin, GORM, SQLite
- 前端：React, TypeScript, Tailwind CSS, react-markdown-editor-lite, markdown-it

**预计时间:** 2-3 天（10-12 小时）

---

## 任务分组

### 第一阶段：后端基础 (2-3 小时)
- Task 1-4: 数据模型、Handler、API、路由

### 第二阶段：前端基础 (1-2 小时)
- Task 5-7: 类型定义、API Service、组件结构

### 第三阶段：核心 UI (3-4 小时)
- Task 8-13: 主要组件实现

### 第四阶段：集成测试 (1-2 小时)
- Task 14-16: 端到端测试、优化

---

## Task 1: 扩展数据模型添加 Tags 字段

**目标:** 为 KnowledgeDocument 添加 Tags 字段用于存储文档标签

**Files:**
- Modify: `internal/model/knowledge_document.go:1-24`

**Step 1: 添加 Tags 字段**

在 `KnowledgeDocument` 结构体中添加 Tags 字段：

```go
type KnowledgeDocument struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	DocType        string    `json:"doc_type" gorm:"size:50"`           // 'markdown', 'pdf', 'url', 'manual'
	Title          string    `json:"title" gorm:"size:255"`
	Content        string    `json:"content" gorm:"type:text"`
	Metadata       string    `json:"metadata" gorm:"type:json"`         // 存储来源、作者等元信息
	Enabled        bool      `json:"enabled" gorm:"default:true;index"`
	Category       string    `json:"category" gorm:"size:100;index"`    // 分类：运维文档、API文档等
	Tags           string    `json:"tags" gorm:"type:text"`             // NEW: JSON 数组 ["tag1", "tag2"]
	Embedding      string    `json:"embedding" gorm:"type:text"`        // JSON 格式的向量 (用于语义搜索)
	EmbeddingModel string    `json:"embedding_model" gorm:"size:64"`    // Embedding 模型标识
}
```

**Step 2: 验证编译**

Run: `go build ./...`
Expected: 编译成功，无错误

**Step 3: 测试数据库迁移**

Run: `go run ./cmd/zenops/main.go` (启动后立即停止)
Expected: GORM AutoMigrate 自动添加 tags 字段

**Step 4: Commit**

```bash
git add internal/model/knowledge_document.go
git commit -m "feat(model): 为 KnowledgeDocument 添加 Tags 字段

- 添加 tags 字段用于存储 JSON 数组格式的标签
- GORM 自动迁移会在系统启动时创建字段"
```

---

## Task 2: 创建 KnowledgeHandler

**目标:** 创建知识库 REST API Handler

**Files:**
- Create: `internal/handler/knowledge_handler.go`

**Step 1: 创建 Handler 文件和基础结构**

创建 `internal/handler/knowledge_handler.go`：

```go
package handler

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
```

**Step 2: Commit**

```bash
git add internal/handler/knowledge_handler.go
git commit -m "feat(handler): 创建 KnowledgeHandler 基础结构

- 创建 Handler 结构体
- 注册 10 个 API 路由
- 准备实现具体接口"
```

---

## Task 3: 实现 API 接口方法

**目标:** 实现所有 REST API 接口

**Files:**
- Modify: `internal/handler/knowledge_handler.go:25-end`

**Step 1: 实现 ListDocuments**

在 `RegisterRoutes` 之后添加：

```go
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
```

**Step 2: 实现 GetDocument**

```go
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
```

**Step 3: 实现 CreateDocument**

```go
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
```

**Step 4: 实现 UpdateDocument**

```go
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
```

**Step 5: 实现 DeleteDocument**

```go
// DeleteDocument 删除文档
func (h *KnowledgeHandler) DeleteDocument(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	if err := h.retriever.DeleteDocument(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "deleted"})
}
```

**Step 6: 实现 ToggleDocument**

```go
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
```

**Step 7: 实现 GetStats**

```go
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
```

**Step 8: 实现 GetCategories**

```go
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
```

**Step 9: 实现 GetTags（暂时返回空）**

```go
// GetTags 获取所有标签
func (h *KnowledgeHandler) GetTags(c *gin.Context) {
	// TODO: 实现从所有文档中提取标签统计
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": []string{},
	})
}
```

**Step 10: 实现 SearchDocuments**

```go
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
```

**Step 11: 验证编译**

Run: `go build ./...`
Expected: 编译成功

**Step 12: Commit**

```bash
git add internal/handler/knowledge_handler.go
git commit -m "feat(handler): 实现知识库所有 API 接口

- ListDocuments: 支持分类和状态筛选
- GetDocument: 获取单个文档详情
- CreateDocument: 创建新文档
- UpdateDocument: 更新文档内容
- DeleteDocument: 删除文档
- ToggleDocument: 启用/禁用文档
- GetStats: 统计信息
- GetCategories: 分类列表
- GetTags: 标签列表（待实现）
- SearchDocuments: 搜索并支持分类过滤"
```

---

## Task 4: 注册路由

**目标:** 将 KnowledgeHandler 注册到路由系统

**Files:**
- Modify: `internal/server/router.go`

**Step 1: 查看当前路由注册位置**

Read: `internal/server/router.go`
找到 `setupRoutes` 或类似函数，确定注册位置

**Step 2: 添加知识库路由注册**

在现有路由注册之后添加：

```go
// 知识库路由
knowledgeHandler := handler.NewKnowledgeHandler(/* 获取 retriever 实例 */)
knowledgeHandler.RegisterRoutes(apiV1)
```

注意：需要确保能访问到 `knowledge.Retriever` 实例，可能需要从 `agent.GetGlobalAgent().Orchestrator.knowledgeRet` 获取

**Step 3: 添加必要的 import**

确保导入：
```go
import (
	"github.com/eryajf/zenops/internal/handler"
	"github.com/eryajf/zenops/internal/agent"
)
```

**Step 4: 验证编译和启动**

Run: `go build ./... && ./zenops run`
Expected:
- 编译成功
- 服务启动
- 日志中显示路由注册成功

**Step 5: 测试 API 端点**

Run: `curl -H "Authorization: Bearer YOUR_TOKEN" http://localhost:8080/api/v1/knowledge/stats`
Expected: 返回统计信息 JSON

**Step 6: Commit**

```bash
git add internal/server/router.go
git commit -m "feat(router): 注册知识库 API 路由

- 注册 /api/v1/knowledge/* 路由组
- 连接 KnowledgeHandler 和 Retriever"
```

---

## Task 5: 前端类型定义

**目标:** 创建 TypeScript 类型定义

**Files:**
- Create: `zenops-web/types/knowledge.ts`

**Step 1: 创建类型定义文件**

创建 `zenops-web/types/knowledge.ts`：

```typescript
export interface KnowledgeDocument {
  id: number;
  title: string;
  content: string;
  doc_type: 'markdown' | 'pdf' | 'url' | 'manual';
  category: string;
  tags: string[];
  enabled: boolean;
  metadata: {
    source_url?: string;
    author?: string;
    [key: string]: any;
  };
  created_at: string;
  updated_at: string;
  score?: number;
}

export interface CreateDocumentRequest {
  title: string;
  content: string;
  doc_type?: string;
  category: string;
  tags?: string[];
  metadata?: Record<string, any>;
}

export interface UpdateDocumentRequest extends CreateDocumentRequest {}

export interface KnowledgeStats {
  total_count: number;
  enabled_count: number;
  categories: string[];
}

export interface SearchRequest {
  query: string;
  category?: string;
}

export interface SearchResponse {
  documents: KnowledgeDocument[];
  query: string;
  total: number;
}
```

**Step 2: Commit**

```bash
git add zenops-web/types/knowledge.ts
git commit -m "feat(frontend): 添加知识库 TypeScript 类型定义

- KnowledgeDocument: 文档模型
- CreateDocumentRequest: 创建请求
- KnowledgeStats: 统计信息
- SearchRequest/Response: 搜索接口"
```

---

## Task 6: 前端 API Service

**目标:** 创建知识库 API 调用服务

**Files:**
- Modify: `zenops-web/services/api.ts`

**Step 1: 添加知识库 API**

在 `zenops-web/services/api.ts` 文件末尾添加：

```typescript
import { KnowledgeDocument, CreateDocumentRequest, KnowledgeStats, SearchResponse } from '../types/knowledge';

// 知识库 API
export const knowledgeApi = {
  /**
   * 获取文档列表
   */
  async listDocuments(params?: {
    category?: string;
    enabled?: boolean;
  }): Promise<KnowledgeDocument[]> {
    const queryParams = new URLSearchParams();
    if (params?.category) queryParams.append('category', params.category);
    if (params?.enabled !== undefined) queryParams.append('enabled', String(params.enabled));

    const url = queryParams.toString()
      ? `${API_BASE}/knowledge/documents?${queryParams}`
      : `${API_BASE}/knowledge/documents`;

    const response = await fetch(url, {
      headers: getAuthHeaders(),
    });

    if (!response.ok) {
      throw new Error('Failed to fetch documents');
    }

    const data = await response.json();
    return data.data || [];
  },

  /**
   * 获取单个文档
   */
  async getDocument(id: number): Promise<KnowledgeDocument> {
    const response = await fetch(`${API_BASE}/knowledge/documents/${id}`, {
      headers: getAuthHeaders(),
    });

    if (!response.ok) {
      throw new Error('Failed to fetch document');
    }

    const data = await response.json();
    return data.data;
  },

  /**
   * 创建文档
   */
  async createDocument(doc: CreateDocumentRequest): Promise<number> {
    const response = await fetch(`${API_BASE}/knowledge/documents`, {
      method: 'POST',
      headers: getAuthHeaders(),
      body: JSON.stringify(doc),
    });

    if (!response.ok) {
      throw new Error('Failed to create document');
    }

    const data = await response.json();
    return data.data.id;
  },

  /**
   * 更新文档
   */
  async updateDocument(id: number, doc: CreateDocumentRequest): Promise<void> {
    const response = await fetch(`${API_BASE}/knowledge/documents/${id}`, {
      method: 'PUT',
      headers: getAuthHeaders(),
      body: JSON.stringify(doc),
    });

    if (!response.ok) {
      throw new Error('Failed to update document');
    }
  },

  /**
   * 删除文档
   */
  async deleteDocument(id: number): Promise<void> {
    const response = await fetch(`${API_BASE}/knowledge/documents/${id}`, {
      method: 'DELETE',
      headers: getAuthHeaders(),
    });

    if (!response.ok) {
      throw new Error('Failed to delete document');
    }
  },

  /**
   * 启用/禁用文档
   */
  async toggleDocument(id: number, enabled: boolean): Promise<void> {
    const response = await fetch(`${API_BASE}/knowledge/documents/${id}/toggle`, {
      method: 'PATCH',
      headers: getAuthHeaders(),
      body: JSON.stringify({ enabled }),
    });

    if (!response.ok) {
      throw new Error('Failed to toggle document');
    }
  },

  /**
   * 获取统计信息
   */
  async getStats(): Promise<KnowledgeStats> {
    const response = await fetch(`${API_BASE}/knowledge/stats`, {
      headers: getAuthHeaders(),
    });

    if (!response.ok) {
      throw new Error('Failed to fetch stats');
    }

    const data = await response.json();
    return data.data;
  },

  /**
   * 搜索文档
   */
  async search(query: string, category?: string): Promise<SearchResponse> {
    const response = await fetch(`${API_BASE}/knowledge/search`, {
      method: 'POST',
      headers: getAuthHeaders(),
      body: JSON.stringify({ query, category }),
    });

    if (!response.ok) {
      throw new Error('Failed to search documents');
    }

    const data = await response.json();
    return data.data;
  },
};
```

**Step 2: 验证类型检查**

Run: `cd zenops-web && npm run type-check`
Expected: 无类型错误

**Step 3: Commit**

```bash
git add zenops-web/services/api.ts zenops-web/types/knowledge.ts
git commit -m "feat(frontend): 添加知识库 API Service

- listDocuments: 获取文档列表（支持筛选）
- getDocument: 获取单个文档
- createDocument: 创建文档
- updateDocument: 更新文档
- deleteDocument: 删除文档
- toggleDocument: 启用/禁用
- getStats: 统计信息
- search: 搜索文档"
```

---

## Task 7: 创建组件目录结构

**目标:** 创建知识库组件的目录和骨架文件

**Files:**
- Create: `zenops-web/components/knowledge/` (目录)
- Create: `zenops-web/components/KnowledgeView.tsx`
- Create: `zenops-web/components/knowledge/DocumentList.tsx`
- Create: `zenops-web/components/knowledge/DocumentEditor.tsx`
- Create: `zenops-web/components/knowledge/CategoryTree.tsx`
- Create: `zenops-web/components/knowledge/StatsCards.tsx`

**Step 1: 创建目录**

Run: `mkdir -p zenops-web/components/knowledge`

**Step 2: 创建组件骨架**

创建每个组件文件，包含基础结构（先不实现具体逻辑）：

`zenops-web/components/KnowledgeView.tsx`:
```typescript
import React from 'react';

const KnowledgeView = () => {
  return (
    <div className="flex flex-col h-full p-6">
      <h1>知识库管理</h1>
      {/* TODO: 实现完整功能 */}
    </div>
  );
};

export default KnowledgeView;
```

`zenops-web/components/knowledge/DocumentList.tsx`:
```typescript
import React from 'react';
import { KnowledgeDocument } from '../../types/knowledge';

interface Props {
  documents: KnowledgeDocument[];
  loading: boolean;
  onEdit: (doc: KnowledgeDocument) => void;
  onDelete: () => void;
  onToggle: () => void;
}

const DocumentList: React.FC<Props> = ({ documents, loading }) => {
  return <div>DocumentList - TODO</div>;
};

export default DocumentList;
```

`zenops-web/components/knowledge/DocumentEditor.tsx`:
```typescript
import React from 'react';
import { KnowledgeDocument } from '../../types/knowledge';

interface Props {
  document: KnowledgeDocument | null;
  onClose: () => void;
  onSave: () => void;
}

const DocumentEditor: React.FC<Props> = ({ document, onClose, onSave }) => {
  return <div>DocumentEditor - TODO</div>;
};

export default DocumentEditor;
```

`zenops-web/components/knowledge/CategoryTree.tsx`:
```typescript
import React from 'react';

interface Props {
  selectedCategory: string;
  onSelectCategory: (category: string) => void;
}

const CategoryTree: React.FC<Props> = ({ selectedCategory, onSelectCategory }) => {
  return <div>CategoryTree - TODO</div>;
};

export default CategoryTree;
```

`zenops-web/components/knowledge/StatsCards.tsx`:
```typescript
import React from 'react';

const StatsCards = () => {
  return <div>StatsCards - TODO</div>;
};

export default StatsCards;
```

**Step 3: 验证编译**

Run: `cd zenops-web && npm run type-check`
Expected: 无类型错误

**Step 4: Commit**

```bash
git add zenops-web/components/KnowledgeView.tsx zenops-web/components/knowledge/
git commit -m "feat(frontend): 创建知识库组件骨架

- KnowledgeView: 主容器组件
- DocumentList: 文档列表
- DocumentEditor: 文档编辑器
- CategoryTree: 分类导航
- StatsCards: 统计卡片

下一步实现具体逻辑"
```

---

## Task 8: 实现 KnowledgeView 主组件

**目标:** 实现知识库主页面逻辑和布局

**Files:**
- Modify: `zenops-web/components/KnowledgeView.tsx`

**Step 1: 实现完整组件**

替换 `KnowledgeView.tsx` 内容：

```typescript
import React, { useState, useEffect } from 'react';
import { Plus, Search } from 'lucide-react';
import { knowledgeApi } from '../services/api';
import { KnowledgeDocument } from '../types/knowledge';
import DocumentList from './knowledge/DocumentList';
import CategoryTree from './knowledge/CategoryTree';
import StatsCards from './knowledge/StatsCards';
import DocumentEditor from './knowledge/DocumentEditor';

const KnowledgeView = () => {
  const [documents, setDocuments] = useState<KnowledgeDocument[]>([]);
  const [selectedCategory, setSelectedCategory] = useState<string>('');
  const [isEditorOpen, setIsEditorOpen] = useState(false);
  const [editingDoc, setEditingDoc] = useState<KnowledgeDocument | null>(null);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');

  useEffect(() => {
    loadDocuments();
  }, [selectedCategory]);

  const loadDocuments = async () => {
    setLoading(true);
    try {
      const docs = await knowledgeApi.listDocuments({
        category: selectedCategory || undefined,
      });
      setDocuments(docs);
    } catch (error) {
      console.error('Failed to load documents:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleCreate = () => {
    setEditingDoc(null);
    setIsEditorOpen(true);
  };

  const handleEdit = (doc: KnowledgeDocument) => {
    setEditingDoc(doc);
    setIsEditorOpen(true);
  };

  const handleSave = async () => {
    setIsEditorOpen(false);
    await loadDocuments();
  };

  const handleSearch = async () => {
    if (!searchQuery.trim()) {
      loadDocuments();
      return;
    }

    setLoading(true);
    try {
      const result = await knowledgeApi.search(searchQuery, selectedCategory || undefined);
      setDocuments(result.documents);
    } catch (error) {
      console.error('Failed to search:', error);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex flex-col h-full p-6 bg-slate-50 dark:bg-slate-950">
      {/* 统计卡片 */}
      <StatsCards />

      <div className="flex flex-1 gap-6 mt-6 min-h-0">
        {/* 左侧分类导航 */}
        <div className="w-64 shrink-0">
          <CategoryTree
            selectedCategory={selectedCategory}
            onSelectCategory={setSelectedCategory}
          />
        </div>

        {/* 右侧文档列表 */}
        <div className="flex-1 flex flex-col min-w-0 bg-white dark:bg-slate-900 rounded-2xl border border-slate-200 dark:border-slate-800 p-6">
          {/* 工具栏 */}
          <div className="flex gap-4 mb-6">
            <div className="flex-1 relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
              <input
                type="text"
                placeholder="搜索文档（标题、内容、标签）"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                onKeyPress={(e) => e.key === 'Enter' && handleSearch()}
                className="w-full pl-10 pr-4 py-2 border border-slate-200 dark:border-slate-700 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-indigo-500"
              />
            </div>
            <button
              onClick={handleCreate}
              className="flex items-center gap-2 px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors"
            >
              <Plus className="w-4 h-4" />
              新建文档
            </button>
          </div>

          {/* 文档列表 */}
          <DocumentList
            documents={documents}
            loading={loading}
            onEdit={handleEdit}
            onDelete={loadDocuments}
            onToggle={loadDocuments}
          />
        </div>
      </div>

      {/* 编辑器弹窗 */}
      {isEditorOpen && (
        <DocumentEditor
          document={editingDoc}
          onClose={() => setIsEditorOpen(false)}
          onSave={handleSave}
        />
      )}
    </div>
  );
};

export default KnowledgeView;
```

**Step 2: 验证编译**

Run: `cd zenops-web && npm run type-check`
Expected: 无类型错误

**Step 3: Commit**

```bash
git add zenops-web/components/KnowledgeView.tsx
git commit -m "feat(frontend): 实现 KnowledgeView 主组件逻辑

- 文档列表加载和状态管理
- 分类筛选
- 搜索功能
- 创建/编辑文档弹窗控制
- 响应式布局"
```

---

## Task 9: 实现 StatsCards 统计卡片

**目标:** 实现顶部统计信息展示

**Files:**
- Modify: `zenops-web/components/knowledge/StatsCards.tsx`

**Step 1: 实现组件**

```typescript
import React, { useEffect, useState } from 'react';
import { BookOpen, CheckCircle, FolderOpen } from 'lucide-react';
import { knowledgeApi } from '../../services/api';
import { KnowledgeStats } from '../../types/knowledge';

const StatsCards = () => {
  const [stats, setStats] = useState<KnowledgeStats | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadStats();
  }, []);

  const loadStats = async () => {
    try {
      const data = await knowledgeApi.getStats();
      setStats(data);
    } catch (error) {
      console.error('Failed to load stats:', error);
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <div className="grid grid-cols-3 gap-6 mb-6">
        {[1, 2, 3].map((i) => (
          <div key={i} className="bg-white dark:bg-slate-900 rounded-2xl border border-slate-200 dark:border-slate-800 p-6 animate-pulse">
            <div className="h-4 bg-slate-200 dark:bg-slate-800 rounded w-1/2 mb-2"></div>
            <div className="h-8 bg-slate-200 dark:bg-slate-800 rounded w-1/3"></div>
          </div>
        ))}
      </div>
    );
  }

  return (
    <div className="grid grid-cols-3 gap-6">
      {/* 总文档数 */}
      <div className="bg-white dark:bg-slate-900 rounded-2xl border border-slate-200 dark:border-slate-800 p-6">
        <div className="flex items-center gap-4">
          <div className="p-3 bg-indigo-50 dark:bg-indigo-900/30 rounded-xl">
            <BookOpen className="w-6 h-6 text-indigo-600 dark:text-indigo-400" />
          </div>
          <div>
            <p className="text-sm text-slate-500 dark:text-slate-400">总文档</p>
            <p className="text-2xl font-bold text-slate-900 dark:text-white">
              {stats?.total_count || 0}
            </p>
          </div>
        </div>
      </div>

      {/* 已启用 */}
      <div className="bg-white dark:bg-slate-900 rounded-2xl border border-slate-200 dark:border-slate-800 p-6">
        <div className="flex items-center gap-4">
          <div className="p-3 bg-green-50 dark:bg-green-900/30 rounded-xl">
            <CheckCircle className="w-6 h-6 text-green-600 dark:text-green-400" />
          </div>
          <div>
            <p className="text-sm text-slate-500 dark:text-slate-400">已启用</p>
            <p className="text-2xl font-bold text-slate-900 dark:text-white">
              {stats?.enabled_count || 0}
            </p>
          </div>
        </div>
      </div>

      {/* 分类数 */}
      <div className="bg-white dark:bg-slate-900 rounded-2xl border border-slate-200 dark:border-slate-800 p-6">
        <div className="flex items-center gap-4">
          <div className="p-3 bg-purple-50 dark:bg-purple-900/30 rounded-xl">
            <FolderOpen className="w-6 h-6 text-purple-600 dark:text-purple-400" />
          </div>
          <div>
            <p className="text-sm text-slate-500 dark:text-slate-400">分类</p>
            <p className="text-2xl font-bold text-slate-900 dark:text-white">
              {stats?.categories?.length || 0}
            </p>
          </div>
        </div>
      </div>
    </div>
  );
};

export default StatsCards;
```

**Step 2: Commit**

```bash
git add zenops-web/components/knowledge/StatsCards.tsx
git commit -m "feat(frontend): 实现统计卡片组件

- 总文档数
- 已启用数
- 分类数
- 加载状态和暗黑模式支持"
```

---

## Task 10: 实现 CategoryTree 分类导航

**目标:** 实现左侧分类导航树

**Files:**
- Modify: `zenops-web/components/knowledge/CategoryTree.tsx`

**Step 1: 实现组件**

```typescript
import React, { useEffect, useState } from 'react';
import { FolderOpen, BookOpen, Wrench, AlertCircle, Settings } from 'lucide-react';
import { knowledgeApi } from '../../services/api';

interface Props {
  selectedCategory: string;
  onSelectCategory: (category: string) => void;
}

const CategoryTree: React.FC<Props> = ({ selectedCategory, onSelectCategory }) => {
  const [stats, setStats] = useState<any>(null);

  useEffect(() => {
    loadStats();
  }, []);

  const loadStats = async () => {
    try {
      const data = await knowledgeApi.getStats();
      setStats(data);
    } catch (error) {
      console.error('Failed to load stats:', error);
    }
  };

  const getCategoryIcon = (category: string) => {
    switch (category) {
      case '运维文档':
        return <BookOpen className="w-4 h-4" />;
      case 'API文档':
        return <Wrench className="w-4 h-4" />;
      case '故障案例':
        return <AlertCircle className="w-4 h-4" />;
      case '配置模板':
        return <Settings className="w-4 h-4" />;
      default:
        return <FolderOpen className="w-4 h-4" />;
    }
  };

  const getCategoryCount = (category: string) => {
    // TODO: 后端返回分类统计
    return 0;
  };

  const categories = ['运维文档', 'API文档', '故障案例', '配置模板'];

  return (
    <div className="bg-white dark:bg-slate-900 rounded-2xl border border-slate-200 dark:border-slate-800 p-4">
      <h3 className="text-sm font-bold text-slate-900 dark:text-white mb-4 px-2">分类</h3>

      <div className="space-y-1">
        {/* 全部 */}
        <button
          onClick={() => onSelectCategory('')}
          className={`w-full flex items-center gap-3 px-3 py-2 rounded-lg text-sm transition-colors ${
            selectedCategory === ''
              ? 'bg-indigo-50 dark:bg-indigo-900/30 text-indigo-600 dark:text-indigo-400'
              : 'text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-800'
          }`}
        >
          <FolderOpen className="w-4 h-4" />
          <span className="flex-1 text-left">全部</span>
          <span className="text-xs text-slate-400">{stats?.total_count || 0}</span>
        </button>

        {/* 分类列表 */}
        {categories.map((category) => (
          <button
            key={category}
            onClick={() => onSelectCategory(category)}
            className={`w-full flex items-center gap-3 px-3 py-2 rounded-lg text-sm transition-colors ${
              selectedCategory === category
                ? 'bg-indigo-50 dark:bg-indigo-900/30 text-indigo-600 dark:text-indigo-400'
                : 'text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-800'
            }`}
          >
            {getCategoryIcon(category)}
            <span className="flex-1 text-left">{category}</span>
            <span className="text-xs text-slate-400">{getCategoryCount(category)}</span>
          </button>
        ))}
      </div>
    </div>
  );
};

export default CategoryTree;
```

**Step 2: Commit**

```bash
git add zenops-web/components/knowledge/CategoryTree.tsx
git commit -m "feat(frontend): 实现分类导航组件

- 全部文档入口
- 预定义分类（运维、API、故障、配置）
- 图标和计数显示
- 选中状态高亮"
```

---

## Task 11: 实现 DocumentList 文档列表

**目标:** 实现文档列表表格和操作

**Files:**
- Modify: `zenops-web/components/knowledge/DocumentList.tsx`

**Step 1: 实现组件**

```typescript
import React from 'react';
import { Edit, Trash2, Eye, EyeOff } from 'lucide-react';
import { KnowledgeDocument } from '../../types/knowledge';
import { knowledgeApi } from '../../services/api';

interface Props {
  documents: KnowledgeDocument[];
  loading: boolean;
  onEdit: (doc: KnowledgeDocument) => void;
  onDelete: () => void;
  onToggle: () => void;
}

const DocumentList: React.FC<Props> = ({ documents, loading, onEdit, onDelete, onToggle }) => {
  const handleToggle = async (doc: KnowledgeDocument) => {
    try {
      await knowledgeApi.toggleDocument(doc.id, !doc.enabled);
      onToggle();
    } catch (error) {
      console.error('Failed to toggle document:', error);
      alert('操作失败');
    }
  };

  const handleDelete = async (doc: KnowledgeDocument) => {
    if (!confirm(`确定删除文档"${doc.title}"吗？`)) {
      return;
    }

    try {
      await knowledgeApi.deleteDocument(doc.id);
      onDelete();
    } catch (error) {
      console.error('Failed to delete document:', error);
      alert('删除失败');
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-indigo-600"></div>
      </div>
    );
  }

  if (documents.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-center">
        <div className="text-6xl mb-4">📚</div>
        <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-2">
          暂无文档
        </h3>
        <p className="text-sm text-slate-500 dark:text-slate-400">
          点击"新建文档"添加您的第一个文档
        </p>
      </div>
    );
  }

  return (
    <div className="flex-1 overflow-auto">
      <table className="w-full">
        <thead className="bg-slate-50 dark:bg-slate-800 sticky top-0">
          <tr>
            <th className="px-4 py-3 text-left text-xs font-semibold text-slate-600 dark:text-slate-400">
              标题
            </th>
            <th className="px-4 py-3 text-left text-xs font-semibold text-slate-600 dark:text-slate-400">
              分类
            </th>
            <th className="px-4 py-3 text-left text-xs font-semibold text-slate-600 dark:text-slate-400">
              标签
            </th>
            <th className="px-4 py-3 text-left text-xs font-semibold text-slate-600 dark:text-slate-400">
              状态
            </th>
            <th className="px-4 py-3 text-left text-xs font-semibold text-slate-600 dark:text-slate-400">
              创建时间
            </th>
            <th className="px-4 py-3 text-right text-xs font-semibold text-slate-600 dark:text-slate-400">
              操作
            </th>
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-200 dark:divide-slate-800">
          {documents.map((doc) => (
            <tr key={doc.id} className="hover:bg-slate-50 dark:hover:bg-slate-800 transition-colors">
              <td className="px-4 py-3">
                <div className="font-medium text-slate-900 dark:text-white">{doc.title}</div>
              </td>
              <td className="px-4 py-3">
                <span className="inline-flex items-center px-2 py-1 rounded-md text-xs font-medium bg-indigo-50 dark:bg-indigo-900/30 text-indigo-600 dark:text-indigo-400">
                  {doc.category || '未分类'}
                </span>
              </td>
              <td className="px-4 py-3">
                <div className="flex flex-wrap gap-1">
                  {doc.tags?.slice(0, 3).map((tag, i) => (
                    <span
                      key={i}
                      className="inline-flex items-center px-2 py-0.5 rounded text-xs bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-400"
                    >
                      #{tag}
                    </span>
                  ))}
                  {doc.tags?.length > 3 && (
                    <span className="text-xs text-slate-400">+{doc.tags.length - 3}</span>
                  )}
                </div>
              </td>
              <td className="px-4 py-3">
                <button
                  onClick={() => handleToggle(doc)}
                  className={`inline-flex items-center gap-1 px-2 py-1 rounded-md text-xs font-medium transition-colors ${
                    doc.enabled
                      ? 'bg-green-50 dark:bg-green-900/30 text-green-600 dark:text-green-400'
                      : 'bg-slate-100 dark:bg-slate-800 text-slate-500 dark:text-slate-400'
                  }`}
                >
                  {doc.enabled ? <Eye className="w-3 h-3" /> : <EyeOff className="w-3 h-3" />}
                  {doc.enabled ? '已启用' : '已禁用'}
                </button>
              </td>
              <td className="px-4 py-3 text-sm text-slate-500 dark:text-slate-400">
                {new Date(doc.created_at).toLocaleDateString('zh-CN')}
              </td>
              <td className="px-4 py-3">
                <div className="flex items-center justify-end gap-2">
                  <button
                    onClick={() => onEdit(doc)}
                    className="p-1.5 text-slate-500 hover:text-indigo-600 dark:hover:text-indigo-400 transition-colors"
                    title="编辑"
                  >
                    <Edit className="w-4 h-4" />
                  </button>
                  <button
                    onClick={() => handleDelete(doc)}
                    className="p-1.5 text-slate-500 hover:text-red-600 dark:hover:text-red-400 transition-colors"
                    title="删除"
                  >
                    <Trash2 className="w-4 h-4" />
                  </button>
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};

export default DocumentList;
```

**Step 2: Commit**

```bash
git add zenops-web/components/knowledge/DocumentList.tsx
git commit -m "feat(frontend): 实现文档列表组件

- 表格展示（标题、分类、标签、状态、时间）
- 启用/禁用切换
- 编辑和删除操作
- 空状态提示
- 加载状态"
```

---

## Task 12: 实现 DocumentEditor 编辑器（简化版）

**目标:** 实现文档编辑器（先不集成 Markdown 编辑器）

**Files:**
- Modify: `zenops-web/components/knowledge/DocumentEditor.tsx`

**Step 1: 实现基础编辑器**

```typescript
import React, { useState, useEffect } from 'react';
import { X } from 'lucide-react';
import { KnowledgeDocument, CreateDocumentRequest } from '../../types/knowledge';
import { knowledgeApi } from '../../services/api';

interface Props {
  document: KnowledgeDocument | null;
  onClose: () => void;
  onSave: () => void;
}

const DocumentEditor: React.FC<Props> = ({ document, onClose, onSave }) => {
  const [title, setTitle] = useState('');
  const [content, setContent] = useState('');
  const [category, setCategory] = useState('运维文档');
  const [tags, setTags] = useState<string[]>([]);
  const [tagInput, setTagInput] = useState('');
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (document) {
      setTitle(document.title);
      setContent(document.content);
      setCategory(document.category || '运维文档');
      setTags(document.tags || []);
    }
  }, [document]);

  const handleAddTag = () => {
    const tag = tagInput.trim();
    if (tag && !tags.includes(tag)) {
      setTags([...tags, tag]);
      setTagInput('');
    }
  };

  const handleRemoveTag = (tagToRemove: string) => {
    setTags(tags.filter((t) => t !== tagToRemove));
  };

  const handleSave = async () => {
    if (!title.trim() || !content.trim()) {
      alert('标题和内容不能为空');
      return;
    }

    setSaving(true);
    try {
      const req: CreateDocumentRequest = {
        title,
        content,
        category,
        tags,
        doc_type: 'markdown',
      };

      if (document) {
        await knowledgeApi.updateDocument(document.id, req);
      } else {
        await knowledgeApi.createDocument(req);
      }

      onSave();
    } catch (error) {
      console.error('Failed to save document:', error);
      alert('保存失败');
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div className="bg-white dark:bg-slate-900 rounded-2xl w-full max-w-4xl max-h-[90vh] flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-slate-200 dark:border-slate-800">
          <h2 className="text-xl font-bold text-slate-900 dark:text-white">
            {document ? '编辑文档' : '新建文档'}
          </h2>
          <div className="flex items-center gap-3">
            <button
              onClick={handleSave}
              disabled={saving}
              className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 disabled:opacity-50 transition-colors"
            >
              {saving ? '保存中...' : '保存'}
            </button>
            <button
              onClick={onClose}
              className="p-2 text-slate-500 hover:text-slate-700 dark:hover:text-slate-300 transition-colors"
            >
              <X className="w-5 h-5" />
            </button>
          </div>
        </div>

        {/* Body */}
        <div className="flex-1 overflow-auto p-6 space-y-6">
          {/* 标题 */}
          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
              标题 *
            </label>
            <input
              type="text"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="请输入文档标题"
              className="w-full px-4 py-2 border border-slate-200 dark:border-slate-700 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-indigo-500"
            />
          </div>

          {/* 分类 */}
          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
              分类 *
            </label>
            <select
              value={category}
              onChange={(e) => setCategory(e.target.value)}
              className="w-full px-4 py-2 border border-slate-200 dark:border-slate-700 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-indigo-500"
            >
              <option value="运维文档">运维文档</option>
              <option value="API文档">API文档</option>
              <option value="故障案例">故障案例</option>
              <option value="配置模板">配置模板</option>
            </select>
          </div>

          {/* 标签 */}
          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
              标签
            </label>
            <div className="flex gap-2 mb-2">
              <input
                type="text"
                value={tagInput}
                onChange={(e) => setTagInput(e.target.value)}
                onKeyPress={(e) => e.key === 'Enter' && handleAddTag()}
                placeholder="输入标签，按回车添加"
                className="flex-1 px-4 py-2 border border-slate-200 dark:border-slate-700 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-indigo-500"
              />
              <button
                onClick={handleAddTag}
                className="px-4 py-2 bg-slate-100 dark:bg-slate-800 text-slate-700 dark:text-slate-300 rounded-lg hover:bg-slate-200 dark:hover:bg-slate-700 transition-colors"
              >
                添加
              </button>
            </div>
            <div className="flex flex-wrap gap-2">
              {tags.map((tag) => (
                <span
                  key={tag}
                  className="inline-flex items-center gap-1 px-3 py-1 bg-indigo-50 dark:bg-indigo-900/30 text-indigo-600 dark:text-indigo-400 rounded-lg text-sm"
                >
                  #{tag}
                  <button
                    onClick={() => handleRemoveTag(tag)}
                    className="ml-1 text-indigo-400 hover:text-indigo-600 dark:hover:text-indigo-300"
                  >
                    <X className="w-3 h-3" />
                  </button>
                </span>
              ))}
            </div>
          </div>

          {/* 内容 (简化版 Textarea) */}
          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
              内容 * (Markdown 格式)
            </label>
            <textarea
              value={content}
              onChange={(e) => setContent(e.target.value)}
              placeholder="请输入文档内容（支持 Markdown 语法）"
              rows={15}
              className="w-full px-4 py-2 border border-slate-200 dark:border-slate-700 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-white font-mono text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
            />
          </div>
        </div>
      </div>
    </div>
  );
};

export default DocumentEditor;
```

**Step 2: Commit**

```bash
git add zenops-web/components/knowledge/DocumentEditor.tsx
git commit -m "feat(frontend): 实现文档编辑器基础版

- 标题、分类、标签输入
- 内容编辑（Textarea，待升级为 Markdown 编辑器）
- 创建/更新逻辑
- 弹窗样式

TODO: 集成 Markdown 编辑器"
```

---

## Task 13: 添加导航入口和路由

**目标:** 在主应用中添加知识库导航和路由

**Files:**
- Modify: `zenops-web/App.tsx`
- Modify: `zenops-web/types.ts` (如果有 ViewState 类型定义)

**Step 1: 添加 ViewState 类型**

在 `types.ts` 或 `App.tsx` 中找到 ViewState 定义，添加 'knowledge'：

```typescript
export type ViewState = 'dashboard' | 'mcp' | 'history' | 'config' | 'chat' | 'mcp-logs' | 'profile' | 'knowledge';
```

**Step 2: 添加导航图标 import**

在 `App.tsx` 顶部添加：

```typescript
import { BookOpen } from 'lucide-react';
import KnowledgeView from './components/KnowledgeView';
```

**Step 3: 在导航栏中添加知识库入口**

找到导航栏渲染位置，在 MCP 和 History 之间添加：

```typescript
<NavItem
  icon={<BookOpen className="w-5 h-5" />}
  label={t('common:nav.knowledge')}
  isActive={currentView === 'knowledge'}
  isOpen={isSidebarOpen}
  onClick={() => navigateTo('knowledge')}
/>
```

**Step 4: 添加路由渲染**

在主内容区域添加知识库视图：

```typescript
{currentView === 'knowledge' && <KnowledgeView />}
```

**Step 5: 添加国际化文本**

在 `i18n` 文件中添加：

```json
{
  "nav": {
    "knowledge": "知识库"
  }
}
```

**Step 6: 测试导航**

Run: `cd zenops-web && npm run dev`
访问: `http://localhost:5173/#/knowledge`
Expected: 显示知识库页面

**Step 7: Commit**

```bash
git add zenops-web/App.tsx zenops-web/types.ts zenops-web/i18n/
git commit -m "feat(frontend): 添加知识库导航和路由

- 在侧边栏添加知识库入口
- 配置路由渲染 KnowledgeView
- 添加国际化文本"
```

---

## Task 14: 端到端测试

**目标:** 测试完整流程

**Step 1: 启动后端**

Run: `./zenops run`
Expected: 服务启动，知识库 API 可访问

**Step 2: 启动前端**

Run: `cd zenops-web && npm run dev`
Expected: 前端启动在 http://localhost:5173

**Step 3: 测试创建文档**

1. 访问 http://localhost:5173/#/knowledge
2. 点击"新建文档"
3. 填写标题、分类、标签、内容
4. 点击"保存"
5. Expected: 文档出现在列表中

**Step 4: 测试编辑文档**

1. 点击文档的"编辑"按钮
2. 修改内容
3. 保存
4. Expected: 更新成功

**Step 5: 测试删除文档**

1. 点击"删除"按钮
2. 确认
3. Expected: 文档从列表消失

**Step 6: 测试分类筛选**

1. 点击左侧分类
2. Expected: 列表只显示该分类文档

**Step 7: 测试搜索**

1. 输入搜索关键词
2. 按回车
3. Expected: 显示匹配结果

**Step 8: 测试 AI 对话引用**

1. 创建几个测试文档
2. 在 Chat 页面提问相关问题
3. Expected: AI 回复引用知识库内容

**Step 9: 记录问题**

如果有问题，记录下来：
- [ ] 问题描述
- [ ] 复现步骤
- [ ] 预期行为

---

## Task 15: 优化和修复（根据测试结果）

**目标:** 修复测试中发现的问题

**Step 1: 修复编译或运行时错误**

根据测试结果修复具体问题

**Step 2: 样式调整**

调整 UI 细节，确保：
- 响应式布局正常
- 暗黑模式正常
- 交互反馈清晰

**Step 3: 性能优化**

- 检查不必要的重新渲染
- 优化列表渲染
- 添加适当的 loading 状态

**Step 4: 最终 Commit**

```bash
git add .
git commit -m "fix: 修复知识库功能测试中发现的问题

- 修复 XXX 错误
- 优化 YYY 性能
- 调整 ZZZ 样式"
```

---

## Task 16: 文档和总结

**目标:** 更新文档和总结

**Step 1: 更新 README（如果需要）**

说明知识库功能使用方法

**Step 2: 创建用户指南（可选）**

Create: `docs/knowledge-base-guide.md`

简要说明：
- 如何添加文档
- 如何组织分类和标签
- 如何搜索

**Step 3: Commit**

```bash
git add docs/
git commit -m "docs: 添加知识库功能用户指南"
```

---

## 后续优化建议（Phase 2）

Phase 1 完成后，以下功能可在 Phase 2 实现：

1. **Markdown 编辑器升级**
   - 集成 react-markdown-editor-lite
   - 分屏预览
   - 工具栏

2. **搜索增强**
   - 高亮关键词
   - 按标签筛选
   - 相关性排序

3. **Chat 引用显示**
   - 在 ChatView 中显示引用文档
   - 点击查看文档详情

4. **批量操作**
   - 批量启用/禁用
   - 批量删除

5. **响应式优化**
   - 移动端布局
   - 平板端适配

---

**实施计划结束**

预计总时间：**2-3 天（10-12 小时）**

分解为 16 个任务，每个任务 30-60 分钟，包含明确的步骤、验证和提交。
