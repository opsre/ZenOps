# 外部 MCP 集成使用指南

## 简介

ZenOps 支持集成外部 MCP (Model Context Protocol) 服务器,实现统一的运维能力入口。通过标准的 MCP 配置格式,您可以轻松接入:

- ✅ Python MCP 服务器 (如 mcp-jenkins)
- ✅ Node.js MCP 服务器 (如 @modelcontextprotocol/server-github)
- ✅ 远程 MCP 服务 (通过 HTTP/SSE)
- ✅ 其他 ZenOps 实例

## 快速开始

### 1. 准备配置文件

复制示例配置文件:

```bash
cp mcp_servers.example.json mcp_servers.json
```

### 2. 配置外部 MCP

编辑 `mcp_servers.json`,添加您的 MCP 服务器配置:

```json
{
  "mcpServers": {
    "my-mcp": {
      "isActive": true,
      "name": "my-mcp",
      "type": "stdio",
      "command": "python3",
      "args": ["/path/to/your/mcp-server.py"],
      "env": {
        "API_KEY": "your_api_key"
      },
      "toolPrefix": "my_",
      "autoRegister": true,
      "timeout": 300
    }
  }
}
```

### 3. 启用外部 MCP 功能

编辑 `config.yaml`:

```yaml
# 指向 MCP Servers 配置文件
mcp_servers_config: "./mcp_servers.json"

# 服务器配置
server:
  mcp:
    enabled: true
    port: 8081
    # 启用外部 MCP 工具自动注册
    auto_register_external_tools: true
```

### 4. 启动 ZenOps

```bash
./zenops run
```

启动日志示例:

```
🧘 Starting ZenOps Server, Version 1.0.0
📥 Loading external MCP servers from: ./mcp_servers.json
✅ Registered MCP server: my-mcp (stdio) with 5 tools
🔧 Registering external MCP tools...
✅ Registered 5 tools from MCP: my-mcp
🎉 Successfully registered 5 tools from 1 external MCP servers
🧰 Starting MCP Server In SSE Mode, Listening On 0.0.0.0:8081
```

### 5. 使用外部 MCP 工具

外部 MCP 的工具会自动注册到 ZenOps,并带有配置的前缀:

```bash
# 列出所有工具 (包括外部 MCP 的工具)
./zenops query --list-tools

# 内置工具
- search_ecs_by_ip
- list_rds
...

# 外部 MCP 工具 (带前缀)
- my_list_resources
- my_get_info
...
```

## 配置说明

### 标准字段 (兼容 Claude Desktop)

```json
{
  "mcpServers": {
    "server-name": {
      // === 必填字段 ===
      "isActive": true,              // 是否启用
      "name": "server-name",         // 服务器名称
      "type": "stdio",               // 传输类型: "stdio" | "sse" | "streamableHttp"

      // === Stdio 模式必填 ===
      "command": "python3",          // 执行命令
      "args": ["/path/to/server.py"], // 命令参数
      "env": {                       // 环境变量
        "KEY": "VALUE"
      },

      // === SSE / StreamableHttp 模式必填 ===
      "baseUrl": "http://...",       // 服务地址
      "headers": {                   // HTTP Headers (可选)
        "Authorization": "Bearer xxx"
      },

      // === 可选字段 ===
      "description": "...",          // 描述
      "provider": "...",             // 提供商
      "providerUrl": "...",          // 提供商网址
      "logoUrl": "...",              // Logo URL
      "tags": ["tag1", "tag2"],      // 标签
      "longRunning": true,           // 是否长期运行
      "timeout": 300,                // 超时时间(秒)
      "installSource": "manual",     // 安装来源

      // === ZenOps 扩展字段 ===
      "toolPrefix": "prefix_",       // 工具名前缀
      "autoRegister": true           // 是否自动注册工具
    }
  }
}
```

### 字段详解

#### `type` - 传输类型

- **stdio**: 通过标准输入输出通信,适合本地进程
  - 需要配置: `command`, `args`, `env`
  - 示例: Python/Node.js MCP 服务器
- **sse**: Server-Sent Events,适合远程 HTTP 服务(单向流)
  - 需要配置: `baseUrl`, `headers` (可选)
  - 示例: 远程部署的 MCP 服务
- **streamableHttp**: 流式 HTTP 传输,适合需要双向通信的远程服务
  - 需要配置: `baseUrl`, `headers` (可选)
  - 示例: 支持流式响应的远程 MCP 服务

#### `toolPrefix` - 工具名前缀

为避免工具名冲突,外部 MCP 的工具会添加前缀:

```
原始工具名: list_jobs
加上前缀:   jenkins_list_jobs
```

建议使用服务名称作为前缀,例如:
- `jenkins_`
- `github_`
- `k8s_`

#### `autoRegister` - 自动注册

- `true`: 启动时自动注册该 MCP 的所有工具到 ZenOps MCP Server
- `false`: 不自动注册,但仍可通过 API 手动调用

## 使用场景

### 场景 1: 集成 Python MCP (Jenkins)

