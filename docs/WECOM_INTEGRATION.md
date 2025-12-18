# 企业微信智能机器人集成完成

## 概述

已成功为 ZenOps 项目集成企业微信智能机器人功能,与现有的钉钉和飞书机器人保持一致的架构设计。

## 实现内容

### 1. 核心文件

#### 新增文件

- **`internal/wecom/crypt.go`** - 企业微信消息加密解密工具
  - 实现 WXBizMsgCrypt 算法
  - 支持 AES-256-CBC 加密
  - 支持消息签名验证
  - PKCS7 填充/去填充

- **`internal/wecom/client.go`** - 企业微信AI机器人客户端
  - URL 验证功能
  - 消息解密
  - 流式响应生成

- **`internal/wecom/handler.go`** - 消息处理器
  - 文本消息处理
  - 流式轮询请求处理
  - LLM 集成
  - 对话状态管理
  - 消息缓存和清理

- **`docs/wecom-bot-setup.md`** - 详细的配置指南

#### 修改文件

- **`internal/config/config.go`** - 添加企业微信配置结构
  ```go
  type WecomConfig struct {
      Enabled        bool   `mapstructure:"enabled"`
      Token          string `mapstructure:"token"`
      EncodingAESKey string `mapstructure:"encoding_aes_key"`
  }
  ```

- **`internal/server/http.go`** - 集成企业微信路由
  - GET `/api/wecom/callback` - URL 验证
  - POST `/api/wecom/callback` - 消息回调

- **`config.example.yaml`** - 添加企业微信配置示例

### 2. 架构设计

```
用户消息
    ↓
企业微信服务器
    ↓ (加密消息)
ZenOps HTTP Server
    ↓
MessageHandler
    ↓
AIBotClient (解密)
    ↓
LLM 处理
    ↓
流式响应
    ↓
AIBotClient (加密)
    ↓
企业微信服务器
    ↓
用户收到回复
```

### 3. 关键特性

1. **消息加密安全**
   - AES-256-CBC 加密算法
   - SHA1 签名验证
   - 支持 URL 验证和消息回调

2. **流式响应**
   - 支持企业微信的流式消息协议
   - 异步消息处理
   - 对话状态管理

3. **LLM 集成**
   - 自动调用 LLM 进行智能问答
   - 支持工具调用(MCP)
   - 流式输出 AI 响应

4. **状态管理**
   - 消息 ID 缓存映射
   - 对话状态维护
   - 自动清理过期状态

### 4. 配置说明

在 `config.yaml` 中添加:

```yaml
# 企业微信配置
wecom:
  enabled: true
  token: "YOUR_WECOM_BOT_TOKEN"
  encoding_aes_key: "YOUR_ENCODING_AES_KEY"  # 43位字符

# 必须启用 LLM
llm:
  enabled: true
  model: "DeepSeek-V3"
  api_key: "YOUR_API_KEY"
  base_url: "https://api.deepseek.com"

# HTTP 服务必须启用
server:
  http:
    enabled: true
    port: 8080
```

### 5. API 接口

#### URL 验证 (GET)
```
GET /api/wecom/callback?msg_signature=xxx&timestamp=xxx&nonce=xxx&echostr=xxx
```

#### 消息回调 (POST)
```
POST /api/wecom/callback?msg_signature=xxx&timestamp=xxx&nonce=xxx
Content-Type: application/json

{
  "encrypt": "..."
}
```

### 6. 消息处理流程

#### 文本消息 (msgtype=text)
1. 接收用户消息
2. 生成对话 ID
3. 创建对话状态
4. 异步调用 LLM 处理
5. 返回初始响应 "正在思考..."

#### 流式轮询 (msgtype=stream)
1. 根据消息 ID 查找对话状态
2. 返回当前生成的内容
3. 如果完成,清理状态并添加结束标记

### 7. 代码示例

#### 初始化 Handler
```go
handler, err := wecom.NewMessageHandler(cfg, mcpServer)
if err != nil {
    log.Fatal(err)
}
```

#### 处理消息
```go
response, err := handler.HandleTextMessage(ctx, userReq)
if err != nil {
    log.Error(err)
}
```

### 8. 测试清单

- [x] 代码编译通过
- [ ] URL 验证测试
- [ ] 消息加密解密测试
- [ ] 文本消息处理测试
- [ ] 流式响应测试
- [ ] LLM 集成测试
- [ ] 端到端集成测试

### 9. 与钉钉/飞书的对比

| 功能 | 钉钉 | 飞书 | 企业微信 |
|------|------|------|----------|
| 连接方式 | Stream SDK | WebSocket | HTTP 回调 |
| 消息加密 | 选配 | 不需要 | 必须 |
| 流式响应 | AI卡片 | 卡片更新 | 流式协议 |
| 配置复杂度 | 中 | 低 | 高 |
| 安全性 | 高 | 中 | 很高 |

### 10. 下一步工作

1. **功能增强**
   - [ ] 支持群聊模式
   - [ ] 添加富文本消息支持
   - [ ] 支持图片消息

2. **性能优化**
   - [ ] 连接池优化
   - [ ] 缓存策略优化
   - [ ] 并发处理优化

3. **测试完善**
   - [ ] 单元测试
   - [ ] 集成测试
   - [ ] 压力测试

4. **文档完善**
   - [ ] API 文档
   - [ ] 部署文档
   - [ ] 故障排查指南

## 使用方法

### 1. 配置企业微信后台

1. 登录企业微信管理后台
2. 创建智能助手应用
3. 获取 Token 和 EncodingAESKey
4. 配置回调地址: `http://120.26.168.217:8080/api/wecom/callback`

### 2. 修改配置文件

编辑 `config.yaml`,启用企业微信和 LLM:

```yaml
wecom:
  enabled: true
  token: "your_token"
  encoding_aes_key: "your_aes_key"

llm:
  enabled: true
  model: "DeepSeek-V3"
  api_key: "your_api_key"
```

### 3. 启动服务

```bash
./zenops serve
```

### 4. 验证配置

在企业微信后台保存配置,如果成功会提示 "配置成功"。

### 5. 开始使用

在企业微信中找到智能助手,发送消息即可与机器人对话。

## 参考资料

- [企业微信智能助手开发文档](https://developer.work.weixin.qq.com/document/path/100719)
- [消息加密解密说明](https://developer.work.weixin.qq.com/document/path/90968)
- [PandaWiki 企微机器人实现](https://github.com/chaitin/panda-wiki)

## 贡献者

- 实现基于 PandaWiki 项目的企微机器人参考
- 加密算法参考企业微信官方 Python SDK

## 许可证

与 ZenOps 项目保持一致。
