# ZenOps 配置参数参考

## 配置文件位置

主配置文件: `config.yml`

## LLM 配置

### llm.enabled
- **类型**: `bool`
- **默认值**: `false`
- **说明**: 是否启用 LLM 功能
- **示例**:
  ```yaml
  llm:
    enabled: true
  ```

### llm.provider
- **类型**: `string`
- **可选值**: `openai`, `deepseek`, `azure`
- **说明**: LLM 提供商
- **示例**:
  ```yaml
  llm:
    provider: "deepseek"  # 推荐,成本低
  ```

### llm.model
- **类型**: `string`
- **说明**: 使用的模型名称
- **示例**:
  ```yaml
  # DeepSeek
  llm:
    model: "deepseek-chat"

  # OpenAI
  llm:
    model: "gpt-4"

  # Azure
  llm:
    model: "gpt-35-turbo"
  ```

### llm.api_key
- **类型**: `string`
- **必需**: 是(当 `enabled: true` 时)
- **说明**: LLM API 密钥
- **示例**:
  ```yaml
  llm:
    api_key: "sk-xxxxxxxxxxxxxxxx"
  ```
- **安全提示**:
  - 不要提交到代码库
  - 可以使用环境变量: `${LLM_API_KEY}`

### llm.base_url
- **类型**: `string`
- **必需**: 否
- **说明**: API 端点地址
- **示例**:
  ```yaml
  # DeepSeek
  llm:
    base_url: "https://api.deepseek.com"

  # OpenAI(默认,可不填)
  llm:
    base_url: ""

  # Azure
  llm:
    base_url: "https://your-resource.openai.azure.com"

  # 自定义代理
  llm:
    base_url: "https://your-proxy.com/v1"
  ```

## 钉钉配置

### dingtalk.enable_llm_conversation
- **类型**: `bool`
- **默认值**: `false`
- **说明**: 是否启用 LLM 对话模式
- **依赖**: 需要 `llm.enabled: true`
- **示例**:
  ```yaml
  dingtalk:
    enable_llm_conversation: true
  ```
- **效果**:
  - `true`: 使用 LLM 处理用户消息
  - `false`: 使用传统意图解析

### dingtalk.enable_stream_card
- **类型**: `bool`
- **默认值**: `false`
- **说明**: 是否启用流式卡片
- **依赖**: 需要配置 `card_template_id`
- **示例**:
  ```yaml
  dingtalk:
    enable_stream_card: true
    card_template_id: "xxx.schema"
  ```
- **效果**:
  - `true`: 使用流式卡片显示响应
  - `false`: 使用普通流式文本消息

### dingtalk.card_template_id
- **类型**: `string`
- **必需**: 否
- **说明**: 钉钉流式卡片模板 ID
- **格式**: `xxx.schema`
- **示例**:
  ```yaml
  dingtalk:
    card_template_id: "d1a4a0d0-1234-5678-90ab-cdef12345678.schema"
  ```
- **获取方式**:
  1. 登录钉钉开放平台
  2. 进入应用 → 互动卡片
  3. 创建流式卡片模板
  4. 复制模板 ID
- **注意**:
  - 不配置也能正常使用(会使用文本消息)
  - 配置错误会自动降级为文本消息

## 日志配置

### logging.level
- **类型**: `string`
- **可选值**: `debug`, `info`, `warn`, `error`
- **默认值**: `info`
- **说明**: 日志级别
- **示例**:
  ```yaml
  logging:
    level: "debug"  # 测试时推荐
  ```
- **日志内容**:
  - `debug`: 所有日志(包括详细调试信息)
  - `info`: 一般信息日志
  - `warn`: 警告和错误
  - `error`: 仅错误

## 配置组合模式

### 模式 1: 传统模式(无 LLM)

**使用场景**: 快速响应,精确命令

```yaml
llm:
  enabled: false

dingtalk:
  enabled: true
  mode: "stream"
  enable_llm_conversation: false
  enable_stream_card: false
  card_template_id: ""
```

**特点**:
- ✅ 配置简单
- ✅ 响应快速
- ⚠️ 需要精确命令
- ⚠️ 无智能对话

### 模式 2: LLM 对话(文本消息)

**使用场景**: 自然语言交互,快速部署

```yaml
llm:
  enabled: true
  provider: "deepseek"
  model: "deepseek-chat"
  api_key: "sk-xxx"
  base_url: "https://api.deepseek.com"

dingtalk:
  enabled: true
  mode: "stream"
  enable_llm_conversation: true
  enable_stream_card: false
  card_template_id: ""

logging:
  level: "debug"
```

**特点**:
- ✅ 自然语言交互
- ✅ 自动调用工具
- ✅ 流式文本响应
- ✅ 无需配置卡片
- ⚠️ 需要 LLM API Key

### 模式 3: LLM 对话(流式卡片)

**使用场景**: 最佳用户体验

```yaml
llm:
  enabled: true
  provider: "deepseek"
  model: "deepseek-chat"
  api_key: "sk-xxx"
  base_url: "https://api.deepseek.com"

dingtalk:
  enabled: true
  mode: "stream"
  enable_llm_conversation: true
  enable_stream_card: true
  card_template_id: "xxx.schema"

logging:
  level: "info"
```

**特点**:
- ✅ 自然语言交互
- ✅ 自动调用工具
- ✅ 实时卡片更新
- ✅ 最佳用户体验
- ⚠️ 需要 LLM API Key
- ⚠️ 需要配置卡片模板

