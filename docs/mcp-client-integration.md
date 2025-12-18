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

完整的配置文件:

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

# 外部 MCP 配置
external_mcp:
  # Jenkins MCP (Python)
  - name: "jenkins"
    enabled: true
    type: "stdio"  # stdio | sse | http
    command: "python3"
    args:
      - "/opt/mcp-servers/mcp-jenkins/server.py"
    env:
      JENKINS_URL: "https://jenkins.example.com"
      JENKINS_USER: "admin"
      JENKINS_API_TOKEN: "xxx"
    # 工具名称映射 (可选)
    tool_prefix: "jenkins_"  # 工具名会变成 jenkins_list_jobs

  # GitHub MCP (Node.js)
  - name: "github"
    enabled: true
    type: "stdio"
    command: "npx"
    args:
      - "-y"
      - "@modelcontextprotocol/server-github"
    env:
      GITHUB_PERSONAL_ACCESS_TOKEN: "ghp_xxx"
    tool_prefix: "github_"

  # 自定义 MCP Server (HTTP/SSE 模式)
  - name: "custom-metrics"
    enabled: true
    type: "sse"
    url: "http://mcp-metrics-service:8080/sse"
    headers:
      Authorization: "Bearer xxx"
    tool_prefix: "metrics_"
    timeout: 30  # 秒

# 服务器配置
server:
  mcp:
    enabled: true
    port: 8081
    # 是否自动注册外部 MCP 的工具
    auto_register_external_tools: true
```

### 4.4 代码示例

**完整的外部 MCP Provider 实现:**

```go
// internal/provider/external/jenkins_mcp.go
package external

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/eryajf/zenops/internal/model"
    "github.com/eryajf/zenops/internal/provider"
    "github.com/mark3labs/mcp-go/client"
    "github.com/mark3labs/mcp-go/mcp"
)

type JenkinsMCPProvider struct {
    name       string
    client     *client.Client
    toolPrefix string
}

func NewJenkinsMCPProvider() provider.CICDProvider {
    return &JenkinsMCPProvider{
        name: "jenkins-mcp-external",
    }
}

func (p *JenkinsMCPProvider) GetName() string {
    return p.name
}

func (p *JenkinsMCPProvider) Initialize(config map[string]any) error {
    command := config["command"].(string)
    args := config["args"].([]string)
    env := config["env"].(map[string]string)

    // 转换环境变量格式
    envList := make([]string, 0, len(env))
    for k, v := range env {
        envList = append(envList, fmt.Sprintf("%s=%s", k, v))
    }

    // 创建 Stdio 客户端
    c, err := client.NewStdioMCPClient(command, envList, args...)
    if err != nil {
        return fmt.Errorf("failed to create MCP client: %w", err)
    }

    // 初始化
    ctx := context.Background()
    initReq := mcp.InitializeRequest{}
    initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
    initReq.Params.ClientInfo = mcp.Implementation{
        Name:    "zenops",
        Version: "1.0.0",
    }

    if _, err := c.Initialize(ctx, initReq); err != nil {
        c.Close()
        return fmt.Errorf("failed to initialize MCP client: %w", err)
    }

    p.client = c
    p.toolPrefix = config["tool_prefix"].(string)

    return nil
}

func (p *JenkinsMCPProvider) ListJobs(ctx context.Context, opts *provider.QueryOptions) ([]*model.Job, error) {
    // 调用外部 MCP 的工具
    callReq := mcp.CallToolRequest{}
    callReq.Params.Name = "list_jobs"
    callReq.Params.Arguments = map[string]any{}

    result, err := p.client.CallTool(ctx, callReq)
    if err != nil {
        return nil, fmt.Errorf("failed to call MCP tool: %w", err)
    }

    // 解析结果
    jobs, err := p.parseJobs(result)
    if err != nil {
        return nil, fmt.Errorf("failed to parse jobs: %w", err)
    }

    return jobs, nil
}

func (p *JenkinsMCPProvider) parseJobs(result *mcp.CallToolResult) ([]*model.Job, error) {
    // 从 MCP 结果中提取数据
    if len(result.Content) == 0 {
        return []*model.Job{}, nil
    }

    // 假设返回的是 TextContent 格式的 JSON
    textContent, ok := result.Content[0].(mcp.TextContent)
    if !ok {
        return nil, fmt.Errorf("unexpected content type")
    }

    var jobs []*model.Job
    if err := json.Unmarshal([]byte(textContent.Text), &jobs); err != nil {
        return nil, err
    }

    return jobs, nil
}

// 实现其他接口方法...
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

通过 mcp-go 客户端能力,ZenOps 完全可以集成外部 MCP 服务:

1. **技术可行**: mcp-go 提供了完整的 Client 实现,支持 Stdio、SSE、HTTP 多种传输方式
2. **架构清晰**: 通过 Provider 抽象,可以将外部 MCP 作为普通 Provider 集成
3. **实现简单**: 核心代码量不大,主要是配置管理和数据转换
4. **扩展性强**: 可以动态注册外部 MCP 的工具,实现统一的运维能力入口

**建议优先实施**: 先实现 Jenkins MCP 集成作为 PoC,验证方案可行性后再推广到其他 MCP 服务。
