# 知识库功能设计方案

> 设计日期：2026-01-03
> 设计目标：为 ZenOps 添加完整的知识库管理功能，支持文档管理、智能检索和 AI 对话集成

## 一、设计背景

### 1.1 当前状态

**后端能力（已实现）**：
- ✅ FTS5 全文检索 + 向量检索（混合检索）
- ✅ 支持文档类型：markdown, pdf, url, manual
- ✅ 文档 CRUD 接口（Retriever 层）
- ✅ 自动集成到 AI 对话（StreamHandler 自动检索相关文档）
- ✅ Embedding 自动生成（语义搜索）

**前端状态（缺失）**：
- ❌ 无知识库管理界面
- ❌ 导航栏无知识库入口
- ❌ 用户无法可视化管理文档

### 1.2 设计目标

**核心目标**：
1. 提供可视化的知识库管理界面（Web UI）
2. 支持文档的增删改查、分类管理、标签管理
3. 与 AI 对话深度集成，显示引用来源
4. 为未来扩展（URL 导入、文档爬取）预留接口

**用户价值**：
- 运维人员可以沉淀运维经验（操作手册、故障处理方案）
- AI 对话能引用团队知识库，回答更准确
- 团队知识共享和传承

---

## 二、功能架构设计

### 2.1 整体架构

```
┌─────────────────────────────────────────────────────────┐
│                     前端界面层                           │
│  ┌──────────────┬──────────────┬──────────────┐        │
│  │ 知识库管理   │  文档编辑器   │  搜索/筛选   │        │
│  │  (列表/分类) │  (Markdown)  │  (标签/分类) │        │
│  └──────────────┴──────────────┴──────────────┘        │
└─────────────────────────────────────────────────────────┘
                           ↕ REST API
┌─────────────────────────────────────────────────────────┐
│                     后端服务层                           │
│  ┌──────────────────────────────────────────┐          │
│  │  KnowledgeHandler (REST API)             │          │
│  │  - 文档 CRUD                             │          │
│  │  - 分类/标签管理                         │          │
│  │  - 统计信息                              │          │
│  └──────────────────────────────────────────┘          │
│                      ↕                                  │
│  ┌──────────────────────────────────────────┐          │
│  │  knowledge.Retriever (已实现)            │          │
│  │  - FTS5 全文检索                         │          │
│  │  - 向量检索 (Cosine Similarity)          │          │
│  │  - 混合检索 (RRF 算法)                   │          │
│  └──────────────────────────────────────────┘          │
└─────────────────────────────────────────────────────────┘
                           ↕
┌─────────────────────────────────────────────────────────┐
│                     数据存储层                           │
│  ┌──────────────┬──────────────┬──────────────┐        │
│  │ SQLite       │  FTS5 索引   │  Embedding   │        │
│  │ (文档数据)   │  (全文检索)  │  (向量字段)  │        │
│  └──────────────┴──────────────┴──────────────┘        │
└─────────────────────────────────────────────────────────┘
```

### 2.2 核心功能模块

#### 模块 1: 知识库管理页面

**功能描述**：
- 新增导航栏入口：Knowledge（位于 MCP 和 History 之间）
- 图标：BookOpen 或 Library
- 路由：`#/knowledge`

**子功能**：
1. 文档列表视图（默认视图）
2. 分类导航（左侧边栏）
3. 搜索和筛选（顶部工具栏）
4. 文档编辑器（弹窗/侧边栏）
5. 统计面板（顶部卡片）

#### 模块 2: 文档管理

**文档组织方式**：
- **分类（Category）**：单选，必填
  - 预定义分类：运维文档、API 文档、故障案例、配置模板
  - 支持自定义添加

- **标签（Tags）**：多选，可选
  - 灵活打标签：#nginx #kubernetes #监控 等
  - 输入时自动提示已有标签
  - 支持创建新标签

**文档类型**：
- Markdown（Phase 1 重点支持）
- PDF（预留，Phase 3）
- URL（预留，Phase 3）
- Manual（手动输入）

**文档状态**：
- 启用（Enabled）：AI 对话会检索
- 禁用（Disabled）：不参与检索，但保留数据