```json
{
  "mcpServers": {
    "jenkins": {
      "isActive": true,
      "name": "jenkins",
      "type": "stdio",
      "command": "python3",
      "args": ["/opt/mcp-jenkins/server.py"],
      "env": {
        "JENKINS_URL": "https://jenkins.example.com",
        "JENKINS_USER": "admin",
        "JENKINS_API_TOKEN": "xxx"
      },
      "toolPrefix": "jenkins_",
      "autoRegister": true
    }
  }
}
```

可用工具:
- `jenkins_list_jobs`
- `jenkins_get_job`
- `jenkins_trigger_build`

### 场景 2: 集成 Node.js MCP (GitHub)

```json
{
  "mcpServers": {
    "github": {
      "isActive": true,
      "name": "github",
      "type": "stdio",
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"],
      "env": {
        "GITHUB_PERSONAL_ACCESS_TOKEN": "ghp_xxx"
      },
      "toolPrefix": "github_",
      "autoRegister": true
    }
  }
}
```

可用工具:
- `github_create_issue`
- `github_list_repos`
- `github_search_code`

### 场景 3: 连接远程 ZenOps 实例 (SSE)

```json
{
  "mcpServers": {
    "zenops-prod": {
      "isActive": true,
      "name": "zenops-prod",
      "type": "sse",
      "baseUrl": "http://zenops-prod:8081/sse",
      "headers": {
        "Authorization": "Bearer your_token"
      },
      "toolPrefix": "prod_",
      "autoRegister": true
    }
  }
}
```

聚合多个 ZenOps 实例的能力!

### 场景 3.5: 连接远程 MCP 服务 (StreamableHttp)

```json
{
  "mcpServers": {
    "remote-mcp": {
      "isActive": true,
      "name": "remote-mcp",
      "type": "streamableHttp",
      "baseUrl": "http://mcp-service.example.com:9090",
      "headers": {
        "Authorization": "Bearer your_token",
        "X-Custom-Header": "value"
      },
      "toolPrefix": "remote_",
      "autoRegister": true,
      "timeout": 600
    }
  }
}
```

适合需要双向流式通信的远程 MCP 服务!

### 场景 4: 自定义 MCP 服务

用任何语言实现自己的 MCP 服务器:

```json
{
  "mcpServers": {
    "my-custom": {
      "isActive": true,
      "type": "stdio",
      "command": "./my-mcp-server",
      "args": [],
      "toolPrefix": "custom_",
      "autoRegister": true
    }
  }
}
```

## 配置格式

支持两种配置格式:

### JSON 格式 (推荐,兼容 Claude Desktop)

```bash
# 使用 JSON 配置
mcp_servers_config: "./mcp_servers.json"
```

### YAML 格式

```yaml
# config.yaml 中直接配置
mcp_servers:
  jenkins:
    is_active: true
    name: "jenkins"
    type: "stdio"
    command: "python3"
    args:
      - "/path/to/server.py"
    env:
      JENKINS_URL: "..."
    tool_prefix: "jenkins_"
    auto_register: true
```

## 故障排查

### 问题 1: MCP 服务器无法启动

**症状**: 日志显示 "Failed to register MCP server"

**解决方法**:
1. 检查 `command` 路径是否正确
2. 确认命令有执行权限
3. 验证环境变量配置
4. 查看详细错误日志

```bash
# 手动测试 MCP 服务器
python3 /path/to/mcp-server.py
```

### 问题 2: 工具名冲突

**症状**: 日志显示 "tool name conflict"

**解决方法**:
修改 `toolPrefix`,确保每个 MCP 使用不同的前缀:

```json
{
  "toolPrefix": "unique_prefix_"
}
```

### 问题 3: 超时错误

**症状**: "failed to initialize client: context deadline exceeded"

**解决方法**:
增加 `timeout` 值:

```json
{
  "timeout": 600  // 增加到 10 分钟
}
```

### 问题 4: SSE 连接失败

**症状**: "failed to create sse client"

**解决方法**:
1. 确认 `baseUrl` 正确且可访问
2. 检查网络连接
3. 验证 Headers 配置 (如 Authorization)

```bash
# 测试 SSE 端点
curl -N http://your-mcp-server:8081/sse
```

## 安全建议

1. **保护敏感信息**: 不要将包含 API Key/Token 的配置文件提交到版本控制
2. **使用环境变量**: 敏感配置可以通过环境变量传递
3. **限制访问**: 使用 `auth` 配置保护 ZenOps API
4. **定期更新**: 及时更新外部 MCP 服务器版本

## 社区 MCP 服务器

以下是一些可用的开源 MCP 服务器:

- [mcp-jenkins](https://github.com/lanbaoshen/mcp-jenkins) - Jenkins CI/CD
- [@modelcontextprotocol/server-github](https://github.com/modelcontextprotocol/servers) - GitHub 集成
- [@modelcontextprotocol/server-filesystem](https://github.com/modelcontextprotocol/servers) - 文件系统访问
- 更多: https://github.com/modelcontextprotocol/servers

## 参考资料

- [Model Context Protocol 官方文档](https://modelcontextprotocol.io)
- [mcp-go GitHub](https://github.com/mark3labs/mcp-go)
- [ZenOps MCP 集成调研报告](./mcp-client-integration.md)

## 下一步

- 尝试集成更多社区 MCP 服务器
- 开发自定义 MCP 服务器
- 分享您的 MCP 配置到社区

欢迎反馈和贡献! 🎉
