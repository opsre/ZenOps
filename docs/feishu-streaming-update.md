# 飞书流式消息更新实现说明

## 问题描述

在测试中发现两个问题:
1. **消息不是流式更新**: LLM 回复是一次性显示,而不是打字机效果
2. **Markdown 格式未被识别**: 发送的内容格式不正确

## 解决方案

### 1. 实现流式卡片更新

#### 原理
- 飞书支持流式卡片 (Streaming Card)
- 使用 `Cardkit.V1.Card.Create` API 创建流式卡片
- 使用 `Cardkit.V1.CardElement.Content` API 更新卡片元素
- 采用定时更新机制(每 300ms 更新一次)

#### 实现步骤

**第一步: 创建流式卡片**
```go
cardTemplate := map[string]interface{}{
    "schema": "2.0",
    "header": map[string]interface{}{
        "title": map[string]interface{}{
            "content": "AI 回答",
            "tag":     "plain_text",
        },
    },
    "config": map[string]interface{}{
        "streaming_mode": true,  // 启用流式模式
        "summary": map[string]interface{}{
            "content": "",
        },
    },
    "body": map[string]interface{}{
        "elements": []map[string]interface{}{
            {
                "tag":        "markdown",  // 使用 markdown 标签
                "content":    initialContent,
                "element_id": "markdown_content",  // 元素 ID,用于后续更新
            },
        },
    },
}

cardID, err := h.client.CreateStreamingCard(ctx, "AI 回答", initialContent)
```

**第二步: 发送卡片消息**
```go
cardContent := map[string]interface{}{
    "type": "card",
    "data": map[string]string{
        "card_id": cardID,
    },
}

_, err = h.client.SendCardMessage(ctx, receiveIDType, receiveID, cardID)
```

**第三步: 流式更新卡片元素**
```go
updateTicker := time.NewTicker(300 * time.Millisecond)
sequence := 0

for {
    select {
    case content := <-responseCh:
        fullResponse.WriteString(content)
    case <-updateTicker.C:
        sequence++
        h.client.UpdateCardElement(ctx, cardID, "markdown_content", currentContent, sequence)
    }
}
```

#### 新增方法

**`Client.CreateStreamingCard()`** - 创建流式卡片
```go
func (c *Client) CreateStreamingCard(ctx context.Context, title, initialContent string) (string, error) {
    req := larkcardkit.NewCreateCardReqBuilder().
        Body(larkcardkit.NewCreateCardReqBodyBuilder().
            Type("card_json").
            Data(string(cardData)).
            Build()).
        Build()

    resp, err := c.client.Cardkit.V1.Card.Create(ctx, req)
    // ...
    return *resp.Data.CardId, nil
}
```

**`Client.SendCardMessage()`** - 发送卡片消息
```go
func (c *Client) SendCardMessage(ctx context.Context, receiveIDType, receiveID, cardID string) (*string, error) {
    cardContent := map[string]interface{}{
        "type": "card",
        "data": map[string]string{
            "card_id": cardID,
        },
    }

    req := larkim.NewCreateMessageReqBuilder().
        ReceiveIdType(receiveIDType).
        Body(larkim.NewCreateMessageReqBodyBuilder().
            ReceiveId(receiveID).
            MsgType("interactive").
            Content(string(contentJSON)).
            Build()).
        Build()
    // ...
}
```

**`Client.UpdateCardElement()`** - 更新卡片元素
```go
func (c *Client) UpdateCardElement(ctx context.Context, cardID, elementID, content string, sequence int) error {
    uuid := fmt.Sprintf("%d-%d", ctx.Value("timestamp"), sequence)

    req := larkcardkit.NewContentCardElementReqBuilder().
        CardId(cardID).
        ElementId(elementID).
        Body(larkcardkit.NewContentCardElementReqBodyBuilder().
            Uuid(uuid).
            Content(content).
            Sequence(sequence).
            Build()).
        Build()

    resp, err := c.client.Cardkit.V1.CardElement.Content(ctx, req)
    // ...
}
```

### 2. 修复 Markdown 格式问题

#### 问题原因
之前使用 `"text"` tag,飞书不会解析 Markdown 格式。

#### 正确的格式

**之前(错误)**:
```go
{
    "tag":  "text",
    "text": content,
}
```

**现在(正确)**:
```go
{
    "tag":        "markdown",  // 使用 markdown 标签
    "content":    content,     // 使用 content 字段而不是 text
    "element_id": "markdown_content",
}
```

#### 关键改进
1. 使用流式卡片模板,设置 `streaming_mode: true`
2. 使用 `"markdown"` tag 支持 Markdown 格式
3. 使用 `element_id` 标识要更新的元素
4. 使用递增的 `sequence` 保证更新顺序

### 3. 更新后的消息流程

```
用户发送消息
    ↓
接收消息事件
    ↓
调用 LLM 流式 API
    ↓
创建流式卡片 (cardID)
    ↓
发送卡片消息 (messageID)
    ↓
┌─────────────────────────────────┐
│  流式接收循环                    │
│  ┌──────────────────────────┐   │
│  │ 每 300ms 更新一次         │   │
│  │ sequence 递增             │   │
│  └──────────────────────────┘   │
│                                  │
│  收到新内容 → 累积               │
│       ↓                          │
│  定时器触发 → 更新卡片元素       │
│       ↓                          │
│  继续接收...                     │
└─────────────────────────────────┘
    ↓
流结束
    ↓
发送最终更新(带时间戳)
```