#### 模块 3: AI 对话集成

**现有机制（自动生效）**：
```
用户提问 → StreamHandler.ChatStream
         → knowledgeRet.Retrieve(query)  # 自动检索
         → 构建 System Prompt（包含相关文档）
         → LLM 生成回复
```

**前端增强（新增）**：
- Chat 界面显示"引用文档"标记
- 显示文档标题、相关性评分
- 点击可查看文档详情

---

## 三、数据模型设计

### 3.1 数据库表结构

**knowledge_documents 表**（已存在，需扩展）

```sql
CREATE TABLE knowledge_documents (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME,
    updated_at DATETIME,
    deleted_at DATETIME,

    -- 基础信息
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    doc_type VARCHAR(50) DEFAULT 'markdown',  -- markdown/pdf/url/manual

    -- 分类和标签
    category VARCHAR(100),                     -- 单个分类
    tags TEXT,                                 -- JSON 数组: ["tag1", "tag2"]

    -- 元数据
    metadata JSON,                             -- 来源 URL、作者等

    -- 状态
    enabled BOOLEAN DEFAULT TRUE,

    -- 向量检索
    embedding TEXT,                            -- JSON 格式向量
    embedding_model VARCHAR(64)                -- 模型标识
);

-- FTS5 虚拟表（已存在）
CREATE VIRTUAL TABLE knowledge_fts USING fts5(
    title, content,
    content=knowledge_documents,
    content_rowid=id
);
```

**索引优化**：
```sql
CREATE INDEX idx_knowledge_category ON knowledge_documents(category);
CREATE INDEX idx_knowledge_enabled ON knowledge_documents(enabled);
CREATE INDEX idx_knowledge_created ON knowledge_documents(created_at DESC);
```

### 3.2 数据结构（TypeScript）

**前端接口定义**：

```typescript
// 文档模型
interface KnowledgeDocument {
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

  // 检索时返回
  score?: number;  // 相关性评分
}

// 创建文档请求
interface CreateDocumentRequest {
  title: string;
  content: string;
  doc_type?: string;
  category: string;
  tags?: string[];
  metadata?: Record<string, any>;
}

// 统计信息
interface KnowledgeStats {
  total_count: number;
  enabled_count: number;
  categories: Array<{
    name: string;
    count: number;
  }>;
  tags: Array<{
    name: string;
    count: number;
  }>;
}
```

---

## 四、API 接口设计

### 4.1 RESTful API 列表

**基础路径**: `/api/v1/knowledge`

| 方法   | 路径                      | 功能             | 参数说明                                      |
|--------|---------------------------|------------------|-----------------------------------------------|
| GET    | `/documents`              | 获取文档列表     | ?category=&tags=&enabled=&page=&page_size=    |
| GET    | `/documents/:id`          | 获取单个文档     | 路径参数: id                                  |
| POST   | `/documents`              | 创建文档         | Body: CreateDocumentRequest                   |
| PUT    | `/documents/:id`          | 更新文档         | Body: UpdateDocumentRequest                   |
| DELETE | `/documents/:id`          | 删除文档         | 路径参数: id                                  |
| PATCH  | `/documents/:id/toggle`   | 启用/禁用文档    | Body: {enabled: true/false}                   |
| GET    | `/stats`                  | 获取统计信息     | 无                                            |
| GET    | `/categories`             | 获取所有分类     | 返回分类列表及每个分类的文档数                |
| GET    | `/tags`                   | 获取所有标签     | 返回标签列表及使用次数                        |
| POST   | `/search`                 | 搜索文档         | Body: {query: string, category?: string}      |

### 4.2 API 详细说明

**4.2.1 获取文档列表**

```http
GET /api/v1/knowledge/documents?category=运维文档&enabled=true&page=1&page_size=20
```

**响应**：
```json
{
  "code": 200,
  "data": {
    "documents": [
      {
        "id": 1,
        "title": "Nginx 重启指南",
        "content": "...",
        "category": "运维文档",
        "tags": ["nginx", "linux"],
        "enabled": true,
        "created_at": "2026-01-03T10:00:00Z"
      }
    ],
    "total": 45,
    "page": 1,
    "page_size": 20
  }
}
```