## 环境变量支持

配置文件支持环境变量替换:

```yaml
llm:
  api_key: "${LLM_API_KEY}"

dingtalk:
  app_key: "${DINGTALK_APP_KEY}"
  app_secret: "${DINGTALK_APP_SECRET}"
```

设置环境变量:
```bash
export LLM_API_KEY="sk-xxx"
export DINGTALK_APP_KEY="xxx"
export DINGTALK_APP_SECRET="xxx"
```

## 配置验证

### 启动时检查

服务启动时会验证配置:

```bash
# 正确配置
INFO  LLM client initialized successfully
INFO  DingTalk stream handler initialized with LLM support

# LLM 未启用
INFO  LLM is disabled
INFO  DingTalk stream handler initialized (traditional mode)

# 配置错误
ERROR Failed to initialize LLM client: invalid API key
```

### 运行时检查

发送消息时的行为:

```bash
# LLM 模式
INFO  Using LLM to process message

# 传统模式
DEBUG Intent parsing mode

# 卡片模式
DEBUG Creating stream card

# 卡片降级
WARN  Failed to create card, fallback to text reply
```

## 性能调优

### 响应速度优化

```yaml
llm:
  provider: "openai"  # OpenAI 响应通常更快
  model: "gpt-3.5-turbo"  # 更小的模型响应更快
```

### 成本优化

```yaml
llm:
  provider: "deepseek"  # DeepSeek 成本最低
  model: "deepseek-chat"
```

### 稳定性优化

```yaml
dingtalk:
  enable_stream_card: false  # 禁用卡片,使用文本更稳定

logging:
  level: "info"  # 生产环境使用 info 级别
```

## 安全建议

### API Key 安全

❌ **不要这样做**:
```yaml
llm:
  api_key: "sk-1234567890abcdef"  # 直接写在配置文件
```

✅ **推荐做法**:
```yaml
llm:
  api_key: "${LLM_API_KEY}"  # 使用环境变量
```

```bash
# .env 文件(不要提交到 Git)
LLM_API_KEY=sk-1234567890abcdef

# 或在启动时设置
export LLM_API_KEY="sk-1234567890abcdef"
./bin/zenops
```

### 权限控制

```yaml
dingtalk:
  # TODO: 未来版本添加
  # allowed_users: ["user1", "user2"]
  # allowed_groups: ["group1", "group2"]
```

## 故障排查

### 配置文件不生效

**检查**:
1. 文件路径是否正确: `config.yml`
2. YAML 格式是否正确(注意缩进)
3. 是否重启服务

**验证 YAML 格式**:
```bash
# 使用 yamllint 检查
yamllint config.yml

# 或使用在线工具
# https://www.yamllint.com/
```

### LLM API Key 错误

**错误日志**:
```
ERROR Failed to call LLM: 401 Unauthorized
```

**解决**:
1. 检查 API Key 是否正确
2. 检查 API Key 是否有效
3. 检查账户余额是否充足

### 卡片模板 ID 错误

**错误日志**:
```
ERROR Failed to create card: param.cardTemplateIdEmpty
```

**解决**:
1. 检查模板 ID 格式(应为 `xxx.schema`)
2. 确认模板已发布
3. 暂时禁用卡片: `enable_stream_card: false`

## 完整配置示例

```yaml
# ZenOps 配置文件

# 日志配置
logging:
  level: "debug"  # debug, info, warn, error

# LLM 配置
llm:
  enabled: true
  provider: "deepseek"  # openai, deepseek, azure
  model: "deepseek-chat"
  api_key: "${LLM_API_KEY}"  # 使用环境变量
  base_url: "https://api.deepseek.com"

# 钉钉配置
dingtalk:
  enabled: true
  mode: "stream"
  app_key: "${DINGTALK_APP_KEY}"
  app_secret: "${DINGTALK_APP_SECRET}"
  agent_id: "${DINGTALK_AGENT_ID}"

  # LLM 对话模式
  enable_llm_conversation: true

  # 流式卡片(可选)
  enable_stream_card: false
  card_template_id: ""

# 云服务配置
aliyun:
  enabled: true
  access_key_id: "${ALIYUN_ACCESS_KEY_ID}"
  access_key_secret: "${ALIYUN_ACCESS_KEY_SECRET}"

tencent:
  enabled: true
  secret_id: "${TENCENT_SECRET_ID}"
  secret_key: "${TENCENT_SECRET_KEY}"

# CI/CD 配置
jenkins:
  enabled: true
  url: "https://jenkins.example.com"
  username: "${JENKINS_USERNAME}"
  token: "${JENKINS_TOKEN}"
```

## 配置模板下载

完整配置示例文件: [config.example.yml](../config.example.yml)

```bash
# 复制配置模板
cp config.example.yml config.yml

# 编辑配置
vim config.yml
```

## 更多帮助

- 快速入门: [QUICKSTART_LLM.md](./QUICKSTART_LLM.md)
- 测试指南: [TESTING_GUIDE.md](./TESTING_GUIDE.md)
- 功能说明: [DINGTALK_LLM.md](./DINGTALK_LLM.md)
- 卡片配置: [CARD_TEMPLATE_OPTIONAL.md](./CARD_TEMPLATE_OPTIONAL.md)

---

最后更新: 2025-12-11
