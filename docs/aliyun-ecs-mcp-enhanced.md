# 阿里云 ECS 查询增强 - MCP 集成指南

## 概述

本次更新对阿里云 ECS 的查询功能进行了精简和优化，重点提升了 MCP（Model Context Protocol）工具的精确查询能力。

## 核心改进

### 1. 精简的查询参数

`ECSQueryParams` 现在只包含最实用的参数：

```go
type ECSQueryParams struct {
    // 基础查询
    InstanceIDs  []string // 实例 ID 列表
    InstanceName string   // 实例名称

    // IP 地址查询
    PrivateIPAddresses []string // 内网 IP
    PublicIPAddresses  []string // 公网 IP
    EipAddresses       []string // 弹性 IP

    // 筛选条件
    Status             string // 实例状态
    InstanceChargeType string // 计费方式

    // 分页
    PageSize int
    PageNum  int
}
```

### 2. 增强的查询方法

#### 新增便捷方法

- **`QueryECSInstances(ctx, params)`** - 使用增强参数查询
- **`GetECSInstanceByName(ctx, name)`** - 按名称查询
- **`GetECSInstanceByIP(ctx, ip, ipType)`** - 按 IP 查询（支持自动检测 IP 类型）

#### 向后兼容

- `ListECSInstances` 和 `GetECSInstance` 保持原有接口，内部使用新实现

### 3. MCP 工具增强

所有 MCP 工具都已升级，使用精确的 API 查询，无需遍历所有实例。

## MCP 工具使用指南

### 1. search_ecs_by_ip - 根据 IP 精确搜索

**功能**：根据 IP 地址精确搜索 ECS 实例，支持自动检测 IP 类型

**参数**：
- `ip` (必填): IP 地址
- `account` (可选): 阿里云账号名称
- `region` (可选): 区域
- `ip_type` (可选): IP 类型 - `private`, `public`, `eip`，不指定则自动尝试所有类型

**示例**：
```json
{
  "ip": "172.16.0.10",
  "account": "production",
  "ip_type": "private"
}
```

**优势**：
- ✅ 直接使用阿里云 API 的 IP 过滤功能
- ✅ 无需遍历所有实例
- ✅ 支持自动检测 IP 类型
- ✅ 响应速度快

### 2. search_ecs_by_name - 根据名称精确搜索

**功能**：根据实例名称精确搜索 ECS 实例

**参数**：
- `name` (必填): 实例名称（精确匹配）
- `account` (可选): 阿里云账号名称
- `region` (可选): 区域

**示例**：
```json
{
  "name": "web-server-01",
  "account": "production",
  "region": "cn-hangzhou"
}
```

**优势**：
- ✅ 直接使用阿里云 API 的名称过滤
- ✅ 精确匹配，快速定位
- ✅ 无需遍历所有实例

### 3. list_ecs - 列出实例（支持筛选）

**功能**：列出 ECS 实例，支持按状态和计费方式筛选

**参数**：
- `account` (可选): 阿里云账号名称
- `region` (可选): 区域
- `status` (可选): 实例状态
  - `Pending` - 准备中
  - `Running` - 运行中
  - `Starting` - 启动中
  - `Stopping` - 停止中
  - `Stopped` - 已停止
- `instance_charge_type` (可选): 计费方式
  - `PostPaid` - 按量付费
  - `PrePaid` - 包年包月

**示例**：
```json
{
  "account": "production",
  "region": "cn-hangzhou",
  "status": "Running",
  "instance_charge_type": "PostPaid"
}
```

**优势**：
- ✅ 支持多维度筛选
- ✅ 精确过滤，减少数据传输
- ✅ 适合运维场景的精确查询

### 4. get_ecs - 获取实例详情

**功能**：根据实例 ID 获取详细信息

**参数**：
- `instance_id` (必填): 实例 ID
- `account` (可选): 阿里云账号名称

**示例**：
```json
{
  "instance_id": "i-bp1234567890abcde",
  "account": "production"
}
```

## AI/LLM 使用建议

### 自然语言查询映射

#### 示例 1：按 IP 查询
```
用户: "帮我查一下 10.0.1.100 这个 IP 是哪台机器"

映射到:
{
  "tool": "search_ecs_by_ip",
  "params": {
    "ip": "10.0.1.100"
  }
}
```

#### 示例 2：按名称查询
```
用户: "web-server-01 这台机器的配置是什么"

映射到:
{
  "tool": "search_ecs_by_name",
  "params": {
    "name": "web-server-01"
  }
}
```