**4.2.2 创建文档**

```http
POST /api/v1/knowledge/documents
Content-Type: application/json

{
  "title": "Kubernetes Pod 故障排查",
  "content": "# 排查步骤\n\n1. 查看 Pod 状态...",
  "category": "故障案例",
  "tags": ["kubernetes", "troubleshooting"],
  "metadata": {
    "author": "张三"
  }
}
```

**响应**：
```json
{
  "code": 200,
  "data": {
    "id": 46,
    "title": "Kubernetes Pod 故障排查",
    "created_at": "2026-01-03T14:30:00Z"
  }
}
```

**4.2.3 搜索文档**

```http
POST /api/v1/knowledge/search
Content-Type: application/json

{
  "query": "如何重启 Nginx",
  "category": "运维文档"
}
```

**响应**：
```json
{
  "code": 200,
  "data": {
    "documents": [
      {
        "id": 1,
        "title": "Nginx 重启指南",
        "content": "...",
        "score": 0.89,  // 相关性评分
        "category": "运维文档"
      }
    ],
    "query": "如何重启 Nginx",
    "total": 3
  }
}
```

---

## 五、UI/UX 设计

### 5.1 页面布局

**主页面结构**：

```
┌─────────────────────────────────────────────────────────┐
│  顶部统计卡片区（3个卡片横向排列）                      │
│  📊 总文档: 45   ✅ 已启用: 42   🏷️ 分类: 4           │
└─────────────────────────────────────────────────────────┘

┌──────────────┬──────────────────────────────────────────┐
│ 分类导航     │  工具栏: [🔍 搜索框] [+ 新建] [筛选▾]   │
│ (左侧 250px) ├──────────────────────────────────────────┤
│              │                                          │
│ 📚 全部(45)  │  文档列表表格                            │
│ 📖 运维(15)  │  ┌────────────────────────────────────┐ │
│ 🔧 API(12)   │  │ 标题 | 分类 | 标签 | 状态 | 操作 │ │
│ 🚨 故障(10)  │  ├────────────────────────────────────┤ │
│ ⚙️  配置(8)   │  │ Nginx重启 | 运维 | [nginx][linux]││ │
│              │  │ [●启用] [查看][编辑][删除]         │ │
│              │  ├────────────────────────────────────┤ │
│              │  │ K8s故障 | 故障 | [k8s][debug]    │ │
│              │  │ [○禁用] [查看][编辑][删除]         │ │
│              │  └────────────────────────────────────┘ │
│              │  分页: [1] 2 3 ... 10                   │
└──────────────┴──────────────────────────────────────────┘
```

### 5.2 关键组件设计

#### 5.2.1 文档编辑器

**布局**：全屏弹窗或大侧边栏

```
┌─────────────────────────────────────────────────────────┐
│ 编辑文档                                      [保存] [取消]│
├─────────────────────────────────────────────────────────┤
│ 标题: [                                               ] │
│ 分类: [运维文档 ▾]     标签: [#nginx] [#linux] [+]     │
├──────────────────────┬──────────────────────────────────┤
│ Markdown 编辑器       │  实时预览                        │
│                      │                                  │
│ # 标题               │  标题                            │
│ - 列表项             │  • 列表项                        │
│                      │                                  │
└──────────────────────┴──────────────────────────────────┘
│ 元数据 (可选): 来源 URL [                            ] │
└─────────────────────────────────────────────────────────┘
```

**功能要点**：
- 左右分栏：编辑器 + 实时预览
- Markdown 工具栏：加粗、斜体、代码、链接等
- 标签输入：支持自动补全、回车创建
- 快捷键：Ctrl+S 保存，Esc 关闭

#### 5.2.2 分类导航

```
┌──────────────┐
│ 📚 知识库    │
├──────────────┤
│ 📂 全部 (45) │ ← 默认选中
│ 📖 运维 (15) │
│ 🔧 API (12)  │
│ 🚨 故障 (10) │
│ ⚙️  配置 (8)  │
├──────────────┤
│ [+ 新分类]   │
└──────────────┘
```

