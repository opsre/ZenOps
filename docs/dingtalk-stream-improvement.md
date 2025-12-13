# ZenOps 钉钉流式输出改进方案

## 当前实现的问题

当前实现使用了自定义的流式消息发送,但存在以下问题:

1. **API 不存在**: 使用的 `SendStreamMessage` API 实际上不是钉钉官方的流式 API
2. **缺少卡片支持**: 没有使用钉钉的互动卡片机制
3. **用户体验差**: 无法实现真正的流式打字效果

## 正确的实现方案

根据 PandaWiki 的实现和钉钉官方文档,正确的流式输出需要:

### 1. 使用官方 SDK

需要引入以下依赖:

```go
// go.mod
github.com/alibabacloud-go/dingtalk v1.6.88
github.com/alibabacloud-go/darabonba-openapi/v2 v2.1.7
github.com/alibabacloud-go/tea v1.3.9
github.com/alibabacloud-go/tea-utils/v2 v2.0.7
github.com/open-dingtalk/dingtalk-stream-sdk-go v0.9.1
```

### 2. 核心 API

**StreamingUpdate API** - 流式更新卡片内容

```go
import (
    dingtalkcard_1_0 "github.com/alibabacloud-go/dingtalk/card_1_0"
    "github.com/alibabacloud-go/tea/tea"
)

func (c *Client) StreamingUpdate(trackID, content string, isFinalize bool) error {
    headers := &dingtalkcard_1_0.StreamingUpdateHeaders{
        XAcsDingtalkAccessToken: tea.String(accessToken),
    }

    request := &dingtalkcard_1_0.StreamingUpdateRequest{
        OutTrackId: tea.String(trackID),     // 卡片唯一标识
        Guid:       tea.String(uuid.New().String()),
        Key:        tea.String("content"),    // 更新的字段
        Content:    tea.String(content),      // 更新的内容
        IsFull:     tea.Bool(true),           // 是否全量更新
        IsFinalize: tea.Bool(isFinalize),     // 是否最终版本
        IsError:    tea.Bool(false),          // 是否错误
    }

    _, err = c.cardClient.StreamingUpdateWithOptions(request, headers, &util.RuntimeOptions{})
    return err
}
```

### 3. 实现流程

#### 步骤 1: 创建并投递卡片

```go
func (c *Client) CreateAndDeliverCard(trackID string, data *ChatData) error {
    request := &dingtalkcard_1_0.CreateAndDeliverRequest{
        CardTemplateId: tea.String(templateID),  // AI 卡片模板 ID
        OutTrackId:     tea.String(trackID),
        CardData: &dingtalkcard_1_0.CreateAndDeliverRequestCardData{
            CardParamMap: map[string]*string{
                "content": tea.String(""),  // 初始内容为空
            },
        },
        CallbackType: tea.String("STREAM"),  // 重要:使用 STREAM 模式
        UserIdType:   tea.Int32(1),
    }

    // 根据会话类型设置 OpenSpaceId
    switch data.ConversationType {
    case "2": // 群聊
        openSpaceId := fmt.Sprintf("dtv1.card//IM_GROUP.%s", data.ConversationId)
        request.SetOpenSpaceId(openSpaceId)
    case "1": // 单聊
        openSpaceId := fmt.Sprintf("dtv1.card//IM_ROBOT.%s", data.SenderStaffId)
        request.SetOpenSpaceId(openSpaceId)
    }

    _, err = c.cardClient.CreateAndDeliverWithOptions(request, headers, &util.RuntimeOptions{})
    return err
}
```

#### 步骤 2: 流式更新卡片内容

```go
func (c *Client) StreamResponse(trackID string, contentCh <-chan string) {
    fullContent := ""
    ticker := time.NewTicker(1500 * time.Millisecond)  // 1.5秒更新一次
    defer ticker.Stop()

    for {
        select {
        case content, ok := <-contentCh:
            if !ok {
                // 最终更新
                c.StreamingUpdate(trackID, fullContent, true)
                return
            }
            fullContent += content

        case <-ticker.C:
            // 定时更新
            if fullContent != "" {
                c.StreamingUpdate(trackID, fullContent, false)
            }
        }
    }
}
```

### 4. 使用 Stream 模式接收消息

钉钉提供了 Stream 模式 SDK,无需配置回调 URL:

