# ZenOps 钉钉集成实现总结

## 实现概述

ZenOps 钉钉集成已完成,用户可以通过钉钉机器人进行自然语言交互,查询云资源和 CI/CD 信息,并通过流式消息获得实时反馈。

## 核心特性

### 1. ✅ 智能对话
- 自然语言意图识别
- 支持中文查询
- 自动映射到 MCP 工具调用

### 2. ✅ 流式响应
- 实时推送查询进度
- 大文本自动分块传输
- Markdown 格式化输出

### 3. ✅ 安全可靠
- HMAC-SHA256 签名验证
- AES-256-CBC 消息加解密
- 时间戳防重放攻击

### 4. ✅ 多云支持
- 阿里云 ECS/RDS 查询
- 腾讯云 CVM/CDB 查询
- Jenkins 任务和构建历史

## 技术架构

### 文件结构

```
internal/
├── server/
│   ├── dingtalk.go           # 钉钉客户端、加解密、消息处理 (430行)
│   ├── http.go                # HTTP 服务器(包含钉钉路由)
│   └── mcp_with_lib.go        # MCP 服务器
├── dingtalk/                  # 原计划的独立包(因循环依赖已废弃)
│   ├── client.go              # 钉钉客户端
│   ├── callback.go            # 回调加解密
│   ├── handler.go             # 消息处理
│   ├── intent.go              # 意图识别
│   └── stream.go              # 流式推送
└── config/
    └── config.go              # 配置结构(含 DingTalk 配置)

docs/
├── dingtalk.md                # 钉钉集成设计文档
└── dingtalk-implementation.md # 本文档
```

### 核心组件

#### 1. DingTalkClient (server/dingtalk.go)
**功能**: 钉钉 API 客户端
- AccessToken 自动管理
- 流式消息发送
- HTTP 请求封装

**关键方法**:
```go
func (c *DingTalkClient) GetAccessToken(ctx context.Context) (string, error)
func (c *DingTalkClient) SendStreamMessage(ctx context.Context, conversationID, streamID, content string, finished bool) error
```

#### 2. DingTalkCrypto (server/dingtalk.go)
**功能**: 消息加解密和签名验证
- HMAC-SHA256 签名验证
- AES-256-CBC 消息解密
- PKCS7 填充处理

**关键方法**:
```go
func (c *DingTalkCrypto) VerifySignature(timestamp, nonce, body, signature string) bool
func (c *DingTalkCrypto) DecryptMessage(encryptedMsg string) (*DingTalkMessage, error)
```

#### 3. ParseIntent (server/dingtalk.go)
**功能**: 用户意图识别
- 正则表达式模式匹配
- 参数提取
- MCP 工具映射

**支持的查询模式**:
```go
"IP 为 192.168.1.1"        → search_ecs_by_ip
"名称为 web-server"        → search_ecs_by_name
"查询阿里云 ECS"           → list_ecs
"列出腾讯云 CVM"           → list_cvm
"Jenkins 任务列表"         → list_jenkins_jobs
```

#### 4. DingTalkMessageHandler (server/dingtalk.go)
**功能**: 消息处理和 MCP 调用
- 异步查询处理
- 流式结果推送
- 错误处理

**处理流程**:
```
接收消息 → 提取内容 → 解析意图 → 调用 MCP → 流式返回
```

#### 5. HTTP 路由 (server/http.go)
**端点**:
- `POST /api/v1/dingtalk/callback` - 钉钉消息回调
- `POST /api/v1/dingtalk/webhook` - Webhook 测试
- `GET /api/v1/dingtalk/health` - 健康检查

## 配置说明

### 环境变量

```bash
# 钉钉应用配置
export DINGTALK_APP_KEY="dingxxxxxxxx"
export DINGTALK_APP_SECRET="xxxxxxxxxxxxxxxx"
export DINGTALK_AGENT_ID="123456789"

# 回调配置
export DINGTALK_CALLBACK_TOKEN="xxxxxxxx"
export DINGTALK_AES_KEY="xxxxxxxxxxxxxxxxxxxxxx"
```