**交互**：
- 点击分类筛选文档列表
- 显示每个分类的文档数量
- 支持自定义添加分类

#### 5.2.3 搜索栏

```
┌────────────────────────────────────────┐
│ 🔍 搜索文档（标题、内容、标签）         │
└────────────────────────────────────────┘
```

**功能**：
- 实时搜索（debounce 500ms）
- 搜索范围：标题 + 内容 + 标签
- 高亮显示搜索关键词
- 显示相关性评分

#### 5.2.4 标签显示

```
[#nginx] [#linux] [#systemd]
```

**样式**：
- 圆角标签，不同颜色区分
- 点击标签筛选相关文档
- 鼠标悬停显示使用次数

### 5.3 响应式设计

**桌面端（≥1024px）**：
- 左右布局：分类导航 + 文档列表
- 编辑器：左右分栏（编辑 + 预览）

**平板端（768px - 1024px）**：
- 分类导航改为顶部下拉选择
- 编辑器：左右分栏

**移动端（<768px）**：
- 分类导航：顶部下拉
- 编辑器：上下布局，Tab 切换（编辑/预览）
- 表格改为卡片布局

### 5.4 空状态设计

**首次使用**：

```
┌────────────────────────────────────────┐
│          📚 知识库是空的               │
│                                        │
│  添加您的第一个文档，让 AI 更智能！    │
│                                        │
│       [+ 创建第一个文档]               │
│                                        │
│  💡 提示：                             │
│     • 添加运维手册、API 文档           │
│     • AI 会自动引用相关内容            │
│     • 支持 Markdown 格式               │
└────────────────────────────────────────┘
```

**搜索无结果**：

```
┌────────────────────────────────────────┐
│    🔍 未找到相关文档                   │
│                                        │
│    试试其他关键词或添加新文档          │
└────────────────────────────────────────┘
```

---

## 六、技术实现方案

### 6.1 后端实现（Go）

#### 6.1.1 数据模型扩展

**修改文件**：`internal/model/knowledge_document.go`

```go
type KnowledgeDocument struct {
    ID             uint      `gorm:"primaryKey" json:"id"`
    CreatedAt      time.Time `json:"created_at"`
    UpdatedAt      time.Time `json:"updated_at"`
    DeletedAt      *time.Time `gorm:"index" json:"-"`

    // 基础信息
    Title          string    `json:"title" gorm:"size:255"`
    Content        string    `json:"content" gorm:"type:text"`
    DocType        string    `json:"doc_type" gorm:"size:50"` // markdown/pdf/url/manual

    // 分类和标签
    Category       string    `json:"category" gorm:"size:100;index"`
    Tags           string    `json:"tags" gorm:"type:text"` // JSON 数组: ["tag1", "tag2"]

    // 元数据
    Metadata       string    `json:"metadata" gorm:"type:json"`

    // 状态
    Enabled        bool      `json:"enabled" gorm:"default:true;index"`

    // 向量检索
    Embedding      string    `json:"-" gorm:"type:text"`
    EmbeddingModel string    `json:"-" gorm:"size:64"`
}
```

#### 6.1.2 Handler 层

**新建文件**：`internal/handler/knowledge_handler.go`

