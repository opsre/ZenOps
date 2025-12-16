# 飞书机器人测试清单

本文档提供飞书机器人功能的快速测试清单,用于验证接入是否成功。

## 前置条件

- [ ] 已在飞书开放平台创建企业自建应用
- [ ] 已获取 App ID 和 App Secret
- [ ] 已配置事件订阅(Stream 模式)
- [ ] 已订阅 `im.message.receive_v1` 事件
- [ ] 已添加必要的权限(`im:message`, `im:message:send_as_bot`)
- [ ] 已发布应用到企业
- [ ] 已在群组中添加机器人

## 配置验证

### 1. 配置文件检查

检查 `config.yaml` 配置:

```yaml
feishu:
  enabled: true
  app_id: "cli_xxxxxxxxxxxxxxxx"      # 确认已填写
  app_secret: "xxxxxxxxxxxxxxxxxxxxx"  # 确认已填写
  enable_llm_conversation: true

llm:
  enabled: true
  model: "DeepSeek-V3"
  api_key: "YOUR_LLM_API_KEY"  # 确认已填写
```

- [ ] `feishu.enabled` 为 `true`
- [ ] `feishu.app_id` 已正确配置
- [ ] `feishu.app_secret` 已正确配置
- [ ] `llm.enabled` 为 `true`
- [ ] `llm.api_key` 已正确配置

### 2. 启动服务

```bash
# 编译
make build

# 启动
./zenops run --config config.yaml
```

- [ ] 编译成功,无错误
- [ ] 启动成功,看到 "Starting Feishu Stream server..."
- [ ] 连接成功,看到 "Feishu Stream server started successfully"

查看日志:
```bash
tail -f zenops.log | grep -i feishu
```

预期输出:
```
[INFO] Starting Feishu Stream server...
[INFO] Got Feishu access token, expire_in 7200
[INFO] Feishu Stream server started successfully
```

## 基础功能测试

### 测试 1: 帮助命令(私聊)

**操作**:
1. 在飞书中找到你的机器人
2. 发送消息: `帮助`

**预期结果**:
- [ ] 机器人在 3 秒内响应
- [ ] 返回 Markdown 格式的帮助文档
- [ ] 帮助文档包含功能说明和使用示例

**日志验证**:
```bash
tail -f zenops.log | grep "Received message"
```

应该看到:
```
[INFO] Received message from Feishu: user ou_xxx, message 帮助
[INFO] Processing message: id om_xxx, type text, chat_type p2p
```

### 测试 2: 帮助命令(群聊)

**操作**:
1. 在群组中 @机器人
2. 发送消息: `@ZenOps 帮助`

**预期结果**:
- [ ] 机器人响应
- [ ] 返回帮助文档(与私聊相同)

**日志验证**:
```
[INFO] Processing message: id om_xxx, type text, chat_type group
```

### 测试 3: LLM 简单问答

**操作**:
发送消息: `你好,请介绍一下自己`

**预期结果**:
- [ ] 机器人先回复 "正在思考中,请稍候..."
- [ ] 然后返回 LLM 生成的回复
- [ ] 回复格式为 Markdown
- [ ] 包含问题和回答两部分
- [ ] 底部包含时间戳

**示例回复**:
```
**问题:** 你好,请介绍一下自己

**回答:**

你好!我是 ZenOps 运维助手...

---
⏰ 2025-12-16 10:30:00
```

## LLM 功能测试

### 测试 4: 云平台查询(需要配置云平台)

**前置条件**:
- [ ] 已配置阿里云或腾讯云凭证

**操作**:
发送消息: `帮我查询阿里云 ECS 实例列表`

**预期结果**:
- [ ] 机器人调用 MCP 工具
- [ ] 返回 ECS 实例列表
- [ ] 数据格式清晰易读

**日志验证**:
```bash
tail -f zenops.log | grep -E "(LLM|MCP|tool)"
```

### 测试 5: Jenkins 查询(需要配置 Jenkins)

**前置条件**:
- [ ] 已配置 Jenkins 连接信息

**操作**:
发送消息: `查看 Jenkins 最近的构建任务`

**预期结果**:
- [ ] 机器人调用 MCP 工具
- [ ] 返回构建任务列表
- [ ] 包含任务名称、状态、时间等信息

### 测试 6: 复杂对话

**操作**:
发送消息: `请详细说明阿里云 ECS 的计费方式`

**预期结果**:
- [ ] 返回详细的回答
- [ ] 格式良好,包含标题、列表等
- [ ] 响应时间在合理范围内(< 30秒)

