# ZenOps MCP SSE 模式使用指南

## 概述

ZenOps 的 MCP (Model Context Protocol) 服务器现已支持两种模式:
- **stdio 模式** - 标准输入输出模式,适合本地进程通信
- **SSE 模式** - Server-Sent Events 模式,通过 HTTP 端口提供服务,适合远程访问

## SSE 模式优势

1. **远程访问** - 可以通过 HTTP 访问,不受本地进程限制
2. **多客户端** - 支持多个客户端同时连接
3. **易于调试** - 可以使用 curl 或浏览器直接测试
4. **防火墙友好** - 使用标准 HTTP 端口
5. **负载均衡** - 可以配合负载均衡器使用

## 配置 SSE 模式

### 1. 修改配置文件

编辑 `configs/config.yaml`:

```yaml
server:
  http:
    enabled: true
    port: 8080
    host: 0.0.0.0
  mcp:
    enabled: true
    mode: sse     # 设置为 sse 模式
    port: 8081    # SSE 服务监听端口
```

### 2. 启动 MCP SSE 服务器

```bash
# 启动 SSE 模式的 MCP 服务器
./bin/zenops server mcp

# 或者指定配置文件
./bin/zenops server mcp --config ./configs/config.yaml
```

服务器会在 `http://0.0.0.0:8081` 上启动,提供以下端点:
- `http://localhost:8081/sse` - SSE 连接端点
- `http://localhost:8081/message` - 消息发送端点

## SSE 模式架构

```
┌─────────────┐         HTTP/SSE         ┌──────────────────┐
│             │ ────────────────────────> │                  │
│  AI Client  │                           │  ZenOps MCP      │
│  (Claude)   │ <──────────────────────── │  SSE Server      │
│             │   Real-time Events        │  (Port 8081)     │
└─────────────┘                           └──────────────────┘
                                                    │
                                                    │ Query
                                                    ▼
                                          ┌──────────────────┐
                                          │  Aliyun API      │
                                          │  (ECS, RDS, etc) │
                                          └──────────────────┘
```

## 测试 SSE 连接

### 1. 测试 SSE 连接端点

```bash
# 使用 curl 连接到 SSE 端点
curl -N http://localhost:8081/sse

# 你会看到类似的输出:
# data: {"type":"connection","client_id":"client-1234567890","status":"connected"}
#
# : heartbeat
# : heartbeat
```

### 2. 测试消息端点

```bash
# 发送 initialize 请求
curl -X POST http://localhost:8081/message \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "initialize",
    "params": {}
  }'

# 响应示例:
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "protocolVersion": "2024-11-05",
    "capabilities": {
      "tools": {}
    },
    "serverInfo": {
      "name": "zenops",
      "version": "1.0.0"
    }
  }
}
```

### 3. 测试工具列表

```bash
curl -X POST http://localhost:8081/message \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "tools/list"
  }'
```

### 4. 测试工具调用

```bash
# 根据 IP 搜索 ECS 实例
curl -X POST http://localhost:8081/message \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 3,
    "method": "tools/call",
    "params": {
      "name": "search_ecs_by_ip",
      "arguments": {
        "ip": "10.8.11.171"
      }
    }
  }'
```

## 在 AI 客户端中配置

### Claude Desktop (SSE 模式)

1. 打开 Claude Desktop 配置文件:
   - macOS: `~/Library/Application Support/Claude/claude_desktop_config.json`
   - Windows: `%APPDATA%\Claude\claude_desktop_config.json`

2. 添加 SSE 配置:

```json
{
  "mcpServers": {
    "zenops": {
      "url": "http://localhost:8081/sse",
      "transport": {
        "type": "sse"
      }
    }
  }
}
```

3. 重启 Claude Desktop

### Cline (VS Code Extension)

1. 打开 VS Code
2. 打开 Cline 设置
3. 在 MCP 服务器配置中添加:

```json
{
  "name": "zenops",
  "url": "http://localhost:8081/sse",
  "transport": "sse"
}
```

## 使用示例

### 示例 1: 通过 AI 查询主机信息

**用户:**
```
帮我查一下阿里云上 10.8.11.171 这个 IP 的主机详细信息
```

**AI 会:**
1. 连接到 SSE 端点 `http://localhost:8081/sse`
2. 发送请求到 `http://localhost:8081/message` 调用 `search_ecs_by_ip` 工具
3. 接收实时响应并展示结果

**流程:**
```
AI → POST /message → { method: "tools/call", name: "search_ecs_by_ip", args: { ip: "10.8.11.171" } }
                   ↓
            ZenOps MCP Server
                   ↓
            Query Aliyun API
                   ↓
AI ← JSON Response ← 返回主机详细信息
```

### 示例 2: 列出所有 ECS 实例

**用户:**
```
列出所有阿里云 ECS 实例
```

**AI 会调用:**
```json
{
  "method": "tools/call",
  "params": {
    "name": "list_ecs",
    "arguments": {}
  }
}
```

