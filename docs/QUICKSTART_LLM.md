# ZenOps LLM 智能对话快速入门

本指南帮助你快速启用 ZenOps 的 LLM 智能对话功能。

## 前置条件

1. ✅ 已配置钉钉机器人(Stream 模式)
2. ✅ 已配置云服务提供商(阿里云/腾讯云等)
3. ✅ 已配置 CI/CD 工具(Jenkins 等)
4. ✅ 拥有 LLM API Key(OpenAI、DeepSeek 等)

## 5 分钟快速启动

### 步骤 1: 编辑配置文件

复制配置示例:
```bash
cp config.example.yml config.yml
```

编辑 `config.yml`,启用 LLM:

```yaml
# 钉钉配置
dingtalk:
  enabled: true
  mode: "stream"
  app_key: "你的AppKey"
  app_secret: "你的AppSecret"
  agent_id: "你的AgentID"

  # 启用 LLM 对话
  enable_llm_conversation: true

  # 暂不启用流式卡片
  enable_stream_card: false

# LLM 配置
llm:
  enabled: true
  provider: "deepseek"  # 推荐使用 DeepSeek,成本低
  model: "deepseek-chat"
  api_key: "你的API密钥"
  base_url: "https://api.deepseek.com"
```

### 步骤 2: 启动服务

```bash
./bin/zenops
```

### 步骤 3: 测试对话

在钉钉群里 @机器人:

```
@机器人 帮我查询一下阿里云有多少台 ECS 服务器
```

机器人会:
1. 🤖 显示"正在思考中"
2. 🔧 自动调用 `aliyun_ecs_list` 工具
3. ✅ 返回查询结果和分析

## 进阶配置

### 启用流式卡片(可选)

流式卡片提供更好的交互体验,但需要额外配置。

#### 1. 创建卡片模板

登录钉钉开放平台 -> 你的应用 -> 互动卡片 -> 创建卡片模板

**模板示例**:
```json
{
  "config": {
    "autoLayout": true,
    "enableForward": true
  },
  "header": {
    "title": {
      "type": "text",
      "text": "ZenOps 智能助手"
    }
  },
  "contents": [
    {
      "type": "markdown",
      "text": "{{content}}"
    }
  ]
}
```

获取模板 ID(格式如: `xxx.schema`)

#### 2. 更新配置

```yaml
dingtalk:
  enable_stream_card: true
  card_template_id: "你的模板ID.schema"
```

#### 3. 重启服务

```bash
./bin/zenops
```

## 常见问题

### Q1: LLM 不调用工具怎么办?

**原因**: 提问不够明确或工具定义不清晰

**解决**:
- 使用明确的提问,如"查询阿里云 ECS 列表"
- 检查 MCP 工具是否正确注册
- 查看日志确认工具列表

### Q2: 响应很慢怎么办?

**原因**: LLM API 延迟或工具执行时间长

**解决**:
- 切换到响应更快的 LLM 提供商
- 优化工具执行逻辑
- 使用流式卡片提升体验

### Q3: 卡片创建失败怎么办?

**原因**: 模板 ID 错误或权限不足

**表现**: 系统会自动降级为普通流式消息

**解决**:
- 检查模板 ID 是否正确
- 确认应用有卡片权限
- 查看日志了解详细错误

### Q4: API 成本太高怎么办?

**解决方案**:
1. 使用 DeepSeek 等成本较低的提供商
2. 添加使用频率限制
3. 配置白名单限制用户

### Q5: 如何调试?

**查看日志**:
```bash
# 设置日志级别为 debug
logging:
  level: "debug"
```

**常用日志关键词**:
- `LLM client initialized`: LLM 初始化成功
- `Processing LLM`: 开始处理 LLM 对话
- `Calling MCP tool`: 调用 MCP 工具
- `Failed to call LLM`: LLM 调用失败

## LLM 提供商选择

### DeepSeek (推荐新手)
- ✅ 成本低(约 OpenAI 的 1/10)
- ✅ 中文支持好
- ✅ 兼容 OpenAI API
- ⚠️ 响应速度一般

```yaml
llm:
  provider: "deepseek"
  model: "deepseek-chat"
  api_key: "sk-xxx"
  base_url: "https://api.deepseek.com"
```

### OpenAI (推荐专业用户)
- ✅ 响应速度快
- ✅ 功能最全面
- ✅ 生态最好
- ⚠️ 成本较高
- ⚠️ 需要特殊网络

```yaml
llm:
  provider: "openai"
  model: "gpt-4"
  api_key: "sk-xxx"
  base_url: ""  # 使用默认
```

### Azure OpenAI (推荐企业用户)
- ✅ 稳定性好
- ✅ 合规性好
- ✅ 国内可用
- ⚠️ 需要企业账号

```yaml
llm:
  provider: "azure"
  model: "gpt-4"
  api_key: "your-key"
  base_url: "https://your-resource.openai.azure.com"
```

## 实际使用示例

### 示例 1: 资源查询

```
用户: 列出杭州地域的所有 ECS 实例

机器人:
🔧 调用工具: aliyun_ecs_list
✅ 工具执行完成

杭州地域当前有 8 台 ECS 实例:

[详细列表...]
```

### 示例 2: 多云对比

```
用户: 对比阿里云和腾讯云的服务器数量

机器人:
🔧 调用工具: aliyun_ecs_list
✅ 工具执行完成

🔧 调用工具: tencent_cvm_list
✅ 工具执行完成

对比结果:
- 阿里云: 15 台
- 腾讯云: 12 台
- 总计: 27 台
```

### 示例 3: CI/CD 查询

```
用户: 最近有哪些 Jenkins 任务失败了?

机器人:
🔧 调用工具: jenkins_list_builds
✅ 工具执行完成

最近 24 小时内有 3 个任务失败:
[失败列表和分析...]
```

## 下一步

- 📖 阅读完整文档: [docs/DINGTALK_LLM.md](./DINGTALK_LLM.md)
- 🛠️ 了解实现细节: [docs/IMPLEMENTATION_SUMMARY.md](./IMPLEMENTATION_SUMMARY.md)
- 🎨 配置流式卡片获得更好体验
- 🔒 配置权限控制保证安全

## 获取帮助

- 查看日志: 设置 `logging.level: "debug"`
- 查看工具列表: 在钉钉发送"帮助"
- 提交 Issue: GitHub Issues
- 查看示例: `config.example.yml`

---

祝你使用愉快! 🎉
