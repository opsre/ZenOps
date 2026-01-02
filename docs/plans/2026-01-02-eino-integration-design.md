# ZenOps Eino 框架集成设计方案

**文档版本**: v1.0
**创建日期**: 2026-01-02
**作者**: Claude
**状态**: 待审核

---

## 1. 背景和目标

### 1.1 当前问题

ZenOps 现有的 LLM 对话能力基于简单的请求-响应模式，存在以下局限性：

1. **处理复杂问题能力不足**
   - 需要多轮 MCP 调用才能获取足够信息
   - 无法跨多个 MCP Server 进行智能编排
   - 缺乏自动推理和规划能力

2. **缺乏上下文记忆**
   - 无法利用用户的历史对话信息
   - 每次查询都是独立的，无法进行追问式交互
   - 高频问题重复处理，效率低下

3. **知识库能力缺失**
   - 无法让用户配置常用资料信息
   - 不支持文档解析和知识检索
   - 回答准确性依赖 LLM 本身知识

### 1.2 改造目标

引入 **Eino 框架**，实现以下能力提升：

✅ **智能编排**: 支持多步骤、跨 MCP Server 的复杂任务自动推理和执行
✅ **记忆管理**: 基于 SQLite + Redis 的会话记忆和用户上下文管理
✅ **知识增强**: 支持文档解析、向量检索和知识库配置
✅ **流式优化**: 保持现有钉钉/飞书/企微流式输出能力
✅ **代码简化**: 用 Eino 统一抽象替换分散的 LLM 调用逻辑

---

## 2. Eino 框架调研

### 2.1 框架概述