### 配置文件 (configs/config.yaml)

```yaml
dingtalk:
  enabled: true  # 启用钉钉集成
  app_key: ${DINGTALK_APP_KEY}
  app_secret: ${DINGTALK_APP_SECRET}
  agent_id: ${DINGTALK_AGENT_ID}
  callback:
    token: ${DINGTALK_CALLBACK_TOKEN}
    aes_key: ${DINGTALK_AES_KEY}
    url: https://your-domain.com/api/v1/dingtalk/callback
```

## 使用指南

### 1. 钉钉应用创建

1. 登录钉钉开放平台
2. 创建企业内部应用
3. 配置权限: 消息接收和发送
4. 设置回调 URL: `https://your-domain.com/api/v1/dingtalk/callback`
5. 配置加密方式: AES 加密
6. 获取 AppKey, AppSecret, AgentID

### 2. 部署配置

```bash
# 1. 配置环境变量
export DINGTALK_APP_KEY="your_app_key"
export DINGTALK_APP_SECRET="your_app_secret"
export DINGTALK_AGENT_ID="your_agent_id"
export DINGTALK_CALLBACK_TOKEN="your_token"
export DINGTALK_AES_KEY="your_aes_key"

# 2. 启动服务
./bin/zenops server --mode=http

# 3. 配置 Nginx 反向代理
# location /api/v1/dingtalk/ {
#     proxy_pass http://localhost:8080;
# }
```

### 3. 使用示例

**查询 ECS 实例**:
```
用户: @运维助手 查询阿里云杭州的 ECS
机器人: 🔍 正在查询,请稍候...
机器人: ✅ 阿里云 ECS 查询完成

找到 3 台服务器:

服务器 1:
  实例 ID: i-bp1234567890abcde
  名称: web-server-01
  状态: Running
  规格: ecs.c6.large (2C4G)
  内网 IP: 172.16.1.10
  公网 IP: 47.98.123.45
  区域: cn-hangzhou
...
```

**查询 Jenkins 任务**:
```
用户: @运维助手 看一下 Jenkins 任务列表
机器人: 🔍 正在查询,请稍候...
机器人: ✅ Jenkins Job 查询完成

找到 5 个任务:

1. deploy-prod-web
   可构建: 是
   最后构建: #128

2. deploy-prod-api
   可构建: 是
   最后构建: #95
...
```

## 技术亮点

### 1. 避免循环依赖
原计划将钉钉功能独立为 `internal/dingtalk` 包,但因需要调用 `server.MCPServerWithLib` 导致循环依赖。

**解决方案**: 将钉钉相关代码直接放入 `server` 包,简化架构。

### 2. 流式消息优化
- 自动分块发送(1000字符/块)
- 防止发送过快(100ms 间隔)
- 按行分割保持完整性

### 3. 简化的意图识别
使用正则表达式实现快速模式匹配,无需引入复杂的 NLP 模型:
```go
patterns := []struct {
    regex   *regexp.Regexp
    tool    string
    extract func([]string) map[string]any
}{
    {regexp.MustCompile(`(?i)(IP|ip).*([\d\.]+)`), "search_ecs_by_ip", ...},
    ...
}
```

### 4. 异步处理模式
用户消息处理分为两阶段:
1. **同步**: 立即返回"正在查询"
2. **异步**: goroutine 执行实际查询并流式推送结果

避免钉钉回调超时(5秒限制)。

## 性能优化

### 1. AccessToken 缓存
- 内存缓存 AccessToken
- 提前 5 分钟刷新
- 读写锁保护

### 2. 并发安全
- 消息处理使用 goroutine
- Context 传递支持超时控制

### 3. 错误处理
- 签名验证失败返回 401
- 解密失败返回 500
- MCP 调用失败流式推送错误信息

## 安全考虑