```go
package handler

import (
    "net/http"
    "strconv"

    "github.com/eryajf/zenops/internal/knowledge"
    "github.com/gin-gonic/gin"
)

type KnowledgeHandler struct {
    retriever *knowledge.Retriever
}

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
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "code": 200,
        "data": docs,
    })
}

// CreateDocument 创建文档
func (h *KnowledgeHandler) CreateDocument(c *gin.Context) {
    var req knowledge.AddDocumentRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    docID, err := h.retriever.AddDocument(&req)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "code": 200,
        "data": gin.H{"id": docID},
    })
}

// GetDocument 获取单个文档
func (h *KnowledgeHandler) GetDocument(c *gin.Context) {
    id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

    doc, err := h.retriever.GetDocumentByID(uint(id))
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "code": 200,
        "data": doc,
    })
}

// UpdateDocument 更新文档
func (h *KnowledgeHandler) UpdateDocument(c *gin.Context) {
    id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

    var req knowledge.AddDocumentRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    if err := h.retriever.UpdateDocument(uint(id), &req); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"code": 200, "message": "updated"})
}

// DeleteDocument 删除文档
func (h *KnowledgeHandler) DeleteDocument(c *gin.Context) {
    id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

    if err := h.retriever.DeleteDocument(uint(id)); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    if err := h.retriever.ToggleDocument(uint(id), req.Enabled); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"code": 200, "message": "toggled"})
}

// GetStats 获取统计信息
func (h *KnowledgeHandler) GetStats(c *gin.Context) {
    stats, err := h.retriever.GetStats()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "code": 200,
        "data": stats["categories"],
    })
}

// GetTags 获取所有标签
func (h *KnowledgeHandler) GetTags(c *gin.Context) {
    // TODO: 实现标签统计
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
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    docs, err := h.retriever.Retrieve(c.Request.Context(), req.Query)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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

#### 6.1.3 路由注册

**修改文件**：`internal/server/router.go`

```go
// 在 setupRoutes 函数中添加
func setupRoutes(r *gin.Engine, /* ... */) {
    // ... 现有代码 ...

    // 知识库路由
    knowledgeHandler := handler.NewKnowledgeHandler(globalAgent.Orchestrator.knowledgeRet)
    knowledgeHandler.RegisterRoutes(apiV1)
}
```

#### 6.1.4 Service 层增强

**修改文件**：`internal/knowledge/retriever.go`

添加标签相关方法：

```go
// GetAllTags 获取所有标签及使用次数
func (r *Retriever) GetAllTags() (map[string]int, error) {
    var docs []model.KnowledgeDocument
    if err := r.db.Find(&docs).Error; err != nil {
        return nil, err
    }

    tagCount := make(map[string]int)
    for _, doc := range docs {
        if doc.Tags == "" {
            continue
        }

        var tags []string
        if err := json.Unmarshal([]byte(doc.Tags), &tags); err != nil {
            continue
        }

        for _, tag := range tags {
            tagCount[tag]++
        }
    }

    return tagCount, nil
}

// SearchByTags 按标签搜索
func (r *Retriever) SearchByTags(tags []string) ([]*Document, error) {
    // 实现标签搜索逻辑
    // ...
}
```

### 6.2 前端实现（React + TypeScript）

#### 6.2.1 目录结构

```
zenops-web/
├── components/
│   ├── KnowledgeView.tsx           # 主容器组件
│   └── knowledge/
│       ├── DocumentList.tsx        # 文档列表
│       ├── DocumentEditor.tsx      # 文档编辑器
│       ├── CategoryTree.tsx        # 分类导航
│       ├── StatsCards.tsx          # 统计卡片
│       ├── TagSelector.tsx         # 标签选择器
│       └── SearchBar.tsx           # 搜索栏
├── services/
│   └── api.ts                      # API 服务（新增 knowledgeApi）
├── types/
│   └── knowledge.ts                # 知识库类型定义
└── App.tsx                         # 添加路由和导航
```

#### 6.2.2 类型定义

**新建文件**：`zenops-web/types/knowledge.ts`

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

export interface KnowledgeStats {
  total_count: number;
  enabled_count: number;
  categories: string[];
}
```

#### 6.2.3 API Service

**修改文件**：`zenops-web/services/api.ts`

