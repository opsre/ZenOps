# 飞书机器人集成说明

## 概述

ZenOps 现已支持飞书机器人集成,可以通过飞书与云平台进行智能交互。

## 核心特性

### 1. WebSocket 长连接
- 使用飞书官方 Stream 模式
- 稳定可靠,无需公网 IP
- 自动重连机制

### 2. LLM 智能对话
- 支持自然语言交互
- 集成大语言模型(LLM)
- 智能理解用户意图

### 3. MCP 工具调用
- 自动调用云平台 API
- 支持阿里云、腾讯云等
- 支持 Jenkins 等 CI/CD 工具

### 4. 消息去重
- 防止重复处理
- 自动清理过期消息
- 内存高效管理

## 架构设计

```
┌─────────────┐
│  飞书用户   │
└──────┬──────┘
       │ 发送消息
       ↓
┌─────────────┐
│  飞书服务器 │
└──────┬──────┘
       │ WebSocket 推送
       ↓
┌──────────────────────────┐
│  ZenOps 飞书 Stream 服务  │
│  ┌────────────────────┐  │
│  │  消息去重          │  │
│  └────────┬───────────┘  │
│           ↓               │
│  ┌────────────────────┐  │
│  │  消息处理器        │  │
│  └────┬───────────┬───┘  │
│       │           │       │
│       ↓           ↓       │
│  ┌────────┐ ┌──────────┐ │
│  │  LLM   │ │   MCP    │ │
│  │ Client │ │  Server  │ │
│  └────────┘ └──────────┘ │
└──────────────────────────┘
       │
       ↓ 返回结果
┌─────────────┐
│  飞书用户   │
└─────────────┘
```

## 实现细节

### 1. 文件结构

```
internal/
├── feishu/
│   ├── client.go       # 飞书 API 客户端
│   │   - Token 管理
│   │   - 消息发送
│   │   - API 调用封装
│   │
│   └── handler.go      # 消息处理器
│       - 消息解析
│       - LLM 调用
│       - 帮助信息
│
├── server/
│   └── feishu_stream.go  # 飞书 Stream 服务
│       - WebSocket 连接
│       - 事件分发
│       - 消息去重
│
└── config/
    └── config.go       # 配置定义
        - FeishuConfig
```

### 2. 关键组件

#### FeishuStreamServer
- 管理 WebSocket 连接
- 接收飞书事件推送
- 消息去重和清理
- 异步消息处理

#### MessageHandler
- 解析消息内容
- 处理特殊命令(帮助等)
- 调用 LLM 生成回复
- 格式化输出

#### Client
- 封装飞书 API
- Token 自动刷新
- 发送各类消息(文本、Markdown、卡片)

### 3. 消息流程

```
1. 用户发送消息
   ↓
2. 飞书服务器推送事件(WebSocket)
   ↓
3. FeishuStreamServer 接收事件
   ↓
4. 检查消息去重
   ↓
5. 异步调用 MessageHandler
   ↓
6. 解析消息内容
   ↓
7. 判断消息类型
   ├─ 帮助命令 → 返回帮助信息
   ├─ LLM 对话 → 调用 LLM Client
   └─ 其他 → 默认处理
   ↓
8. 发送回复给用户
```

## 与钉钉实现的对比

| 特性 | 钉钉实现 | 飞书实现 |
|------|---------|---------|
| 连接方式 | Stream SDK | WebSocket (官方 SDK) |
| 消息接收 | Stream 回调 | 事件订阅 |
| 卡片支持 | AI 流式卡片 | 交互式卡片(未实现) |
| 用户信息 | 自动获取 | 可选获取 |
| Token 管理 | 自动刷新 | 自动刷新 |
| 配置项 | 5 项 | 3 项(更简洁) |

## 配置说明

### 最小配置

```yaml
feishu:
  enabled: true
  app_id: "cli_xxxxxxxxxxxxxxxx"
  app_secret: "xxxxxxxxxxxxxxxxxxxxx"
  enable_llm_conversation: true

llm:
  enabled: true
  model: "DeepSeek-V3"
  api_key: "YOUR_API_KEY"
```

### 完整配置

```yaml
feishu:
  enabled: true                     # 是否启用飞书机器人
  app_id: "cli_xxxxxxxxxxxxxxxx"    # 飞书应用 App ID
  app_secret: "xxxxxxxxxxxxx"       # 飞书应用 App Secret
  enable_llm_conversation: true     # 是否启用 LLM 对话

llm:
  enabled: true                     # 是否启用 LLM
  model: "DeepSeek-V3"             # 模型名称
  api_key: "YOUR_API_KEY"          # API 密钥
  base_url: ""                     # 自定义端点(可选)

# 云平台配置(按需)
providers:
  aliyun:
    - name: "default"
      enabled: true
      ak: "YOUR_AK"
      sk: "YOUR_SK"
      regions: ["cn-hangzhou"]
```

