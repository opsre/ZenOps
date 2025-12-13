# 钉钉 + LLM 智能对话功能说明

本文档说明如何使用 ZenOps 的钉钉集成和 LLM 智能对话功能。

## 功能概述

ZenOps 现在支持通过钉钉机器人与 LLM 进行智能对话,并且 LLM 可以自动调用 MCP 工具来查询云资源、CI/CD 任务等信息。

### 主要特性

1. **智能对话**: 支持多种 LLM 提供商(OpenAI、DeepSeek、Claude 等)
2. **工具调用**: LLM 自动调用 MCP 工具获取实时数据
3. **流式响应**: 支持流式卡片更新,实时展示 LLM 回复
4. **多模式支持**:
   - 传统的意图解析模式
   - LLM 智能对话模式(带 MCP 工具调用)
   - 流式卡片交互模式

## 配置说明

### 1. 基础配置

编辑 `config.yml` 文件:

```yaml
# 钉钉配置
dingtalk:
  enabled: true
  mode: "stream"  # 使用 Stream 模式
  app_key: "YOUR_APP_KEY"
  app_secret: "YOUR_APP_SECRET"
  agent_id: "YOUR_AGENT_ID"

  # 启用 LLM 对话
  enable_llm_conversation: true

  # (可选)启用流式卡片
  enable_stream_card: false
  card_template_id: ""

# LLM 配置
llm:
  enabled: true
  provider: "openai"  # 或 deepseek, claude 等
  model: "gpt-4"
  api_key: "YOUR_API_KEY"
  base_url: ""  # 可选,用于自定义 API 端点
```

### 2. LLM 提供商配置

#### OpenAI

```yaml
llm:
  enabled: true
  provider: "openai"
  model: "gpt-4"
  api_key: "sk-..."
  base_url: ""  # 使用默认的 OpenAI API
```

#### DeepSeek

```yaml
llm:
  enabled: true
  provider: "deepseek"
  model: "deepseek-chat"
  api_key: "YOUR_DEEPSEEK_API_KEY"
  base_url: "https://api.deepseek.com"
```

#### Azure OpenAI

```yaml
llm:
  enabled: true
  provider: "azure"
  model: "gpt-4"
  api_key: "YOUR_AZURE_KEY"
  base_url: "https://YOUR-RESOURCE.openai.azure.com"
```

### 3. 流式卡片配置(可选)

流式卡片可以提供更好的交互体验,实时显示 LLM 的回复过程。

#### 步骤 1: 创建卡片模板

1. 登录钉钉开放平台
2. 进入你的应用
3. 选择"互动卡片" -> "创建卡片模板"
4. 创建一个包含 `content` 字段的流式卡片模板
5. 获取模板 ID

#### 步骤 2: 配置模板 ID

```yaml
dingtalk:
  enable_stream_card: true
  card_template_id: "YOUR_CARD_TEMPLATE_ID"
```

## 使用场景

### 场景 1: 传统意图解析(不使用 LLM)

当 `enable_llm_conversation: false` 时,系统使用传统的关键词匹配模式:

```
用户: 查询阿里云 ECS 列表
机器人: [返回格式化的 ECS 列表]
```

### 场景 2: LLM 智能对话(使用 MCP 工具)

当 `enable_llm_conversation: true` 时,LLM 会理解用户意图并自动调用相应工具:

```
用户: 帮我看看生产环境有多少台阿里云 ECS 服务器
机器人:
  🔧 调用工具: aliyun_ecs_list
  ✅ 工具执行完成

  根据查询结果,生产环境当前有 15 台阿里云 ECS 服务器,分布如下:
  - cn-hangzhou: 8 台
  - cn-shanghai: 7 台

  所有服务器运行状态正常。
```

### 场景 3: 流式卡片交互

启用流式卡片后,回复会以卡片形式实时更新:

```
用户: 分析一下最近的 Jenkins 构建情况

[卡片实时更新显示]
帮我分析一下最近的 Jenkins 构建情况

正在思考中...

[逐字流式显示]
我来帮你查询最近的 Jenkins 构建情况...

🔧 调用工具: jenkins_list_jobs
✅ 工具执行完成

根据查询结果,最近的构建情况如下:
...
```

## 工作原理

### MCP 工具自动调用流程