```typescript
// 添加知识库 API
export const knowledgeApi = {
  async listDocuments(params?: {
    category?: string;
    enabled?: boolean;
  }): Promise<KnowledgeDocument[]> {
    const queryParams = new URLSearchParams();
    if (params?.category) queryParams.append('category', params.category);
    if (params?.enabled !== undefined) queryParams.append('enabled', String(params.enabled));

    const response = await fetch(`${API_BASE}/knowledge/documents?${queryParams}`, {
      headers: getAuthHeaders(),
    });
    const data = await response.json();
    return data.data;
  },

  async getDocument(id: number): Promise<KnowledgeDocument> {
    const response = await fetch(`${API_BASE}/knowledge/documents/${id}`, {
      headers: getAuthHeaders(),
    });
    const data = await response.json();
    return data.data;
  },

  async createDocument(doc: CreateDocumentRequest): Promise<number> {
    const response = await fetch(`${API_BASE}/knowledge/documents`, {
      method: 'POST',
      headers: getAuthHeaders(),
      body: JSON.stringify(doc),
    });
    const data = await response.json();
    return data.data.id;
  },

  async updateDocument(id: number, doc: CreateDocumentRequest): Promise<void> {
    await fetch(`${API_BASE}/knowledge/documents/${id}`, {
      method: 'PUT',
      headers: getAuthHeaders(),
      body: JSON.stringify(doc),
    });
  },

  async deleteDocument(id: number): Promise<void> {
    await fetch(`${API_BASE}/knowledge/documents/${id}`, {
      method: 'DELETE',
      headers: getAuthHeaders(),
    });
  },

  async toggleDocument(id: number, enabled: boolean): Promise<void> {
    await fetch(`${API_BASE}/knowledge/documents/${id}/toggle`, {
      method: 'PATCH',
      headers: getAuthHeaders(),
      body: JSON.stringify({ enabled }),
    });
  },

  async getStats(): Promise<KnowledgeStats> {
    const response = await fetch(`${API_BASE}/knowledge/stats`, {
      headers: getAuthHeaders(),
    });
    const data = await response.json();
    return data.data;
  },

  async search(query: string, category?: string): Promise<KnowledgeDocument[]> {
    const response = await fetch(`${API_BASE}/knowledge/search`, {
      method: 'POST',
      headers: getAuthHeaders(),
      body: JSON.stringify({ query, category }),
    });
    const data = await response.json();
    return data.data.documents;
  },
};
```

#### 6.2.4 核心组件

**新建文件**：`zenops-web/components/KnowledgeView.tsx`

