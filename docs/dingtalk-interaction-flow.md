# 钉钉机器人交互流程详解

本文档详细介绍 ZenOps 钉钉机器人从接收用户消息到返回响应的完整流程。

## 目录

- [架构概览](#架构概览)
- [核心组件](#核心组件)
- [详细流程](#详细流程)
  - [1. 消息接收阶段](#1-消息接收阶段)
  - [2. 消息处理阶段](#2-消息处理阶段)
  - [3. LLM 调用阶段](#3-llm-调用阶段)
  - [4. 工具调用阶段](#4-工具调用阶段)
  - [5. 响应返回阶段](#5-响应返回阶段)
- [消息卡片机制](#消息卡片机制)
- [流式响应机制](#流式响应机制)
- [性能优化要点](#性能优化要点)

---

## 架构概览

```
┌─────────────┐
│  钉钉用户   │
└──────┬──────┘
       │ 1. 发送消息
       ▼
┌─────────────────────────────────────────────────┐
│            钉钉开放平台 (Stream Mode)            │
└──────┬──────────────────────────────────────────┘
       │ 2. 推送事件
       ▼
┌─────────────────────────────────────────────────┐
│         DingTalk Stream Handler                  │
│    (internal/server/dingtalk_stream_handler.go) │
└──────┬──────────────────────────────────────────┘
       │ 3. 处理消息
       ▼
┌─────────────────────────────────────────────────┐
│              LLM Client                          │
│         (internal/llm/openai.go)                 │
└──────┬──────────────────────────────────────────┘
       │ 4. 调用大模型
       ▼
┌─────────────────────────────────────────────────┐
│           OpenAI Compatible API                  │
│    (DeepSeek / 通义千问 / ChatGPT 等)            │
└──────┬──────────────────────────────────────────┘
       │ 5. 需要调用工具?
       ▼
┌─────────────────────────────────────────────────┐
│              MCP Server                          │
│    (internal/server/mcp_with_lib.go)            │
│                                                  │
│  - search_ecs_by_ip (阿里云 ECS 查询)           │
│  - search_cvm_by_ip (腾讯云 CVM 查询)           │
│  - list_jenkins_jobs (Jenkins 任务)             │
│  - ...                                           │
└──────┬──────────────────────────────────────────┘
       │ 6. 执行云服务查询
       ▼
┌─────────────────────────────────────────────────┐
│             Provider 层                          │
│      (internal/provider/)                        │
│                                                  │
│  - Aliyun Provider (阿里云 SDK)                  │
│  - Tencent Provider (腾讯云 SDK)                 │
│  - Jenkins Provider (Jenkins API)                │
└──────┬──────────────────────────────────────────┘
       │ 7. 返回查询结果
       ▼
┌─────────────────────────────────────────────────┐
│           LLM 分析工具结果                       │
│           生成最终回复                           │
└──────┬──────────────────────────────────────────┘
       │ 8. 流式返回文本
       ▼
┌─────────────────────────────────────────────────┐
│         DingTalk Card Client                     │
│    (实时更新消息卡片)                            │
└──────┬──────────────────────────────────────────┘
       │ 9. 显示回复
       ▼
┌─────────────┐
│  钉钉用户   │
└─────────────┘
```

---

## 核心组件

### 1. DingTalkStreamHandler
- **文件:** `internal/server/dingtalk_stream_handler.go`
- **职责:**
  - 接收钉钉 Stream 事件
  - 解析用户消息
  - 调度 LLM 处理
  - 管理响应流

### 2. LLM Client
- **文件:** `internal/llm/openai.go`
- **职责:**
  - 封装 OpenAI 兼容 API
  - 管理对话上下文
  - 处理工具调用循环
  - 提供流式响应

### 3. MCP Server
- **文件:** `internal/server/mcp_with_lib.go`
- **职责:**
  - 注册 MCP 工具
  - 执行工具调用
  - 返回结构化结果

### 4. Provider 层
- **目录:** `internal/provider/`
- **职责:**
  - 封装各云厂商 SDK
  - 提供统一查询接口
  - 处理分页和错误

### 5. Card Client
- **文件:** `internal/server/dingtalk_stream_handler.go` (CardClient 结构)
- **职责:**
  - 创建交互式消息卡片
  - 实时更新卡片内容
  - 支持 Markdown 渲染

---

## 详细流程

### 1. 消息接收阶段

#### 1.1 用户在钉钉发送消息

用户在钉钉群聊或单聊中 @ 机器人并发送消息,例如:
```
@ZenOps 帮我查询 IP 10.0.1.100 是哪台服务器
```

#### 1.2 钉钉平台推送事件

钉钉开放平台通过 **Stream 模式** 推送事件到应用:

```go
// internal/server/dingtalk_stream_handler.go:75
func (h *DingTalkStreamHandler) OnChatBotMessageReceived(
    ctx context.Context,
    data *chatbot.BotCallbackDataModel,
) (*chatbot.BotCallbackResponse, error)
```

**关键数据结构:**
```go
data.Text.Content      // 用户消息内容 "帮我查询 IP 10.0.1.100 是哪台服务器"
data.SenderNick        // 用户昵称
data.ConversationId    // 会话 ID (用于回复)
data.SenderStaffId     // 用户员工 ID
```

#### 1.3 消息预处理

```go
// internal/server/dingtalk_stream_handler.go:90-120
// 1. 提取用户消息
userMessage := strings.TrimSpace(data.Text.Content)

// 2. 记录日志
logx.Info("Received message from %s: %s", data.SenderNick, userMessage)

// 3. 检查消息内容
if userMessage == "" {
    return quickReply("请输入有效的问题"), nil
}
```

---

### 2. 消息处理阶段

#### 2.1 选择响应模式

系统支持两种响应模式:

**模式 A: 消息卡片模式** (推荐,支持实时更新)
```go
// internal/server/dingtalk_stream_handler.go:650
if h.config.DingTalk.UseCard {
    return h.handleMessageWithCard(ctx, data)
}
```

**模式 B: 文本消息模式** (简单,不支持实时更新)
```go
return h.handleMessageWithText(ctx, data)
```

#### 2.2 创建消息卡片 (模式 A)

```go
// internal/server/dingtalk_stream_handler.go:662-690
func (h *DingTalkStreamHandler) handleMessageWithCard(
    ctx context.Context,
    data *chatbot.BotCallbackDataModel,
) (*chatbot.BotCallbackResponse, error) {

    // 1. 创建初始卡片
    trackID := generateTrackID()
    initialContent := "🤔 正在思考中..."

    if err := h.cardClient.SendCard(
        data.ConversationId,
        trackID,
        initialContent,
    ); err != nil {
        return nil, err
    }

    // 2. 在后台处理消息 (不阻塞钉钉回调)
    go h.processMessageWithCard(ctx, data, trackID)

    // 3. 立即返回 (让钉钉知道消息已接收)
    return &chatbot.BotCallbackResponse{}, nil
}
```

**重要:** 这里使用 `go` 关键字启动异步处理,避免阻塞钉钉的回调请求。

---

### 3. LLM 调用阶段

#### 3.1 构建上下文并调用 LLM

```go
// internal/server/dingtalk_stream_handler.go:725-740
func (h *DingTalkStreamHandler) processMessageWithCard(
    ctx context.Context,
    data *chatbot.BotCallbackDataModel,
    trackID string,
) {
    userMessage := strings.TrimSpace(data.Text.Content)

    // 调用 LLM (带工具和流式响应)
    responseCh, err := h.llmClient.ChatWithToolsAndStream(ctx, userMessage)
    if err != nil {
        h.cardClient.StreamingUpdate(trackID, "❌ 调用失败", true)
        return
    }

    // 流式处理响应
    h.streamLLMResponseWithCard(ctx, trackID, userMessage, responseCh)
}
```

#### 3.2 LLM 内部处理流程

```go
// internal/llm/openai.go:351-479
func (c *Client) ChatWithToolsAndStream(
    ctx context.Context,
    userMessage string,
) (<-chan string, error) {

    responseCh := make(chan string, 100)

    go func() {
        defer close(responseCh)

        // 1. 构建消息历史
        messages := []Message{
            {Role: "system", Content: c.buildSystemPrompt()},
            {Role: "user", Content: userMessage},
        }

        // 2. 获取可用工具列表
        tools, _ := c.getMCPTools(ctx)

        // 3. 进入工具调用循环 (最多 10 轮)
        for i := 0; i < maxIterations; i++ {

            // 3.1 判断是否有工具
            hasTools := len(tools) > 0

            if !hasTools {
                // 📌 分支 A: 无工具 - 使用纯流式 API
                contentCh, errCh, _ := openaiClient.ChatStream(ctx, messages)
                for content := range contentCh {
                    responseCh <- content  // 实时推送
                }
                return
            }

            // 📌 分支 B: 有工具 - 使用非流式 API (需要解析工具调用)
            resp, _ := openaiClient.ChatWithTools(ctx, messages, tools)

            // 3.2 检查是否有工具调用
            if len(resp.Choices[0].Message.ToolCalls) > 0 {
                // ⚠️ 进入工具调用流程 (见下一节)
                for _, toolCall := range resp.Choices[0].Message.ToolCalls {
                    responseCh <- fmt.Sprintf("🔧 调用工具: %s", toolCall.Function.Name)

                    // 执行工具
                    result, _ := c.executeToolCall(ctx, toolCall)

                    // 添加工具结果到消息历史
                    messages = append(messages, Message{
                        Role:       "tool",
                        Content:    result,
                        ToolCallID: toolCall.ID,
                    })

                    responseCh <- "✅ 工具执行完成"
                }

                // 继续循环,让 LLM 分析工具结果
                continue
            }

            // 3.3 没有工具调用,返回最终内容
            // ⚠️ 问题在这里: 非流式返回,导致延迟
            if resp.Choices[0].Message.Content != "" {
                responseCh <- resp.Choices[0].Message.Content
                return
            }
        }
    }()

    return responseCh, nil
}
```

**关键问题:**
- **第 417 行:** 当有工具时,使用非流式 `ChatWithTools()`
- **第 467 行:** 工具调用后的最终响应也是非流式返回
- **结果:** 必须等待 LLM 生成完整内容后才一次性推送,造成 ~10 秒延迟

---

### 4. 工具调用阶段

#### 4.1 解析工具调用请求

当 LLM 决定调用工具时,会返回 `tool_calls` 结构:

```json
{
  "id": "call_abc123",
  "type": "function",
  "function": {
    "name": "search_ecs_by_ip",
    "arguments": "{\"ip\":\"10.0.1.100\",\"account\":\"default\"}"
  }
}
```

#### 4.2 执行工具调用

```go
// internal/llm/openai.go:492-526
func (c *Client) executeToolCall(
    ctx context.Context,
    toolCall ToolCall,
) (string, error) {

    // 1. 解析参数
    var args map[string]any
    json.Unmarshal([]byte(toolCall.Function.Arguments), &args)

    // 2. 调用 MCP Server
    result, err := c.mcpServer.CallTool(ctx, toolCall.Function.Name, args)
    if err != nil {
        return "", err
    }

    // 3. 格式化结果
    var resultText string
    for _, content := range result.Content {
        if content.Type == "text" {
            resultText += content.Text
        }
    }

    return resultText, nil
}
```

#### 4.3 MCP Server 处理工具调用

```go
// internal/server/mcp_with_lib.go:750-803
func (s *MCPServerWithLib) CallTool(
    ctx context.Context,
    toolName string,
    arguments map[string]any,
) (*mcp.CallToolResult, error) {

    // 根据工具名称路由到具体处理函数
    switch toolName {
    case "search_ecs_by_ip":
        return s.handleSearchECSByIP(ctx, request)
    case "search_cvm_by_ip":
        return s.handleSearchCVMByIP(ctx, request)
    case "list_jenkins_jobs":
        return s.handleListJenkinsJobs(ctx, request)
    // ... 其他工具
    default:
        return mcp.NewToolResultError("unsupported tool"), nil
    }
}
```

#### 4.4 Provider 层执行实际查询

以阿里云 ECS 查询为例:

```go
// internal/server/mcp_with_lib.go:322-392
func (s *MCPServerWithLib) handleSearchECSByIP(
    ctx context.Context,
    request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {

    // 1. 获取参数
    ip := args["ip"].(string)
    accountName := args["account"].(string)

    // 2. 初始化 Provider
    p, aliyunConfig, _ := s.getAliyunProvider(accountName)

    // 3. 分页查询所有实例
    var matchedInstances []*model.Instance
    pageNum := 1
    pageSize := 100

    for {
        opts := &provider.QueryOptions{
            PageSize: pageSize,
            PageNum:  pageNum,
        }

        instances, _ := p.ListInstances(ctx, opts)

        // 4. 匹配 IP
        for _, inst := range instances {
            for _, privateIP := range inst.PrivateIP {
                if privateIP == ip {
                    matchedInstances = append(matchedInstances, inst)
                }
            }
        }

        if len(instances) < pageSize {
            break
        }
        pageNum++
    }

    // 5. 格式化结果
    result := formatInstances(matchedInstances, aliyunConfig.Name)
    return mcp.NewToolResultText(result), nil
}
```

#### 4.5 工具结果示例

```
找到 1 个 ECS 实例 (账号: default):

【实例 1】
  实例 ID: i-bp1234567890abcde
  实例名称: web-server-01
  区域: cn-hangzhou
  可用区: cn-hangzhou-h
  实例规格: ecs.c6.large
  状态: Running
  CPU: 2 核
  内存: 4096 MB
  操作系统: CentOS 7.9 64位
  私网 IP: [10.0.1.100]
  公网 IP: [47.96.123.45]
  创建时间: 2024-01-15 10:30:00
  控制台地址: https://ecs.console.aliyun.com/...
```

---

### 5. 响应返回阶段

#### 5.1 流式响应缓冲机制

```go
// internal/server/dingtalk_stream_handler.go:757-802
func (h *DingTalkStreamHandler) streamLLMResponseWithCard(
    ctx context.Context,
    trackID string,
    question string,
    responseCh <-chan string,
) {
    fullContent := ""
    updateBuffer := ""

    // 缓冲参数
    minUpdateInterval := 200 * time.Millisecond  // 最小更新间隔
    minBufferSize := 10                          // 最小缓冲大小

    ticker := time.NewTicker(minUpdateInterval)
    defer ticker.Stop()

    for {
        select {
        case content, ok := <-responseCh:
            if !ok {
                // 流结束,发送最终更新
                if updateBuffer != "" {
                    fullContent += updateBuffer
                }
                fullContent += fmt.Sprintf("\n\n---\n⏰ %s", time.Now().Format("2006-01-02 15:04:05"))

                h.cardClient.StreamingUpdate(trackID, fullContent, true)
                logx.Info("LLM conversation completed with card")
                return
            }

            // 累积到缓冲区
            updateBuffer += content

        case <-ticker.C:
            // 定时检查是否需要更新
            if updateBuffer != "" && len(updateBuffer) >= minBufferSize {
                fullContent += updateBuffer
                updateBuffer = ""

                // 更新卡片 (非最终更新)
                h.cardClient.StreamingUpdate(trackID, fullContent, false)
            }
        }
    }
}
```

**缓冲策略:**
1. **时间缓冲:** 每 200ms 检查一次
2. **大小缓冲:** 至少累积 10 个字符才更新
3. **目的:** 避免频繁更新卡片,减少 API 调用

#### 5.2 更新钉钉消息卡片

```go
// internal/server/dingtalk_stream_handler.go:597-648
func (c *CardClient) StreamingUpdate(
    trackID string,
    content string,
    isFinal bool,
) error {

    // 1. 构建卡片内容
    cardData := map[string]any{
        "config": map[string]any{
            "autoLayout": true,
            "enableForward": true,
        },
        "header": map[string]any{
            "title": map[string]string{
                "type": "text",
                "text": "ZenOps 助手",
            },
            "logo": "@lALPDfJ6V_FPDmvNAfTNAfQ",
        },
        "contents": []map[string]any{
            {
                "type": "markdown",
                "text": content,  // Markdown 格式内容
            },
        },
    }

    // 2. 调用钉钉 API 更新卡片
    req := dingtalk.NewDingTalkRequest(
        "dingtalk.oapi.im.chat.scenegroup.interactivecard.update",
        c.accessToken,
    )
    req.SetBizContent(map[string]any{
        "card_data":     cardData,
        "out_track_id":  trackID,
        "card_update_options": map[string]any{
            "update_card_data_by_key": false,
        },
    })

    resp, err := c.dingtalkClient.Execute(req)
    if err != nil {
        return err
    }

    // 3. 记录更新日志
    if isFinal {
        logx.Info("Card final update successful, track_id=%s", trackID)
    }

    return nil
}
```

#### 5.3 用户看到实时更新

用户在钉钉客户端看到消息卡片**实时更新**:

```
┌─────────────────────────────────────┐
│ 🤔 ZenOps 助手                      │
├─────────────────────────────────────┤
│ 🔧 调用工具: search_ecs_by_ip       │
│ ✅ 工具执行完成                      │
│                                     │
│ 根据查询结果,IP 10.0.1.100 对应的   │
│ 服务器信息如下:                      │
│                                     │
│ **实例名称:** web-server-01          │
│ **实例 ID:** i-bp1234567890abcde    │
│ **状态:** 运行中 🟢                  │
│ **规格:** ecs.c6.large (2核4G)      │
│ **系统:** CentOS 7.9                │
│ **私网 IP:** 10.0.1.100             │
│ **公网 IP:** 47.96.123.45           │
│ **创建时间:** 2024-01-15 10:30:00   │
│                                     │
│ 需要我帮你做其他操作吗?               │
│                                     │
│ ---                                 │
│ ⏰ 2024-12-12 16:19:10              │
└─────────────────────────────────────┘
```

---

## 消息卡片机制

### 卡片创建流程

```go
// 1. 创建初始卡片
trackID := generateTrackID()  // 生成唯一 ID
h.cardClient.SendCard(conversationId, trackID, "🤔 正在思考中...")

// 2. 实时更新卡片
h.cardClient.StreamingUpdate(trackID, "🔧 调用工具...", false)
h.cardClient.StreamingUpdate(trackID, "✅ 工具执行完成...", false)

// 3. 最终更新
h.cardClient.StreamingUpdate(trackID, finalContent, true)
```

### Track ID 的作用

- **唯一标识:** 每个卡片有唯一的 `trackID`
- **更新凭证:** 通过 `trackID` 定位要更新的卡片
- **格式:** `zenops_reply_{timestamp}_{random}`

### 卡片支持的功能

- ✅ Markdown 渲染 (粗体、列表、代码块等)
- ✅ 实时内容更新 (不刷新页面)
- ✅ 消息转发
- ✅ 自动布局

---

## 流式响应机制

### 为什么需要流式响应?

**对比:**

| 模式 | 用户体验 | 延迟 | 实现复杂度 |
|------|---------|------|-----------|
| 非流式 | 等待后一次性显示 | 10-30秒 | 简单 |
| 流式 | 实时逐字显示 | <1秒首字 | 中等 |

### 流式响应的实现

#### 1. Channel 机制

```go
// 创建响应通道
responseCh := make(chan string, 100)

// 生产者 (LLM 侧)
go func() {
    defer close(responseCh)
    for chunk := range llmStream {
        responseCh <- chunk  // 发送文本片段
    }
}()

// 消费者 (钉钉侧)
for content := range responseCh {
    fullContent += content
    if shouldUpdate() {
        updateCard(fullContent)  // 更新卡片
    }
}
```

#### 2. 缓冲策略

**问题:** 如果每收到一个字符就更新卡片,会导致:
- API 调用过于频繁
- 卡片闪烁
- 钉钉限流

**解决方案:** 缓冲机制
```go
updateBuffer := ""
ticker := time.NewTicker(200 * time.Millisecond)

for {
    select {
    case content := <-responseCh:
        updateBuffer += content  // 累积内容

    case <-ticker.C:
        if len(updateBuffer) >= 10 {  // 累积到一定量再更新
            fullContent += updateBuffer
            updateBuffer = ""
            updateCard(fullContent)
        }
    }
}
```

**参数说明:**
- `minUpdateInterval`: 200ms (最小更新间隔)
- `minBufferSize`: 10 个字符 (最小缓冲大小)

---

## 性能优化要点

### 1. 当前性能瓶颈

#### 问题: 工具调用后的响应延迟

**现象:**
```
16:18:58.660 - 工具查询完成 ✅
          ↓
     [等待 ~10秒]  ⏱️
          ↓
16:19:10.137 - 卡片更新完成 ✅
```

**原因:**
- 工具调用使用非流式 API (为了解析 `tool_calls`)
- 工具调用后的最终响应也使用非流式 API
- 必须等待 LLM 生成完整内容才返回

**代码位置:**
```go
// internal/llm/openai.go:417
// ⚠️ 非流式调用
resp, err := openaiClient.ChatWithTools(ctx, messages, tools)

// internal/llm/openai.go:467
// ⚠️ 非流式返回最终内容
if choice.Message.Content != "" {
    responseCh <- choice.Message.Content  // 一次性推送全部内容
    return
}
```

### 2. 优化方案

#### 方案 A: 工具调用后强制流式 (简单)

```go
// 在工具调用完成后,清空工具列表
if len(choice.Message.ToolCalls) > 0 {
    // 执行工具调用...

    // ✨ 关键修改: 强制下一轮使用流式
    tools = nil
    continue
}
```

**优点:**
- 修改简单,只需 1 行代码
- 立即生效

**缺点:**
- 如果需要多轮工具调用,后续无法再调用工具

#### 方案 B: 全流程流式 (优雅)

使用 OpenAI 的流式 API,并手动解析 `tool_calls` delta:

```go
stream, _ := client.CreateChatCompletionStream(ctx, openai.ChatCompletionRequest{
    Model:    model,
    Messages: messages,
    Tools:    tools,  // ✨ 流式 API 也支持工具!
    Stream:   true,
})

toolCallsBuffer := make(map[string]*ToolCall)

for {
    response, _ := stream.Recv()
    delta := response.Choices[0].Delta

    // 处理内容流
    if delta.Content != "" {
        responseCh <- delta.Content  // 实时推送
    }

    // 处理工具调用流 (逐步累积)
    if len(delta.ToolCalls) > 0 {
        for _, tc := range delta.ToolCalls {
            if existing, ok := toolCallsBuffer[tc.Index]; ok {
                // 累积参数
                existing.Function.Arguments += tc.Function.Arguments
            } else {
                // 新工具调用
                toolCallsBuffer[tc.Index] = &tc
            }
        }
    }

    if response.Choices[0].FinishReason == "tool_calls" {
        // 工具调用完整,执行工具
        for _, toolCall := range toolCallsBuffer {
            executeToolCall(toolCall)
        }
    }
}
```

**优点:**
- 全流程流式,体验最佳
- 支持多轮工具调用

**缺点:**
- 需要重构流式解析逻辑
- 代码复杂度较高

### 3. 其他优化建议

#### 3.1 Provider 层缓存

```go
// 缓存云资源查询结果 (TTL: 5分钟)
type CachedProvider struct {
    cache *cache.Cache
    provider Provider
}

func (p *CachedProvider) ListInstances(ctx context.Context, opts *QueryOptions) ([]*Instance, error) {
    key := fmt.Sprintf("instances:%s:%s", opts.Region, opts.PageNum)

    // 尝试从缓存获取
    if cached, found := p.cache.Get(key); found {
        return cached.([]*Instance), nil
    }

    // 缓存未命中,查询并缓存
    instances, err := p.provider.ListInstances(ctx, opts)
    if err == nil {
        p.cache.Set(key, instances, 5*time.Minute)
    }

    return instances, err
}
```

#### 3.2 并行工具调用

当需要调用多个独立工具时,可以并行执行:

```go
// 串行执行 (慢)
for _, toolCall := range toolCalls {
    result := executeToolCall(toolCall)
    results = append(results, result)
}

// 并行执行 (快)
var wg sync.WaitGroup
results := make([]string, len(toolCalls))

for i, toolCall := range toolCalls {
    wg.Add(1)
    go func(idx int, tc ToolCall) {
        defer wg.Done()
        results[idx] = executeToolCall(tc)
    }(i, toolCall)
}

wg.Wait()
```

#### 3.3 减少 API 调用

**卡片更新优化:**
- 增加缓冲区大小 (10 → 20 字符)
- 增加更新间隔 (200ms → 300ms)
- 仅在有实质性变化时更新

```go
// 计算内容差异
if len(newContent) - len(oldContent) < 20 {
    continue  // 变化太小,跳过更新
}
```

---

## 时序图

### 完整交互时序图

```
用户      钉钉      Stream      LLM       MCP      Provider
 │        │       Handler    Client    Server     Layer
 │        │          │          │         │          │
 │ 发送消息 │          │          │         │          │
 ├────────>│          │          │         │          │
 │        │ 推送事件  │          │         │          │
 │        ├─────────>│          │         │          │
 │        │          │ 创建卡片  │         │          │
 │        │<─────────┤          │         │          │
 │        │          │ 调用LLM   │         │          │
 │        │          ├─────────>│         │          │
 │        │          │          │ 获取工具 │          │
 │        │          │          ├────────>│          │
 │        │          │          │<────────┤          │
 │        │          │          │         │          │
 │        │          │          │ 调用工具 │          │
 │        │          │          ├────────>│          │
 │        │          │          │         │ 查询资源  │
 │        │          │          │         ├─────────>│
 │        │          │          │         │<─────────┤
 │        │          │          │<────────┤          │
 │        │          │          │         │          │
 │        │          │          │ 生成回复 │          │
 │        │          │<─────────┤ (流式)  │          │
 │        │          │          │         │          │
 │        │ 更新卡片  │          │         │          │
 │        │<─────────┤          │         │          │
 │<───────┤          │          │         │          │
 │ 看到回复 │          │          │         │          │
```

---

## 配置说明

### 钉钉相关配置

```yaml
dingtalk:
  enabled: true
  mode: "stream"              # stream 或 webhook
  use_card: true              # 是否使用消息卡片

  # Stream 模式配置
  app_key: "your_app_key"
  app_secret: "your_app_secret"

  # 回调配置 (Webhook 模式)
  callback:
    token: "your_token"
    aes_key: "your_aes_key"
```

### LLM 相关配置

```yaml
llm:
  provider: "deepseek"        # 或 "qwen", "openai"
  api_key: "your_api_key"
  base_url: "https://api.deepseek.com"
  model: "deepseek-chat"

  # 对话参数
  temperature: 0.7
  max_tokens: 4000
  timeout: 60s
```

### 性能调优配置

```yaml
# 卡片更新策略
card:
  min_update_interval: 200ms   # 最小更新间隔
  min_buffer_size: 10          # 最小缓冲大小

# 缓存配置
cache:
  enabled: true
  ttl: 5m                      # 缓存过期时间
```

---

## 调试技巧

### 1. 查看完整日志

```bash
# 启动服务,查看详细日志
./zenops serve --log-level debug

# 关键日志:
# - "Received message from": 收到用户消息
# - "Starting LLM conversation": 开始 LLM 调用
# - "Calling tool": 调用工具
# - "Successfully queried": 工具查询成功
# - "LLM conversation completed": 对话完成
```

### 2. 测试工具调用

```bash
# 直接测试 MCP 工具
./zenops query aliyun ecs --ip 10.0.1.100
```

### 3. 监控性能

```go
// 添加性能埋点
start := time.Now()
result, _ := executeToolCall(toolCall)
logx.Info("Tool execution took %v", time.Since(start))
```

---

## 总结

ZenOps 钉钉机器人的交互流程涉及多个层次:

1. **接入层:** 钉钉 Stream 事件接收
2. **处理层:** 消息解析和任务调度
3. **智能层:** LLM 理解和决策
4. **执行层:** MCP 工具调用
5. **数据层:** Provider 云资源查询
6. **展示层:** 消息卡片实时更新

整个流程充分利用了 **异步处理、流式响应、缓冲优化** 等技术,在保证功能完整的同时,尽可能提升用户体验。

当前的主要优化方向是 **工具调用后的流式响应**,通过改进可以将响应延迟从 10 秒降低到接近实时。
