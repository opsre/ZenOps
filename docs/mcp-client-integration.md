# ZenOps 集成外部 MCP 服务调研报告

## 一、调研背景

ZenOps 当前作为 MCP Server 提供运维资源查询能力。为了扩展平台功能,希望能够集成已有的开源 MCP 服务(如 Jenkins MCP、GitHub MCP 等),实现统一的运维资源查询入口。

## 二、当前架构分析

### 2.1 现有 MCP Server 实现

ZenOps 基于 `github.com/mark3labs/mcp-go` 库实现了 MCP Server:

**核心组件:**
- **MCP Server**: [internal/imcp/server.go](../internal/imcp/server.go) - 基于 mcp-go 实现的服务端
- **Provider 抽象**: [internal/provider/interface.go](../internal/provider/interface.go) - 统一的资源提供商接口
- **注册机制**: [internal/provider/registry.go](../internal/provider/registry.go) - Provider 注册和管理

**已实现的 Provider:**
- 阿里云 (ECS, RDS, OSS)
- 腾讯云 (CVM, CDB, COS)
- Jenkins (Job, Build)

**MCP Tools 注册流程:**
```go
// 1. 创建 MCP Server
mcpServer := server.NewMCPServer("zenops", "1.0.0")

// 2. 注册工具
mcpServer.AddTool(
    mcp.NewTool("search_ecs_by_ip", ...),
    handleSearchECSByIP,
)

// 3. 启动服务
mcpServer.StartSSE() // SSE 模式
// 或
server.ServeStdio(mcpServer) // Stdio 模式
```

### 2.2 支持的访问方式

1. **CLI 命令行**: `./zenops query ...`
2. **HTTP API**: RESTful 接口
3. **MCP 协议**: SSE 或 Stdio 传输
4. **智能机器人**: 钉钉、飞书、企业微信集成

## 三、MCP Client 集成方案

### 3.1 MCP 客户端基础

`mcp-go` 库同时提供了 Client 和 Server 能力:

**Client 创建方式:**

```go
import (
    "github.com/mark3labs/mcp-go/client"
    "github.com/mark3labs/mcp-go/mcp"
)

// 1. Stdio 传输 (适合本地进程通信)
c, err := client.NewStdioMCPClient(
    "python",                    // 命令
    []string{},                  // 环境变量
    "server.py",                 // 参数
)

// 2. HTTP/SSE 传输 (适合远程服务)
c, err := client.NewSSEMCPClient(
    "http://localhost:8080/sse",
    transport.WithHeaders(map[string]string{
        "Authorization": "Bearer token",
    }),
)

// 3. In-Process 传输 (同进程内通信)
c, err := client.NewInProcessClient(mcpServer)
```

**客户端使用流程:**

```go
// 1. 初始化客户端
ctx := context.Background()
initRequest := mcp.InitializeRequest{
    Params: mcp.InitializeRequestParams{
        ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
        ClientInfo: mcp.Implementation{
            Name:    "zenops",
            Version: "1.0.0",
        },
    },
}
serverInfo, err := c.Initialize(ctx, initRequest)

// 2. 列出可用工具
toolsResult, err := c.ListTools(ctx, mcp.ListToolsRequest{})

// 3. 调用工具
callRequest := mcp.CallToolRequest{
    Params: mcp.CallToolParams{
        Name: "list_jobs",
        Arguments: map[string]any{
            "filter": "active",
        },
    },
}
result, err := c.CallTool(ctx, callRequest)
```

### 3.2 集成 Python MCP 服务 (以 Jenkins MCP 为例)

**Python MCP 服务特点:**
- 大多数开源 MCP 服务使用 Python SDK (`mcp` 包) 开发
- 通过 Stdio 传输协议通信
- 需要 Python 运行环境

**集成方式 1: Stdio 子进程模式 (推荐)**

```go
// internal/provider/external/jenkins_mcp.go
package external

import (
    "context"
    "github.com/eryajf/zenops/internal/model"
    "github.com/eryajf/zenops/internal/provider"
    "github.com/mark3labs/mcp-go/client"
    "github.com/mark3labs/mcp-go/mcp"
)

// ExternalJenkinsMCPProvider 外部 Jenkins MCP 提供商
type ExternalJenkinsMCPProvider struct {
    name       string
    client     *client.Client
    serverPath string // Python 服务器脚本路径
}

func NewExternalJenkinsMCPProvider() provider.CICDProvider {
    return &ExternalJenkinsMCPProvider{
        name: "jenkins-mcp-external",
    }
}

func (p *ExternalJenkinsMCPProvider) Initialize(config map[string]any) error {
    serverPath := config["server_path"].(string) // 例: /path/to/mcp-jenkins/server.py

    // 创建 Stdio 客户端
    c, err := client.NewStdioMCPClient(
        "python",                // 或 "python3"
        []string{},              // 环境变量
        serverPath,              // server.py 路径
        // 传递给 Python 服务的参数
        "--jenkins-url", config["jenkins_url"].(string),
        "--jenkins-user", config["jenkins_user"].(string),
        "--jenkins-token", config["jenkins_token"].(string),
    )
    if err != nil {
        return err
    }

    p.client = c
    p.serverPath = serverPath

    // 初始化 MCP 客户端
    ctx := context.Background()
    initReq := mcp.InitializeRequest{}
    initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
    initReq.Params.ClientInfo = mcp.Implementation{
        Name:    "zenops",
        Version: "1.0.0",
    }

    _, err = c.Initialize(ctx, initReq)
    return err
}

func (p *ExternalJenkinsMCPProvider) ListJobs(ctx context.Context, opts *provider.QueryOptions) ([]*model.Job, error) {
    // 调用外部 MCP 服务的工具
    callReq := mcp.CallToolRequest{}
    callReq.Params.Name = "list_jobs" // 外部 MCP 提供的工具名
    callReq.Params.Arguments = map[string]any{
        // 根据外部 MCP 的要求传递参数
    }

    result, err := p.client.CallTool(ctx, callReq)
    if err != nil {
        return nil, err
    }

    // 解析结果并转换为内部模型
    jobs := parseJobsFromMCPResult(result)
    return jobs, nil
}

// 其他方法实现...
```

