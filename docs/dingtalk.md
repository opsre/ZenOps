# ZenOps 钉钉集成设计文档

## 概述

ZenOps 通过钉钉机器人提供智能化的运维查询能力,用户可以通过自然语言与机器人对话,查询云资源和 CI/CD 信息。

## 架构设计

### 整体流程

```
用户消息 → 钉钉服务器 → ZenOps回调接口 → AI意图识别 → MCP工具调用 → 流式返回结果
```

### 核心组件

1. **DingTalk Client** (`internal/dingtalk/client.go`)
   - 钉钉 API 客户端封装
   - 处理 OAuth 认证
   - 发送消息(普通消息和流式消息)

2. **Callback Handler** (`internal/dingtalk/callback.go`)
   - 处理钉钉回调请求
   - 验证签名和加密
   - 解析消息事件

3. **Message Handler** (`internal/dingtalk/handler.go`)
   - 消息分发和处理
   - 调用 AI 进行意图识别
   - 调用 MCP 工具执行查询

4. **Intent Parser** (`internal/dingtalk/intent.go`)
   - 解析用户意图
   - 提取查询参数
   - 映射到对应的 MCP 工具

5. **Stream Manager** (`internal/dingtalk/stream.go`)
   - 管理流式消息推送
   - 处理长文本分块
   - 确保消息顺序

## 功能特性

### 1. 消息接收

- ✅ 支持群聊 @机器人
- ✅ 支持私聊对话
- ✅ 验证消息签名
- ✅ 解密加密消息

### 2. 意图识别

支持的查询意图:

**阿里云查询:**
- "帮我查一下杭州的 ECS 实例"
- "IP 为 192.168.1.1 的服务器是哪个"
- "查询名为 web-server 的实例"
- "列出所有 RDS 数据库"

**腾讯云查询:**
- "查询腾讯云广州的 CVM"
- "找一下 IP 10.0.0.1 的腾讯云机器"
- "列出所有 CDB 数据库"

**Jenkins 查询:**
- "看一下 Jenkins 上的任务列表"
- "查询 deploy-prod 任务的构建历史"
- "最近的构建状态如何"

### 3. 流式响应

- ✅ 使用钉钉流式消息 API
- ✅ 实时推送查询进度
- ✅ 支持长文本分块发送
- ✅ Markdown 格式化输出

## 技术实现

### 1. 钉钉应用配置

需要在钉钉开放平台创建企业内部应用:

```yaml
应用信息:
  AppKey: ${DINGTALK_APP_KEY}
  AppSecret: ${DINGTALK_APP_SECRET}
  AgentId: ${DINGTALK_AGENT_ID}

回调配置:
  回调 URL: https://your-domain.com/api/v1/dingtalk/callback
  加密方式: AES 加密
  Token: ${DINGTALK_CALLBACK_TOKEN}
  AESKey: ${DINGTALK_AES_KEY}

权限配置:
  - 消息接收和发送
  - 企业通讯录只读
```

### 2. 消息签名验证

钉钉使用 HMAC-SHA256 签名:

```go
func verifySignature(timestamp, nonce, body, signature string) bool {
    message := timestamp + "\n" + nonce + "\n" + body
    mac := hmac.New(sha256.New, []byte(appSecret))
    mac.Write([]byte(message))
    expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))
    return expected == signature
}
```

### 3. 消息解密

使用 AES-256-CBC 解密:

```go
func decryptMessage(encryptedMsg string) (string, error) {
    // Base64 解码
    ciphertext, _ := base64.StdEncoding.DecodeString(encryptedMsg)

    // AES 解密
    block, _ := aes.NewCipher(aesKey)
    plaintext := make([]byte, len(ciphertext))
    mode := cipher.NewCBCDecrypter(block, iv)
    mode.CryptBlocks(plaintext, ciphertext)

    return string(plaintext), nil
}
```

### 4. 意图识别流程

使用关键词匹配和正则表达式:

```go
type Intent struct {
    Action   string // list, get, search
    Provider string // aliyun, tencent, jenkins
    Resource string // ecs, cvm, rds, cdb, job, build
    Params   map[string]string
}

func parseIntent(message string) (*Intent, error) {
    // 1. 识别云平台
    provider := detectProvider(message)

    // 2. 识别资源类型
    resource := detectResource(message)

    // 3. 识别操作类型
    action := detectAction(message)

    // 4. 提取参数
    params := extractParams(message, resource)

    return &Intent{
        Action:   action,
        Provider: provider,
        Resource: resource,
        Params:   params,
    }, nil
}
```

### 5. MCP 工具映射

```go
var intentToMCPTool = map[string]string{
    "aliyun_ecs_search_ip":   "search_ecs_by_ip",
    "aliyun_ecs_search_name": "search_ecs_by_name",
    "aliyun_ecs_list":        "list_ecs",
    "tencent_cvm_search_ip":  "search_cvm_by_ip",
    "jenkins_jobs_list":      "list_jenkins_jobs",
    // ... 更多映射
}
```

### 6. 流式消息推送

钉钉流式消息 API:

```go
func sendStreamMessage(conversationId, streamId, content string, finished bool) error {
    url := "https://oapi.dingtalk.com/chat/send/stream"

    payload := map[string]any{
        "conversation_id": conversationId,
        "stream_id":       streamId,
        "content":         content,
        "finished":        finished,
    }

    // 发送请求
    resp, err := client.Post(url, payload)
    return err
}
```

