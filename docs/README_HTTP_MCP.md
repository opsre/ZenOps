# ZenOps HTTP API 和 MCP 使用指南

## 概述

ZenOps 现已支持两种服务模式:
1. **HTTP API 服务器** - 提供 RESTful API 接口
2. **MCP 服务器** - 基于 Model Context Protocol,可与 AI 大模型(如 Claude)集成

## 1. HTTP API 服务器

### 1.1 启动 HTTP 服务器

```bash
# 启动 HTTP 服务器
./bin/zenops server http

# 指定配置文件
./bin/zenops server http --config ./configs/config.yaml
```

默认监听地址: `http://0.0.0.0:8080`

### 1.2 API 接口列表

#### 健康检查
```bash
curl http://localhost:8080/api/v1/health
```

#### 列出所有 ECS 实例
```bash
# 使用默认账号
curl http://localhost:8080/api/v1/aliyun/ecs/list

# 指定账号
curl "http://localhost:8080/api/v1/aliyun/ecs/list?account=prod"

# 指定区域
curl "http://localhost:8080/api/v1/aliyun/ecs/list?region=cn-beijing"
```

#### 根据 IP 搜索 ECS 实例
```bash
# 搜索指定 IP 的主机
curl "http://localhost:8080/api/v1/aliyun/ecs/search?ip=10.8.11.171"

# 指定账号搜索
curl "http://localhost:8080/api/v1/aliyun/ecs/search?ip=10.8.11.171&account=prod"
```

#### 根据名称搜索 ECS 实例
```bash
curl "http://localhost:8080/api/v1/aliyun/ecs/search?name=web-server-01"
```

#### 获取 ECS 实例详情
```bash
curl "http://localhost:8080/api/v1/aliyun/ecs/get?instance_id=i-xxx"
```

#### 列出所有 RDS 实例
```bash
curl http://localhost:8080/api/v1/aliyun/rds/list
```

#### 根据名称搜索 RDS 实例
```bash
curl "http://localhost:8080/api/v1/aliyun/rds/search?name=mysql-prod"
```

### 1.3 响应格式

所有 API 返回统一的 JSON 格式:

```json
{
  "code": 200,
  "message": "Success",
  "data": {
    "total": 1,
    "instances": [...],
    "account": "default"
  }
}
```

## 2. MCP 服务器

### 2.1 什么是 MCP?

MCP (Model Context Protocol) 是一个标准协议,允许 AI 模型(如 Claude)调用外部工具。通过 MCP,大模型可以查询阿里云资源。

### 2.2 配置 MCP

#### 步骤 1: 修改环境变量

编辑 `mcp_config.json` 文件,填入你的阿里云凭证:

```json
{
  "mcpServers": {
    "zenops": {
      "command": "/path/to/zenops/bin/zenops",
      "args": ["server", "mcp", "--config", "/path/to/configs/config.yaml"],
      "env": {
        "ALIYUN_ACCESS_KEY_ID": "your_access_key_id",
        "ALIYUN_ACCESS_KEY_SECRET": "your_access_key_secret"
      }
    }
  }
}
```

#### 步骤 2: 在 Claude Desktop 中配置

1. 打开 Claude Desktop 配置文件:
   - macOS: `~/Library/Application Support/Claude/claude_desktop_config.json`
   - Windows: `%APPDATA%\Claude\claude_desktop_config.json`

2. 将 `mcp_config.json` 的内容添加到配置文件中

3. 重启 Claude Desktop

#### 步骤 3: 在 Cline (VS Code) 中配置

1. 打开 VS Code
2. 打开 Cline 设置
3. 在 MCP 服务器配置中添加 zenops
4. 重启 Cline

### 2.3 可用的 MCP 工具

ZenOps 提供以下 MCP 工具:

#### 1. `search_ecs_by_ip`
根据 IP 地址搜索阿里云 ECS 实例

**参数:**
- `ip` (必需): IP 地址(支持私网 IP 和公网 IP)
- `account` (可选): 阿里云账号名称

**示例对话:**
```
用户: 帮我查一下阿里云上 10.8.11.171 这个 IP 的主机详细信息

AI 将调用: search_ecs_by_ip(ip="10.8.11.171")
```

#### 2. `search_ecs_by_name`
根据实例名称搜索 ECS 实例

**参数:**
- `name` (必需): 实例名称
- `account` (可选): 阿里云账号名称

**示例对话:**
```
用户: 查一下名为 web-server-01 的主机信息

AI 将调用: search_ecs_by_name(name="web-server-01")
```

#### 3. `list_ecs`
列出所有 ECS 实例

**参数:**
- `account` (可选): 阿里云账号名称
- `region` (可选): 区域

**示例对话:**
```
用户: 列出所有阿里云 ECS 实例

AI 将调用: list_ecs()
```

#### 4. `get_ecs`
获取 ECS 实例详情

**参数:**
- `instance_id` (必需): 实例 ID
- `account` (可选): 阿里云账号名称

#### 5. `list_rds`
列出所有 RDS 数据库实例

**参数:**
- `account` (可选): 阿里云账号名称
- `region` (可选): 区域

**示例对话:**
```
用户: 列出所有 RDS 数据库

AI 将调用: list_rds()
```

#### 6. `search_rds_by_name`
根据名称搜索 RDS 实例

**参数:**
- `name` (必需): RDS 实例名称
- `account` (可选): 阿里云账号名称

### 2.4 使用示例

#### 示例 1: 查询指定 IP 的主机信息