**配置文件示例:**

```yaml
# config.yaml
cicd:
  jenkins:
    enabled: false  # 禁用内置 Jenkins Provider

  # 外部 Jenkins MCP
  jenkins_mcp_external:
    enabled: true
    provider_type: "external_mcp"
    server_path: "/path/to/mcp-jenkins/server.py"
    jenkins_url: "https://jenkins.example.com"
    jenkins_user: "admin"
    jenkins_token: "YOUR_TOKEN"
```

**集成方式 2: HTTP/SSE 远程模式**

如果外部 MCP 服务部署为独立服务(通过 SSE 提供):

```go
func (p *ExternalJenkinsMCPProvider) Initialize(config map[string]any) error {
    mcpServerURL := config["mcp_server_url"].(string) // http://mcp-jenkins:8080/sse

    // 创建 SSE 客户端
    c, err := client.NewSSEMCPClient(
        mcpServerURL,
        transport.WithHeaders(map[string]string{
            "Authorization": "Bearer " + config["token"].(string),
        }),
    )
    if err != nil {
        return err
    }

    p.client = c

    // 初始化...
    return nil
}
```

### 3.3 MCP Tools 动态代理

为了让外部 MCP 的工具直接暴露给 ZenOps 的 MCP Server,可以实现动态代理:

```go
// internal/imcp/proxy.go
package imcp

import (
    "context"
    "github.com/mark3labs/mcp-go/client"
    "github.com/mark3labs/mcp-go/mcp"
)

// MCPClientProxy MCP 客户端代理
type MCPClientProxy struct {
    name   string
    client *client.Client
}

// RegisterExternalMCPTools 将外部 MCP 的工具注册到本地 MCP Server
func (s *MCPServer) RegisterExternalMCPTools(ctx context.Context, proxy *MCPClientProxy) error {
    // 1. 列出外部 MCP 的所有工具
    toolsResult, err := proxy.client.ListTools(ctx, mcp.ListToolsRequest{})
    if err != nil {
        return err
    }

    // 2. 为每个工具创建代理处理器
    for _, tool := range toolsResult.Tools {
        externalTool := tool // 捕获循环变量

        // 3. 注册到本地 MCP Server
        s.mcpServer.AddTool(
            // 添加前缀避免命名冲突
            mcp.NewTool(
                proxy.name+"_"+externalTool.Name,
                mcp.WithDescription(externalTool.Description),
                // 复制参数定义...
            ),
            // 代理处理器
            func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
                // 转发请求到外部 MCP
                proxyReq := mcp.CallToolRequest{
                    Params: mcp.CallToolParams{
                        Name:      externalTool.Name,
                        Arguments: request.Params.Arguments,
                    },
                }
                return proxy.client.CallTool(ctx, proxyReq)
            },
        )
    }

    return nil
}
```

**使用示例:**

```go
// cmd/root.go
func init() {
    // 创建本地 MCP Server
    mcpServer := imcp.NewMCPServer(cfg)

    // 连接外部 Jenkins MCP
    jenkinsClient, _ := client.NewStdioMCPClient("python", nil, "/path/to/mcp-jenkins/server.py")
    jenkinsProxy := &imcp.MCPClientProxy{
        name:   "jenkins_ext",
        client: jenkinsClient,
    }

    // 注册外部工具到本地 Server
    mcpServer.RegisterExternalMCPTools(context.Background(), jenkinsProxy)
}
```

这样,外部 MCP 的工具会以 `jenkins_ext_list_jobs`、`jenkins_ext_get_job` 等名称暴露。

## 四、实施方案

### 4.1 架构设计