1. 用户发送消息到钉钉机器人
2. ZenOps 将消息转发给 LLM
3. LLM 分析用户意图,决定是否需要调用工具
4. 如需调用工具,LLM 返回工具调用请求
5. ZenOps 执行对应的 MCP 工具
6. 工具执行结果返回给 LLM
7. LLM 基于工具结果生成最终回复
8. 回复以流式方式返回给用户

### 流程图

```
用户消息 -> 钉钉机器人 -> ZenOps
                              |
                              v
                            LLM 分析
                              |
                    是否需要调用工具?
                   /                \
                Yes                  No
                 |                    |
              调用 MCP 工具         直接回复
                 |                    |
              工具结果 -> LLM         |
                 |                    |
                 +--------------------+
                          |
                      流式返回
                          |
                      用户收到回复
```

## 示例对话

### 示例 1: 查询云资源

```
用户: 列出腾讯云广州地区的 CVM 实例

机器人:
🔧 调用工具: tencent_cvm_list
✅ 工具执行完成

广州地区当前有 12 台 CVM 实例:

| 实例 ID | 实例名称 | 状态 | IP 地址 |
|---------|----------|------|---------|
| ins-xxx1 | web-server-1 | 运行中 | 10.0.1.10 |
| ins-xxx2 | web-server-2 | 运行中 | 10.0.1.11 |
...
```

### 示例 2: CI/CD 任务查询

```
用户: 最近有哪些 Jenkins 任务失败了?

机器人:
🔧 调用工具: jenkins_list_builds
✅ 工具执行完成

最近 24 小时内有 3 个任务失败:

1. backend-api-build #125
   - 失败时间: 2024-01-15 14:30
   - 失败原因: 单元测试失败

2. frontend-deploy #88
   - 失败时间: 2024-01-15 16:20
   - 失败原因: 部署超时

3. database-migration #45
   - 失败时间: 2024-01-15 18:10
   - 失败原因: 数据库连接失败

建议优先处理 database-migration 任务。
```

### 示例 3: 复杂查询

```
用户: 对比一下阿里云和腾讯云的 ECS/CVM 数量

机器人:
🔧 调用工具: aliyun_ecs_list
✅ 工具执行完成

🔧 调用工具: tencent_cvm_list
✅ 工具执行完成

根据查询结果,两个云平台的资源对比如下:

阿里云 ECS: 15 台
  - 生产环境: 10 台
  - 测试环境: 5 台

腾讯云 CVM: 12 台
  - 生产环境: 8 台
  - 测试环境: 4 台

总计: 27 台云主机
建议: 两个平台的资源分布较为均衡。
```

## 注意事项

1. **API 费用**: LLM 调用会产生 API 费用,请注意控制使用量
2. **响应时间**: 涉及工具调用的对话可能需要较长时间
3. **隐私安全**: 确保敏感信息不会被发送到 LLM 服务商
4. **卡片模板**: 流式卡片需要在钉钉开放平台创建模板
5. **权限控制**: 建议配置白名单限制使用范围

## 故障排查

### LLM 调用失败

1. 检查 API Key 是否正确
2. 检查网络连接是否正常
3. 检查 base_url 配置是否正确
4. 查看日志了解详细错误信息

### 工具调用失败

1. 检查 MCP 工具是否正常注册
2. 检查云服务凭证是否有效
3. 检查工具参数是否正确
4. 查看日志了解详细错误

### 流式卡片不工作

1. 确认已创建卡片模板
2. 确认 card_template_id 配置正确
3. 确认钉钉应用权限充足
4. 查看是否有错误日志

系统会自动降级:如果流式卡片创建失败,会自动使用普通流式消息。

## 开发参考

### 添加新的 LLM 提供商

在 `internal/llm/openai.go` 中实现新的客户端:

```go
// 示例: 添加 Claude 支持
func NewClaudeClient(config *Config) *ClaudeClient {
    // 实现 Claude 客户端
}
```

### 自定义 MCP 工具

参考 MCP 文档创建新工具并注册到 MCP Server。

## 更多信息

- [钉钉开放平台文档](https://open.dingtalk.com/)
- [MCP 协议说明](https://github.com/mark3labs/mcp-go)
- [OpenAI API 文档](https://platform.openai.com/docs)