[Eino](https://github.com/cloudwego/eino) 是字节跳动开源的 Go 语言 LLM 应用开发框架，已在抖音、豆包等产品中经过生产验证。

**核心特性**:
- 强类型、符合 Go 语言习惯的 API 设计
- 丰富的组件抽象（ChatModel、Tool、Retriever、Lambda 等）
- 强大的编排能力（Chain、Graph、Workflow）
- 内置 ReAct Agent 实现
- 原生支持 MCP 协议集成

### 2.2 核心组件

| 组件 | 说明 | 在 ZenOps 中的应用 |
|------|------|-------------------|
| **ChatModel** | LLM 接口抽象 | 替换现有 `internal/llm/openai.go` |
| **Tool** | 工具调用接口 | 将 MCP Server 适配为 Eino Tool |
| **Retriever** | 文档检索接口 | 实现知识库检索 |
| **Graph** | 有向图编排 | 实现复杂的多步骤任务流程 |
| **ChatMemory** | 会话记忆 | 基于 SQLite + Redis 实现 |

### 2.3 MCP 集成

Eino 通过适配器模式支持 MCP 协议：
- 使用 `github.com/mark3labs/mcp-go` SDK（与 ZenOps 现有依赖一致）
- 支持 stdio、SSE、streamableHttp 三种传输协议
- 可将 MCP Server 的工具直接包装为 Eino Tool

**参考资料**:
- [Eino MCP Tool 集成文档](https://cloudwego.cn/docs/eino/ecosystem_integration/tool/tool_mcp/)
- [MCP Go SDK](https://github.com/mark3labs/mcp-go)

---

## 3. 整体架构设计

### 3.1 架构图

```
┌─────────────────────────────────────────────────────────┐
│                    用户请求入口                           │
│        (钉钉/飞书/企微/Web/CLI/HTTP API)                 │
└─────────────────┬───────────────────────────────────────┘
                  │
┌─────────────────▼───────────────────────────────────────┐
│              Eino Agent Orchestrator                     │
│         (Graph 编排 + ReAct 推理引擎)                    │
├──────────────────────────────────────────────────────────┤
│  ┌────────────┐  ┌──────────────┐  ┌─────────────────┐  │
│  │  Memory    │  │  Knowledge   │  │   MCP Tools     │  │
│  │  Manager   │  │  Retriever   │  │   Adapter       │  │
│  └─────┬──────┘  └──────┬───────┘  └────────┬────────┘  │
│        │                │                   │            │
│  ┌─────▼────────────────▼───────────────────▼─────────┐  │
│  │           Eino ChatModel (OpenAI 兼容)             │  │
│  └──────────────────────────────────────────────────────┘  │
└──────────────────┬──────────────────┬───────────────────┘
                   │                  │
       ┌───────────▼────────┐   ┌────▼──────────────┐
       │   Redis (L1 Cache) │   │  SQLite (L2 DB)   │
       │  - 会话状态         │   │  - 用户上下文      │
       │  - QA 缓存          │   │  - 对话历史        │
       │  - 活跃会话         │   │  - 知识库          │
       └────────────────────┘   └───────────────────┘
                   │
       ┌───────────▼────────────────────────────────┐
       │        MCP Client Manager                  │
       │   (复用现有 internal/mcpclient)            │
       │   - stdio/SSE/streamableHttp              │
       └────────────────────────────────────────────┘
```

### 3.2 数据流程

**简单问答流程**:
```
用户提问
  → 检查 QA 缓存 (Redis)
     ├─ 命中 → 直接返回
     └─ 未命中 ↓
  → 加载对话历史 (Redis/SQLite)
  → 加载用户上下文 (SQLite)
  → 检索知识库 (SQLite FTS5)
  → Eino Graph 编排
     → ChatModel 推理
     → 判断是否需要工具调用
        ├─ 不需要 → 直接回答
        └─ 需要 → 调用 MCP Tools
           → 返回结果给 ChatModel
           → (可能多轮循环)
  → 保存到记忆 (Redis + SQLite)
  → 更新 QA 缓存
  → 返回用户
```

**复杂任务流程示例**（跨 MCP Server）:
```
用户: "对比阿里云和腾讯云的 CVM 数量，生成报告"

Eino Graph 自动编排:
  1. 调用 MCP Tool: aliyun_list_ecs
  2. 调用 MCP Tool: tencent_list_cvm
  3. LLM 汇总分析两者数据
  4. 生成对比报告
  5. 返回给用户
```

### 3.3 存储策略

#### SQLite (持久化存储)
- **现有表**: `chat_logs`, `conversations`, `users` 等（保留）
- **新增表**: `user_contexts`, `qa_cache`, `knowledge_documents`, `knowledge_fts`

#### Redis (缓存层)
- **Key 设计**:
  - `conv:{conversation_id}:history` → 对话历史 (List, TTL=1h)
  - `user:{username}:context` → 用户上下文 (Hash)
  - `qa:{question_hash}` → 问答缓存 (String, TTL=1h)
  - `session:{username}:active` → 当前活跃会话 ID (String)

---

## 4. 详细模块设计

### 4.1 Memory Manager（记忆管理）

**职责**: 管理会话历史、用户上下文和 QA 缓存

**接口定义**:
```go
// internal/memory/manager.go

type MemoryManager struct {
    redis    *redis.Client
    db       *gorm.DB
    ttl      time.Duration
}

// 核心方法
func (m *MemoryManager) GetConversationHistory(conversationID uint, limit int) ([]*model.ChatLog, error)
func (m *MemoryManager) SaveMessage(conversationID uint, chatType int, content string) error
func (m *MemoryManager) GetUserContext(username string) (*UserContext, error)
func (m *MemoryManager) UpdateUserContext(username, key, value string) error
func (m *MemoryManager) GetCachedAnswer(username, question string) (string, bool, error)
func (m *MemoryManager) UpdateQACache(username, question, answer string) error
```

**新增数据库表**:

```sql
-- 用户上下文表（扩展用户偏好）
CREATE TABLE user_contexts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME,
    updated_at DATETIME,
    username TEXT NOT NULL,
    context_key TEXT NOT NULL,      -- 如: "favorite_region", "default_vpc"
    context_value TEXT,              -- JSON 格式存储值
    context_type TEXT DEFAULT 'user', -- user/system/auto_learned
    UNIQUE(username, context_key)
);
CREATE INDEX idx_user_contexts_username ON user_contexts(username);

-- 问答缓存表（语义缓存）
CREATE TABLE qa_cache (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME,
    updated_at DATETIME,
    question_hash TEXT NOT NULL,     -- 问题的语义哈希
    question TEXT NOT NULL,
    answer TEXT,
    username TEXT,                   -- 可选：用户级别缓存
    hit_count INTEGER DEFAULT 1,
    last_hit_at DATETIME,
    UNIQUE(question_hash, username)
);
CREATE INDEX idx_qa_cache_hash ON qa_cache(question_hash);
CREATE INDEX idx_qa_cache_hits ON qa_cache(hit_count DESC);

-- 为 chat_logs 添加全文索引（可选，用于历史搜索）
CREATE VIRTUAL TABLE chat_logs_fts USING fts5(
    content,
    content='chat_logs',
    content_rowid='id'
);
```

**工作流程**:
1. **读取历史**: 先查 Redis `conv:{id}:history`，未命中则从 `chat_logs` 表加载并回填
2. **用户上下文**: 从 `user_contexts` 表读取，注入到 System Prompt
3. **QA 缓存**: 对问题计算哈希，查询 `qa_cache` 表，命中则返回并更新 `hit_count`

---

### 4.2 Knowledge Retriever（知识检索）

**职责**: 文档解析、存储和智能检索

**接口定义**:
```go
// internal/knowledge/retriever.go

type KnowledgeRetriever struct {
    db          *gorm.DB
    embedder    *Embedder         // 文本向量化（可选）
    useVector   bool              // 是否启用向量检索
}

// 实现 Eino Retriever 接口
func (k *KnowledgeRetriever) Retrieve(ctx context.Context, query string, opts ...Option) ([]*Document, error)

// 文档管理
func (k *KnowledgeRetriever) AddDocument(doc *Document) error
func (k *KnowledgeRetriever) DeleteDocument(docID int) error
func (k *KnowledgeRetriever) ListDocuments(category string) ([]*Document, error)
```

**新增数据库表**:

```sql
-- 知识库文档表
CREATE TABLE knowledge_documents (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME,
    updated_at DATETIME,
    doc_type TEXT,              -- 'markdown', 'pdf', 'url', 'manual'
    title TEXT,
    content TEXT,
    metadata JSON,              -- 存储来源、作者等元信息
    enabled INTEGER DEFAULT 1,
    category TEXT               -- 分类：运维文档、API文档等
);
CREATE INDEX idx_knowledge_category ON knowledge_documents(category, enabled);

-- 文档全文索引（SQLite FTS5）
CREATE VIRTUAL TABLE knowledge_fts USING fts5(
    title,
    content,
    content='knowledge_documents',
    content_rowid='id',
    tokenize='porter unicode61'  -- 支持中英文分词
);

-- 可选：向量表（如果启用 sqlite-vec）
-- 需要 sqlite-vec 扩展支持
CREATE VIRTUAL TABLE IF NOT EXISTS knowledge_vectors USING vec0(
    doc_id INTEGER PRIMARY KEY,
    embedding FLOAT[1536]       -- OpenAI embedding 维度
);
```

**检索策略**:
1. **关键词检索**（FTS5）: 快速全文搜索，适合精确匹配
2. **向量检索**（可选）: 语义相似度搜索，适合模糊查询
3. **混合检索**: RRF (Reciprocal Rank Fusion) 算法合并结果

**文档导入方式**:
- 管理员在 Web 界面上传/配置文档
- 从高频 QA 缓存中自动提取知识（`hit_count > 阈值`）
- 定期抓取外部运维文档（Confluence、Wiki 等）

---

### 4.3 Agent Orchestrator（智能编排）

**职责**: 核心编排引擎，使用 Eino Graph 管理复杂对话流程

**接口定义**:
```go
// internal/agent/orchestrator.go

type AgentOrchestrator struct {
    chatModel     chatmodel.ChatModel      // Eino ChatModel
    memoryMgr     *memory.MemoryManager
    knowledgeRet  *knowledge.KnowledgeRetriever
    mcpServer     *imcp.Server             // 复用现有 MCP Server
    toolsNode     *compose.ToolsNode
}

// 构建 Eino Graph
func (a *AgentOrchestrator) BuildGraph() *compose.Graph

// 执行对话
func (a *AgentOrchestrator) Execute(ctx context.Context, req *ChatRequest) (*ChatResponse, error)

// 流式对话
func (a *AgentOrchestrator) Stream(ctx context.Context, req *ChatRequest) (<-chan string, error)
```

**Graph 定义**:

```go
func (a *AgentOrchestrator) BuildGraph() *compose.Graph {
    builder := compose.NewGraphBuilder[map[string]any]()

    // 节点定义
    builder.AddNode("load_memory", a.loadMemoryNode)           // 加载历史
    builder.AddNode("load_context", a.loadContextNode)         // 加载用户上下文
    builder.AddNode("retrieve_knowledge", a.retrieveKnowledgeNode) // 检索知识库
    builder.AddNode("llm", a.llmNode)                          // LLM 推理
    builder.AddNode("tools", a.toolsNode)                      // MCP 工具调用
    builder.AddNode("save_memory", a.saveMemoryNode)           // 保存历史

    // 边定义（流程编排）
    builder.AddEdge(START, "load_memory")
    builder.AddEdge("load_memory", "load_context")
    builder.AddEdge("load_context", "retrieve_knowledge")
    builder.AddEdge("retrieve_knowledge", "llm")

    // 条件分支：是否需要调用工具
    builder.AddConditionalEdge("llm", a.shouldCallTools,
        map[string]string{
            "tools":  "tools",        // 需要调用工具
            "finish": "save_memory",  // 直接结束
        })

    builder.AddEdge("tools", "llm")         // 工具结果回到 LLM（支持多轮）
    builder.AddEdge("save_memory", END)

    return builder.Compile()
}

// 条件路由：判断是否需要调用工具
func (a *AgentOrchestrator) shouldCallTools(state map[string]any) string {
    response := state["llm_response"].(ChatResponse)
    if len(response.ToolCalls) > 0 {
        return "tools"
    }
    return "finish"
}
```

**关键节点实现**:

```go
// 1. 加载记忆节点
func (a *AgentOrchestrator) loadMemoryNode(ctx context.Context, state map[string]any) (map[string]any, error) {
    conversationID := state["conversation_id"].(uint)
    history, err := a.memoryMgr.GetConversationHistory(conversationID, 10)
    if err != nil {
        return state, err
    }
    state["history"] = history
    return state, nil
}

// 2. 加载用户上下文节点
func (a *AgentOrchestrator) loadContextNode(ctx context.Context, state map[string]any) (map[string]any, error) {
    username := state["username"].(string)
    userCtx, err := a.memoryMgr.GetUserContext(username)
    if err != nil {
        return state, err
    }
    state["user_context"] = userCtx
    return state, nil
}

// 3. 检索知识库节点
func (a *AgentOrchestrator) retrieveKnowledgeNode(ctx context.Context, state map[string]any) (map[string]any, error) {
    userMessage := state["user_message"].(string)
    docs, err := a.knowledgeRet.Retrieve(ctx, userMessage)
    if err != nil {
        return state, err
    }
    state["knowledge_docs"] = docs
    return state, nil
}

// 4. LLM 推理节点
func (a *AgentOrchestrator) llmNode(ctx context.Context, state map[string]any) (map[string]any, error) {
    // 构建完整的 Prompt（包含历史、上下文、知识库）
    messages := a.buildMessages(state)

    // 调用 Eino ChatModel
    resp, err := a.chatModel.Generate(ctx, messages, chatmodel.WithTools(a.getTools()))
    if err != nil {
        return state, err
    }

    state["llm_response"] = resp
    return state, nil
}

// 5. 工具调用节点（使用 Eino ToolsNode）
func (a *AgentOrchestrator) buildToolsNode() *compose.ToolsNode {
    return compose.NewToolsNode(a.buildMCPTools())
}

// 6. 保存记忆节点
func (a *AgentOrchestrator) saveMemoryNode(ctx context.Context, state map[string]any) (map[string]any, error) {
    conversationID := state["conversation_id"].(uint)
    userMessage := state["user_message"].(string)
    aiResponse := state["llm_response"].(ChatResponse)

    // 保存用户消息
    _ = a.memoryMgr.SaveMessage(conversationID, 1, userMessage)

    // 保存 AI 回复
    _ = a.memoryMgr.SaveMessage(conversationID, 2, aiResponse.Content)

    // 更新 QA 缓存
    username := state["username"].(string)
    _ = a.memoryMgr.UpdateQACache(username, userMessage, aiResponse.Content)

    return state, nil
}
```

**MCP Tools 适配器**:

```go
// internal/agent/mcp_adapter.go

type MCPToolAdapter struct {
    name      string
    desc      string
    schema    any
    mcpServer *imcp.Server
}

// 实现 Eino Tool 接口
func (t *MCPToolAdapter) Info(ctx context.Context) (*schema.ToolInfo, error) {
    return &schema.ToolInfo{
        Name:        t.name,
        Description: t.desc,
        ParamsOneOf: t.schema,
    }, nil
}

func (t *MCPToolAdapter) InvokableRun(ctx context.Context, args string, opts ...Option) (string, error) {
    // 解析参数
    var params map[string]any
    if err := json.Unmarshal([]byte(args), &params); err != nil {
        return "", err
    }

    // 调用 MCP Server
    result, err := t.mcpServer.CallTool(ctx, t.name, params)
    if err != nil {
        return "", err
    }

    // 提取文本结果
    if len(result.Content) > 0 {
        if textContent, ok := result.Content[0].(mcp.TextContent); ok {
            return textContent.Text, nil
        }
    }

    return "", nil
}

// 从 MCP Server 构建 Eino Tools
func (a *AgentOrchestrator) buildMCPTools() []tool.Tool {
    var tools []tool.Tool

    mcpTools, _ := a.mcpServer.ListEnabledTools(context.Background())

    for _, mcpTool := range mcpTools.Tools {
        adapter := &MCPToolAdapter{
            name:      mcpTool.Name,
            desc:      mcpTool.Description,
            schema:    mcpTool.InputSchema,
            mcpServer: a.mcpServer,
        }
        tools = append(tools, adapter)
    }

    return tools
}
```

---

### 4.4 Stream Handler（流式输出）

**职责**: 适配 Eino 流式输出到现有 IM 接口

**接口定义**:
```go
// internal/agent/stream_handler.go

type StreamHandler struct {
    orchestrator *AgentOrchestrator
}

// 流式对话（兼容现有接口）
func (s *StreamHandler) ChatStream(ctx context.Context, req *ChatRequest) (<-chan string, error) {
    responseCh := make(chan string, 100)

    go func() {
        defer close(responseCh)

        // 构建初始状态
        state := map[string]any{
            "user_message":    req.Message,
            "username":        req.Username,
            "conversation_id": req.ConversationID,
        }

        // 执行 Eino Graph（带回调）
        graph := s.orchestrator.BuildGraph()
        callbacks := &StreamCallbacks{responseCh: responseCh}

        err := graph.Stream(ctx, state, compose.WithCallbacks(callbacks))
        if err != nil {
            responseCh <- fmt.Sprintf("❌ 执行失败: %v", err)
        }
    }()

    return responseCh, nil
}
```

**流式回调**:
```go
type StreamCallbacks struct {
    responseCh chan<- string
}

func (c *StreamCallbacks) OnChatModelStream(ctx context.Context, delta string) {
    c.responseCh <- delta  // 实时推送 LLM 输出
}

func (c *StreamCallbacks) OnToolStart(ctx context.Context, toolName string) {
    c.responseCh <- fmt.Sprintf("\n> 🔧 调用工具: **%s**\n", toolName)
}

func (c *StreamCallbacks) OnToolEnd(ctx context.Context, toolName string, result any) {
    c.responseCh <- "✅ 工具执行完成\n\n"
}
```

**集成到现有 Handler**:
```go
// internal/server/chat_handler.go (改造后)

func (h *ChatHandler) StreamChat(c *gin.Context) {
    // 参数解析（保持不变）
    // ...

    // 使用 Eino Agent
    streamHandler := agent.NewStreamHandler(h.orchestrator)
    responseCh, err := streamHandler.ChatStream(ctx, &agent.ChatRequest{
        Username:       username,
        Message:        req.Message,
        ConversationID: conversationID,
    })

    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    // SSE 流式输出（保持不变）
    c.Header("Content-Type", "text/event-stream")
    c.Header("Cache-Control", "no-cache")
    c.Header("Connection", "keep-alive")

    for chunk := range responseCh {
        c.SSEvent("message", chunk)
        c.Writer.Flush()
    }
}
```

---

## 5. 迁移和代码清理计划

### 5.1 模块清单

#### 📦 保留的模块（复用）
```
internal/
├── config/          ✅ 保留（配置管理）
├── database/        ✅ 保留（数据库连接）
├── model/           ✅ 保留（所有现有表模型，新增 UserContext、QACache、KnowledgeDocument）
├── mcpclient/       ✅ 保留（MCP 客户端管理）
├── imcp/            ✅ 保留（MCP Server 实现）
├── dingtalk/        ✅ 保留（钉钉集成）
├── feishu/          ✅ 保留（飞书集成）
├── wecom/           ✅ 保留（企微集成）
├── provider/        ✅ 保留（云厂商 Provider）
└── service/         ✅ 保留（现有业务逻辑）
```

#### 🗑️ 删除的模块（被 Eino 替换）
```
internal/
└── llm/
    ├── client.go           ❌ 删除（Eino ChatModel 替代）
    ├── openai.go           ❌ 删除（Eino 提供 OpenAI 实现）
    └── 所有相关调用逻辑    ❌ 删除
```

#### 🔄 改造的文件（部分重写）
```
internal/server/
├── chat_handler.go         🔄 改造：使用 agent.StreamHandler
├── dingtalk_stream.go      🔄 改造：使用 agent.StreamHandler
├── feishu_stream.go        🔄 改造：使用 agent.StreamHandler
└── (其他 handler 保持不变)

internal/dingtalk/
├── handler.go              🔄 改造：调用 agent.StreamHandler
└── (其他文件保持不变)

internal/feishu/
├── handler.go              🔄 改造：调用 agent.StreamHandler
└── (其他文件保持不变)

internal/wecom/
├── handler.go              🔄 改造：调用 agent.StreamHandler
└── (其他文件保持不变)
```

#### ✨ 新增的模块
```
internal/
├── agent/                  ✨ 新增（Eino 编排）
│   ├── orchestrator.go     # Graph 编排核心
│   ├── stream_handler.go   # 流式处理适配
│   ├── mcp_adapter.go      # MCP Tool 适配器
│   └── types.go            # 类型定义
├── memory/                 ✨ 新增（记忆管理）
│   ├── manager.go          # Memory Manager 核心
│   ├── redis_cache.go      # Redis 缓存层
│   └── types.go            # 类型定义
└── knowledge/              ✨ 新增（知识检索）
    ├── retriever.go        # Knowledge Retriever 核心
    ├── document.go         # 文档管理
    ├── fts_search.go       # FTS5 全文检索
    └── types.go            # 类型定义

internal/model/
├── user_context.go         ✨ 新增（用户上下文模型）
├── qa_cache.go             ✨ 新增（QA 缓存模型）
└── knowledge_document.go   ✨ 新增（知识库文档模型）
```

### 5.2 迁移步骤

#### 阶段一：基础设施准备（不影响现有功能）

**目标**: 建立新的基础设施，但不改变现有代码

**任务清单**:
1. ✅ 添加 Eino 依赖到 `go.mod`
   ```bash
   go get github.com/cloudwego/eino@latest
   go get github.com/cloudwego/eino-ext@latest
   ```

2. ✅ 添加 Redis 客户端依赖
   ```bash
   go get github.com/redis/go-redis/v9
   ```

3. ✅ 创建新的数据库表
   - 执行 migration: `user_contexts`, `qa_cache`, `knowledge_documents`
   - 创建 FTS5 索引

4. ✅ 实现 Memory Manager
   - `internal/memory/manager.go`
   - 单元测试

5. ✅ 实现 Knowledge Retriever
   - `internal/knowledge/retriever.go`
   - 单元测试

6. ✅ 实现 MCP Tool Adapter
   - `internal/agent/mcp_adapter.go`
   - 集成测试

**验证标准**:
- 所有新模块有单元测试覆盖
- 现有功能不受影响，可正常运行

---

#### 阶段二：Eino Agent 实现（并行开发）

**目标**: 实现 Eino Agent Orchestrator，但暂不接入生产

**任务清单**:
1. ✅ 实现 Agent Orchestrator
   - `internal/agent/orchestrator.go`
   - 构建 Eino Graph

2. ✅ 实现 Stream Handler
   - `internal/agent/stream_handler.go`
   - 流式回调

3. ✅ 集成测试
   - 使用测试数据验证完整流程
   - 对比新旧实现的输出一致性

**验证标准**:
- Agent 可以独立运行，输出符合预期
- 流式输出与现有实现行为一致

---

#### 阶段三：逐步切换（灰度发布）

**目标**: 逐个接口切换到新实现，确保平滑过渡

**任务清单**:
1. ✅ 切换 Web Chat API
   - 修改 `internal/server/chat_handler.go`
   - A/B 测试：通过配置开关控制新旧实现
   - 验证功能正常

2. ✅ 切换钉钉机器人
   - 修改 `internal/dingtalk/handler.go`
   - 灰度测试
   - 验证流式输出正常

3. ✅ 切换飞书机器人
   - 修改 `internal/feishu/handler.go`
   - 灰度测试

4. ✅ 切换企微机器人
   - 修改 `internal/wecom/handler.go`
   - 灰度测试

**验证标准**:
- 每个接口切换后，进行充分测试
- 用户无感知，功能保持一致或增强

---

#### 阶段四：清理旧代码

**目标**: 删除被替换的代码，清理依赖

**任务清单**:
1. ✅ 删除 `internal/llm/` 整个目录
2. ✅ 清理未使用的导入
3. ✅ 更新 `go.mod`，移除不再需要的依赖
   ```bash
   go mod tidy
   ```
4. ✅ 更新相关文档

**验证标准**:
- 编译通过，无未使用的导入
- 所有测试通过
- 文档更新完整

---

### 5.3 风险控制

#### 回滚策略
- **配置开关**: 使用 Feature Flag 控制新旧实现
  ```yaml
  # config.yaml
  agent:
    use_eino: true  # false 时使用旧实现
  ```

- **数据备份**: 在执行 migration 前备份数据库
  ```bash
  cp data/zenops.db data/zenops.db.backup
  ```

#### 兼容性保证
- 新表不影响现有表结构
- Redis 为可选依赖，未配置时降级到纯 SQLite 模式
- MCP Server 接口保持不变

---

## 6. 功能增强点

### 6.1 智能上下文注入

**用户场景**:
> 用户经常查询某个地域（如 "华北2"）的资源，系统自动记住用户偏好

**实现方式**:
```go
// 自动学习用户偏好
func (m *MemoryManager) LearnUserPreference(username, key, value string) {
    // 从对话中提取关键信息，保存到 user_contexts
    m.UpdateUserContext(username, key, value)
}

// 注入到 System Prompt
func (a *AgentOrchestrator) buildSystemPrompt(userCtx *UserContext) string {
    prompt := "你是一个智能运维助手。\n\n"

    if userCtx.FavoriteRegion != "" {
        prompt += fmt.Sprintf("用户常用地域: %s\n", userCtx.FavoriteRegion)
    }

    if userCtx.DefaultVPC != "" {
        prompt += fmt.Sprintf("用户默认 VPC: %s\n", userCtx.DefaultVPC)
    }

    return prompt
}
```

### 6.2 智能问答缓存

**用户场景**:
> 多个用户问 "如何查看 ECS 实例?"，第一次 LLM 推理，后续直接返回缓存

**实现方式**:
```go
func (m *MemoryManager) GetCachedAnswer(username, question string) (string, bool, error) {
    // 1. 计算问题的语义哈希（简单实现：使用文本哈希）
    hash := calculateHash(question)

    // 2. 先查 Redis
    if answer, ok := m.getCachedFromRedis(hash); ok {
        return answer, true, nil
    }

    // 3. 再查 SQLite
    var cache model.QACache
    err := m.db.Where("question_hash = ?", hash).First(&cache).Error
    if err == nil {
        // 更新命中统计
        m.db.Model(&cache).Updates(map[string]any{
            "hit_count":    gorm.Expr("hit_count + 1"),
            "last_hit_at":  time.Now(),
        })

        // 回填 Redis
        m.setCachedToRedis(hash, cache.Answer)

        return cache.Answer, true, nil
    }

    return "", false, nil
}
```

### 6.3 文档知识库

**用户场景**:
> 管理员上传运维手册，用户提问时自动检索相关内容

**实现方式**:
```go
// 1. 文档上传接口
POST /api/knowledge/documents
{
    "title": "ECS 实例管理手册",
    "content": "...",
    "category": "运维文档",
    "doc_type": "markdown"
}

// 2. 检索流程
func (k *KnowledgeRetriever) Retrieve(ctx context.Context, query string) ([]*Document, error) {
    // FTS5 全文检索
    var docs []*model.KnowledgeDocument
    k.db.Raw(`
        SELECT d.*
        FROM knowledge_documents d
        JOIN knowledge_fts f ON d.id = f.rowid
        WHERE knowledge_fts MATCH ?
        AND d.enabled = 1
        ORDER BY rank
        LIMIT 3
    `, query).Scan(&docs)

    return docs, nil
}

// 3. 注入到 LLM Context
func (a *AgentOrchestrator) buildMessagesWithKnowledge(state map[string]any) []Message {
    messages := []Message{
        {Role: "system", Content: "你是智能运维助手"},
    }

    // 注入知识库内容
    if docs, ok := state["knowledge_docs"].([]*Document); ok && len(docs) > 0 {
        knowledgeText := "参考资料:\n"
        for _, doc := range docs {
            knowledgeText += fmt.Sprintf("- %s: %s\n", doc.Title, doc.Content)
        }
        messages = append(messages, Message{
            Role:    "system",
            Content: knowledgeText,
        })
    }

    // 其他消息...
    return messages
}
```

---

## 7. 性能优化

### 7.1 缓存策略

**Redis 缓存层**:
- 对话历史: TTL=1h，LRU 淘汰
- QA 缓存: TTL=1h，高频问题长期缓存
- 用户上下文: 长期缓存，手动失效

**SQLite 优化**:
- FTS5 索引加速全文检索
- 对高频查询字段添加索引
- 使用 PRAGMA 优化（如 `journal_mode=WAL`）

### 7.2 并发控制

**Eino Graph 并发**:
- 多个独立的工具调用可以并发执行
- 使用 Go Context 控制超时

**数据库连接池**:
```go
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(5 * time.Minute)
```

---

## 8. 测试策略

### 8.1 单元测试

**覆盖模块**:
- `internal/memory/` - Memory Manager 核心逻辑
- `internal/knowledge/` - 检索算法
- `internal/agent/mcp_adapter.go` - MCP 适配器

**测试工具**:
- `testing` 标准库
- `github.com/stretchr/testify` 断言库
- Mock MCP Server 进行隔离测试

### 8.2 集成测试

**测试场景**:
1. 完整对话流程（加载历史 → LLM → 工具调用 → 保存）
2. 多轮对话（工具调用失败重试）
3. 知识库检索准确性
4. QA 缓存命中率

### 8.3 性能测试

**指标**:
- 首次响应时间（TTFB）
- 完整对话耗时
- 缓存命中率
- 数据库查询性能

**工具**:
- `go test -bench`
- 压力测试工具（如 `wrk`）

---

## 9. 依赖变更

### 9.1 新增依赖

```go
// go.mod (新增)
require (
    github.com/cloudwego/eino v0.x.x          // Eino 框架
    github.com/cloudwego/eino-ext v0.x.x      // Eino 扩展
    github.com/redis/go-redis/v9 v9.x.x       // Redis 客户端
)
```

### 9.2 保留依赖

```go
// go.mod (保留)
require (
    github.com/mark3labs/mcp-go v0.x.x        // MCP SDK (复用)
    github.com/gin-gonic/gin v1.x.x           // Web 框架
    gorm.io/gorm v1.x.x                       // ORM
    gorm.io/driver/sqlite v1.x.x              // SQLite 驱动
    // ... 其他现有依赖
)
```

### 9.3 移除依赖

```go
// go.mod (移除)
// github.com/sashabaranov/go-openai  ❌ 删除（Eino 内置）
```

---

## 10. 配置变更

### 10.1 新增配置项

```yaml
# config.yaml

# Eino Agent 配置
agent:
  use_eino: true                # 是否启用 Eino（Feature Flag）
  max_iterations: 10            # 最大工具调用迭代次数
  timeout: 300                  # 超时时间（秒）

# Redis 配置（可选）
redis:
  enabled: true                 # 是否启用 Redis 缓存
  host: localhost
  port: 6379
  password: ""
  db: 0
  ttl: 3600                     # 默认 TTL（秒）

# 知识库配置
knowledge:
  enabled: true                 # 是否启用知识库
  use_vector: false             # 是否启用向量检索（需要 sqlite-vec）
  max_results: 3                # 最大检索结果数

# 记忆管理配置
memory:
  history_limit: 10             # 对话历史保留条数
  qa_cache_enabled: true        # 是否启用 QA 缓存
  qa_cache_threshold: 3         # QA 缓存命中阈值（hit_count）
```

### 10.2 兼容性

- 旧配置项保持不变，向后兼容
- 新配置项有默认值，不配置也能运行
- Redis 未配置时降级到纯 SQLite 模式

---

## 11. 风险和挑战

### 11.1 技术风险

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| Eino 框架不稳定 | 高 | 1. 使用稳定版本<br>2. 充分测试<br>3. 准备回滚方案 |
| 性能下降 | 中 | 1. 性能测试对比<br>2. 缓存优化<br>3. 并发控制 |
| Redis 依赖增加复杂性 | 低 | 1. 设为可选依赖<br>2. 降级方案 |
| 数据迁移失败 | 中 | 1. 数据备份<br>2. 分步迁移<br>3. 验证脚本 |

### 11.2 业务风险

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| 用户体验变化 | 中 | 1. 灰度发布<br>2. A/B 测试<br>3. 用户反馈收集 |
| 功能回归 | 高 | 1. 充分测试<br>2. 功能对比清单<br>3. 快速回滚 |
| 学习成本 | 低 | 1. 代码注释完善<br>2. 开发文档<br>3. 团队培训 |

### 11.3 挑战

1. **Eino 学习曲线**
   - 团队需要学习 Eino 的概念和最佳实践
   - 建议：先通过示例项目熟悉，再正式开发

2. **流式输出兼容性**
   - Eino 的流式 API 需要适配到现有的 SSE 输出
   - 建议：封装统一的 Stream Handler

3. **多 MCP Server 编排**
   - 跨 MCP Server 的工具调用需要仔细设计
   - 建议：使用 Eino Graph 的条件分支

---

## 12. 时间规划

### 12.1 开发周期估算

| 阶段 | 任务 | 预估工时 |
|------|------|----------|
| 阶段一 | 基础设施准备 | 3-5 天 |
| 阶段二 | Eino Agent 实现 | 5-7 天 |
| 阶段三 | 逐步切换 | 3-5 天 |
| 阶段四 | 清理旧代码 | 1-2 天 |
| **总计** | | **12-19 天** |

### 12.2 里程碑

- **Week 1**: 完成阶段一（基础设施）
- **Week 2**: 完成阶段二（Agent 实现）
- **Week 3**: 完成阶段三（切换测试）
- **Week 4**: 完成阶段四（清理上线）

---

## 13. 成功标准

### 13.1 功能标准

✅ 支持复杂的多步骤、跨 MCP Server 任务编排
✅ 会话记忆和用户上下文正常工作
✅ 知识库检索准确性达到预期
✅ QA 缓存命中率 > 30%（高频问题）
✅ 流式输出与现有实现行为一致
✅ 所有现有功能无回归

### 13.2 性能标准

✅ 首次响应时间 < 2s
✅ 完整对话耗时 < 10s（含工具调用）
✅ 缓存命中时响应时间 < 500ms
✅ 数据库查询 P95 < 100ms

### 13.3 质量标准

✅ 单元测试覆盖率 > 70%
✅ 集成测试通过率 100%
✅ 无严重 Bug
✅ 代码通过 linter 检查
✅ 文档完整（代码注释 + 开发文档）

---

## 14. 参考资料

### 14.1 Eino 框架
- [Eino GitHub](https://github.com/cloudwego/eino)
- [Eino 官方文档](https://www.cloudwego.io/docs/eino/)
- [Eino 框架结构](https://www.cloudwego.io/docs/eino/overview/eino_framework_structure/)
- [Eino 编排设计原则](https://www.cloudwego.io/docs/eino/core_modules/chain_and_graph_orchestration/orchestration_design_principles/)
- [Eino ReAct Agent 手册](https://www.cloudwego.io/docs/eino/core_modules/flow_integration_components/react_agent_manual/)

### 14.2 MCP 协议
- [MCP Go SDK](https://github.com/mark3labs/mcp-go)
- [Eino MCP Tool 集成](https://cloudwego.cn/docs/eino/ecosystem_integration/tool/tool_mcp/)

### 14.3 知识检索
- [Eino Retriever 指南](https://www.cloudwego.io/docs/eino/core_modules/components/retriever_guide/)
- [SQLite FTS5 文档](https://www.sqlite.org/fts5.html)
- [SQLite Vector 扩展](https://www.sqlite.ai/sqlite-vector)

### 14.4 其他
- [Redis AI Agent Memory](https://redis.io/resources/redis-whitepaper-ai-agent-memory.pdf)
- [Go Context 最佳实践](https://go.dev/blog/context)

---

## 15. 附录

### 15.1 术语表

| 术语 | 说明 |
|------|------|
| **Eino** | 字节跳动开源的 Go 语言 LLM 应用开发框架 |
| **ReAct** | Reasoning and Acting，推理与行动模式 |
| **MCP** | Model Context Protocol，模型上下文协议 |
| **RAG** | Retrieval Augmented Generation，检索增强生成 |
| **FTS5** | SQLite 全文检索引擎第 5 版 |
| **Graph** | Eino 的有向图编排模式 |
| **ToolsNode** | Eino 的工具调用节点 |
| **ChatModel** | Eino 的 LLM 接口抽象 |

### 15.2 FAQ

**Q: Eino 是否支持 OpenAI 兼容的 API?**
A: 是的，Eino 提供了 OpenAI 兼容的 ChatModel 实现，可以直接替换现有的 `github.com/sashabaranov/go-openai`。

**Q: Redis 是必须的吗?**
A: 不是。Redis 是可选的缓存层，未配置时会降级到纯 SQLite 模式，性能略有下降但功能完整。

**Q: 如何回滚到旧实现?**
A: 通过配置项 `agent.use_eino: false` 即可切换回旧实现（需要在阶段三保留旧代码）。

**Q: 向量检索是否必须?**
A: 不是。可以只使用 FTS5 全文检索，向量检索是可选的增强功能（需要 sqlite-vec 扩展）。

**Q: 现有的 MCP Server 需要改造吗?**
A: 不需要。MCP Server 保持不变，只需要通过 Adapter 包装为 Eino Tool。

---

## 16. 审批签字

| 角色 | 姓名 | 签字 | 日期 |
|------|------|------|------|
| 设计者 | Claude | ✅ | 2026-01-02 |
| 技术评审 |  |  |  |
| 产品评审 |  |  |  |
| 最终批准 |  |  |  |

---

**文档结束**