```
┌─────────────────────────────────────────────────────────────┐
│                        ZenOps Platform                       │
│  ┌───────────────────────────────────────────────────────┐  │
│  │              ZenOps MCP Server (SSE/Stdio)           │  │
│  │  ┌─────────────────┐  ┌──────────────────────────┐  │  │
│  │  │  Internal Tools │  │   External MCP Proxies   │  │  │
│  │  │  - search_ecs   │  │  - jenkins_ext_*         │  │  │
│  │  │  - list_rds     │  │  - github_ext_*          │  │  │
│  │  │  - ...          │  │  - gitlab_ext_*          │  │  │
│  │  └─────────────────┘  └──────────────────────────┘  │  │
│  └────────────────────────────┬──────────────────────────┘  │
│                               │                              │
│  ┌────────────────────────────┴──────────────────────────┐  │
│  │              MCP Client Manager                       │  │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐           │  │
│  │  │ Client 1 │  │ Client 2 │  │ Client N │           │  │
│  │  │ (Stdio)  │  │  (SSE)   │  │ (Stdio)  │           │  │
│  │  └─────┬────┘  └─────┬────┘  └─────┬────┘           │  │
│  └────────┼─────────────┼─────────────┼─────────────────┘  │
└───────────┼─────────────┼─────────────┼────────────────────┘
            │             │             │
            ▼             ▼             ▼
    ┌──────────────┐ ┌──────────┐ ┌──────────────┐
    │ mcp-jenkins  │ │ mcp-github│ │ mcp-gitlab   │
    │ (Python)     │ │ (Node.js) │ │ (Python)     │
    └──────────────┘ └──────────┘ └──────────────┘
```

### 4.2 实施步骤

#### 阶段 1: 基础框架

1. **创建 MCP Client 管理器**
   ```go
   // internal/mcpclient/manager.go
   type Manager struct {
       clients map[string]*client.Client
       mu      sync.RWMutex
   }

   func (m *Manager) Register(name string, c *client.Client) error
   func (m *Manager) Get(name string) (*client.Client, error)
   func (m *Manager) Close(name string) error
   ```

2. **实现外部 MCP Provider 基类**
   ```go
   // internal/provider/external/base.go
   type ExternalMCPProvider struct {
       name   string
       client *client.Client
       config map[string]any
   }
   ```

3. **添加配置支持**
   ```yaml
   # config.yaml
   external_mcp:
     - name: "jenkins-mcp"
       type: "stdio"
       command: "python"
       args: ["/path/to/mcp-jenkins/server.py"]
       env:
         JENKINS_URL: "https://jenkins.example.com"
         JENKINS_USER: "admin"
         JENKINS_TOKEN: "token"

     - name: "github-mcp"
       type: "sse"
       url: "http://localhost:8081/sse"
       headers:
         Authorization: "Bearer token"
   ```

#### 阶段 2: 集成 Jenkins MCP

1. **实现 Jenkins MCP Provider**
   - 基于 Stdio 客户端
   - 实现 CICDProvider 接口
   - 工具映射和数据转换

2. **注册到系统**
   ```go
   // internal/provider/external/init.go
   func init() {
       provider.RegisterCICD("jenkins-mcp-external", NewJenkinsMCPProvider())
   }
   ```

3. **测试验证**
   - 单元测试
   - 集成测试
   - MCP 协议兼容性测试

#### 阶段 3: 工具代理功能

1. **实现动态工具注册**
   - 从外部 MCP 读取工具列表
   - 创建代理处理器
   - 注册到本地 MCP Server

2. **命名空间管理**
   - 工具名称前缀 (如 `jenkins_ext_`)
   - 避免命名冲突
   - 工具分组展示

#### 阶段 4: 更多 MCP 集成

- GitHub MCP
- GitLab MCP
- Kubernetes MCP
- Prometheus MCP
- 等...

### 4.3 配置示例

#### 方案 1: 标准 MCP 配置格式 (推荐)

完全兼容 Claude Desktop 等 MCP 客户端的配置格式:

```yaml
# config.yaml

# 内置 Provider
providers:
  aliyun:
    - name: "default"
      enabled: true
      ak: "xxx"
      sk: "xxx"

cicd:
  jenkins:
    enabled: false  # 使用外部 MCP 替代

# 标准 MCP Servers 配置 (兼容 Claude Desktop 格式)
mcp_servers:
  # Jenkins MCP (Python Stdio)
  jenkins:
    is_active: true
    name: "jenkins"
    type: "stdio"  # stdio | sse
    description: "Jenkins CI/CD Integration"
    command: "python3"
    args:
      - "/opt/mcp-servers/mcp-jenkins/server.py"
    env:
      JENKINS_URL: "https://jenkins.example.com"
      JENKINS_USER: "admin"
      JENKINS_API_TOKEN: "xxx"
    provider: "lanbaoshen"
    provider_url: "https://github.com/lanbaoshen/mcp-jenkins"
    logo_url: ""
    tags: ["cicd", "jenkins"]
    long_running: true
    timeout: 300
    # ZenOps 扩展配置
    tool_prefix: "jenkins_"  # 工具名前缀
    auto_register: true      # 是否自动注册工具到 ZenOps MCP Server

  # GitHub MCP (Node.js Stdio)
  github:
    is_active: true
    name: "github"
    type: "stdio"
    description: "GitHub Integration"
    command: "npx"
    args:
      - "-y"
      - "@modelcontextprotocol/server-github"
    env:
      GITHUB_PERSONAL_ACCESS_TOKEN: "ghp_xxx"
    provider: "modelcontextprotocol"
    provider_url: "https://github.com/modelcontextprotocol/servers"
    tags: ["github", "git"]
    long_running: true
    timeout: 300
    tool_prefix: "github_"
    auto_register: true

  # Kubernetes MCP (SSE 远程服务)
  kubernetes:
    is_active: true
    name: "kubernetes"
    type: "sse"
    description: "Kubernetes Cluster Management"
    base_url: "http://mcp-k8s-service:8080/sse"
    command: ""  # SSE 模式不需要
    args: []
    env: {}
    headers:
      Authorization: "Bearer xxx"
    provider: "custom"
    provider_url: ""
    tags: ["kubernetes", "k8s"]
    long_running: true
    timeout: 300
    tool_prefix: "k8s_"
    auto_register: true

  # Prometheus Monitoring (Stdio)
  prometheus:
    is_active: false  # 可以禁用
    name: "prometheus"
    type: "stdio"
    description: "Prometheus Metrics Query"
    command: "python3"
    args:
      - "/opt/mcp-servers/mcp-prometheus/server.py"
      - "--prom-url"
      - "http://prometheus:9090"
    env:
      PROM_AUTH_TOKEN: "xxx"
    tags: ["monitoring", "metrics"]
    long_running: true
    timeout: 60
    tool_prefix: "prom_"
    auto_register: true

# 服务器配置
server:
  mcp:
    enabled: true
    port: 8081
    # 全局配置
    auto_register_external_tools: true  # 全局开关
    tool_name_format: "{prefix}{name}"  # 工具命名格式
```

