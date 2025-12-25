# ZenOps 后端 API 接口设计文档

## 目录
- [1. 基础约定](#1-基础约定)
- [2. 系统配置管理 (Config Management)](#2-系统配置管理-config-management)
- [3. MCP 服务管理 (MCP Services)](#3-mcp-服务管理-mcp-services)
- [4. 仪表盘与监控 (Dashboard)](#4-仪表盘与监控-dashboard)
- [5. 对话历史 (Chat History)](#5-对话历史-chat-history)
- [6. Go 数据结构建议](#6-go-数据结构建议)

---

## 1. 基础约定

### Base URL
```
/api/v1
```

### Content-Type
```
application/json
```

### 身份验证
- 在 Header 中携带: `Authorization: Bearer <token>`
- 对应配置中的 `AppConfig.auth`

### 通用响应格式
```json
{
  "code": 0,           // 0 表示成功，非 0 表示错误
  "message": "success",
  "data": {}           // 实际返回数据
}
```

### 错误码定义
- `0`: 成功
- `400`: 请求参数错误
- `401`: 未授权
- `403`: 禁止访问
- `404`: 资源不存在
- `500`: 服务器内部错误

---

## 2. 系统配置管理 (Config Management)

对应前端组件: `ConfigView.tsx`

### 2.1 获取全量配置

**接口**: `GET /api/v1/config`

**描述**: 获取系统所有配置信息

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "server": {
      "http": {
        "enabled": true,
        "port": 8080
      },
      "mcp": {
        "enabled": true,
        "port": 8081,
        "auto_register_external_tools": true,
        "tool_name_format": "{prefix}{name}"
      }
    },
    "logger": {
      "level": "info",
      "file": "./zenops.log"
    },
    "database": {
      "driver": "sqlite",
      "dsn": "zenops.db"
    },
    "llm_providers": [...],
    "dingtalk": {...},
    "feishu": {...},
    "wecom": {...},
    "providers": {
      "aliyun": [...],
      "tencent": [...]
    },
    "auth": {...},
    "cache": {...},
    "mcp_servers_config": "./mcp_servers.json"
  }
}
```

---

### 2.2 更新系统基础设置

**接口**: `PATCH /api/v1/config/system`

**描述**: 更新 HTTP/MCP/Database 等基础配置

**请求体**:
```json
{
  "server": {
    "http": {
      "enabled": true,
      "port": 8080
    }
  }
}
```

**响应**: 标准成功响应

---

### 2.3 LLM 供应商管理

#### 2.3.1 获取所有 LLM 引擎
**接口**: `GET /api/v1/config/llm`

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "id": "1",
      "name": "DeepSeek 主引擎",
      "enabled": true,
      "provider": "deepseek",
      "model": "DeepSeek-V3",
      "api_key": "sk-****",
      "base_url": "https://api.deepseek.com"
    }
  ]
}
```

**注意**: 返回的 `api_key` 需要脱敏处理 (只显示前4位和后4位)

#### 2.3.2 新增 LLM 引擎
**接口**: `POST /api/v1/config/llm`

**请求体**:
```json
{
  "name": "新 LLM 引擎",
  "enabled": true,
  "provider": "openai",
  "model": "gpt-4o",
  "api_key": "sk-xxx",
  "base_url": "https://api.openai.com"
}
```

**响应**: 返回创建的 LLM 实例 (包含生成的 ID)

#### 2.3.3 更新 LLM 引擎
**接口**: `PUT /api/v1/config/llm/:id`

**请求体**: 同新增

**响应**: 标准成功响应

#### 2.3.4 删除 LLM 引擎
**接口**: `DELETE /api/v1/config/llm/:id`

**响应**: 标准成功响应

#### 2.3.5 切换 LLM 引擎状态
**接口**: `PATCH /api/v1/config/llm/:id/toggle`

**响应**: 返回更新后的 enabled 状态

---

### 2.4 IM 平台配置

#### 2.4.1 更新钉钉配置
**接口**: `PATCH /api/v1/config/im/dingtalk`

**请求体**:
```json
{
  "enabled": true,
  "app_key": "ding_key_...",
  "app_secret": "ding_secret_...",
  "agent_id": "123456",
  "card_template_id": ""
}
```

#### 2.4.2 更新飞书配置
**接口**: `PATCH /api/v1/config/im/feishu`

**请求体**:
```json
{
  "enabled": true,
  "app_id": "cli_...",
  "app_secret": "secret_..."
}
```

#### 2.4.3 更新企业微信配置
**接口**: `PATCH /api/v1/config/im/wecom`

**请求体**:
```json
{
  "enabled": true,
  "token": "token_...",
  "encoding_aes_key": "aes_..."
}
```

---

### 2.5 云账号管理

#### 2.5.1 获取阿里云账号列表
**接口**: `GET /api/v1/config/cloud/aliyun`

**响应示例**:
```json
{
  "code": 0,
  "data": [
    {
      "id": "ali-1",
      "name": "默认账号",
      "enabled": true,
      "ak": "LTAI****",
      "sk": "****",
      "regions": ["cn-hangzhou", "cn-shanghai"]
    }
  ]
}
```

#### 2.5.2 新增阿里云账号
**接口**: `POST /api/v1/config/cloud/aliyun`

**请求体**:
```json
{
  "name": "新账号",
  "enabled": true,
  "ak": "LTAI...",
  "sk": "SECRET...",
  "regions": ["cn-beijing"]
}
```

#### 2.5.3 更新阿里云账号
**接口**: `PUT /api/v1/config/cloud/aliyun/:id`

#### 2.5.4 删除阿里云账号
**接口**: `DELETE /api/v1/config/cloud/aliyun/:id`

#### 2.5.5 腾讯云账号管理
**接口**:
- `GET /api/v1/config/cloud/tencent`
- `POST /api/v1/config/cloud/tencent`
- `PUT /api/v1/config/cloud/tencent/:id`
- `DELETE /api/v1/config/cloud/tencent/:id`

与阿里云类似，字段略有不同 (ak/sk 对应 SecretId/SecretKey)

---

## 3. MCP 服务管理 (MCP Services)

对应前端组件: `MCPView.tsx`

### 3.1 MCP 服务列表与操作

#### 3.1.1 获取所有 MCP 服务
**接口**: `GET /api/v1/mcp/servers`

**响应示例**:
```json
{
  "code": 0,
  "data": [
    {
      "id": "1",
      "name": "filesystem",
      "description": "允许安全访问本地文件系统",
      "type": "stdio",
      "isEnabled": true,
      "config": {
        "command": "npx",
        "args": ["-y", "@modelcontextprotocol/server-filesystem"],
        "env": {}
      },
      "tools": [
        {
          "name": "read_file",
          "description": "读取文件",
          "isEnabled": true,
          "inputSchema": {...}
        }
      ],
      "status": "connected",
      "longRunning": false,
      "timeout": 60,
      "provider": "Model Context Protocol",
      "providerUrl": "https://github.com/modelcontextprotocol/servers",
      "logoUrl": "",
      "tags": ["system", "io"],
      "url": "",
      "sseHeaders": ""
    }
  ]
}
```

#### 3.1.2 创建 MCP 服务
**接口**: `POST /api/v1/mcp/servers`

**请求体**:
```json
{
  "name": "new-server",
  "description": "新服务描述",
  "type": "stdio",
  "config": {
    "command": "npx",
    "args": ["-y", "@mcp/server"],
    "env": {}
  },
  "longRunning": false,
  "timeout": 60,
  "provider": "",
  "providerUrl": "",
  "logoUrl": "",
  "tags": []
}
```

**响应**: 返回创建的服务 (包含生成的 ID)

#### 3.1.3 获取单个 MCP 服务详情
**接口**: `GET /api/v1/mcp/servers/:id`

**响应**: 返回服务详情及工具列表

#### 3.1.4 更新 MCP 服务
**接口**: `PUT /api/v1/mcp/servers/:id`

**请求体**: 同创建

#### 3.1.5 删除 MCP 服务
**接口**: `DELETE /api/v1/mcp/servers/:id`

---

### 3.2 MCP 服务状态控制

#### 3.2.1 启用/停止 MCP 服务
**接口**: `POST /api/v1/mcp/servers/:id/toggle`

**描述**: 启用/停止该 MCP 服务进程

**响应**:
```json
{
  "code": 0,
  "data": {
    "id": "1",
    "isEnabled": true,
    "status": "connected"
  }
}
```

#### 3.2.2 启用/禁用工具
**接口**: `PATCH /api/v1/mcp/servers/:id/tools/:toolName/toggle`

**描述**: 切换某个工具的启用状态

**响应**:
```json
{
  "code": 0,
  "data": {
    "toolName": "read_file",
    "isEnabled": true
  }
}
```

---

### 3.3 MCP 工具调试

**接口**: `POST /api/v1/mcp/debug/execute`

**描述**: 后端中转调用对应的 MCP 工具，并返回执行结果

**请求体**:
```json
{
  "serverId": "1",
  "toolName": "read_file",
  "arguments": {
    "path": "/tmp/test.txt"
  }
}
```

**响应**:
```json
{
  "code": 0,
  "data": {
    "success": true,
    "result": "文件内容...",
    "latency": 45
  }
}
```

---

## 4. 仪表盘与监控 (Dashboard)

对应前端组件: `DashboardView.tsx`

### 4.1 获取仪表盘统计数据

**接口**: `GET /api/v1/dashboard/stats`

**描述**: 获取实时汇总统计

**响应示例**:
```json
{
  "code": 0,
  "data": {
    "activeBots": 2,
    "totalServers": 3,
    "totalTools": 15,
    "successRate": 98.5,
    "avgLatency": 120
  }
}
```

---

### 4.2 基础设施健康检查

**接口**: `GET /api/v1/dashboard/health`

**描述**: 返回各组件的在线状态及 Uptime

**响应示例**:
```json
{
  "code": 0,
  "data": {
    "components": [
      {
        "label": "DingTalk Gateway",
        "status": "online",
        "uptime": "99.9%",
        "detail": ""
      },
      {
        "label": "DeepSeek Provider",
        "status": "online",
        "uptime": "98.5%"
      },
      {
        "label": "MCP Grid",
        "status": "online",
        "uptime": "100%"
      },
      {
        "label": "SQLite Database",
        "status": "online",
        "uptime": "100%"
      },
      {
        "label": "Aliyun API",
        "status": "warning",
        "uptime": "94.2%",
        "detail": "Latent regions: cn-hangzhou"
      }
    ]
  }
}
```

---

### 4.3 MCP 调用日志

**接口**: `GET /api/v1/logs/mcp`

**描述**: 获取 MCP 工具调用审计日志

**查询参数**:
- `page`: 页码 (默认 1)
- `pageSize`: 每页数量 (默认 20)
- `search`: 搜索关键词
- `status`: 状态过滤 (success/error/warning)

**响应示例**:
```json
{
  "code": 0,
  "data": {
    "total": 100,
    "page": 1,
    "pageSize": 20,
    "items": [
      {
        "id": "l1",
        "timestamp": "2024-05-20 14:20:11",
        "serverName": "filesystem",
        "toolName": "read_file",
        "status": "success",
        "latency": 45,
        "request": "read config.json"
      }
    ]
  }
}
```

---

## 5. 对话历史 (Chat History)

对应前端组件: `ChatHistoryView.tsx`

### 5.1 获取对话记录

**接口**: `GET /api/v1/history/chats`

**查询参数**:
- `page`: 页码 (默认 1)
- `pageSize`: 每页数量 (默认 20)
- `search`: 搜索关键词 (搜索内容或用户名)
- `source`: 来源过滤 (私聊/群聊)
- `chat_type`: 类型过滤 (1=用户提问, 2=AI回答)

**响应示例**:
```json
{
  "code": 0,
  "data": {
    "total": 500,
    "page": 1,
    "pageSize": 20,
    "items": [
      {
        "id": 9,
        "created_at": "2023-04-02 09:32:27.821287+08:00",
        "updated_at": "2023-04-02 09:32:27.821287+08:00",
        "deleted_at": null,
        "username": "李启龙",
        "source": "私聊",
        "chat_type": 1,
        "parent_content": 0,
        "content": "世界上有多少个国家"
      }
    ]
  }
}
```

---

### 5.2 获取消息上下文

**接口**: `GET /api/v1/history/chats/:id/context`

**描述**: 获取某条消息的上下文 (关联的提问或回复)

**响应示例**:
```json
{
  "code": 0,
  "data": {
    "current": {...},
    "parent": {...},
    "children": [...]
  }
}
```

---

## 6. Go 数据结构建议

### 6.1 MCP Server 模型

```go
package model

import "time"

// MCPServer MCP 服务模型
type MCPServer struct {
    ID          string                 `json:"id" gorm:"primaryKey"`
    Name        string                 `json:"name" gorm:"not null"`
    Description string                 `json:"description"`
    Type        string                 `json:"type" gorm:"not null"` // "stdio" | "sse"
    IsEnabled   bool                   `json:"isEnabled" gorm:"default:true"`
    Config      MCPServerConfig        `json:"config" gorm:"serializer:json"`
    Status      string                 `json:"status"` // "connected" | "disconnected" | "error"

    // Advanced Fields
    LongRunning  bool                  `json:"longRunning" gorm:"default:false"`
    Timeout      int                   `json:"timeout" gorm:"default:60"`
    Provider     string                `json:"provider"`
    ProviderURL  string                `json:"providerUrl"`
    LogoURL      string                `json:"logoUrl"`
    Tags         []string              `json:"tags" gorm:"serializer:json"`

    // SSE Specific
    URL         string                 `json:"url,omitempty"`
    SSEHeaders  string                 `json:"sseHeaders,omitempty"`

    CreatedAt   time.Time              `json:"created_at"`
    UpdatedAt   time.Time              `json:"updated_at"`
    DeletedAt   *time.Time             `json:"deleted_at" gorm:"index"`
}

// MCPServerConfig 服务配置
type MCPServerConfig struct {
    Command string            `json:"command"`
    Args    []string          `json:"args"`
    Env     map[string]string `json:"env"`
}

// MCPTool MCP 工具模型
type MCPTool struct {
    ID          uint                   `json:"id" gorm:"primaryKey"`
    ServerID    string                 `json:"serverId" gorm:"index"`
    Name        string                 `json:"name" gorm:"not null"`
    Description string                 `json:"description"`
    IsEnabled   bool                   `json:"isEnabled" gorm:"default:true"`
    InputSchema map[string]interface{} `json:"inputSchema" gorm:"serializer:json"`

    CreatedAt   time.Time              `json:"created_at"`
    UpdatedAt   time.Time              `json:"updated_at"`
}

// MCPLog MCP 调用日志
type MCPLog struct {
    ID         string    `json:"id" gorm:"primaryKey"`
    Timestamp  time.Time `json:"timestamp"`
    ServerName string    `json:"serverName" gorm:"index"`
    ToolName   string    `json:"toolName" gorm:"index"`
    Status     string    `json:"status"` // "success" | "error" | "warning"
    Latency    int       `json:"latency"`
    Request    string    `json:"request"`
    Response   string    `json:"response"`
}
```

---

### 6.2 Chat History 模型

```go
// ChatLog 对话记录
type ChatLog struct {
    ID            uint       `json:"id" gorm:"primaryKey"`
    CreatedAt     time.Time  `json:"created_at"`
    UpdatedAt     time.Time  `json:"updated_at"`
    DeletedAt     *time.Time `json:"deleted_at" gorm:"index"`
    Username      string     `json:"username" gorm:"index"`
    Source        string     `json:"source"` // "私聊" | "群聊"
    ChatType      int        `json:"chat_type"` // 1=用户提问, 2=AI回答
    ParentContent uint       `json:"parent_content"` // 父消息ID
    Content       string     `json:"content" gorm:"type:text"`
}
```

---

### 6.3 Config 模型

```go
// AppConfig 应用配置 (存储在配置文件或数据库)
type AppConfig struct {
    Server       ServerConfig       `json:"server"`
    Logger       LoggerConfig       `json:"logger"`
    Database     DatabaseConfig     `json:"database"`
    LLMProviders []LLMProvider      `json:"llm_providers"`
    DingTalk     DingTalkConfig     `json:"dingtalk"`
    Feishu       FeishuConfig       `json:"feishu"`
    WeCom        WeComConfig        `json:"wecom"`
    Providers    CloudProviders     `json:"providers"`
    Auth         AuthConfig         `json:"auth"`
    Cache        CacheConfig        `json:"cache"`
    MCPServersConfig string         `json:"mcp_servers_config"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
    HTTP ServerHTTPConfig `json:"http"`
    MCP  ServerMCPConfig  `json:"mcp"`
}

type ServerHTTPConfig struct {
    Enabled bool `json:"enabled"`
    Port    int  `json:"port"`
}

type ServerMCPConfig struct {
    Enabled                   bool   `json:"enabled"`
    Port                      int    `json:"port"`
    AutoRegisterExternalTools bool   `json:"auto_register_external_tools"`
    ToolNameFormat            string `json:"tool_name_format"`
}

// LLMProvider LLM 提供商实例
type LLMProvider struct {
    ID       string `json:"id"`
    Name     string `json:"name"`
    Enabled  bool   `json:"enabled"`
    Provider string `json:"provider"` // "openai" | "anthropic" | "deepseek" | etc.
    Model    string `json:"model"`
    APIKey   string `json:"api_key"`
    BaseURL  string `json:"base_url"`
}

// CloudAccount 云账号
type CloudAccount struct {
    ID      string   `json:"id"`
    Name    string   `json:"name"`
    Enabled bool     `json:"enabled"`
    AK      string   `json:"ak"`
    SK      string   `json:"sk"`
    Regions []string `json:"regions"`
}
```

---

## 7. 实现建议

### 7.1 安全性
- **API Key 脱敏**: 所有返回给前端的 API Key、Secret 字段需要脱敏 (例如: `sk-****1234`)
- **更新时接受明文**: 只有在更新配置时才接受明文，读取时返回脱敏后的数据
- **权限验证**: 根据 `auth` 配置验证请求的合法性

### 7.2 MCP 进程管理
- **独立进程**: 使用 `os/exec` 为每个 stdio 类型的 MCP 服务启动独立进程
- **状态监控**: 定期检查进程状态，更新 `status` 字段
- **日志收集**: 收集 MCP 工具调用日志到数据库

### 7.3 实时通信
- **WebSocket/SSE**: 建议后端在 MCP 服务状态变化时，通过 WebSocket 向前端推送更新
- **接口**: `ws://localhost:8080/api/v1/ws` 或 SSE endpoint

### 7.4 配置持久化
- **文件 + 数据库混合**: 基础配置可以存在配置文件，运行时数据 (MCP Servers, Logs) 存在数据库
- **热更新**: 配置变更后，支持不重启服务的热更新

---

## 8. 前端调用示例

### 8.1 获取配置
```typescript
// services/api.ts
export const getConfig = async () => {
  const response = await fetch('/api/v1/config');
  return response.json();
};
```

### 8.2 更新 LLM 配置
```typescript
export const updateLLM = async (id: string, data: LLMProviderInstance) => {
  const response = await fetch(`/api/v1/config/llm/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data)
  });
  return response.json();
};
```

### 8.3 获取 MCP 服务列表
```typescript
export const getMCPServers = async () => {
  const response = await fetch('/api/v1/mcp/servers');
  return response.json();
};
```

---

## 9. 总结

本文档定义了 ZenOps 后端所有 API 接口的设计规范，包括:

1. **系统配置管理**: 全量配置获取、LLM 引擎管理、IM 平台配置、云账号管理
2. **MCP 服务管理**: CRUD 操作、状态控制、工具调试
3. **仪表盘监控**: 统计数据、健康检查、日志审计
4. **对话历史**: 记录查询、上下文获取

后端开发时请严格按照此文档实现接口，确保前后端数据结构一致。