**用户输入:**
```
帮我查一下阿里云上 10.8.11.171 这个 IP 的主机详细信息
```

**AI 响应:**
```
我已找到 IP 地址为 10.8.11.171 的主机信息:

**实例 ID:** i-2ze3abc123def456
**实例名称:** web-server-01
**区域:** cn-beijing
**可用区:** cn-beijing-a
**实例规格:** ecs.c6.large
**状态:** Running
**CPU:** 2 核
**内存:** 4096 MB
**操作系统:** CentOS 7.9 64位
**私网 IP:** 10.8.11.171
**公网 IP:** 120.77.123.45
**创建时间:** 2024-01-15T10:30:00Z
**账号:** default

这台主机目前状态正常,正在运行中。
```

#### 示例 2: 列出所有 ECS 实例

**用户输入:**
```
列出所有阿里云 ECS 实例,按区域分组
```

**AI 响应:**
```
已为您查询到所有 ECS 实例,共 15 台。按区域分组如下:

**cn-beijing (北京) - 8 台**
1. web-server-01 (Running) - 10.8.11.171
2. web-server-02 (Running) - 10.8.11.172
3. api-server-01 (Running) - 10.8.11.173
...

**cn-shanghai (上海) - 5 台**
1. db-proxy-01 (Running) - 10.10.20.10
2. cache-server-01 (Running) - 10.10.20.11
...

**cn-hangzhou (杭州) - 2 台**
1. backup-server-01 (Stopped) - 10.20.30.40
...
```

#### 示例 3: 查询数据库信息

**用户输入:**
```
查一下生产环境的 MySQL 数据库有哪些
```

**AI 响应:**
```
已为您查询到生产环境的 MySQL 数据库:

1. **mysql-prod-master**
   - 实例 ID: rm-2ze3abc123
   - 引擎: MySQL 8.0
   - 区域: cn-beijing
   - 状态: Running
   - 连接地址: rm-2ze3abc123.mysql.rds.aliyuncs.com:3306

2. **mysql-prod-slave**
   - 实例 ID: rm-2ze3def456
   - 引擎: MySQL 8.0
   - 区域: cn-beijing
   - 状态: Running
   - 连接地址: rm-2ze3def456.mysql.rds.aliyuncs.com:3306
```

## 3. 配置说明

### 3.1 配置文件

编辑 `configs/config.yaml`:

```yaml
# 服务配置
server:
  http:
    enabled: true   # 是否启用 HTTP 服务器
    port: 8080      # HTTP 端口
    host: 0.0.0.0   # 监听地址
  mcp:
    enabled: true   # 是否启用 MCP 服务器
    mode: stdio     # MCP 模式 (仅支持 stdio)

# 云服务提供商配置
providers:
  aliyun:
    - name: default
      enabled: true
      access_key_id: ${ALIYUN_ACCESS_KEY_ID}
      access_key_secret: ${ALIYUN_ACCESS_KEY_SECRET}
      regions:
        - cn-hangzhou
        - cn-beijing
    - name: prod
      enabled: true
      access_key_id: ${ALIYUN_PROD_ACCESS_KEY_ID}
      access_key_secret: ${ALIYUN_PROD_ACCESS_KEY_SECRET}
      regions:
        - cn-shanghai
```

### 3.2 环境变量

可以通过环境变量配置阿里云凭证:

```bash
export ALIYUN_ACCESS_KEY_ID="your_access_key_id"
export ALIYUN_ACCESS_KEY_SECRET="your_access_key_secret"
```

## 4. 测试

### 4.1 测试 HTTP API

```bash
# 1. 启动 HTTP 服务器
./bin/zenops server http &

# 2. 测试健康检查
curl http://localhost:8080/api/v1/health

# 3. 测试 IP 搜索
curl "http://localhost:8080/api/v1/aliyun/ecs/search?ip=10.8.11.171"

# 4. 停止服务器
pkill -f "zenops server http"
```

### 4.2 测试 MCP 工具

使用支持 MCP 的客户端(如 Claude Desktop 或 Cline)测试:

1. 配置 MCP 服务器
2. 向 AI 发送查询请求
3. 验证 AI 能否正确调用工具并返回结果

## 5. 故障排查

### 5.1 HTTP 服务器无法启动

- 检查端口是否被占用: `lsof -i :8080`
- 检查配置文件是否正确
- 查看日志输出

### 5.2 MCP 工具调用失败

- 检查阿里云凭证是否正确
- 检查 MCP 配置文件路径
- 查看 zenops 日志输出
- 确认账号有查询 ECS/RDS 的权限

### 5.3 查询返回空结果

- 确认指定的 IP/名称确实存在
- 检查是否指定了正确的账号名称
- 确认账号配置中包含了实例所在的区域

## 6. 安全建议

1. **不要在代码中硬编码凭证** - 使用环境变量或密钥管理服务
2. **限制 HTTP 服务器访问** - 使用防火墙或反向代理
3. **使用只读权限** - 阿里云 AccessKey 应只有查询权限
4. **定期轮换凭证** - 定期更新 AccessKey
5. **监控 API 调用** - 记录和审计所有 API 请求

## 7. 后续扩展

可以继续扩展的功能:
- 添加更多云服务商(腾讯云、AWS 等)
- 支持更多资源类型(SLB、VPC、OSS 等)
- 添加资源监控和告警
- 实现资源变更操作(启动、停止、重启等)
- 支持 Webhook 通知