#### 方案 2: JSON 配置文件 (完全兼容 Claude Desktop)

也可以使用独立的 JSON 配置文件:

```json
// mcp_servers.json
{
  "mcpServers": {
    "jenkins": {
      "isActive": true,
      "name": "jenkins",
      "type": "stdio",
      "description": "Jenkins CI/CD Integration",
      "baseUrl": "",
      "command": "python3",
      "args": [
        "/opt/mcp-servers/mcp-jenkins/server.py"
      ],
      "env": {
        "JENKINS_URL": "https://jenkins.example.com",
        "JENKINS_USER": "admin",
        "JENKINS_API_TOKEN": "xxx"
      },
      "provider": "lanbaoshen",
      "providerUrl": "https://github.com/lanbaoshen/mcp-jenkins",
      "logoUrl": "",
      "tags": ["cicd", "jenkins"],
      "longRunning": true,
      "timeout": 300,
      "installSource": "manual",
      "toolPrefix": "jenkins_",
      "autoRegister": true
    },
    "github": {
      "isActive": true,
      "name": "github",
      "type": "stdio",
      "description": "GitHub Integration",
      "baseUrl": "",
      "command": "npx",
      "args": [
        "-y",
        "@modelcontextprotocol/server-github"
      ],
      "env": {
        "GITHUB_PERSONAL_ACCESS_TOKEN": "ghp_xxx"
      },
      "provider": "modelcontextprotocol",
      "providerUrl": "https://github.com/modelcontextprotocol/servers",
      "logoUrl": "",
      "tags": ["github", "git"],
      "longRunning": true,
      "timeout": 300,
      "installSource": "npm",
      "toolPrefix": "github_",
      "autoRegister": true
    },
    "zenops-remote": {
      "isActive": true,
      "name": "zenops-remote",
      "type": "sse",
      "description": "Remote ZenOps Instance",
      "baseUrl": "http://zenops-prod:8081/sse",
      "command": "",
      "args": [],
      "env": {},
      "headers": {
        "Authorization": "Bearer xxx"
      },
      "provider": "eryajf",
      "providerUrl": "https://github.com/opsre/zenops",
      "logoUrl": "https://raw.githubusercontent.com/opsre/ZenOps/main/src/zenops.png",
      "tags": ["zenops", "ops"],
      "longRunning": true,
      "timeout": 300,
      "installSource": "unknown",
      "toolPrefix": "remote_",
      "autoRegister": true
    }
  }
}
```

在 `config.yaml` 中引用:

```yaml
# config.yaml
mcp_servers_config: "./mcp_servers.json"  # 指向 JSON 配置文件
```

### 4.4 代码示例

#### 通用 MCP Server 配置结构

```go
// internal/config/mcp_servers.go
package config

import (
    "encoding/json"
    "os"
    "gopkg.in/yaml.v3"
)

// MCPServerConfig 标准 MCP Server 配置 (兼容 Claude Desktop 格式)
type MCPServerConfig struct {
    IsActive      bool              `yaml:"is_active" json:"isActive"`
    Name          string            `yaml:"name" json:"name"`
    Type          string            `yaml:"type" json:"type"` // "stdio" | "sse"
    Description   string            `yaml:"description" json:"description"`
    BaseURL       string            `yaml:"base_url" json:"baseUrl"`
    Command       string            `yaml:"command" json:"command"`
    Args          []string          `yaml:"args" json:"args"`
    Env           map[string]string `yaml:"env" json:"env"`
    Headers       map[string]string `yaml:"headers" json:"headers"` // 用于 SSE/HTTP
    Provider      string            `yaml:"provider" json:"provider"`
    ProviderURL   string            `yaml:"provider_url" json:"providerUrl"`
    LogoURL       string            `yaml:"logo_url" json:"logoUrl"`
    Tags          []string          `yaml:"tags" json:"tags"`
    LongRunning   bool              `yaml:"long_running" json:"longRunning"`
    Timeout       int               `yaml:"timeout" json:"timeout"`
    InstallSource string            `yaml:"install_source" json:"installSource"`

    // ZenOps 扩展字段
    ToolPrefix   string `yaml:"tool_prefix" json:"toolPrefix"`     // 工具名前缀
    AutoRegister bool   `yaml:"auto_register" json:"autoRegister"` // 是否自动注册
}

// MCPServersConfig MCP Servers 配置集合
type MCPServersConfig struct {
    MCPServers map[string]*MCPServerConfig `yaml:"mcp_servers" json:"mcpServers"`
}

// LoadMCPServersConfig 加载 MCP Servers 配置
func LoadMCPServersConfig(configPath string) (*MCPServersConfig, error) {
    data, err := os.ReadFile(configPath)
    if err != nil {
        return nil, err
    }

    var config MCPServersConfig

    // 根据文件扩展名判断格式
    if isJSON(configPath) {
        err = json.Unmarshal(data, &config)
    } else {
        err = yaml.Unmarshal(data, &config)
    }

    if err != nil {
        return nil, err
    }

    return &config, nil
}

func isJSON(filename string) bool {
    return strings.HasSuffix(filename, ".json")
}
```

