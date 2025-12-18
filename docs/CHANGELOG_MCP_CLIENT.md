# 外部 MCP 客户端集成 - 更新日志

## 🎉 新功能: 通用 MCP 客户端集成

### 概述

实现了通用的外部 MCP (Model Context Protocol) 客户端集成功能,支持接入任何标准 MCP 服务器。

### 核心特性

#### ✅ 完全兼容标准

- 配置格式兼容 Claude Desktop
- 支持标准 MCP 协议
- 可直接复用社区 MCP 配置

#### ✅ 语言无关

- **Python MCP**: `python3 server.py`
- **Node.js MCP**: `npx @mcp/server-xxx`
- **Go MCP**: `./mcp-server`
- **远程服务**: HTTP/SSE 连接

#### ✅ 零代码集成

添加新的 MCP 服务只需修改配置文件,无需编写代码:

```json
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
```

#### ✅ 自动工具代理

外部 MCP 的工具自动注册到 ZenOps MCP Server:

```
ZenOps 内置工具:
- search_ecs_by_ip
- list_rds
...

外部 MCP 工具 (自动注册):
- jenkins_list_jobs
- jenkins_get_job
- github_create_issue
...
```

### 代码变更

#### 新增文件

1. **internal/config/mcp_servers.go**
   - MCP Server 配置结构
   - 支持 JSON 和 YAML 格式
   - 配置校验逻辑

2. **internal/mcpclient/manager.go**
   - MCP 客户端管理器
   - 支持 Stdio 和 SSE 两种传输模式
   - 客户端生命周期管理

3. **internal/imcp/external.go**
   - 外部 MCP 工具代理
   - 自动工具注册
   - 工具名冲突检测

4. **mcp_servers.example.json**
   - 标准配置文件示例
   - 包含多种场景示例

5. **docs/external-mcp-integration.md**
   - 完整使用文档
   - 配置说明
   - 故障排查

#### 修改文件

1. **internal/config/config.go**
   - 添加 `MCPServersConfig` 字段
   - 添加 `AutoRegisterExternalTools` 配置

2. **cmd/root.go**
   - 集成 MCP 客户端管理器
   - 启动时加载外部 MCP
   - 优雅关闭时清理资源

3. **config.example.yaml**
   - 添加外部 MCP 配置说明
   - 添加示例配置

### 配置示例

#### config.yaml

```yaml
# 外部 MCP Servers 配置文件路径
mcp_servers_config: "./mcp_servers.json"

server:
  mcp:
    enabled: true
    port: 8081
    # 启用外部 MCP 工具自动注册
    auto_register_external_tools: true
```

#### mcp_servers.json

```json
{
  "mcpServers": {
    "jenkins": {
      "isActive": true,
      "type": "stdio",
      "command": "python3",
      "args": ["/path/to/mcp-jenkins/server.py"],
      "env": {
        "JENKINS_URL": "https://jenkins.example.com"
      },
      "toolPrefix": "jenkins_",
      "autoRegister": true,
      "timeout": 300
    }
  }
}
```

### 启动效果

```bash
$ ./zenops run

🧘 Starting ZenOps Server, Version 1.0.0
📥 Loading external MCP servers from: ./mcp_servers.json
✅ Registered MCP server: jenkins (stdio) with 5 tools
🔧 Registering external MCP tools...
✅ Registered 5 tools from MCP: jenkins
🎉 Successfully registered 5 tools from 1 external MCP servers
🧰 Starting MCP Server In SSE Mode, Listening On 0.0.0.0:8081
```

### 使用场景

1. **集成 Python MCP** (如 Jenkins)
2. **集成 Node.js MCP** (如 GitHub)
3. **连接远程 ZenOps 实例**
4. **自定义 MCP 服务**

### 技术优势

1. **复用开源生态**: 直接使用社区维护的 MCP 服务
2. **降低开发成本**: 无需重复开发相同功能
3. **统一接口**: 所有工具通过统一的 MCP 协议暴露
4. **易于扩展**: 添加新的 MCP 服务只需修改配置
5. **零侵入**: 不影响现有内置 Provider 的实现

### 文档

- [使用指南](./external-mcp-integration.md)
- [技术调研](./mcp-client-integration.md)

### 下一步

- [ ] 添加更多社区 MCP 集成示例
- [ ] 实现 MCP 健康检查和监控
- [ ] 支持动态加载/卸载 MCP
- [ ] 添加 MCP 管理 API

### 贡献者

- [@eryajf](https://github.com/eryajf)
- [@Claude](https://claude.ai) (设计和实现)

---

**更新时间**: 2025-12-18
**版本**: v0.2.0+