## 监控和日志

### 查看连接状态

SSE 服务器会记录所有连接和请求:

```bash
# 启动时的日志
2025-12-08T15:05:10.123+0800  INFO  Starting MCP server in SSE mode  {"addr": "0.0.0.0:8081"}

# 客户端连接日志
2025-12-08T15:05:15.456+0800  INFO  SSE client connected  {"client_id": "client-1733650515456789000"}

# 请求日志
2025-12-08T15:05:20.789+0800  INFO  MCP SSE request  {"method": "POST", "path": "/message", "remote_addr": "127.0.0.1:54321"}
2025-12-08T15:05:20.790+0800  DEBUG Received MCP message  {"method": "tools/call"}
2025-12-08T15:05:21.123+0800  INFO  Calling tool  {"tool": "search_ecs_by_ip"}
2025-12-08T15:05:21.890+0800  INFO  MCP SSE response  {"method": "POST", "path": "/message", "duration": "1.101s"}

# 客户端断开日志
2025-12-08T15:10:15.456+0800  INFO  SSE client disconnected  {"client_id": "client-1733650515456789000"}
```

### 心跳检测

SSE 连接会每 30 秒发送一次心跳,确保连接保持活跃:

```
: heartbeat
```

## 端口和安全

### 默认端口

- HTTP API: `8080`
- MCP SSE: `8081`

### 安全建议

1. **内网部署** - 建议在内网环境中使用
2. **反向代理** - 生产环境建议使用 Nginx/Caddy 等反向代理
3. **HTTPS** - 通过反向代理启用 HTTPS
4. **认证** - 添加 API Token 或 Basic Auth
5. **限流** - 配置请求频率限制

### Nginx 反向代理示例

```nginx
server {
    listen 443 ssl http2;
    server_name mcp.example.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location /sse {
        proxy_pass http://localhost:8081/sse;
        proxy_http_version 1.1;
        proxy_set_header Connection "";
        proxy_set_header X-Real-IP $remote_addr;
        proxy_buffering off;
        proxy_cache off;
    }

    location /message {
        proxy_pass http://localhost:8081/message;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

## 故障排查

### 问题 1: 连接失败

**症状:** 无法连接到 SSE 端点

**检查:**
```bash
# 检查服务是否启动
ps aux | grep zenops

# 检查端口是否监听
netstat -an | grep 8081
# 或
lsof -i :8081

# 测试连接
curl -v http://localhost:8081/sse
```

### 问题 2: 连接断开

**症状:** SSE 连接频繁断开

**可能原因:**
- 防火墙超时设置
- 代理服务器超时
- 网络不稳定

**解决方案:**
- 检查心跳间隔设置 (默认 30 秒)
- 配置反向代理的超时时间
- 使用更稳定的网络环境

### 问题 3: 工具调用失败

**症状:** 工具调用返回错误

**检查:**
```bash
# 检查日志
./bin/zenops server mcp --log-level debug

# 手动测试工具调用
curl -X POST http://localhost:8081/message \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
      "name": "search_ecs_by_ip",
      "arguments": {"ip": "10.8.11.171"}
    }
  }' | jq
```

## 性能优化

### 1. 调整连接数

修改操作系统文件描述符限制:

```bash
# 临时修改
ulimit -n 10000

# 永久修改 /etc/security/limits.conf
* soft nofile 10000
* hard nofile 10000
```

### 2. 启用缓存

在配置文件中启用缓存:

```yaml
cache:
  enabled: true
  type: memory
  ttl: 300  # 缓存 5 分钟
```

### 3. 负载均衡

使用多个 ZenOps 实例 + 负载均衡器:

```
        ┌─────────────┐
        │   Nginx     │
        │Load Balancer│
        └─────────────┘
              │
    ┌─────────┼─────────┐
    │         │         │
┌───▼───┐ ┌──▼────┐ ┌──▼────┐
│ZenOps │ │ZenOps │ │ZenOps │
│  :8081│ │  :8082│ │  :8083│
└───────┘ └───────┘ └───────┘
```

## 对比 stdio 和 SSE 模式

| 特性 | stdio 模式 | SSE 模式 |
|------|-----------|----------|
| 通信方式 | 标准输入输出 | HTTP/SSE |
| 网络访问 | 仅本地 | 支持远程 |
| 多客户端 | 不支持 | 支持 |
| 调试难度 | 较难 | 容易 |
| 性能 | 更高 | 略低 |
| 部署复杂度 | 简单 | 中等 |
| 适用场景 | 本地开发/测试 | 生产环境/远程访问 |

## 总结

- ✅ SSE 模式提供 HTTP 端口访问
- ✅ 支持多客户端同时连接
- ✅ 易于测试和调试
- ✅ 适合生产环境部署
- ✅ 支持负载均衡和反向代理

现在你可以通过 HTTP 端口 (8081) 访问 MCP 服务器,让 AI 模型远程调用你的云资源查询工具! 🎉