#### 通用 MCP Client 管理器

```go
// internal/mcpclient/manager.go
package mcpclient

import (
    "context"
    "fmt"
    "sync"
    "time"

    "github.com/eryajf/zenops/internal/config"
    "github.com/mark3labs/mcp-go/client"
    "github.com/mark3labs/mcp-go/client/transport"
    "github.com/mark3labs/mcp-go/mcp"
    "cnb.cool/zhiqiangwang/pkg/logx"
)

// Manager MCP 客户端管理器
type Manager struct {
    clients map[string]*MCPClient
    mu      sync.RWMutex
}

// MCPClient MCP 客户端封装
type MCPClient struct {
    Config *config.MCPServerConfig
    Client *client.Client
    Tools  []mcp.Tool
}

// NewManager 创建管理器
func NewManager() *Manager {
    return &Manager{
        clients: make(map[string]*MCPClient),
    }
}

// LoadFromConfig 从配置加载所有 MCP 客户端
func (m *Manager) LoadFromConfig(cfg *config.MCPServersConfig) error {
    for name, serverCfg := range cfg.MCPServers {
        if !serverCfg.IsActive {
            logx.Info("Skip inactive MCP server: %s", name)
            continue
        }

        if err := m.Register(name, serverCfg); err != nil {
            logx.Error("Failed to register MCP server %s: %v", name, err)
            continue
        }
    }
    return nil
}

// Register 注册一个 MCP 客户端
func (m *Manager) Register(name string, cfg *config.MCPServerConfig) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    // 创建客户端
    c, err := m.createClient(cfg)
    if err != nil {
        return fmt.Errorf("failed to create client: %w", err)
    }

    // 初始化客户端
    ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Timeout)*time.Second)
    defer cancel()

    if err := m.initializeClient(ctx, c); err != nil {
        c.Close()
        return fmt.Errorf("failed to initialize client: %w", err)
    }

    // 获取工具列表
    tools, err := m.listTools(ctx, c)
    if err != nil {
        c.Close()
        return fmt.Errorf("failed to list tools: %w", err)
    }

    // 保存客户端
    m.clients[name] = &MCPClient{
        Config: cfg,
        Client: c,
        Tools:  tools,
    }

    logx.Info("✅ Registered MCP server: %s (%s) with %d tools",
        name, cfg.Type, len(tools))

    return nil
}

// createClient 根据配置创建客户端
func (m *Manager) createClient(cfg *config.MCPServerConfig) (*client.Client, error) {
    switch cfg.Type {
    case "stdio":
        return m.createStdioClient(cfg)
    case "sse":
        return m.createSSEClient(cfg)
    default:
        return nil, fmt.Errorf("unsupported MCP type: %s", cfg.Type)
    }
}

// createStdioClient 创建 Stdio 客户端
func (m *Manager) createStdioClient(cfg *config.MCPServerConfig) (*client.Client, error) {
    // 转换环境变量
    env := make([]string, 0, len(cfg.Env))
    for k, v := range cfg.Env {
        env = append(env, fmt.Sprintf("%s=%s", k, v))
    }

    // 创建 Stdio 客户端
    c, err := client.NewStdioMCPClient(
        cfg.Command,
        env,
        cfg.Args...,
    )
    if err != nil {
        return nil, err
    }

    return c, nil
}

// createSSEClient 创建 SSE 客户端
func (m *Manager) createSSEClient(cfg *config.MCPServerConfig) (*client.Client, error) {
    // 构建选项
    opts := []transport.ClientOption{}

    // 添加 Headers
    if len(cfg.Headers) > 0 {
        opts = append(opts, transport.WithHeaders(cfg.Headers))
    }

    // 创建 SSE 客户端
    c, err := client.NewSSEMCPClient(cfg.BaseURL, opts...)
    if err != nil {
        return nil, err
    }

    return c, nil
}

// initializeClient 初始化客户端
func (m *Manager) initializeClient(ctx context.Context, c *client.Client) error {
    initReq := mcp.InitializeRequest{}
    initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
    initReq.Params.ClientInfo = mcp.Implementation{
        Name:    "zenops",
        Version: "1.0.0",
    }
    initReq.Params.Capabilities = mcp.ClientCapabilities{}

    _, err := c.Initialize(ctx, initReq)
    return err
}

// listTools 获取工具列表
func (m *Manager) listTools(ctx context.Context, c *client.Client) ([]mcp.Tool, error) {
    toolsReq := mcp.ListToolsRequest{}
    result, err := c.ListTools(ctx, toolsReq)
    if err != nil {
        return nil, err
    }
    return result.Tools, nil
}

// Get 获取客户端
func (m *Manager) Get(name string) (*MCPClient, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()

    c, ok := m.clients[name]
    if !ok {
        return nil, fmt.Errorf("MCP client %s not found", name)
    }
    return c, nil
}

// List 列出所有客户端
func (m *Manager) List() []*MCPClient {
    m.mu.RLock()
    defer m.mu.RUnlock()

    clients := make([]*MCPClient, 0, len(m.clients))
    for _, c := range m.clients {
        clients = append(clients, c)
    }
    return clients
}

// CallTool 调用工具
func (m *Manager) CallTool(ctx context.Context, serverName, toolName string, args map[string]any) (*mcp.CallToolResult, error) {
    mcpClient, err := m.Get(serverName)
    if err != nil {
        return nil, err
    }

    callReq := mcp.CallToolRequest{}
    callReq.Params.Name = toolName
    callReq.Params.Arguments = args

    return mcpClient.Client.CallTool(ctx, callReq)
}

// Close 关闭客户端
func (m *Manager) Close(name string) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    c, ok := m.clients[name]
    if !ok {
        return fmt.Errorf("client %s not found", name)
    }

    c.Client.Close()
    delete(m.clients, name)

    logx.Info("Closed MCP client: %s", name)
    return nil
}

// CloseAll 关闭所有客户端
func (m *Manager) CloseAll() {
    m.mu.Lock()
    defer m.mu.Unlock()

    for name, c := range m.clients {
        c.Client.Close()
        logx.Info("Closed MCP client: %s", name)
    }
    m.clients = make(map[string]*MCPClient)
}
```