### 1. 签名验证
```go
message := timestamp + "\n" + nonce + "\n" + body
mac := hmac.New(sha256.New, []byte(token))
mac.Write([]byte(message))
expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))
return hmac.Equal([]byte(expected), []byte(signature))
```

### 2. 时间戳检查
- 5 分钟内有效
- 防止重放攻击

### 3. 消息加密
- AES-256-CBC 加密
- 43 字符 Base64 编码密钥

## 已知限制

### 1. 意图识别
- 当前使用简单正则匹配
- 复杂查询可能无法识别
- **未来**: 可集成 Claude/GPT 进行语义理解

### 2. 上下文管理
- 不支持多轮对话
- 每次查询独立
- **未来**: 添加会话管理

### 3. 权限控制
- 未实现用户级权限
- 所有用户权限相同
- **未来**: 基于钉钉用户 ID 的权限系统

## 测试建议

### 1. 单元测试
```go
// 测试签名验证
func TestVerifySignature(t *testing.T) {
    crypto, _ := NewDingTalkCrypto("test_token", "test_key", "test_suite")
    valid := crypto.VerifySignature(timestamp, nonce, body, signature)
    assert.True(t, valid)
}

// 测试意图识别
func TestParseIntent(t *testing.T) {
    intent, _ := ParseIntent("查询阿里云杭州的 ECS")
    assert.Equal(t, "list_ecs", intent.MCPTool)
    assert.Equal(t, "cn-hangzhou", intent.Params["region"])
}
```

### 2. 集成测试
```bash
# 模拟钉钉回调
curl -X POST http://localhost:8080/api/v1/dingtalk/callback \
  -H "Timestamp: 1234567890" \
  -H "Nonce: abc123" \
  -H "Signature: xxx" \
  -d '{"encrypt":"..."}'
```

### 3. 端到端测试
1. 在钉钉创建测试群
2. @机器人发送测试消息
3. 验证响应内容和格式

## 编译和运行

```bash
# 编译
make build

# 运行(HTTP 模式)
./bin/zenops server --mode=http

# 运行(同时启用 HTTP 和 MCP)
./bin/zenops server

# 查看版本
./bin/zenops version
```

## 故障排查

### 问题 1: 签名验证失败
**原因**: Token 配置错误或时间戳不同步
**解决**:
- 检查 `DINGTALK_CALLBACK_TOKEN` 配置
- 确认服务器时间正确
- 查看日志中的 expected vs actual signature

### 问题 2: 解密失败
**原因**: AESKey 配置错误
**解决**:
- 确认 `DINGTALK_AES_KEY` 长度为 43 字符
- 检查是否包含了 Base64 padding(=)

### 问题 3: MCP 工具调用失败
**原因**: Provider 未初始化或配置错误
**解决**:
- 检查云服务商配置(AccessKey/SecretKey)
- 查看 MCP 服务器日志
- 确认 Provider enabled=true

### 问题 4: 流式消息未收到
**原因**: ConversationID 错误或网络问题
**解决**:
- 确认回调消息中的 conversation_id
- 检查网络连接
- 验证 AccessToken 有效性

## 监控指标

建议监控以下指标:
- 钉钉回调请求数
- 签名验证成功率
- MCP 工具调用延迟
- 流式消息推送成功率
- 错误类型分布

## 总结

ZenOps 钉钉集成实现了:
- ✅ 完整的钉钉机器人功能
- ✅ 智能意图识别
- ✅ MCP 工具集成
- ✅ 流式消息推送
- ✅ 安全的消息加解密
- ✅ 多云平台支持

**代码统计**:
- 核心代码: ~430 行 (server/dingtalk.go)
- 设计文档: ~550 行 (docs/dingtalk.md)
- 配置支持: 完整
- 编译状态: ✅ 成功

**下一步**:
1. 添加更多查询模式
2. 集成 LLM 进行语义理解
3. 实现会话管理
4. 添加用户权限控制
5. 性能优化和监控

---

**实现日期**: 2025-12-09
**版本**: v1.0.0
**状态**: 已完成,可投入使用
