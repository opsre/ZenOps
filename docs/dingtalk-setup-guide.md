# 钉钉机器人流式输出配置指南

本指南说明如何配置 ZenOps 钉钉机器人以支持流式输出功能。

## 一、钉钉开放平台配置

### 1.1 创建机器人应用

1. 登录 [钉钉开放平台](https://open.dingtalk.com/)
2. 进入"应用开发" > "企业内部应用"
3. 创建新应用或选择现有应用
4. 记录以下信息:
   - `AppKey` (ClientID)
   - `AppSecret` (ClientSecret)
   - `AgentID`

### 1.2 配置应用权限

在应用权限设置中,添加以下权限:

- **企业员工信息读权限**
- **企业员工手机号信息**
- **成员信息读权限**
- **群消息发送权限**
- **机器人消息接收权限**

### 1.3 创建 AI 卡片模板

#### 步骤 1: 进入 AI 卡片管理

1. 在钉钉开放平台左侧菜单,选择"能力" > "AI 卡片"
2. 点击"创建模板"

#### 步骤 2: 配置卡片模板

使用以下 JSON 配置创建模板:

```json
{
  "config": {
    "autoLayout": true,
    "enableForward": true
  },
  "header": {
    "title": {
      "type": "text",
      "text": "ZenOps 查询结果"
    },
    "logo": "@lALPDfJ6V_FPDmvNAfTNAfQ"
  },
  "contents": [
    {
      "type": "markdown",
      "text": "${content}",
      "id": "content"
    }
  ]
}
```

**重要字段说明:**
- `${content}`: 这是流式更新的目标字段,必须保持这个名称
- `id: "content"`: 字段 ID 必须为 "content",与代码中的更新逻辑对应

#### 步骤 3: 发布模板

1. 保存模板并发布
2. 记录**模板 ID** (格式类似: `4d18414c-aabc-4ec8-9e67-4ceefeada72a.schema`)

### 1.4 配置回调地址

如果使用 HTTP 回调模式(非必需):

1. 在应用设置中找到"事件订阅"
2. 配置回调 URL: `https://your-domain.com/api/v1/dingtalk/callback`
3. 生成并记录:
   - `加签密钥` (Token)
   - `数据加密密钥` (AES Key)

## 二、ZenOps 配置

### 2.1 环境变量配置

在 `.env` 文件或环境中设置以下变量:

```bash
# 钉钉基础配置
DINGTALK_APP_KEY=your_app_key
DINGTALK_APP_SECRET=your_app_secret
DINGTALK_AGENT_ID=your_agent_id

# AI 卡片模板 ID (必需,用于流式输出)
DINGTALK_TEMPLATE_ID=c1f597d3-ecae-40bb-ba04-e183844ba2cd.schema

# 回调配置 (如果使用 HTTP 回调模式)
DINGTALK_CALLBACK_TOKEN=your_callback_token
DINGTALK_AES_KEY=your_aes_key
```

### 2.2 config.yaml 配置

在 `configs/config.yaml` 中启用钉钉集成:

```yaml
dingtalk:
  enabled: true
  app_key: ${DINGTALK_APP_KEY}
  app_secret: ${DINGTALK_APP_SECRET}
  agent_id: ${DINGTALK_AGENT_ID}
  template_id: ${DINGTALK_TEMPLATE_ID}  # AI 卡片模板 ID
  callback:
    token: ${DINGTALK_CALLBACK_TOKEN}
    aes_key: ${DINGTALK_AES_KEY}
    url: https://your-domain.com/api/v1/dingtalk/callback
```

## 三、工作原理

### 3.1 流式输出流程

```
用户发送消息
    ↓
钉钉回调 → ZenOps 接收消息
    ↓
创建并投递 AI 卡片
    ↓
发送初始提示 "正在查询..."
    ↓
调用 MCP 工具查询
    ↓
流式更新卡片内容 (每 1.5 秒更新一次)
    ↓
发送最终版本 (isFinalize=true)
```

### 3.2 核心 API

ZenOps 使用钉钉官方 SDK 实现流式输出:

#### StreamingUpdate API

```go
request := &dingtalkcard_1_0.StreamingUpdateRequest{
    OutTrackId: tea.String(trackID),     // 卡片跟踪 ID
    Guid:       tea.String(uuid.New()),  // 唯一标识
    Key:        tea.String("content"),   // 更新字段名
    Content:    tea.String(content),     // 更新内容
    IsFull:     tea.Bool(true),          // 全量更新
    IsFinalize: tea.Bool(isFinalize),    // 是否最终版本
    IsError:    tea.Bool(false),         // 是否错误
}
```

### 3.3 更新频率

- **定时更新**: 每 1.5 秒更新一次卡片内容
- **最终更新**: 查询完成后发送 `isFinalize=true` 的最终版本
- **错误处理**: 查询失败时发送错误信息并标记为最终版本

## 四、使用示例

### 4.1 在群聊中使用

```
@ZenOps 查询 IP 为 192.168.1.100 的阿里云 ECS
```

效果:
1. 机器人立即响应创建卡片
2. 卡片显示 "正在查询..."
3. 每 1.5 秒更新一次查询进度
4. 显示完整结果并标记为完成

### 4.2 在私聊中使用

```
列出腾讯云所有 CVM 实例
```

效果相同,支持流式打字机效果。

## 五、功能对比

### 使用 AI 卡片流式输出 (推荐)

**优点:**
- ✅ 真正的流式打字机效果
- ✅ 用户体验好,实时看到查询进度
- ✅ 支持 Markdown 格式
- ✅ 卡片可转发
- ✅ 错误信息友好展示

**配置要求:**
- 需要创建 AI 卡片模板
- 需要配置 `template_id`

### 不使用流式输出 (降级方案)

如果不配置 `template_id`,系统会使用传统的文本消息:

**特点:**
- 简单配置
- 普通文本消息
- 无流式效果

## 六、故障排查

### 6.1 卡片创建失败

**错误**: `failed to create and deliver card`

**排查步骤:**
1. 检查 `template_id` 是否正确
2. 确认 AI 卡片模板已发布
3. 检查应用权限是否完整
4. 查看日志中的详细错误信息

### 6.2 流式更新失败

**错误**: `failed to update card`

**排查步骤:**
1. 检查 `trackID` 是否唯一
2. 确认卡片已成功创建
3. 检查 Access Token 是否有效
4. 确认更新频率不要太快 (建议 1.5 秒)

### 6.3 无法接收消息

**排查步骤:**
1. 检查回调 URL 是否可访问
2. 确认签名验证逻辑正确
3. 检查消息加密/解密配置
4. 查看服务器日志

## 七、技术实现

### 7.1 依赖包

ZenOps 使用以下官方 SDK:

```go
github.com/alibabacloud-go/dingtalk v1.6.88
github.com/alibabacloud-go/darabonba-openapi/v2 v2.1.7
github.com/alibabacloud-go/tea v1.3.9
github.com/alibabacloud-go/tea-utils/v2 v2.0.7
github.com/open-dingtalk/dingtalk-stream-sdk-go v0.9.1
```

### 7.2 核心代码文件

- `internal/server/dingtalk_stream.go`: 流式客户端实现
- `internal/server/dingtalk.go`: 消息处理和意图识别
- `internal/api/handler/dingtalk.go`: HTTP 回调处理器

### 7.3 扩展开发

如果需要添加新的 MCP 工具支持:

1. 在 `ParseIntent()` 中添加新的正则模式
2. 在 `callMCPTool()` 中添加工具调用
3. 确保工具已在 MCP Server 中注册

## 八、参考资料

- [钉钉开放平台 - 互动卡片](https://open.dingtalk.com/document/orgapp/interactive-cards-overview)
- [钉钉 StreamingUpdate API](https://open.dingtalk.com/document/development/api-streamingupdate)
- [钉钉 Stream 模式 SDK](https://github.com/open-dingtalk/dingtalk-stream-sdk-go)
- [阿里云钉钉 SDK](https://github.com/alibabacloud-go/dingtalk)

## 九、最佳实践

1. **必须配置 template_id**: 否则无法使用流式输出功能
2. **控制更新频率**: 建议 1.5 秒,避免触发限流
3. **错误处理**: 始终发送最终版本,即使查询失败
4. **日志记录**: 启用详细日志以便排查问题
5. **测试环境**: 先在测试环境验证配置正确性

## 十、升级说明

### 从旧版本升级

如果你之前使用的是自定义流式实现,升级到官方 SDK 版本需要:

1. 添加新的 SDK 依赖 (运行 `go mod tidy`)
2. 在钉钉平台创建 AI 卡片模板
3. 配置 `template_id` 环境变量
4. 重启服务

**兼容性**: 如果不配置 `template_id`,系统会自动降级到旧的实现方式,保持向后兼容。

---

**配置完成后,请测试以下功能:**

- ✅ 机器人能收到消息
- ✅ 卡片能正常创建
- ✅ 流式更新效果正常
- ✅ 查询结果正确显示
- ✅ 错误信息友好展示
