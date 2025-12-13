# ZenOps LLM 功能测试指南

## 测试前准备

### 1. 确认配置文件

编辑 `config.yml`,确保以下配置正确:

```yaml
# LLM 配置
llm:
  enabled: true
  provider: "deepseek"  # 或 openai, azure
  model: "deepseek-chat"
  api_key: "sk-your-api-key-here"  # 替换为你的实际 API Key
  base_url: "https://api.deepseek.com"  # DeepSeek 的 API 地址

# 钉钉配置
dingtalk:
  enabled: true
  mode: "stream"
  app_key: "your-app-key"
  app_secret: "your-app-secret"
  agent_id: "your-agent-id"

  # 启用 LLM 对话
  enable_llm_conversation: true

  # 卡片配置(可选)
  enable_stream_card: false  # 建议先测试文本模式
  card_template_id: ""       # 暂不配置
```

### 2. 验证日志级别

确保日志级别设置为 `debug` 以便查看详细信息:

```yaml
logging:
  level: "debug"
```

### 3. 重新编译和启动

```bash
# 编译
make build

# 或直接运行
go build -o bin/zenops .

# 启动服务
./bin/zenops
```

## 测试步骤

### 测试 1: 验证 LLM 初始化

**预期日志输出**:
```
INFO  LLM client initialized successfully
INFO  DingTalk stream handler initialized with LLM support
```

**如果看到以下日志,说明配置正确**:
```
DEBUG LLM config: provider=deepseek, model=deepseek-chat
```

### 测试 2: 简单对话测试

在钉钉中 @机器人,发送:
```
你好
```

**预期行为**:
- 终端日志显示: `Using LLM to process message`
- 机器人返回 LLM 的回复(不调用工具)

**预期日志**:
```
INFO  Using LLM to process message
DEBUG Processing LLM message: 你好
DEBUG LLM response received (no tools called)
```

### 测试 3: 工具调用测试

在钉钉中发送需要查询数据的问题:
```
帮我查询一下阿里云有多少台 ECS 服务器
```

**预期行为**:
1. 机器人显示 "正在思考中..."
2. 调用 `aliyun_ecs_list` 工具
3. 返回查询结果和分析

**预期日志**:
```
INFO  Using LLM to process message
DEBUG Processing LLM message: 帮我查询一下阿里云有多少台 ECS 服务器
DEBUG LLM requesting tool call: aliyun_ecs_list
INFO  Calling MCP tool: aliyun_ecs_list
DEBUG Tool execution result: [...]
DEBUG LLM final response: 根据查询结果...
```

### 测试 4: 多工具调用测试

发送需要调用多个工具的问题:
```
对比一下阿里云和腾讯云的服务器数量
```

**预期行为**:
1. LLM 调用 `aliyun_ecs_list`
2. LLM 调用 `tencent_cvm_list`
3. LLM 综合分析并返回对比结果

**预期日志**:
```
DEBUG LLM requesting tool call: aliyun_ecs_list
INFO  Calling MCP tool: aliyun_ecs_list
DEBUG LLM requesting tool call: tencent_cvm_list
INFO  Calling MCP tool: tencent_cvm_list
DEBUG LLM final response: 对比结果...
```

### 测试 5: 流式卡片测试(可选)

如果你已配置卡片模板ID,编辑配置:
```yaml
dingtalk:
  enable_stream_card: true
  card_template_id: "your-template-id.schema"
```

重启服务后,发送相同的问题,观察卡片是否实时更新。

**预期日志**:
```
DEBUG Creating stream card
INFO  Card created successfully, trackID: xxx
DEBUG Streaming update: 0/500 chars
DEBUG Streaming update: 500/1000 chars
DEBUG Streaming update: finalized
```

## 常见问题排查

### 问题 1: 日志显示 "无法理解您的请求"

**原因**: LLM 未被调用,仍在使用意图解析器

**检查**:
```bash
# 查看日志,应该看到:
INFO  Using LLM to process message

# 如果看到以下内容,说明 LLM 未启用:
DEBUG Intent parsing result: unknown
```

**解决**:
- 确认 `config.yml` 中 `llm.enabled: true`
- 确认 `dingtalk.enable_llm_conversation: true`
- 重启服务

### 问题 2: LLM 调用失败

**错误日志**:
```
ERROR Failed to call LLM: ...
```

**检查**:
1. API Key 是否正确
2. Base URL 是否正确
3. 网络是否可达
4. API 额度是否充足

**测试 API 连接**:
```bash
# DeepSeek 测试
curl https://api.deepseek.com/v1/models \
  -H "Authorization: Bearer sk-your-key"

# OpenAI 测试
curl https://api.openai.com/v1/models \
  -H "Authorization: Bearer sk-your-key"
```

### 问题 3: 工具未被调用

**症状**: LLM 回复了问题,但没有调用工具获取数据

**原因**:
1. 提问不够明确
2. MCP 工具未注册
3. 工具定义不清晰

**检查工具列表**:
查看启动日志中的工具列表:
```
DEBUG Available MCP tools: [aliyun_ecs_list, tencent_cvm_list, ...]
```

**改进提问**:
- ❌ "有多少服务器?" (太模糊)
- ✅ "查询阿里云 ECS 列表" (明确)
- ✅ "帮我看看腾讯云有多少台 CVM" (明确)

### 问题 4: 卡片创建失败

**错误日志**:
```
ERROR Failed to create card, fallback to text reply
```

**解决**:
- 检查 `card_template_id` 是否正确
- 确认应用有卡片权限
- 暂时禁用卡片: `enable_stream_card: false`

### 问题 5: 响应很慢

**原因**: LLM API 延迟或工具执行时间长

**优化方案**:
1. 切换到响应更快的 LLM 提供商
2. 使用流式卡片提升体验感
3. 检查工具执行效率

## 验证成功的标志

✅ **LLM 功能正常的标志**:

1. 日志中看到 `Using LLM to process message`
2. 简单对话能正常回复
3. 明确的查询请求会调用对应的 MCP 工具
4. 工具执行结果被 LLM 正确处理和分析
5. 最终回复准确且自然

✅ **流式更新正常的标志**:

1. 回复内容逐步显示(文本模式)或卡片实时更新
2. 日志中看到 `Streaming update` 记录
3. 最终内容完整

## 性能指标

**正常响应时间**:
- 简单对话: 2-5 秒
- 单工具调用: 5-10 秒
- 多工具调用: 10-20 秒

**如果响应时间过长**:
- 检查 LLM API 延迟
- 检查工具执行时间
- 考虑优化或缓存

## 下一步

测试通过后,可以:
1. 配置流式卡片获得更好体验
2. 添加更多 MCP 工具
3. 优化系统提示词
4. 配置权限控制
5. 启用使用统计

## 获取帮助

如果遇到问题:
1. 查看完整日志: `logging.level: "debug"`
2. 检查配置文件: `config.yml`
3. 参考文档:
   - [docs/QUICKSTART_LLM.md](./QUICKSTART_LLM.md)
   - [docs/DINGTALK_LLM.md](./DINGTALK_LLM.md)
   - [docs/CARD_TEMPLATE_OPTIONAL.md](./CARD_TEMPLATE_OPTIONAL.md)

---

祝测试顺利! 🎉