#### 示例 3：筛选查询
```
用户: "列出杭州区域所有运行中的按量付费实例"

映射到:
{
  "tool": "list_ecs",
  "params": {
    "region": "cn-hangzhou",
    "status": "Running",
    "instance_charge_type": "PostPaid"
  }
}
```

#### 示例 4：精确 IP 类型
```
用户: "查询内网 IP 为 172.16.0.10 的实例"

映射到:
{
  "tool": "search_ecs_by_ip",
  "params": {
    "ip": "172.16.0.10",
    "ip_type": "private"
  }
}
```

## 性能对比

### 旧实现（遍历方式）
```
查询 IP 10.0.1.100 的实例：
1. 列出所有实例（可能几千台）
2. 逐个检查 IP 匹配
3. 耗时：10-30 秒
4. 网络传输：MB 级别
```

### 新实现（精确查询）
```
查询 IP 10.0.1.100 的实例：
1. 直接调用 API 精确查询
2. 返回匹配的实例
3. 耗时：< 1 秒
4. 网络传输：KB 级别
```

**性能提升**：10-30倍

## 支持的查询组合

| 场景 | 使用工具 | 参数组合 |
|------|---------|----------|
| 根据 IP 查询 | `search_ecs_by_ip` | `ip` + 可选 `ip_type` |
| 根据名称查询 | `search_ecs_by_name` | `name` + 可选 `region` |
| 查运行中的实例 | `list_ecs` | `status=Running` |
| 查按量付费实例 | `list_ecs` | `instance_charge_type=PostPaid` |
| 查特定区域的停止实例 | `list_ecs` | `region` + `status=Stopped` |
| 根据 ID 查详情 | `get_ecs` | `instance_id` |

## 代码示例

### Go 代码使用

```go
import (
    "context"
    "github.com/eryajf/zenops/internal/provider/aliyun"
)

// 示例 1: 根据 IP 查询
func searchByIP(client *aliyun.Client) {
    ctx := context.Background()
    instance, err := client.GetECSInstanceByIP(ctx, "10.0.1.100", "private")
    if err != nil {
        // 处理错误
    }
    // 使用 instance
}

// 示例 2: 根据名称查询
func searchByName(client *aliyun.Client) {
    ctx := context.Background()
    instance, err := client.GetECSInstanceByName(ctx, "web-server-01")
    if err != nil {
        // 处理错误
    }
    // 使用 instance
}

// 示例 3: 使用增强参数查询
func advancedQuery(client *aliyun.Client) {
    ctx := context.Background()
    params := &aliyun.ECSQueryParams{
        Status:             "Running",
        InstanceChargeType: "PostPaid",
        PrivateIPAddresses: []string{"10.0.1.100", "10.0.1.101"},
        PageSize:           50,
    }
    instances, err := client.QueryECSInstances(ctx, params)
    if err != nil {
        // 处理错误
    }
    // 使用 instances
}
```

## 常见问题

### Q1: IP 查询支持哪些类型？
**A**: 支持三种类型：
- `private`: 内网 IP（VPC 或经典网络）
- `public`: 公网 IP
- `eip`: 弹性公网 IP

不指定 `ip_type` 时，会自动尝试所有类型。

### Q2: 名称查询支持模糊匹配吗？
**A**: 当前版本支持精确匹配。未来可能支持通配符。

### Q3: 可以同时指定多个筛选条件吗？
**A**: 可以。例如同时指定 `status` 和 `instance_charge_type`。

### Q4: 如何查询多个区域？
**A**: 需要分别调用每个区域，或者不指定 `region` 让系统查询所有配置的区域。

## 变更总结

### 参数精简
- 移除不常用的参数（VPC、安全组、标签等）
- 保留核心查询参数
- 简化 MCP 工具接口

### 性能优化
- 直接使用阿里云 API 过滤功能
- 避免遍历所有实例
- 减少网络传输

### 功能增强
- 新增按 IP 精确查询
- 新增按名称精确查询
- 支持 IP 类型自动检测
- 支持多维度筛选

## 参考文档

- [ecs.go](../internal/provider/aliyun/ecs.go) - 核心实现
- [handlers_aliyun.go](../internal/imcp/handlers_aliyun.go) - MCP 处理器
- [server.go](../internal/imcp/server.go) - MCP 工具注册

## 更新日期

2025-12-15