#### 集成到 ZenOps MCP Server

```go
// internal/imcp/external.go
package imcp

import (
    "context"
    "fmt"

    "github.com/eryajf/zenops/internal/mcpclient"
    "github.com/mark3labs/mcp-go/mcp"
    "cnb.cool/zhiqiangwang/pkg/logx"
)

// RegisterExternalMCPTools 将外部 MCP 的工具注册到 ZenOps MCP Server
func (s *MCPServer) RegisterExternalMCPTools(ctx context.Context, manager *mcpclient.Manager) error {
    // 遍历所有外部 MCP 客户端
    for _, mcpClient := range manager.List() {
        if !mcpClient.Config.AutoRegister {
            logx.Info("Skip auto-register for MCP: %s", mcpClient.Config.Name)
            continue
        }

        // 为每个工具创建代理
        for _, tool := range mcpClient.Tools {
            if err := s.registerProxyTool(ctx, mcpClient, tool); err != nil {
                logx.Error("Failed to register tool %s from %s: %v",
                    tool.Name, mcpClient.Config.Name, err)
                continue
            }
        }

        logx.Info("✅ Registered %d tools from MCP: %s",
            len(mcpClient.Tools), mcpClient.Config.Name)
    }

    return nil
}

// registerProxyTool 注册单个代理工具
func (s *MCPServer) registerProxyTool(ctx context.Context, mcpClient *mcpclient.MCPClient, tool mcp.Tool) error {
    // 构建工具名称 (带前缀)
    toolName := mcpClient.Config.ToolPrefix + tool.Name

    // 复制工具定义
    proxyTool := mcp.NewTool(
        toolName,
        mcp.WithDescription(fmt.Sprintf("[%s] %s", mcpClient.Config.Name, tool.Description)),
    )

    // 复制参数定义
    if tool.InputSchema != nil {
        proxyTool.InputSchema = tool.InputSchema
    }

    // 创建代理处理函数
    handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        // 转发请求到外部 MCP
        proxyReq := mcp.CallToolRequest{}
        proxyReq.Params.Name = tool.Name // 使用原始工具名
        proxyReq.Params.Arguments = request.Params.Arguments

        logx.Debug("Proxy call: %s -> %s.%s",
            toolName, mcpClient.Config.Name, tool.Name)

        // 调用外部 MCP
        result, err := mcpClient.Client.CallTool(ctx, proxyReq)
        if err != nil {
            return mcp.NewToolResultError(err.Error()), nil
        }

        return result, nil
    }

    // 注册到本地 MCP Server
    s.mcpServer.AddTool(proxyTool, handler)

    return nil
}
```

#### 启动时初始化