## 数据模型

### 回调消息结构

```go
type CallbackMessage struct {
    MsgId           string `json:"msgId"`
    MsgType         string `json:"msgtype"`
    CreateAt        int64  `json:"createAt"`
    ConversationId  string `json:"conversationId"`
    SenderId        string `json:"senderId"`
    SenderNick      string `json:"senderNick"`
    ChatbotUserId   string `json:"chatbotUserId"`
    IsAdmin         bool   `json:"isAdmin"`
    SessionWebhook  string `json:"sessionWebhook"`
    Text            *TextContent `json:"text,omitempty"`
    AtUsers         []AtUser     `json:"atUsers,omitempty"`
}

type TextContent struct {
    Content string `json:"content"`
}

type AtUser struct {
    DingtalkId string `json:"dingtalkId"`
    StaffId    string `json:"staffId"`
}
```

### 响应消息结构

```go
type Response struct {
    MsgType string      `json:"msgtype"`
    Text    *TextMsg    `json:"text,omitempty"`
    Markdown *MarkdownMsg `json:"markdown,omitempty"`
    ActionCard *ActionCardMsg `json:"actionCard,omitempty"`
}

type TextMsg struct {
    Content string `json:"content"`
}

type MarkdownMsg struct {
    Title string `json:"title"`
    Text  string `json:"text"`
}
```

## API 端点

### POST /api/v1/dingtalk/callback

处理钉钉回调消息

**请求头:**
```
Timestamp: 1234567890
Nonce: abc123
Signature: xxx
```

**请求体:**
```json
{
  "encrypt": "encrypted_message_content"
}
```

**响应:**
```json
{
  "msgtype": "text",
  "text": {
    "content": "收到您的消息,正在查询..."
  }
}
```

### POST /api/v1/dingtalk/webhook

主动发送消息的 Webhook(用于测试)

## 使用示例

### 1. 查询 ECS 实例

**用户:** @运维助手 帮我查一下杭州的 ECS 实例

**机器人响应:**
```markdown
📊 正在查询阿里云 ECS 实例...

✅ 查询完成,找到 5 台服务器:

1️⃣ **web-server-01**
   - 状态: Running
   - 规格: ecs.c6.large (2C4G)
   - 内网IP: 172.16.1.10
   - 公网IP: 47.98.123.45
   - 区域: cn-hangzhou

2️⃣ **db-server-01**
   - 状态: Running
   - 规格: ecs.g6.xlarge (4C16G)
   - 内网IP: 172.16.1.20
   - 区域: cn-hangzhou

...
```

### 2. 查询 Jenkins 构建

**用户:** @运维助手 看一下 deploy-prod 任务的最近构建

**机器人响应:**
```markdown
🔧 正在查询 Jenkins 构建历史...

✅ 查询完成,找到 10 个构建:

#128 - ✅ SUCCESS
   时间: 2025-12-09 15:30:45
   耗时: 3分15秒
   [查看详情](https://jenkins.example.com/job/deploy-prod/128)

#127 - ⚠️ UNSTABLE
   时间: 2025-12-09 14:20:30
   耗时: 3分08秒

...
```

## 安全考虑

### 1. 签名验证

- ✅ 验证每个请求的签名
- ✅ 检查时间戳防重放攻击
- ✅ 使用 HTTPS 传输

### 2. 消息加密

- ✅ 使用 AES-256 加密消息内容
- ✅ 定期更换密钥

### 3. 权限控制

- ✅ 验证用户身份
- ✅ 根据用户角色限制查询范围
- ✅ 记录审计日志

### 4. 限流保护

- ✅ 单用户请求频率限制
- ✅ 全局并发查询限制
- ✅ 防止滥用

## 部署配置

### 环境变量

```bash
# 钉钉应用配置
export DINGTALK_APP_KEY="dingxxxxxxxx"
export DINGTALK_APP_SECRET="xxxxxxxxxxxxxxxx"
export DINGTALK_AGENT_ID="123456789"

# 回调配置
export DINGTALK_CALLBACK_TOKEN="xxxxxxxx"
export DINGTALK_AES_KEY="xxxxxxxxxxxxxxxxxxxxxx"

# 回调 URL
export DINGTALK_CALLBACK_URL="https://your-domain.com/api/v1/dingtalk/callback"
```

### Nginx 配置

```nginx
location /api/v1/dingtalk/ {
    proxy_pass http://localhost:8080;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_read_timeout 300s;
}
```

## 错误处理

### 常见错误

1. **签名验证失败**
   - 检查 AppSecret 配置
   - 检查时间戳是否在有效范围内

2. **消息解密失败**
   - 检查 AESKey 配置
   - 确认加密模式正确

3. **工具调用失败**
   - 检查 Provider 配置
   - 查看日志获取详细错误

4. **流式推送失败**
   - 检查网络连接
   - 确认 ConversationId 有效

## 监控指标

- 消息接收总数
- 消息处理成功率
- 平均响应时间
- MCP 工具调用次数
- 错误率统计

## 未来优化

- [ ] 支持多轮对话上下文
- [ ] 接入更强大的 NLP 模型
- [ ] 支持图表和可视化展示
- [ ] 支持语音消息输入
- [ ] 智能推荐和预警