## 性能特点

### 资源占用
- 内存: ~50-100MB(空闲)
- CPU: < 5%(消息处理时)
- 网络: WebSocket 长连接,带宽占用极低

### 响应性能
- 消息接收延迟: < 100ms
- 简单回复延迟: < 1s
- LLM 回复延迟: 3-30s(取决于模型)

### 并发能力
- 支持多用户同时交互
- 消息处理异步化
- 无阻塞设计

## 安全考虑

### 1. 凭证保护
- App Secret 不写入日志
- 建议使用环境变量
- Token 仅内存缓存

### 2. 消息验证
- 消息去重防止重放
- 仅处理文本消息
- 忽略异常格式

### 3. 错误处理
- 所有 API 调用有错误处理
- 异常不影响主服务
- 友好的错误提示

## 扩展性

### 支持的扩展

1. **自定义命令**
   - 在 `handler.go` 中添加命令处理
   - 示例: 添加 `统计` 命令

2. **消息类型**
   - 当前仅支持文本
   - 可扩展支持图片、文件等

3. **交互式卡片**
   - 预留了卡片接口
   - 可实现复杂交互

4. **群组管理**
   - 可添加群组权限控制
   - 可实现群组白名单

### 未来计划

- [ ] 支持飞书 AI 流式卡片
- [ ] 支持图片、文件消息
- [ ] 支持按钮交互
- [ ] 支持群组权限管理
- [ ] 支持会话上下文
- [ ] 支持多轮对话

## 依赖

### Go 依赖

```go
github.com/larksuite/oapi-sdk-go/v3  // 飞书官方 SDK
```

### 外部服务

- 飞书开放平台
- LLM API 服务(OpenAI 兼容)
- 云平台 API(可选)

## 兼容性

- Go 版本: >= 1.21
- 飞书版本: 企业版/标准版
- 操作系统: Linux/macOS/Windows

## 限制

### 飞书平台限制
- 消息发送频率: 100 次/分钟(企业应用)
- WebSocket 连接: 长期稳定
- Token 有效期: 2 小时

### ZenOps 限制
- 消息去重时间: 5 分钟
- 仅支持文本消息
- 暂不支持多轮对话上下文

## 监控建议

### 关键指标

1. **连接状态**
   ```bash
   grep "Feishu Stream server started" zenops.log
   ```

2. **消息处理**
   ```bash
   grep "Processing message" zenops.log | wc -l
   ```

3. **错误率**
   ```bash
   grep "error\|failed" zenops.log | grep feishu
   ```

4. **LLM 调用**
   ```bash
   grep "LLM" zenops.log
   ```

### 告警阈值

- WebSocket 断连 > 3 次/小时
- 消息处理失败率 > 5%
- LLM 调用失败率 > 10%
- 内存占用 > 500MB

## 故障排查

### 常见问题

1. **WebSocket 连接失败**
   - 检查网络连接
   - 验证 App ID 和 Secret
   - 确认应用已发布

2. **消息无响应**
   - 检查事件订阅配置
   - 确认应用权限
   - 查看日志错误

3. **LLM 调用失败**
   - 验证 API Key
   - 检查网络连接
   - 确认配额限制

### 调试命令

```bash
# 查看飞书相关日志
tail -f zenops.log | grep feishu

# 查看 WebSocket 连接
netstat -an | grep ESTABLISHED

# 查看进程状态
ps aux | grep zenops

# 查看内存占用
top -p $(pidof zenops)
```

## 文档

- [飞书接入指南](./feishu-setup-guide.md) - 详细的配置和部署文档
- [测试清单](./feishu-test-checklist.md) - 完整的功能测试清单
- [飞书开放平台](https://open.feishu.cn/) - 官方文档

## 更新日志

### v1.0 (2025-12-16)
- ✅ 初始版本发布
- ✅ 支持 WebSocket 长连接
- ✅ 支持 LLM 智能对话
- ✅ 支持 MCP 工具调用
- ✅ 支持消息去重
- ✅ 支持帮助命令
- ✅ 完整的文档和测试清单

## 贡献

欢迎提交 Issue 和 Pull Request!

---

**维护者**: ZenOps Team
**更新时间**: 2025-12-16