```go
// cmd/root.go
package cmd

import (
    "github.com/eryajf/zenops/internal/config"
    "github.com/eryajf/zenops/internal/imcp"
    "github.com/eryajf/zenops/internal/mcpclient"
    "cnb.cool/zhiqiangwang/pkg/logx"
)

func runServer(cfg *config.Config) error {
    // 1. 创建 MCP 客户端管理器
    mcpClientManager := mcpclient.NewManager()

    // 2. 加载外部 MCP 配置
    if cfg.MCPServersConfig != "" {
        mcpServersConfig, err := config.LoadMCPServersConfig(cfg.MCPServersConfig)
        if err != nil {
            logx.Warn("Failed to load MCP servers config: %v", err)
        } else {
            // 注册所有外部 MCP 客户端
            if err := mcpClientManager.LoadFromConfig(mcpServersConfig); err != nil {
                logx.Error("Failed to load MCP clients: %v", err)
            }
        }
    }

    // 3. 创建 ZenOps MCP Server
    mcpServer := imcp.NewMCPServer(cfg)

    // 4. 注册外部 MCP 的工具 (如果启用)
    if cfg.Server.MCP.AutoRegisterExternalTools {
        ctx := context.Background()
        if err := mcpServer.RegisterExternalMCPTools(ctx, mcpClientManager); err != nil {
            logx.Error("Failed to register external MCP tools: %v", err)
        }
    }

    // 5. 启动服务...
    return mcpServer.StartSSE()
}
```

#### 使用示例

**配置文件:**

```yaml
# config.yaml

# 指向外部 MCP Servers 配置文件
mcp_servers_config: "./mcp_servers.json"

# 服务器配置
server:
  mcp:
    enabled: true
    port: 8081
    auto_register_external_tools: true
```

**MCP Servers 配置文件:**

```json
// mcp_servers.json
{
  "mcpServers": {
    "jenkins": {
      "isActive": true,
      "name": "jenkins",
      "type": "stdio",
      "description": "Jenkins CI/CD Integration",
      "command": "python3",
      "args": ["/opt/mcp-jenkins/server.py"],
      "env": {
        "JENKINS_URL": "https://jenkins.example.com",
        "JENKINS_USER": "admin",
        "JENKINS_API_TOKEN": "xxx"
      },
      "toolPrefix": "jenkins_",
      "autoRegister": true,
      "longRunning": true,
      "timeout": 300
    },
    "github": {
      "isActive": true,
      "name": "github",
      "type": "stdio",
      "description": "GitHub Integration",
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"],
      "env": {
        "GITHUB_PERSONAL_ACCESS_TOKEN": "ghp_xxx"
      },
      "toolPrefix": "github_",
      "autoRegister": true,
      "longRunning": true,
      "timeout": 300
    }
  }
}
```

**启动后效果:**

```
🧘 Starting ZenOps Server, Version 1.0.0
✅ Registered MCP server: jenkins (stdio) with 5 tools
✅ Registered MCP server: github (stdio) with 12 tools
✅ Registered 5 tools from MCP: jenkins
✅ Registered 12 tools from MCP: github
🧰 Starting MCP Server In SSE Mode, Listening On 0.0.0.0:8081
```

**可用的工具列表:**

```
# ZenOps 内置工具
- search_ecs_by_ip
- list_ecs
- search_rds_by_name
...

# Jenkins MCP 工具 (带前缀)
- jenkins_list_jobs
- jenkins_get_job
- jenkins_trigger_build
...

# GitHub MCP 工具 (带前缀)
- github_create_issue
- github_list_repos
- github_search_code
...
```

## 五、优势与挑战

### 5.1 优势

1. **复用开源生态**: 直接使用社区维护的 MCP 服务
2. **降低开发成本**: 无需重复开发相同功能的 Provider
3. **语言无关**: 支持 Python、Node.js 等不同语言的 MCP 服务
4. **统一接口**: 所有工具通过统一的 MCP 协议暴露
5. **易于扩展**: 添加新的外部 MCP 只需配置

### 5.2 挑战

1. **依赖管理**: 需要管理外部 MCP 服务的运行环境 (Python/Node.js)
2. **进程管理**: Stdio 模式需要管理子进程生命周期
3. **错误处理**: 外部 MCP 故障时的降级和重试
4. **性能开销**: 多一层 MCP 协议通信
5. **数据映射**: 外部 MCP 的数据结构可能需要转换

### 5.3 最佳实践

1. **优先使用内置 Provider**: 对于核心功能,仍然使用 Go 原生实现
2. **外部 MCP 作为补充**: 用于快速集成非核心功能
3. **健康检查**: 定期检查外部 MCP 服务状态
4. **超时控制**: 设置合理的超时时间
5. **日志记录**: 详细记录外部 MCP 调用日志
6. **优雅降级**: 外部 MCP 不可用时不影响主服务

## 六、后续规划

### 6.1 短期目标

- [ ] 实现 MCP Client 管理器
- [ ] 集成第一个外部 MCP (Jenkins)
- [ ] 完善配置和文档

### 6.2 中期目标

- [ ] 支持更多外部 MCP 服务
- [ ] 实现动态工具注册和代理
- [ ] 添加监控和告警

### 6.3 长期目标

- [ ] MCP 服务市场/插件系统
- [ ] 可视化的 MCP 管理界面
- [ ] 自动发现和注册 MCP 服务

## 七、参考资料