```go
import (
    "github.com/open-dingtalk/dingtalk-stream-sdk-go/chatbot"
    "github.com/open-dingtalk/dingtalk-stream-sdk-go/client"
)

func (c *Client) Start() error {
    cli := client.NewStreamClient(client.WithAppCredential(
        client.NewAppCredentialConfig(c.clientID, c.clientSecret),
    ))

    // 注册消息处理函数
    cli.RegisterChatBotCallbackRouter(c.OnMessageReceived)

    return cli.Start(ctx)
}

func (c *Client) OnMessageReceived(ctx context.Context, data *chatbot.BotCallbackDataModel) ([]byte, error) {
    question := data.Text.Content
    trackID := uuid.New().String()

    // 1. 创建卡片
    c.CreateAndDeliverCard(trackID, data)

    // 2. 初始更新
    c.StreamingUpdate(trackID, "稍等,正在查询...", false)

    // 3. 调用 MCP 并流式更新
    contentCh, _ := c.queryMCP(question)
    c.StreamResponse(trackID, contentCh)

    return []byte(""), nil
}
```

### 5. AI 卡片模板

需要在钉钉开放平台创建 AI 卡片模板:

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
    }
  },
  "contents": [
    {
      "type": "markdown",
      "text": "${content}",  // 这是流式更新的字段
      "id": "content"
    }
  ]
}
```

## 对比分析

### 当前实现 vs 正确实现

| 特性 | 当前实现 | 正确实现 |
|------|---------|---------|
| 消息接收 | HTTP 回调 | Stream 模式 |
| 消息展示 | 普通文本 | 互动卡片 |
| 流式效果 | ❌ 无法实现 | ✅ 真正流式 |
| 用户体验 | 分段显示 | 打字机效果 |
| 配置复杂度 | 需要公网 URL | 无需配置 |
| 签名验证 | 手动实现 | SDK 自动处理 |

### PandaWiki 的实现优势

1. **使用官方 SDK**: 阿里云钉钉官方 SDK,稳定可靠
2. **Stream 模式**: 不需要配置公网回调 URL
3. **真正流式**: 使用 `StreamingUpdate` API 实现打字机效果
4. **定时更新**: 1.5秒更新一次,平衡体验和性能
5. **错误处理**: 完善的错误处理和日志

## 改进建议

### 方案 A: 完全重写(推荐)

使用官方 SDK 完全重写钉钉集成:

**优点**:
- 功能完整,体验最好
- 代码更简洁,维护性好
- 支持所有钉钉特性

**缺点**:
- 需要重写大部分代码
- 增加依赖包

### 方案 B: 保留当前,添加卡片支持

保留当前 HTTP 回调模式,仅添加流式卡片更新:

**优点**:
- 改动较小
- 兼容当前实现

**缺点**:
- 仍需配置回调 URL
- 代码复杂度高

### 方案 C: 混合模式

HTTP 回调 + 流式卡片:

```go
// 保留当前回调处理
func (h *Handler) handleCallback(w http.ResponseWriter, r *http.Request) {
    // 1. 验证签名
    // 2. 解密消息
    // 3. 创建卡片并返回
    trackID := uuid.New().String()
    h.createCard(trackID, msg.ConversationID)

    // 4. 异步处理查询
    go h.processWithStream(trackID, msg)

    // 5. 立即返回
    return simpleResponse("ok")
}

func (h *Handler) processWithStream(trackID string, msg *Message) {
    // 使用 StreamingUpdate API 更新卡片
    content := h.queryMCP(msg)
    h.streamingUpdate(trackID, content)
}
```

## 实施步骤

### 阶段 1: 添加依赖

```bash
go get github.com/alibabacloud-go/dingtalk/card_1_0
go get github.com/alibabacloud-go/darabonba-openapi/v2
go get github.com/alibabacloud-go/tea
go get github.com/alibabacloud-go/tea-utils/v2
go get github.com/open-dingtalk/dingtalk-stream-sdk-go
```

### 阶段 2: 创建 AI 卡片模板

1. 登录钉钉开放平台
2. 进入"AI 卡片"
3. 创建新模板
4. 配置 content 字段
5. 获取模板 ID

### 阶段 3: 实现 StreamingUpdate

创建新的流式更新客户端

### 阶段 4: 集成到现有代码

选择方案 C(混合模式)最适合 ZenOps

### 阶段 5: 测试和优化

## 代码示例

完整的改进实现见 `internal/server/dingtalk_stream.go`

## 参考资源

- [钉钉开放平台 - 互动卡片](https://open.dingtalk.com/document/orgapp/interactive-cards-overview)
- [钉钉 Stream 模式 SDK](https://github.com/open-dingtalk/dingtalk-stream-sdk-go)
- [PandaWiki 实现](https://github.com/chaitin/panda-wiki)
- [阿里云钉钉 SDK](https://github.com/alibabacloud-go/dingtalk)

## 总结

当前实现虽然能工作,但没有使用钉钉的核心流式特性。建议采用**方案 C(混合模式)**:

1. 保留现有 HTTP 回调处理
2. 添加官方 SDK 支持
3. 使用 `StreamingUpdate` API 实现真正流式输出
4. 创建 AI 卡片模板提升用户体验

这样既保持了架构的灵活性,又能提供最佳的用户体验。