```typescript
import React, { useState, useEffect } from 'react';
import { BookOpen, Plus, Search } from 'lucide-react';
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

  useEffect(() => {
    loadDocuments();
  }, [selectedCategory]);

  const loadDocuments = async () => {
    setLoading(true);
    try {
      const docs = await knowledgeApi.listDocuments({
        category: selectedCategory || undefined,
        enabled: undefined,
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

  return (
    <div className="flex flex-col h-full">
      {/* 统计卡片 */}
      <StatsCards />

      <div className="flex flex-1 gap-6 mt-6">
        {/* 左侧分类导航 */}
        <div className="w-64">
          <CategoryTree
            selectedCategory={selectedCategory}
            onSelectCategory={setSelectedCategory}
          />
        </div>

        {/* 右侧文档列表 */}
        <div className="flex-1 flex flex-col">
          {/* 工具栏 */}
          <div className="flex gap-4 mb-4">
            <div className="flex-1 relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
              <input
                type="text"
                placeholder="搜索文档..."
                className="w-full pl-10 pr-4 py-2 border rounded-lg"
              />
            </div>
            <button
              onClick={handleCreate}
              className="flex items-center gap-2 px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700"
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

#### 6.2.5 技术栈选择

**Markdown 编辑器**：
- 库：`react-markdown-editor-lite` + `markdown-it`
- 特性：分屏预览、工具栏、语法高亮
- 安装：`npm install react-markdown-editor-lite markdown-it`

**标签输入**：
- 方案 1：手写（推荐，更轻量）
- 方案 2：`react-tag-autocomplete`

**图标库**：
- 使用现有的 `lucide-react`（已安装）

**状态管理**：
- React Hooks（useState + useEffect）
- 无需 Redux（保持简单）

---

## 七、实施计划

### 7.1 开发阶段

#### Phase 1: 核心功能（MVP）

**时间**：2-3 天

**后端任务**：
1. ✅ 数据模型扩展（Tags 字段）
2. ✅ 创建 `KnowledgeHandler`
3. ✅ 实现 8 个 REST API 接口
4. ✅ 路由注册
5. ✅ 完善 `Retriever` 的标签查询方法

**前端任务**：
1. ✅ 创建 `KnowledgeView` 和子组件
2. ✅ 实现文档列表（表格 + 分页）
3. ✅ 实现文档编辑器（Markdown）
4. ✅ 实现分类导航
5. ✅ 添加导航栏入口
6. ✅ API 服务集成

**验收标准**：
- [x] 用户能添加/编辑/删除文档
- [x] 支持分类和标签管理
- [x] Markdown 编辑器可用
- [x] AI 对话自动引用知识库

---

#### Phase 2: 体验优化

**时间**：1-2 天

**增强功能**：
1. 统计面板（卡片展示）
2. 高级搜索（支持标签筛选）
3. 批量操作（批量启用/禁用）
4. 文档预览模式（只读）
5. 空状态引导页面
6. 响应式布局适配

**Chat 界面增强**：
1. 显示"引用文档"标记
2. 可点击查看文档详情
3. 显示相关性评分

---

#### Phase 3: 高级功能（未来迭代）

**待开发功能**：
1. URL 自动导入（爬虫 + 内容提取）
2. PDF 文件上传和解析
3. 文档版本历史
4. 协作编辑（多人实时编辑）
5. 文档导入/导出（批量）
6. 知识图谱可视化
7. 文档模板功能

---

### 7.2 开发顺序

**第一步：后端基础**（0.5 天）
1. 扩展 `KnowledgeDocument` 模型（Tags 字段）
2. 创建 `KnowledgeHandler`
3. 实现基础 CRUD API（List, Get, Create, Update, Delete）
4. 注册路由

**第二步：前端框架**（0.5 天）
1. 创建目录结构和组件骨架
2. 添加导航栏入口
3. 实现 API Service
4. 搭建主页面布局

**第三步：核心功能**（1 天）
1. 文档列表展示
2. 分类导航
3. 文档编辑器（Markdown）
4. 创建/编辑/删除功能

**第四步：增强功能**（0.5 天）
1. 标签管理
2. 搜索功能
3. 统计面板
4. 启用/禁用切换

**第五步：测试和优化**（0.5 天）
1. 功能测试
2. 响应式适配
3. 样式优化
4. 性能优化

---

### 7.3 数据库迁移

**自动迁移**：

使用 GORM AutoMigrate，系统启动时自动执行：

```go
// 在 internal/db/init.go 中
func AutoMigrate(db *gorm.DB) error {
    return db.AutoMigrate(
        &model.KnowledgeDocument{},
        // ... 其他模型
    )
}
```

**手动 SQL**（如需要）：

```sql
-- 添加 tags 字段（如果不存在）
ALTER TABLE knowledge_documents ADD COLUMN tags TEXT;

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_knowledge_category ON knowledge_documents(category);
CREATE INDEX IF NOT EXISTS idx_knowledge_enabled ON knowledge_documents(enabled);
```

---

## 八、风险和注意事项

### 8.1 技术风险

**风险 1：向量检索性能**
- **问题**：当前是内存计算余弦相似度，文档数多时性能下降
- **阈值**：文档数 < 1000 性能足够
- **缓解**：
  - 短期：限制检索文档数（max_results=3）
  - 中期：添加缓存（Redis）
  - 长期：引入向量数据库（Milvus/Qdrant）

**风险 2：Markdown 编辑器兼容性**
- **问题**：不同浏览器渲染差异
- **缓解**：
  - 使用成熟的 Markdown 库（markdown-it）
  - 充分测试（Chrome, Safari, Firefox）
  - 提供预览功能

**风险 3：大文档处理**
- **问题**：单个文档过大（如 API 文档几万字）
- **缓解**：
  - 前端：限制内容长度（提示拆分）
  - 后端：分段存储（未来优化）
  - 检索：只索引摘要或关键部分

### 8.2 产品风险

**风险 1：标签混乱**
- **问题**：用户随意创建标签，导致标签爆炸
- **缓解**：
  - 限制每个文档最多 10 个标签
  - 输入时提示已有标签（自动补全）
  - 提供标签管理功能（合并/删除）

**风险 2：分类不合理**
- **问题**：预定义分类不符合用户习惯
- **缓解**：
  - 支持自定义分类
  - 提供默认分类作为参考
  - 允许重命名分类

**风险 3：知识库滥用**
- **问题**：用户添加大量低质量内容
- **缓解**：
  - 引导用户添加高质量文档
  - 提供文档质量评分（未来）
  - 管理员可批量清理

### 8.3 安全风险

**风险 1：XSS 攻击**
- **问题**：Markdown 内容包含恶意脚本
- **缓解**：
  - 使用 `markdown-it` 自带的 sanitize
  - 禁止 HTML 内联（仅允许 Markdown）
  - CSP（Content Security Policy）

**风险 2：权限控制**
- **问题**：Phase 1 无权限控制，所有用户都能编辑
- **缓解**：
  - Phase 1：仅内部团队使用
  - Phase 2：添加角色（管理员/普通用户）
  - Phase 3：文档级权限（创建者/查看者）

---

## 九、测试计划

### 9.1 单元测试

**后端测试**：

```go
// internal/handler/knowledge_handler_test.go
func TestCreateDocument(t *testing.T) {
    // 测试文档创建
}