## 代码变更

### 1. `internal/feishu/client.go`

**新增导入**:
```go
import (
    larkcardkit "github.com/larksuite/oapi-sdk-go/v3/service/cardkit/v1"
)
```

**新增方法**:
```go
func (c *Client) CreateStreamingCard(ctx context.Context, title, initialContent string) (string, error)
func (c *Client) SendCardMessage(ctx context.Context, receiveIDType, receiveID, cardID string) (*string, error)
func (c *Client) UpdateCardElement(ctx context.Context, cardID, elementID, content string, sequence int) error
```

### 2. `internal/feishu/handler.go`

**修改流式处理逻辑**:
```go
func (h *MessageHandler) processLLMMessage(...) error {
    // 1. 创建流式卡片
    cardID, err := h.client.CreateStreamingCard(ctx, "AI 回答", initialContent)

    // 2. 发送卡片消息
    _, err = h.client.SendCardMessage(ctx, receiveIDType, receiveID, cardID)

    // 3. 流式接收和更新
    sequence := 0
    for {
        select {
        case content := <-responseCh:
            fullResponse.WriteString(content)
        case <-updateTicker.C:
            sequence++
            h.client.UpdateCardElement(ctx, cardID, "markdown_content", currentContent, sequence)
        }
    }
}
```

## 效果对比

### 之前

❌ **一次性显示**
```
[等待3秒...]
完整回复内容一次性出现
```

❌ **格式问题**
```
**问题:** 帮我查询一下腾讯云的cvm服务器
**回答:**
[内容未格式化]
```

### 现在

✅ **流式显示**(打字机效果)
```
正在思考中...
↓ (300ms 后)
**问题:** 帮我查询...
**回答:**

我来帮您...
↓ (300ms 后)
我来帮您查询腾讯云的CVM...
↓ (持续更新)
完整内容
```

✅ **格式正确**
```
问题: 帮我查询一下腾讯云的cvm服务器

回答:

我来帮您查询腾讯云的CVM服务器信息。

[格式化的 Markdown 内容]

---
⏰ 2025-01-12 20:00:00
```

## 性能优化

### 更新频率控制

**为什么是 300ms?**
- 飞书 API 有频率限制
- 太频繁: 浪费 API 调用,影响性能
- 太慢: 用户体验差
- **300ms**: 平衡性能和体验,比之前的 500ms 更流畅

### 去重更新

```go
if currentContent != lastUpdate && len(currentContent) > len(questionHeader) {
    sequence++
    h.client.UpdateCardElement(ctx, cardID, "markdown_content", currentContent, sequence)
    lastUpdate = currentContent
}
```

- 只在内容变化时更新
- 避免重复更新相同内容
- 减少 API 调用次数

### Sequence 机制

- 每次更新 sequence 递增
- 确保更新顺序正确
- 飞书服务端根据 sequence 排序

## 测试验证

### 测试步骤

1. **重新编译**
   ```bash
   go build -o zenops main.go
   ```

2. **重启服务**
   ```bash
   ./zenops run --config config.yaml
   ```

3. **测试流式更新**
   - 在飞书中发送: "帮我查询一下腾讯云的cvm服务器"
   - 观察消息是否逐步更新
   - 检查是否有打字机效果

4. **测试 Markdown 格式**
   - 发送: "帮助"
   - 检查格式是否正确显示
   - 确认标题、列表、代码块等格式

### 预期效果

✅ **流式更新**:
- 消息先显示 "正在思考中..."
- 然后逐步显示完整内容
- 每 300ms 更新一次
- 有打字机效果

✅ **格式正确**:
- **粗体**、*斜体* 正常显示
- 标题、段落正确显示
- 列表、分隔线正常
- 代码块格式正确
- 时间戳在底部

## API 限制说明

### 飞书 Cardkit API 限制

1. **卡片更新频率**: 建议不超过每秒 3 次
2. **卡片大小**: 单个卡片最大 100KB
3. **Sequence 范围**: 0-2^31-1
4. **UUID 唯一性**: 每次更新需要唯一 UUID

### 当前实现限制

1. **仅支持 Markdown**: 使用 markdown 元素
2. **固定更新频率**: 300ms 不可配置
3. **单一元素更新**: 仅更新一个 markdown 元素
4. **无重试机制**: 更新失败仅记录日志

## 未来改进

### 短期改进

1. **优化 UUID 生成**
   - 使用真正的 UUID 库
   - 当前使用时间戳+序列号的简单方案

2. **支持更丰富的卡片元素**
   - 图片、链接等
   - 多个 markdown 元素
   - 交互式按钮

3. **自适应更新频率**
   - 内容变化大时更快更新
   - 内容变化小时降低频率

### 长期改进

1. **错误重试机制**
   - 更新失败时自动重试
   - 指数退避策略

2. **卡片模板管理**
   - 支持自定义卡片模板
   - 支持多种卡片风格

3. **性能监控**
   - 统计更新成功率
   - 监控 API 调用延迟

## 相关文档

- [飞书开放平台 - 流式卡片](https://open.feishu.cn/document/cardkit-v1/streaming-updates-openapi-overview)
- [飞书 Cardkit API](https://open.feishu.cn/document/server-docs/cardkit-v1/card/create)
- [飞书消息 API](https://open.feishu.cn/document/server-docs/im-v1/message/patch)

---

**更新时间**: 2025-01-12
**版本**: v2.0 (基于飞书流式卡片)
