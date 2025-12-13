# 钉钉 Stream 模式测试指南

## 📋 前置准备

### 1. 获取钉钉机器人配置

在钉钉开放平台创建应用并获取以下信息:

1. **AppKey** (Client ID)
2. **AppSecret** (Client Secret)
3. **AgentID** (应用 ID,可选)
4. **TemplateID** (AI卡片模板ID,可选,用于流式更新)

> 💡 获取方式: 登录 [钉钉开放平台](https://open-dev.dingtalk.com/) → 应用开发 → 机器人应用 → 查看凭证

### 2. 配置机器人权限

确保你的钉钉机器人应用已开通以下权限:

- ✅ **Stream 推送能力** (必需)
- ✅ **企业内机器人发送消息权限**
- ✅ **通讯录只读权限** (用于获取用户信息)
- ✅ **AI 卡片权限** (如果使用流式卡片)

### 3. 开启 Stream 推送

在钉钉开放平台:
1. 进入应用 → 开发配置
2. 找到 **事件订阅**
3. 选择 **Stream 模式**
4. 订阅 **机器人接收消息** 事件

## 🚀 快速开始

### 步骤 1: 设置环境变量

```bash
# 复制环境变量模板
cp .env.example .env

# 编辑 .env 文件,填入你的配置
vim .env
```

**必需配置:**
```bash
export DINGTALK_APP_KEY='your_app_key_here'
export DINGTALK_APP_SECRET='your_app_secret_here'
```

**可选配置:**
```bash
export DINGTALK_AGENT_ID='your_agent_id'
export DINGTALK_TEMPLATE_ID='your_template_id'

# 云服务商配置
export ALIYUN_ACCESS_KEY_ID='your_ak'
export ALIYUN_ACCESS_KEY_SECRET='your_sk'
export TENCENT_SECRET_ID='your_id'
export TENCENT_SECRET_KEY='your_key'
```

### 步骤 2: 加载环境变量

```bash
source .env
```

### 步骤 3: 验证配置

```bash
./scripts/validate_config.sh
```

### 步骤 4: 启动服务

```bash
# 方式1: 使用测试脚本(推荐)
./scripts/test_dingtalk_stream.sh

# 方式2: 直接启动
./bin/zenops serve --log-level debug
```

## 📝 测试步骤

### 1. 服务启动检查

启动后你应该看到类似日志:

```
[INFO] Starting DingTalk in Stream mode
[INFO] DingTalk Stream mode started successfully app_key=dingxxxx...
[INFO] Starting DingTalk Stream client app_key=dingxxxx...
[INFO] DingTalk Stream client connecting...
```

### 2. 在钉钉中测试

#### 方式 A: 群聊测试
1. 将机器人添加到群聊
2. 在群里 @机器人 发送消息
3. 示例命令:
   ```
   @机器人 帮助
   @机器人 查询阿里云 ECS
   @机器人 列出腾讯云广州的 CVM
   ```

#### 方式 B: 私聊测试
1. 在通讯录找到机器人
2. 直接发送消息(无需@)
3. 示例命令:
   ```
   帮助
   查询阿里云杭州的 ECS
   找一下 IP 为 192.168.1.1 的服务器
   ```

### 3. 预期结果

**成功情况:**
- ✅ 机器人收到消息并回复
- ✅ 如果配置了 AI 卡片模板,会看到流式更新效果
- ✅ 查询结果以卡片形式展示
- ✅ 服务端日志显示消息处理过程

**失败排查:**
- ❌ 机器人无响应 → 检查 Stream 推送是否开启
- ❌ 报错 "无法识别请求" → 检查消息格式
- ❌ 连接失败 → 检查 AppKey/AppSecret 是否正确

## 🔍 日志说明

### 正常日志流程

```
# 1. 启动 Stream 客户端
[INFO] Starting DingTalk Stream client

# 2. 收到消息
[INFO] Received chatbot message sender=张三 conversation_id=xxx

# 3. 解析意图
[DEBUG] Parsing intent message=查询阿里云ECS
[INFO] Intent parsed provider=aliyun resource=ecs action=list

# 4. 调用 MCP 工具
[DEBUG] Calling MCP tool tool=list_ecs params=map[]

# 5. 流式更新卡片
[DEBUG] Streaming update card track_id=track_xxx finalize=false
[DEBUG] Streaming update card track_id=track_xxx finalize=true
```

### 常见错误日志

#### 错误 1: Token 获取失败
```
[ERROR] Failed to get access token error=invalid appkey
```
**解决:** 检查 DINGTALK_APP_KEY 和 DINGTALK_APP_SECRET

#### 错误 2: Stream 连接失败
```
[ERROR] DingTalk Stream client connection failed
```
**解决:**
1. 检查网络连接
2. 确认 Stream 推送已开启
3. 检查防火墙设置

#### 错误 3: 卡片创建失败
```
[ERROR] Failed to create and deliver card error=...
```
**解决:**
1. 检查 DINGTALK_TEMPLATE_ID 是否正确
2. 确认应用有 AI 卡片权限
3. 如果没有模板ID,暂时可以注释掉

## 💡 支持的查询命令

### 阿里云 ECS
- `列出阿里云 ECS`
- `查询阿里云杭州的 ECS`
- `找一下 IP 为 192.168.1.1 的服务器`
- `查询名为 web-server 的实例`

### 阿里云 RDS
- `列出阿里云 RDS 数据库`
- `查询阿里云杭州的 RDS`
- `搜索 RDS 名称为 mysql-prod`

### 腾讯云 CVM
- `列出腾讯云 CVM`
- `查询腾讯云广州的 CVM`
- `找腾讯云 IP 10.0.0.1 的机器`

### 腾讯云 CDB
- `列出腾讯云 CDB`
- `查询腾讯云广州的 CDB`

### Jenkins (如果启用)
- `看一下 Jenkins 任务列表`
- `查询 deploy-prod 的构建历史`

### 帮助
- `帮助`
- `help`

## 🐛 故障排查

### Stream 模式 vs HTTP 模式

| 特性 | Stream 模式 | HTTP 模式 |
|------|------------|-----------|
| 需要公网地址 | ❌ 不需要 | ✅ 需要 |
| 本地开发 | ✅ 支持 | ❌ 不支持 |
| 防火墙穿透 | ✅ 不需要 | ❌ 需要配置 |
| 实时性 | ✅ 实时 | ⚠️ 依赖轮询 |
| 配置复杂度 | ✅ 简单 | ⚠️ 较复杂 |

### 切换到 HTTP 模式

如果 Stream 模式有问题,可以临时切换到 HTTP 模式:

```yaml
dingtalk:
  enabled: true
  mode: http  # 改为 http
  # ... 其他配置
  callback:
    token: ${DINGTALK_CALLBACK_TOKEN}
    aes_key: ${DINGTALK_AES_KEY}
    url: https://your-domain.com/api/v1/dingtalk/callback
```

## 📞 获取帮助

如果遇到问题:

1. 检查日志: `--log-level debug`
2. 查看配置: `./scripts/validate_config.sh`
3. 阅读官方文档: [钉钉 Stream 模式文档](https://open.dingtalk.com/document/orgapp/stream-mode-overview)
4. 提交 Issue: [GitHub Issues](https://github.com/eryajf/zenops/issues)

## ✅ 验收标准

成功配置 Stream 模式的标志:

- [x] 服务启动无错误
- [x] 日志显示 "DingTalk Stream client connecting..."
- [x] 在钉钉中@机器人能收到回复
- [x] 发送 "帮助" 能看到命令列表
- [x] 执行查询命令能返回云资源信息
- [x] (可选) AI 卡片流式更新正常工作