func TestSearchDocuments(t *testing.T) {
    // 测试搜索功能（FTS5 + 向量）
}
```

**前端测试**：

```typescript
// components/knowledge/DocumentEditor.test.tsx
describe('DocumentEditor', () => {
  it('should save document', async () => {
    // 测试保存功能
  });
});
```

### 9.2 集成测试

**测试场景**：
1. 创建文档 → 检索 → AI 对话引用
2. 编辑文档 → 重新生成 Embedding → 检索验证
3. 禁用文档 → AI 对话不引用
4. 删除文档 → FTS5 索引同步删除

### 9.3 用户测试

**测试用户**：3-5 名内部用户

**测试任务**：
1. 添加 5 个运维文档
2. 使用 AI 对话，观察是否引用知识库
3. 搜索功能测试
4. 编辑和删除文档
5. 反馈 UI/UX 问题

---

## 十、未来扩展

### 10.1 自动化导入

**URL 爬取**：
- 输入文档 URL，自动抓取内容
- 支持 HTML → Markdown 转换
- 定期更新（Cron Job）

**PDF 解析**：
- 支持 PDF 上传
- 提取文本和结构
- 生成 Markdown

### 10.2 协作功能

**版本历史**：
- 记录每次编辑
- 支持版本对比
- 回滚到历史版本

**多人编辑**：
- 实时协作（WebSocket）
- 冲突检测
- 锁定机制

### 10.3 智能增强

**知识图谱**：
- 自动提取实体和关系
- 可视化知识网络
- 智能推荐相关文档

**质量评分**：
- 根据引用次数评分
- 用户反馈（有用/无用）
- AI 自动评估内容质量

---

## 十一、总结

### 11.1 设计亮点

1. **渐进式实施**：MVP 快速上线，后续迭代优化
2. **混合检索**：FTS5 + 向量检索，兼顾速度和准确性
3. **AI 深度集成**：知识库自动融入对话上下文
4. **灵活组织**：分类 + 标签，适应不同场景
5. **可扩展性**：预留 URL 导入、PDF 解析等扩展接口

### 11.2 交付物

**Phase 1 交付**：
- ✅ 知识库管理界面（完整 CRUD）
- ✅ Markdown 编辑器
- ✅ 分类和标签管理
- ✅ AI 对话集成
- ✅ 搜索功能

**Phase 2 交付**：
- ✅ 统计面板
- ✅ 高级筛选
- ✅ Chat 引用显示
- ✅ 响应式布局

### 11.3 成功指标

**功能指标**：
- 用户能在 5 分钟内添加第一个文档
- 搜索响应时间 < 500ms（1000 文档以内）
- AI 对话引用准确率 > 80%

**用户满意度**：
- 界面易用性评分 > 4/5
- 知识库使用率 > 60%（活跃用户）
- 用户愿意持续添加内容

---

**文档结束**
