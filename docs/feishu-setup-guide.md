# 飞书机器人接入指南

本文档介绍如何将 ZenOps 接入飞书机器人,实现通过飞书与云平台进行智能交互。

## 一、飞书应用创建与配置

### 1.1 创建飞书企业自建应用

1. 登录 [飞书开放平台](https://open.feishu.cn/)
2. 点击「创建企业自建应用」
3. 填写应用信息:
   - 应用名称: ZenOps 运维助手
   - 应用描述: 智能运维工具集成平台
   - 应用图标: 上传应用图标

### 1.2 获取应用凭证

创建完成后,在「凭证与基础信息」页面获取:
- **App ID**: `cli_xxxxxxxxxxxxxxxx`
- **App Secret**: `xxxxxxxxxxxxxxxxxxxxx`

> 这两个参数将用于配置 `config.yaml` 文件

### 1.3 添加应用能力

在「添加应用能力」中启用以下权限:

#### 机器人能力
- 启用「机器人」功能

#### 权限配置
在「权限管理」中添加以下权限:

**消息与群组权限**:
- `im:message` - 获取与发送单聊、群组消息
- `im:message:send_as_bot` - 以应用的身份发消息
- `im:chat` - 获取群组信息

**通讯录权限** (可选,用于获取用户信息):
- `contact:user.base:readonly` - 获取用户基本信息

### 1.4 配置事件订阅

1. 进入「事件订阅」配置页面
2. 选择「使用 Stream 模式」(推荐)
   - ZenOps 已经实现了 Stream 模式,无需配置回调 URL
   - Stream 模式更稳定,无需公网 IP
3. 订阅以下事件:
   - `im.message.receive_v1` - 接收消息事件

### 1.5 发布版本

1. 在「版本管理与发布」中创建版本
2. 提交审核(企业自建应用通常可以立即发布)
3. 发布到企业

### 1.6 将机器人添加到群组

1. 在飞书中创建一个测试群组
2. 点击群设置 → 群机器人 → 添加机器人
3. 选择你创建的应用

## 二、ZenOps 配置

### 2.1 配置文件

编辑 `config.yaml`,添加飞书配置:

```yaml
# 飞书配置
feishu:
  enabled: true  # 启用飞书机器人

  # 飞书应用凭证
  app_id: "cli_xxxxxxxxxxxxxxxx"      # 替换为你的 App ID
  app_secret: "xxxxxxxxxxxxxxxxxxxxx"  # 替换为你的 App Secret

  # LLM 对话配置
  enable_llm_conversation: true  # 启用 LLM 智能对话模式

# LLM 大模型配置(必须配置)
llm:
  enabled: true
  model: "DeepSeek-V3"  # 或其他模型
  api_key: "YOUR_LLM_API_KEY"
  base_url: ""  # 可选,自定义 API 端点
```

### 2.2 启动 ZenOps

```bash
# 编译项目
make build

# 启动服务
./zenops run --config config.yaml
```

### 2.3 验证启动

查看日志,确认飞书服务启动成功:

```
[INFO] Starting Feishu Stream server...
[INFO] Feishu Stream server started successfully
```

## 三、功能测试

### 3.1 基础功能测试

#### 1. 测试私聊
1. 在飞书中找到你的机器人
2. 发送消息: `帮助`
3. 机器人应该回复使用指南

#### 2. 测试群聊
1. 在添加了机器人的群组中
2. @机器人 并发送: `帮助`
3. 机器人应该回复使用指南

### 3.2 LLM 对话测试

#### 测试 1: 简单问答
```
你好,请介绍一下 ZenOps
```

**预期结果**: 机器人通过 LLM 回复 ZenOps 的介绍

#### 测试 2: 云平台查询(需要配置云平台凭证)
```
帮我查询阿里云 ECS 实例列表
```

**预期结果**:
- 机器人调用 MCP 工具查询阿里云 ECS
- 返回实例列表信息

#### 测试 3: Jenkins 查询(需要配置 Jenkins)
```
查看 Jenkins 最近的构建任务
```

**预期结果**:
- 机器人调用 MCP 工具查询 Jenkins
- 返回构建任务列表

### 3.3 消息格式测试

#### 测试长文本回复
发送复杂问题,测试 Markdown 格式化:
```
请详细说明如何在阿里云上部署一个高可用的 Web 应用
```

**预期结果**:
- 返回格式良好的 Markdown 消息
- 包含标题、列表、代码块等

## 四、常见问题排查

### 4.1 机器人无响应

**问题**: 在飞书中发送消息,机器人没有任何响应

**排查步骤**:
1. 检查 ZenOps 日志,确认是否收到消息
   ```bash
   tail -f zenops.log | grep "Received message"
   ```

2. 检查飞书应用是否正确配置了事件订阅
   - 确认订阅了 `im.message.receive_v1` 事件
   - 确认使用的是 Stream 模式

3. 检查应用权限
   - 确认已授予 `im:message` 权限
   - 确认应用已发布

### 4.2 LLM 调用失败

**问题**: 机器人收到消息,但回复错误信息

**排查步骤**:
1. 检查 LLM 配置
   ```yaml
   llm:
     enabled: true
     api_key: "YOUR_API_KEY"  # 确认 API Key 正确
   ```

2. 测试 LLM 连接
   ```bash
   # 查看 LLM 调用日志
   tail -f zenops.log | grep "LLM"
   ```

3. 检查网络连接
   - 确认服务器可以访问 LLM API 端点
   - 如有代理,确认代理配置正确

### 4.3 消息去重失败

**问题**: 机器人重复回复同一条消息

**排查步骤**:
1. 检查日志中的消息 ID
   ```bash
   tail -f zenops.log | grep "message_id"
   ```

2. 如果问题持续,重启 ZenOps 服务

### 4.4 获取用户信息失败

**问题**: 日志显示 "failed to get user info"

**原因**: 缺少通讯录权限

**解决方案**:
1. 在飞书开放平台添加权限:
   - `contact:user.base:readonly`
2. 重新发布应用版本
3. 在企业管理后台重新授权

## 五、高级配置

### 5.1 仅启用飞书服务

如果只想使用飞书机器人,不启用其他服务:

```yaml
server:
  http:
    enabled: false
  mcp:
    enabled: false

dingtalk:
  enabled: false

feishu:
  enabled: true
```

### 5.2 同时启用钉钉和飞书

ZenOps 支持同时启用多个 IM 平台:

```yaml
dingtalk:
  enabled: true
  # ...钉钉配置

feishu:
  enabled: true
  # ...飞书配置
```

### 5.3 禁用 LLM 对话

如果不需要 AI 对话功能:

```yaml
feishu:
  enabled: true
  enable_llm_conversation: false  # 禁用 LLM
```

> 禁用后,机器人会返回默认提示消息

### 5.4 自定义日志级别

在启动时指定日志级别:

```bash
export LOG_LEVEL=debug
./zenops run --config config.yaml
```

## 六、开发调试

### 6.1 本地调试

1. 启动 ZenOps
   ```bash
   go run main.go run --config config.yaml
   ```

2. 查看实时日志
   ```bash
   # 飞书相关日志
   tail -f zenops.log | grep "feishu"

   # 所有日志
   tail -f zenops.log
   ```

### 6.2 测试消息流程

查看完整的消息处理流程:

```bash
tail -f zenops.log | grep -E "(Received message|Processing message|LLM|Sent)"
```

### 6.3 代码结构

```
internal/
├── feishu/
│   ├── client.go       # 飞书 API 客户端
│   └── handler.go      # 消息处理器
├── server/
│   └── feishu_stream.go  # 飞书 Stream 服务
└── config/
    └── config.go       # 配置定义
```

## 七、生产部署建议

### 7.1 资源要求

- **CPU**: 1 核心(最低),2 核心(推荐)
- **内存**: 512MB(最低),1GB(推荐)
- **网络**: 稳定的互联网连接

### 7.2 安全建议

1. **保护应用凭证**
   ```bash
   # 使用环境变量
   export FEISHU_APP_ID="cli_xxx"
   export FEISHU_APP_SECRET="xxx"
   ```

2. **使用 systemd 管理服务**
   ```ini
   [Unit]
   Description=ZenOps Service
   After=network.target

   [Service]
   Type=simple
   User=zenops
   WorkingDirectory=/opt/zenops
   ExecStart=/opt/zenops/zenops run --config /etc/zenops/config.yaml
   Restart=always
   RestartSec=10

   [Install]
   WantedBy=multi-user.target
   ```

3. **启用日志轮转**
   ```bash
   # /etc/logrotate.d/zenops
   /var/log/zenops/*.log {
       daily
       rotate 7
       compress
       delaycompress
       missingok
       notifempty
   }
   ```

### 7.3 监控建议

1. 监控服务状态
   ```bash
   systemctl status zenops
   ```

2. 监控日志错误
   ```bash
   journalctl -u zenops -f | grep -i error
   ```

3. 设置告警(可选)
   - 使用 Prometheus + Grafana
   - 配置飞书群组告警通知

## 八、参考资料

- [飞书开放平台文档](https://open.feishu.cn/document/home/index)
- [飞书 Go SDK](https://github.com/larksuite/oapi-sdk-go)
- [ZenOps GitHub](https://github.com/eryajf/zenops)

## 九、技术支持

如有问题,请:
1. 查看本文档的常见问题部分
2. 查看项目 [Issues](https://github.com/eryajf/zenops/issues)
3. 提交新的 Issue

---

**更新日期**: 2025-12-16
**版本**: v1.0