## 边界情况测试

### 测试 7: 空消息

**操作**:
发送空格或空消息

**预期结果**:
- [ ] 机器人不响应(或返回友好提示)

### 测试 8: 非文本消息

**操作**:
发送图片、文件等非文本消息

**预期结果**:
- [ ] 机器人不响应
- [ ] 日志显示 "Ignoring non-text message"

### 测试 9: 重复消息

**操作**:
快速连续发送同一条消息 3 次

**预期结果**:
- [ ] 机器人只响应一次
- [ ] 日志显示消息去重

### 测试 10: 长文本

**操作**:
发送一条很长的问题(> 500 字)

**预期结果**:
- [ ] 机器人正常处理
- [ ] 返回完整回答
- [ ] 无消息截断

## 性能测试

### 测试 11: 并发消息

**操作**:
在不同的群组或私聊中同时发送消息

**预期结果**:
- [ ] 所有消息都得到响应
- [ ] 响应时间正常
- [ ] 无消息丢失

### 测试 12: 持续运行

**操作**:
让服务运行 1 小时,期间发送多条消息

**预期结果**:
- [ ] 服务稳定运行
- [ ] 所有消息正常响应
- [ ] 无内存泄漏(观察进程内存)

## 错误处理测试

### 测试 13: LLM 服务异常

**操作**:
1. 停止 LLM 服务或配置错误的 API Key
2. 发送消息

**预期结果**:
- [ ] 机器人返回错误提示
- [ ] 错误信息友好,不暴露敏感信息
- [ ] 日志记录详细错误

### 测试 14: 网络中断

**操作**:
1. 启动服务后短暂断网
2. 恢复网络
3. 发送消息

**预期结果**:
- [ ] WebSocket 自动重连
- [ ] 消息正常处理
- [ ] 日志显示重连过程

## 安全测试

### 测试 15: 无效凭证

**操作**:
使用错误的 App ID 或 App Secret 启动服务

**预期结果**:
- [ ] 服务启动失败或无法连接
- [ ] 日志显示认证错误
- [ ] 不泄露敏感信息

### 测试 16: 权限不足

**操作**:
移除应用的某些权限,然后发送消息

**预期结果**:
- [ ] 服务返回权限错误
- [ ] 日志记录错误详情

## 日志验证

### 关键日志检查

运行服务时,检查是否有以下日志:

```bash
# 启动日志
grep "Starting Feishu Stream server" zenops.log
grep "Feishu Stream server started successfully" zenops.log

# Token 获取
grep "Got Feishu access token" zenops.log

# 消息接收
grep "Received message from Feishu" zenops.log
grep "Processing message" zenops.log

# LLM 调用
grep "LLM client initialized" zenops.log
grep "Failed to call LLM" zenops.log  # 应该为空

# 错误日志
grep -i "error" zenops.log  # 检查是否有异常错误
```

- [ ] 所有关键日志都存在
- [ ] 无异常错误日志
- [ ] Token 定期刷新(每 2 小时)

## 清理测试

### 测试 17: 优雅停止

**操作**:
发送 SIGTERM 信号停止服务
```bash
kill -TERM $(pidof zenops)
```

**预期结果**:
- [ ] 服务优雅关闭
- [ ] 日志显示 "Stopping Feishu Stream server..."
- [ ] 无 goroutine 泄漏

## 测试总结

### 必须通过的测试

以下测试必须全部通过才能认为接入成功:

- [ ] 测试 1: 帮助命令(私聊)
- [ ] 测试 2: 帮助命令(群聊)
- [ ] 测试 3: LLM 简单问答
- [ ] 测试 8: 非文本消息
- [ ] 测试 9: 重复消息

### 可选测试

以下测试依赖特定配置,可选择性验证:

- [ ] 测试 4: 云平台查询
- [ ] 测试 5: Jenkins 查询

### 性能基准

- 消息响应延迟: < 3 秒
- LLM 查询延迟: < 30 秒
- Token 刷新: 自动,无需人工干预
- 内存占用: < 100MB(空闲)

## 问题记录

在测试过程中遇到的问题:

| 测试项 | 问题描述 | 解决方案 | 状态 |
|--------|---------|---------|------|
|        |         |         |      |
|        |         |         |      |

## 测试完成确认

- [ ] 所有必须测试项通过
- [ ] 日志无异常错误
- [ ] 性能符合预期
- [ ] 可投入生产使用

---

**测试人**: ___________
**测试日期**: ___________
**ZenOps 版本**: ___________
**飞书应用**: ___________
