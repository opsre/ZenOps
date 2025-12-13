# ZenOps - 运维数据智能化查询工具设计文档

## 项目概述

ZenOps 是一个面向运维领域的数据智能化查询工具,通过统一的接口抽象,支持多云平台(阿里云、腾讯云等)、CI/CD 工具(Jenkins等)的资源查询,并通过 CLI、HTTP API 和 MCP 协议提供多种访问方式,同时集成钉钉智能机器人实现对话式查询。

## 核心设计理念

### 1. 可扩展性
- 通过 Interface 抽象统一的查询能力
- 新增云平台或服务只需实现对应接口
- 插件化的架构设计

### 2. 多接入方式
- CLI 命令行工具 (基于 Cobra)
- HTTP RESTful API
- MCP (Model Context Protocol) 协议支持
- 钉钉机器人集成

### 3. 统一数据模型
- 标准化的资源描述格式
- 跨平台的资源映射能力

## 技术栈

- **语言**: Go 1.21+
- **CLI框架**: [cobra](https://github.com/spf13/cobra)
- **配置管理**: [viper](https://github.com/spf13/viper)
- **HTTP框架**: [gin](https://github.com/gin-gonic/gin)
- **MCP实现**: 自定义 MCP Server
- **钉钉SDK**: [dingtalk-sdk-golang](https://github.com/icepy/go-dingtalk)
- **日志**: [zap](https://github.com/uber-go/zap)
- **云服务SDK**:
  - 阿里云: aliyun-sdk-go
  - 腾讯云: tencentcloud-sdk-go
  - Jenkins: gojenkins

## 项目结构

```
zenops/
├── cmd/                          # CLI 命令定义
│   ├── root.go                   # 根命令
│   ├── serve.go                  # HTTP/MCP Server 启动命令
│   ├── query.go                  # 查询相关命令组
│   │   ├── aliyun.go            # 阿里云查询命令
│   │   ├── tencent.go           # 腾讯云查询命令
│   │   └── jenkins.go           # Jenkins 查询命令
│   └── version.go                # 版本信息
│
├── internal/                     # 私有应用代码
│   ├── provider/                 # 云服务提供商实现
│   │   ├── interface.go         # 统一接口定义
│   │   ├── aliyun/              # 阿里云实现
│   │   │   ├── ecs.go          # ECS 服务
│   │   │   ├── rds.go          # RDS 服务
│   │   │   └── client.go       # 客户端封装
│   │   ├── tencent/             # 腾讯云实现
│   │   │   ├── cvm.go          # CVM 服务
│   │   │   └── client.go
│   │   └── jenkins/             # Jenkins 实现
│   │       ├── job.go
│   │       └── client.go
│   │
│   ├── model/                    # 数据模型
│   │   ├── common.go            # 通用模型
│   │   ├── instance.go          # 实例模型(统一云服务器)
│   │   ├── database.go          # 数据库模型
│   │   └── job.go               # 任务模型
│   │
│   ├── service/                  # 业务逻辑层
│   │   ├── query.go             # 查询服务
│   │   ├── aggregator.go        # 数据聚合服务
│   │   └── formatter.go         # 数据格式化
│   │
│   ├── api/                      # HTTP API 实现
│   │   ├── server.go            # HTTP 服务器
│   │   ├── handler/             # 请求处理
│   │   │   ├── query.go        # 查询接口
│   │   │   └── health.go       # 健康检查
│   │   └── middleware/          # 中间件
│   │       ├── auth.go         # 认证
│   │       └── logger.go       # 日志
│   │
│   ├── mcp/                      # MCP 协议实现
│   │   ├── server.go            # MCP 服务器
│   │   ├── handler.go           # MCP 请求处理
│   │   └── tools.go             # MCP Tools 定义
│   │
│   ├── dingtalk/                 # 钉钉集成
│   │   ├── bot.go               # 机器人处理
│   │   ├── handler.go           # 消息处理
│   │   └── callback.go          # 回调处理
│   │
│   └── config/                   # 配置管理
│       ├── config.go            # 配置结构
│       └── loader.go            # 配置加载
│
├── pkg/                          # 公共库代码(可被外部引用)
│   ├── utils/                    # 工具函数
│   │   ├── logger.go            # 日志工具
│   │   └── errors.go            # 错误处理
│   └── constants/                # 常量定义
│
├── configs/                      # 配置文件
│   ├── config.yaml              # 默认配置
│   └── config.example.yaml      # 配置示例
│
├── docs/                         # 文档
│   ├── api.md                   # API 文档
│   ├── mcp.md                   # MCP 协议文档
│   ├── providers.md             # Provider 开发指南
│   └── dingtalk.md              # 钉钉集成文档
│
├── scripts/                      # 脚本
│   ├── build.sh                 # 编译脚本
│   └── deploy.sh                # 部署脚本
│
├── main.go                       # 程序入口
├── go.mod
├── go.sum
├── Makefile                      # 编译管理
├── README.md                     # 项目说明
└── DESIGN.md                     # 本设计文档
```

## 核心接口设计

### 1. Provider 接口 (统一云服务抽象)

```go
package provider

import (
    "context"
    "github.com/eryajf/zenops/internal/model"
)

// Provider 定义了��服务提供商的统一接口
type Provider interface {
    // GetName 返回提供商名称 (如: aliyun, tencent, aws)
    GetName() string

    // Initialize 初始化提供商客户端
    Initialize(config map[string]any) error

    // ListInstances 列出所有实例 (ECS/CVM/EC2)
    ListInstances(ctx context.Context, opts *QueryOptions) ([]*model.Instance, error)

    // GetInstance 获取单个实例详情
    GetInstance(ctx context.Context, instanceID string) (*model.Instance, error)

    // ListDatabases 列出数据库实例
    ListDatabases(ctx context.Context, opts *QueryOptions) ([]*model.Database, error)

    // GetDatabase 获取数据库详情
    GetDatabase(ctx context.Context, dbID string) (*model.Database, error)

    // HealthCheck 健康检查
    HealthCheck(ctx context.Context) error
}

// QueryOptions 查询选项
type QueryOptions struct {
    Region      string            // 区域
    PageSize    int              // 分页大小
    PageNum     int              // 页码
    Filters     map[string]string // 过滤条件
    Tags        map[string]string // 标签过滤
}
```

### 2. CI/CD Provider 接口

```go
package provider

import (
    "context"
    "github.com/eryajf/zenops/internal/model"
)

// CICDProvider 定义 CI/CD 工具的统一接口
type CICDProvider interface {
    // GetName 返回提供商名称 (如: jenkins, gitlab-ci)
    GetName() string

    // Initialize 初始化客户端
    Initialize(config map[string]any) error

    // ListJobs 列出所有任务
    ListJobs(ctx context.Context, opts *QueryOptions) ([]*model.Job, error)

    // GetJob 获取任务详情
    GetJob(ctx context.Context, jobName string) (*model.Job, error)

    // GetJobBuilds 获取任务的构建历史
    GetJobBuilds(ctx context.Context, jobName string, limit int) ([]*model.Build, error)

    // HealthCheck 健康检查
    HealthCheck(ctx context.Context) error
}
```

### 3. 统一数据模型

```go
package model

import "time"

// Instance 统一的实例模型 (跨云平台)
type Instance struct {
    ID           string            `json:"id"`
    Name         string            `json:"name"`
    Provider     string            `json:"provider"`      // 提供商: aliyun, tencent
    Region       string            `json:"region"`        // 区域
    Zone         string            `json:"zone"`          // 可用区
    InstanceType string            `json:"instance_type"` // 实例规格
    Status       string            `json:"status"`        // 状态
    PrivateIP    []string          `json:"private_ip"`
    PublicIP     []string          `json:"public_ip"`
    CPU          int               `json:"cpu"`
    Memory       int               `json:"memory"`        // MB
    OSType       string            `json:"os_type"`
    OSName       string            `json:"os_name"`
    CreatedAt    time.Time         `json:"created_at"`
    ExpiredAt    *time.Time        `json:"expired_at,omitempty"`
    Tags         map[string]string `json:"tags"`
    Metadata     map[string]any `json:"metadata"` // 扩展字段
}

// Database 数据库模型
type Database struct {
    ID           string            `json:"id"`
    Name         string            `json:"name"`
    Provider     string            `json:"provider"`
    Region       string            `json:"region"`
    Engine       string            `json:"engine"`        // mysql, postgresql, redis
    EngineVersion string           `json:"engine_version"`
    Status       string            `json:"status"`
    Endpoint     string            `json:"endpoint"`
    Port         int               `json:"port"`
    CreatedAt    time.Time         `json:"created_at"`
    Tags         map[string]string `json:"tags"`
}

// Job Jenkins 任务模型
type Job struct {
    Name        string    `json:"name"`
    DisplayName string    `json:"display_name"`
    URL         string    `json:"url"`
    Description string    `json:"description"`
    Buildable   bool      `json:"buildable"`
    LastBuild   *Build    `json:"last_build,omitempty"`
}

// Build 构建模型
type Build struct {
    Number    int       `json:"number"`
    Status    string    `json:"status"`
    Result    string    `json:"result"`
    Timestamp time.Time `json:"timestamp"`
    Duration  int64     `json:"duration"` // 毫秒
    URL       string    `json:"url"`
}
```

## 实现流程

### Phase 1: 基础框架搭建 ✅ (已完成)

#### 1.1 项目初始化
- [x] 创建项目结构
- [x] 配置 Go Modules
- [x] 集成 Cobra CLI 框架
- [x] 集成 Viper 配置管理
- [x] 实现日志系统 (zap)

#### 1.2 定义核心接口
- [x] 定义 Provider 接口
- [x] 定义 CICDProvider 接口
- [x] 定义统一数据模型
- [x] 实现 Provider 注册机制

#### 1.3 配置管理
- [x] 设计配置文件格式
- [x] 实现配置加载逻辑
- [x] 支持多环境配置
- [x] 支持多账号配置

### Phase 2: Provider 实现 ✅ (已完成)

#### 2.1 阿里云 Provider ✅ (已完成)
- [x] 实现阿里云客户端封装
- [x] 实现 ECS 查询功能
- [x] 实现 RDS 查询功能
- [x] 编写单元测试

#### 2.2 腾讯云 Provider ✅ (已完成)
- [x] 实现腾讯云客户端封装
- [x] 实现 CVM 查询功能
- [x] 实现 CDB 数据库查询功能
- [x] 支持多账号配置
- [x] 支持多区域查询
- [x] 实现自动分页

#### 2.3 Jenkins Provider ✅ (已完成)
- [x] 实现 Jenkins 客户端封装
- [x] 实现 Job 查询功能
- [x] 实现 Build 查询功能
- [x] 支持文件夹路径查询
- [x] 实现 Job 搜索功能

### Phase 3: CLI 实现 ✅ (已完成)

#### 3.1 基础命令
- [x] 实现 `zenops version` 命令
- [x] 实现 `zenops server http` 命令 (启动 HTTP 服务器)
- [x] 实现 `zenops server mcp` 命令 (启动 MCP 服务器)
- [ ] 实现 `zenops config` 命令 (配置管理)

#### 3.2 查询命令 ✅ (已完成)
- [x] 实现 `zenops query aliyun ecs list` 命令
- [x] 实现 `zenops query aliyun ecs get <id>` 命令
- [x] 实现 `zenops query aliyun rds list` 命令
- [x] 实现 `zenops query aliyun rds get <id>` 命令
- [x] 实现 `zenops query tencent cvm list` 命令
- [x] 实现 `zenops query tencent cvm get <id>` 命令
- [x] 实现 `zenops query tencent cdb list` 命令
- [x] 实现 `zenops query jenkins job list` 命令
- [x] 实现 `zenops query jenkins job get <name>` 命令
- [x] 实现 `zenops query jenkins build list <job>` 命令
- [x] 支持多账号选择 (`--account`)
- [x] 支持区域过滤 (`--region`)
- [x] 支持分页控制 (`--page-size`, `--page-num`)
- [x] 支持获取所有资源 (`--all`)
- [x] 支持多种输出格式 (`--output table/json`)
- [x] 美化表格输出 (lipgloss/table)
- [ ] 实现通用的 `zenops query all instances` (聚合查询)

### Phase 4: HTTP API 实现 ✅ (已完成)

#### 4.1 HTTP 服务器
- [x] 实现 HTTP Server (基于标准库 net/http)
- [x] 实现日志中间件
- [x] 实现错误处理
- [x] 实现优雅关闭
- [ ] 实现认证中间件

#### 4.2 API 端点 ✅ (已完成)
- [x] `GET /api/v1/health` - 健康检查
- [x] `GET /api/v1/aliyun/ecs/list` - 列出阿里云 ECS 实例
- [x] `GET /api/v1/aliyun/ecs/search` - 根据 IP/名称搜索 ECS
- [x] `GET /api/v1/aliyun/ecs/get` - 获取 ECS 实例详情
- [x] `GET /api/v1/aliyun/rds/list` - 列出阿里云 RDS 实例
- [x] `GET /api/v1/aliyun/rds/search` - 根据名称/endpoint 搜索 RDS
- [x] `GET /api/v1/tencent/cvm/list` - 列出腾讯云 CVM 实例
- [x] `GET /api/v1/tencent/cvm/search` - 根据 IP/名称搜索 CVM
- [x] `GET /api/v1/tencent/cvm/get` - 获取 CVM 实例详情
- [x] `GET /api/v1/tencent/cdb/list` - 列出腾讯云 CDB 实例
- [x] `GET /api/v1/tencent/cdb/search` - 根据名称搜索 CDB
- [x] `GET /api/v1/jenkins/jobs` - 列出 Jenkins 任务
- [x] `GET /api/v1/jenkins/jobs/:name` - 获取 Job 详情
- [x] `GET /api/v1/jenkins/jobs/:name/builds` - 获取构建历史
- [x] 支持多账号查询 (`?account=xxx`)
- [x] 支持区域过滤 (`?region=xxx`)
- [x] 自动分页获取所有数据
- [ ] `GET /api/v1/providers` - 列出所有提供商
- [ ] `GET /api/v1/instances` - 列出所有实例 (跨云聚合)
- [ ] `GET /api/v1/databases` - 列出数据库 (跨云聚合)

### Phase 5: MCP 协议实现 ✅ (已完成)

#### 5.1 MCP Server
- [x] 实现 MCP 协议服务器 (支持 stdio 和 SSE 两种模式)
- [x] 实现 MCP 初始化握手
- [x] 实现 MCP 工具注册
- [x] 实现 JSONRPC 2.0 协议
- [x] 提供两个实现版本:
  - [x] 手动实现版本 (mcp.go)
  - [x] 基于 mcp-go 库的版本 (mcp_with_lib.go)

#### 5.2 MCP Tools 定义 ✅ (已完成)

**阿里云工具:**
- [x] `search_ecs_by_ip` - 根据 IP 搜索 ECS 实例
- [x] `search_ecs_by_name` - 根据名称搜索 ECS 实例
- [x] `list_ecs` - 列出 ECS 实例
- [x] `get_ecs` - 获取 ECS 实例详情
- [x] `list_rds` - 列出 RDS 数据库
- [x] `search_rds_by_name` - 根据名称搜索 RDS

**腾讯云工具:**
- [x] `search_cvm_by_ip` - 根据 IP 搜索 CVM 实例
- [x] `search_cvm_by_name` - 根据名称搜索 CVM 实例
- [x] `list_cvm` - 列出 CVM 实例
- [x] `get_cvm` - 获取 CVM 实例详情
- [x] `list_cdb` - 列出 CDB 数据库
- [x] `search_cdb_by_name` - 根据名称搜索 CDB

**Jenkins 工具:**
- [x] `list_jenkins_jobs` - 列出 Jenkins 任务
- [x] `get_jenkins_job` - 获取 Job 详情
- [x] `list_jenkins_builds` - 列出构建历史

**通用功能:**
- [x] 支持多账号选择
- [x] 自动分页获取所有数据
- [x] SSE 模式支持实时推送
- [x] 格式化文本输出(适合 AI 阅读)
- [x] 统一使用 mcp-go 库实现

#### 5.3 MCP Resources
- [ ] 定义资源模板
- [ ] 实现资源访问接口

### Phase 6: 钉钉集成 (Week 6)

#### 6.1 钉钉机器人基础
- [ ] 创建钉钉应用
- [ ] 实现钉钉 OAuth 认证
- [ ] 实现消息回调处理

#### 6.2 对话处理
- [ ] 实现消息解析 (识别查询意图)
- [ ] 对接 MCP Server
- [ ] 实现结果格式化 (Markdown 卡片)
- [ ] 实现错误处理和友好提示

#### 6.3 高级功能
- [ ] 实现会话管理 (上下文保持)
- [ ] 实现权限控制 (根据用户身份)
- [ ] 实现审计日志

### Phase 7: 测试与文档 (Week 7)

#### 7.1 测试
- [ ] 完善单元测试 (覆盖率 > 70%)
- [ ] 编写集成测试
- [ ] 性能测试

#### 7.2 文档
- [ ] API 接口文档
- [ ] MCP 协议文档
- [ ] Provider 开发指南
- [ ] 钉钉集成部署文档
- [ ] 用户使用手册

## 配置文件示例

```yaml
# configs/config.yaml

# 服务配置
server:
  http:
    enabled: true
    port: 8080
    host: 0.0.0.0
  mcp:
    enabled: true
    mode: stdio  # stdio 或 sse
    port: 8081   # 仅 sse 模式需要

# 日志配置
logging:
  level: info
  format: json
  output: stdout

# 云服务提供商配置
providers:
  aliyun:
    enabled: true
    access_key_id: ${ALIYUN_ACCESS_KEY_ID}
    access_key_secret: ${ALIYUN_ACCESS_KEY_SECRET}
    regions:
      - cn-hangzhou
      - cn-shanghai
      - cn-beijing

  tencent:
    enabled: true
    secret_id: ${TENCENT_SECRET_ID}
    secret_key: ${TENCENT_SECRET_KEY}
    regions:
      - ap-guangzhou
      - ap-shanghai

# CI/CD 工具配置
cicd:
  jenkins:
    enabled: true
    url: https://jenkins.example.com
    username: ${JENKINS_USER}
    token: ${JENKINS_TOKEN}

# 钉钉配置
dingtalk:
  enabled: true
  app_key: ${DINGTALK_APP_KEY}
  app_secret: ${DINGTALK_APP_SECRET}
  agent_id: ${DINGTALK_AGENT_ID}
  # 回调配置
  callback:
    token: ${DINGTALK_CALLBACK_TOKEN}
    aes_key: ${DINGTALK_AES_KEY}
    url: https://your-domain.com/api/v1/dingtalk/callback

# 认证配置
auth:
  enabled: true
  type: token  # token, basic, oauth2
  tokens:
    - ${API_TOKEN_1}
    - ${API_TOKEN_2}

# 缓存配置 (可选)
cache:
  enabled: true
  type: memory  # memory, redis
  ttl: 300      # 秒
```

## CLI 使用示例

```bash
# 列出所有阿里云 ECS 实例
zenops query aliyun ecs list --region cn-hangzhou

# 获取指定实例详情
zenops query aliyun ecs get i-xxxxx

# 列出所有腾讯云 CVM 实例
zenops query tencent cvm list

# 聚合查询所有云平台的实例
zenops query all instances --output table

# 查询 Jenkins 任务
zenops query jenkins jobs --filter "name=*prod*"

# 启动 HTTP + MCP 服务
zenops serve --config configs/config.yaml

# 仅启动 MCP 服务
zenops serve --mcp-only --mcp-mode stdio
```

## API 使用示例

```bash
# 获取所有实例 (跨云聚合)
curl -H "Authorization: Bearer <token>" \
  http://localhost:8080/api/v1/instances

# 获取阿里云实例
curl -H "Authorization: Bearer <token>" \
  http://localhost:8080/api/v1/aliyun/instances?region=cn-hangzhou

# 获取 Jenkins 任务
curl -H "Authorization: Bearer <token>" \
  http://localhost:8080/api/v1/jenkins/jobs
```

## MCP 集成示例

在 Claude Desktop 的配置文件中 (`claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "zenops": {
      "command": "/path/to/zenops",
      "args": ["serve", "--mcp-only", "--mcp-mode", "stdio"],
      "env": {
        "ALIYUN_ACCESS_KEY_ID": "your-key",
        "ALIYUN_ACCESS_KEY_SECRET": "your-secret",
        "TENCENT_SECRET_ID": "your-id",
        "TENCENT_SECRET_KEY": "your-key"
      }
    }
  }
}
```

MCP Tools 定义:

```json
{
  "tools": [
    {
      "name": "list_instances",
      "description": "列出云服务器实例,支持跨云平台聚合查询",
      "inputSchema": {
        "type": "object",
        "properties": {
          "provider": {
            "type": "string",
            "description": "云平台名称 (aliyun, tencent, all)",
            "enum": ["aliyun", "tencent", "all"]
          },
          "region": {
            "type": "string",
            "description": "区域,如 cn-hangzhou"
          },
          "filters": {
            "type": "object",
            "description": "过滤条件"
          }
        }
      }
    },
    {
      "name": "get_instance",
      "description": "获取指定实例的详细信息",
      "inputSchema": {
        "type": "object",
        "properties": {
          "provider": {
            "type": "string",
            "description": "云平台名称",
            "enum": ["aliyun", "tencent"]
          },
          "instance_id": {
            "type": "string",
            "description": "实例 ID"
          }
        },
        "required": ["provider", "instance_id"]
      }
    }
  ]
}
```

## 钉钉对话示例

**用户**: @运维助手 帮我查一下杭州的服务器列表

**机器人**:
```
📊 阿里云 ECS 实例列表 (cn-hangzhou)

找到 5 台服务器:

1️⃣ web-server-01
   状态: Running
   规格: ecs.c6.large (2C4G)
   内网IP: 172.16.1.10
   公网IP: 47.xx.xx.xx

2️⃣ db-server-01
   状态: Running
   规格: ecs.g6.xlarge (4C16G)
   内网IP: 172.16.1.20

...
```

**用户**: @运维助手 看一下 Jenkins 上 prod 相关的任务

**机器人**:
```
🔧 Jenkins 任务列表 (匹配 "prod")

1️⃣ deploy-prod-web
   状态: ✅ Success
   最后构建: #128 (2小时前)
   耗时: 3分15秒

2️⃣ deploy-prod-api
   状态: ⚠️ Unstable
   最后构建: #95 (1天前)

点击查看详情: https://jenkins.example.com/...
```

## 扩展性设计

### 新增云平台 Provider

1. 在 `internal/provider/newcloud/` 创建新目录
2. 实现 `Provider` 接口:

```go
package newcloud

import (
    "context"
    "github.com/eryajf/zenops/internal/model"
    "github.com/eryajf/zenops/internal/provider"
)

type NewCloudProvider struct {
    client *Client
}

func NewProvider() provider.Provider {
    return &NewCloudProvider{}
}

func (p *NewCloudProvider) GetName() string {
    return "newcloud"
}

func (p *NewCloudProvider) Initialize(config map[string]any) error {
    // 初始化客户端
    return nil
}

func (p *NewCloudProvider) ListInstances(ctx context.Context, opts *provider.QueryOptions) ([]*model.Instance, error) {
    // 实现查询逻辑
    return nil, nil
}

// ... 实现其他接口方法
```

3. 在 `internal/provider/registry.go` 注册:

```go
func init() {
    Register("newcloud", newcloud.NewProvider())
}
```

4. 添加 CLI 命令 (可选)
5. 更新配置文件和文档

## 安全考虑

### 1. 凭证管理
- 敏感信息通过环境变量传递
- 支持 AWS Secrets Manager / 阿里云 KMS 等
- 不在日志中打印敏感信息

### 2. API 认证
- Token 认证 (Bearer Token)
- 支持 API Key 轮换
- 实现请求限流

### 3. 钉钉安全
- 验证请求签名
- 加密敏感响应内容
- 实现用户权限控制

### 4. 审计日志
- 记录所有查询操作
- 记录用户身份信息
- 支持日志导出

## 性能优化

### 1. 并发查询
- 跨云平台查询使用 goroutine 并发
- 实现超时控制
- 优雅降级 (部分平台失败不影响其他)

### 2. 缓存策略
- 实例列表缓存 (TTL: 5分钟)
- 实例详情缓存 (TTL: 1分钟)
- 支持手动刷新缓存

### 3. 分页查询
- 大量数据分页返回
- 支持流式查询

## 监控与运维

### 1. 健康检查
- `/health` 端点返回服务状态
- 检查各 Provider 连接状态

### 2. 指标暴露
- Prometheus 格式指标
- 查询耗时统计
- 错误率统计

### 3. 日志
- 结构化日志 (JSON)
- 支持日志级别动态调整
- 集成 ELK / Loki

## 部署方案

### 1. 单机部署
```bash
# 编译
make build

# 运行
./zenops serve --config configs/config.yaml
```

### 2. Docker 部署
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o zenops main.go

FROM alpine:latest
COPY --from=builder /app/zenops /usr/local/bin/
COPY configs /etc/zenops/
ENTRYPOINT ["zenops"]
CMD ["serve", "--config", "/etc/zenops/config.yaml"]
```

### 3. Kubernetes 部署
- 使用 ConfigMap 管理配置
- 使用 Secret 管理凭证
- 使用 Service 暴露 HTTP/MCP 端点

## 后续规划

### 短期 (3个月)
- [ ] 支持更多云平台 (AWS, 华为云)
- [ ] 实现资源变更通知
- [ ] 支持 GitLab CI/CD
- [ ] 增强钉钉对话交互 (卡片式交互)

### 中期 (6个月)
- [ ] 实现资源拓扑展示
- [ ] 成本分析功能
- [ ] 告警集成 (对接监控系统)
- [ ] Web 控制台

### 长期 (1年)
- [ ] 资源自动化操作 (重启、扩容等)
- [ ] AI 智能运维建议
- [ ] 多租户支持
- [ ] 资源编排能力

## 贡献指南

### 开发规范
- 遵循 Go 官方代码规范
- 单元测试覆盖率 > 70%
- 提交前运行 `make lint` 和 `make test`
- Commit 信息遵循 Conventional Commits

### 新增 Provider 流程
1. Fork 项目
2. 在 `internal/provider/` 创建新 Provider
3. 实现必需接口方法
4. 编写单元测试
5. 更新文档
6. 提交 Pull Request

## 参考资源

### 相关文档
- [Cobra 文档](https://cobra.dev/)
- [MCP 协议规范](https://modelcontextprotocol.io/)
- [钉钉开放平台](https://open.dingtalk.com/)
- [阿里云 SDK](https://help.aliyun.com/sdk)
- [腾讯云 SDK](https://cloud.tencent.com/document/sdk)

### 类似项目
- [Steampipe](https://steampipe.io/) - SQL 查询云资源
- [CloudQuery](https://www.cloudquery.io/) - 云资源数据导出
- [Infracost](https://www.infracost.io/) - 云成本分析

---

**项目状态**: 实现阶段 (Phase 2-5 已完成)
**最后更新**: 2025-12-09
**维护者**: @eryajf

## 更新记录

### 2025-12-09
- ✅ 完成腾讯云 Provider 实现 (CVM + CDB)
- ✅ 完成 Jenkins Provider 实现 (Job + Build)
- ✅ 完成腾讯云 CLI 命令 (`zenops query tencent`)
- ✅ 完成 Jenkins CLI 命令 (`zenops query jenkins`)
- ✅ 完成腾讯云 HTTP API 端点
- ✅ 完成 Jenkins HTTP API 端点
- ✅ 完成腾讯云 MCP Tools (6个工具)
- ✅ 完成 Jenkins MCP Tools (3个工具)
- ✅ MCP 服务器统一迁移到 mcp-go 库实现
- 📁 新增文件: `internal/server/mcp_tencent_handlers.go`
- 📁 新增文件: `internal/server/mcp_jenkins_handlers.go`

### 2025-12-08
- ✅ 完成项目基础框架搭建
- ✅ 完成阿里云 Provider 实现 (ECS + RDS)
- ✅ 完成阿里云 CLI 命令
- ✅ 完成阿里云 HTTP API
- ✅ 完成阿里云 MCP Tools
- ✅ 实现 MCP 服务器 (stdio + SSE 模式)