- [Model Context Protocol 官方文档](https://modelcontextprotocol.io)
- [mcp-go GitHub](https://github.com/mark3labs/mcp-go)
- [MCP Servers 列表](https://github.com/modelcontextprotocol/servers)
- [mcp-jenkins](https://github.com/lanbaoshen/mcp-jenkins)

## 八、总结

### 8.1 核心价值

通过标准 MCP 配置格式,ZenOps 实现了真正通用的外部 MCP 集成方案:

#### ✅ 完全兼容标准

**配置格式兼容 Claude Desktop:**
```json
{
  "mcpServers": {
    "server-name": {
      "isActive": true,
      "type": "stdio | sse",
      "command": "...",
      "args": [...],
      "env": {...}
    }
  }
}
```

这意味着:
- ✅ 可以直接复用 Claude Desktop 的 MCP 配置
- ✅ 任何能在 Claude Desktop 运行的 MCP 都能在 ZenOps 运行
- ✅ 兼容社区所有标准 MCP 服务器

#### ✅ 语言无关

支持任何语言实现的 MCP 服务器:
- **Python**: `python3 server.py`
- **Node.js**: `npx @modelcontextprotocol/server-xxx`
- **Go**: `./mcp-server`
- **远程服务**: HTTP/SSE 连接

#### ✅ 真正通用

**不需要为每个 MCP 写专门的 Provider**:

传统方式(不推荐):
```go
// 需要为每个 MCP 写一个 Provider
type JenkinsMCPProvider struct {...}
type GitHubMCPProvider struct {...}
type K8sMCPProvider struct {...}
```

通用方案(推荐):
```go
// 一个通用的 MCP Client Manager 搞定所有
mcpClientManager.LoadFromConfig(config)
mcpServer.RegisterExternalMCPTools(mcpClientManager)
```

只需配置文件,无需写代码!

#### ✅ 开箱即用

```bash
# 1. 准备配置
cat > mcp_servers.json <<EOF
{
  "mcpServers": {
    "jenkins": {
      "isActive": true,
      "type": "stdio",
      "command": "python3",
      "args": ["/path/to/mcp-jenkins/server.py"],
      "env": {"JENKINS_URL": "..."},
      "toolPrefix": "jenkins_",
      "autoRegister": true
    }
  }
}
EOF

# 2. 启动 ZenOps
./zenops run

# 3. 外部 MCP 的工具自动可用
# - jenkins_list_jobs
# - jenkins_get_job
# - ...
```

### 8.2 技术优势

1. **技术可行**: mcp-go 提供了完整的 Client 实现,支持 Stdio、SSE、HTTP 多种传输方式
2. **架构清晰**: 通过 MCP Client Manager 统一管理所有外部 MCP
3. **实现简单**: 核心代码约 500 行,主要是配置解析和工具代理
4. **扩展性强**: 添加新的 MCP 服务只需修改配置文件
5. **零侵入**: 不影响现有内置 Provider 的实现
6. **自动代理**: 外部 MCP 的工具自动注册到 ZenOps MCP Server

### 8.3 使用场景

#### 场景 1: 快速集成社区 MCP

```json
{
  "mcpServers": {
    "github": {
      "type": "stdio",
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"],
      "env": {"GITHUB_PERSONAL_ACCESS_TOKEN": "xxx"}
    }
  }
}
```

瞬间获得 GitHub 查询能力!

#### 场景 2: 连接远程 ZenOps 实例

```json
{
  "mcpServers": {
    "zenops-prod": {
      "type": "sse",
      "baseUrl": "http://zenops-prod:8081/sse"
    }
  }
}
```

聚合多个 ZenOps 实例的能力!

#### 场景 3: 自定义 MCP 扩展

用任何语言实现自己的 MCP 服务器,然后:

```json
{
  "mcpServers": {
    "my-custom-ops": {
      "type": "stdio",
      "command": "./my-mcp-server"
    }
  }
}
```

无缝集成!

### 8.4 实施建议

**优先级:**

1. **Phase 1**: 实现通用 MCP Client Manager (核心框架)
2. **Phase 2**: 实现自动工具代理功能
3. **Phase 3**: 集成第一个外部 MCP (Jenkins) 作为 PoC
4. **Phase 4**: 文档和示例,推广使用

**开发工作量估算:**

- MCP Client Manager: 2-3 天
- 工具代理功能: 1-2 天
- 配置加载和集成: 1 天
- 测试和文档: 2 天

**总计: 约 1 周**

### 8.5 关键代码

整个方案核心就 3 个文件:

1. **配置结构** (`internal/config/mcp_servers.go`): 定义标准 MCP 配置
2. **客户端管理器** (`internal/mcpclient/manager.go`): 管理所有外部 MCP 客户端
3. **工具代理** (`internal/imcp/external.go`): 将外部工具注册到 ZenOps

**关键特性:**
- ✅ 支持 Stdio 和 SSE 两种传输模式
- ✅ 自动初始化和健康检查
- ✅ 工具自动发现和注册
- ✅ 工具名称前缀避免冲突
- ✅ 优雅的错误处理和日志

### 8.6 最终效果

用户视角:

```bash
# 1. 配置外部 MCP (就像配置 Claude Desktop 一样)
vim mcp_servers.json

# 2. 启动 ZenOps
./zenops run

# 3. 所有工具都可用了!
./zenops query --mcp

# 内置工具:
# - search_ecs_by_ip
# - list_rds
# - ...

# 外部 MCP 工具:
# - jenkins_list_jobs
# - github_create_issue
# - k8s_get_pods
# - ...
```

**一个平台,所有能力!** 🚀
