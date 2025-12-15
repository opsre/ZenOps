# 阿里云 ECS 增强查询功能使用示例

本文档展示了如何使用 zenops 中增强的阿里云 ECS 查询功能，这些功能特别适合 AI 和 MCP 进行精确查询。

## 核心改进

1. **新增 `ECSQueryParams` 结构体**：支持丰富的查询参数
2. **新增 `QueryECSInstances` 方法**：使用增强参数进行查询
3. **保留 `ListECSInstances` 方法**：向后兼容，内部使用新方法
4. **新增便捷查询方法**：
   - `GetECSInstanceByQuery`：根据自定义参数查询单个实例
   - `GetECSInstanceByName`：根据实例名称查询
   - `GetECSInstanceByIP`：根据 IP 地址查询

## 使用示例

### 1. 基础查询 - 使用实例 ID

```go
// 方式 1: 使用原有方法（向后兼容）
instance, err := client.GetECSInstance(ctx, "i-bp1234567890abcde")

// 方式 2: 使用新方法
params := &aliyun.ECSQueryParams{
    InstanceIDs: []string{"i-bp1234567890abcde"},
}
instances, err := client.QueryECSInstances(ctx, params)
```

### 2. 根据实例名称查询

```go
// 精确匹配
instance, err := client.GetECSInstanceByName(ctx, "web-server-01")

// 或使用参数
params := &aliyun.ECSQueryParams{
    InstanceName: "web-server-01",
}
instances, err := client.QueryECSInstances(ctx, params)

// 模糊匹配（支持通配符 *）
params := &aliyun.ECSQueryParams{
    InstanceName: "web-*",
}
instances, err := client.QueryECSInstances(ctx, params)
```

### 3. 根据 IP 地址查询

```go
// 自动检测 IP 类型
instance, err := client.GetECSInstanceByIP(ctx, "172.16.0.10", "")

// 指定 IP 类型
instance, err := client.GetECSInstanceByIP(ctx, "172.16.0.10", "private")
instance, err := client.GetECSInstanceByIP(ctx, "47.96.123.45", "public")
instance, err := client.GetECSInstanceByIP(ctx, "47.96.200.10", "eip")

// 或使用参数查询多个 IP
params := &aliyun.ECSQueryParams{
    PrivateIPAddresses: []string{"172.16.0.10", "172.16.0.11"},
}
instances, err := client.QueryECSInstances(ctx, params)
```

### 4. 根据网络拓扑查询

```go
// 查询特定 VPC 下的实例
params := &aliyun.ECSQueryParams{
    VpcID: "vpc-bp1234567890abcde",
}
instances, err := client.QueryECSInstances(ctx, params)

// 查询特定交换机下的实例
params := &aliyun.ECSQueryParams{
    VpcID:     "vpc-bp1234567890abcde",
    VSwitchID: "vsw-bp1234567890abcde",
}
instances, err := client.QueryECSInstances(ctx, params)

// 查询特定可用区的实例
params := &aliyun.ECSQueryParams{
    ZoneID: "cn-hangzhou-i",
}
instances, err := client.QueryECSInstances(ctx, params)

// 查询特定安全组的实例
params := &aliyun.ECSQueryParams{
    SecurityGroupID: "sg-bp1234567890abcde",
}
instances, err := client.QueryECSInstances(ctx, params)
```

### 5. 根据实例状态和类型查询

```go
// 查询运行中的实例
params := &aliyun.ECSQueryParams{
    Status: "Running",
}
instances, err := client.QueryECSInstances(ctx, params)

// 查询特定规格的实例
params := &aliyun.ECSQueryParams{
    InstanceType: "ecs.g7.large",
}
instances, err := client.QueryECSInstances(ctx, params)

// 查询特定规格族的实例
params := &aliyun.ECSQueryParams{
    InstanceTypeFamily: "ecs.g7",
}
instances, err := client.QueryECSInstances(ctx, params)

// 查询特定镜像的实例
params := &aliyun.ECSQueryParams{
    ImageID: "centos_7_9_x64_20G_alibase_20210927.vhd",
}
instances, err := client.QueryECSInstances(ctx, params)
```

### 6. 根据标签查询

```go
// 查询包含特定标签的实例
params := &aliyun.ECSQueryParams{
    Tags: map[string]string{
        "Environment": "production",
        "Project":     "web-app",
    },
}
instances, err := client.QueryECSInstances(ctx, params)
```

### 7. 根据计费方式查询

```go
// 查询按量付费实例
params := &aliyun.ECSQueryParams{
    InstanceChargeType: "PostPaid",
}
instances, err := client.QueryECSInstances(ctx, params)

// 查询包年包月实例
params := &aliyun.ECSQueryParams{
    InstanceChargeType: "PrePaid",
}
instances, err := client.QueryECSInstances(ctx, params)
```

### 8. 组合查询

```go
// 复杂组合查询示例
params := &aliyun.ECSQueryParams{
    VpcID:              "vpc-bp1234567890abcde",
    Status:             "Running",
    InstanceTypeFamily: "ecs.g7",
    Tags: map[string]string{
        "Environment": "production",
    },
    PageSize: 50,
    PageNum:  1,
}
instances, err := client.QueryECSInstances(ctx, params)
```

### 9. 批量查询多个实例

```go
// 查询多个实例 ID
params := &aliyun.ECSQueryParams{
    InstanceIDs: []string{
        "i-bp1111111111111111",
        "i-bp2222222222222222",
        "i-bp3333333333333333",
    },
}
instances, err := client.QueryECSInstances(ctx, params)
```

### 10. 分页查询

```go
// 第一页
params := &aliyun.ECSQueryParams{
    PageSize: 100,
    PageNum:  1,
}
instances, err := client.QueryECSInstances(ctx, params)

// 第二页
params.PageNum = 2
instances, err = client.QueryECSInstances(ctx, params)
```

## AI/MCP 集成建议

对于 AI 和 MCP 场景，推荐使用以下查询方式：

### 自然语言到查询参数的映射

```go
// "查询生产环境在杭州可用区 I 的所有运行中的实例"
params := &aliyun.ECSQueryParams{
    ZoneID: "cn-hangzhou-i",
    Status: "Running",
    Tags: map[string]string{
        "Environment": "production",
    },
}

// "找出使用 10.0.1.100 这个 IP 的机器"
instance, err := client.GetECSInstanceByIP(ctx, "10.0.1.100", "")

// "列出 web-app VPC 下所有的 g7 规格族机器"
params := &aliyun.ECSQueryParams{
    VpcID:              "vpc-web-app",
    InstanceTypeFamily: "ecs.g7",
}
```

## 支持的所有查询参数

### 基础参数
- `InstanceIDs`: 实例 ID 列表（最多 100 个）
- `InstanceName`: 实例名称（支持通配符 *）

### 网络参数
- `VpcID`: VPC ID
- `VSwitchID`: 交换机 ID
- `ZoneID`: 可用区 ID
- `InstanceNetworkType`: 网络类型（classic, vpc）
- `SecurityGroupID`: 安全组 ID
- `PrivateIPAddresses`: 内网 IP 列表
- `PublicIPAddresses`: 公网 IP 列表
- `EipAddresses`: 弹性公网 IP 列表
- `InnerIPAddresses`: 经典网络内网 IP 列表

### 实例状态与类型
- `Status`: 实例状态（Pending, Running, Starting, Stopping, Stopped）
- `ImageID`: 镜像 ID
- `InstanceType`: 实例规格
- `InstanceTypeFamily`: 实例规格族

### 资源与配置
- `ResourceGroupID`: 资源组 ID
- `KeyPairName`: SSH 密钥对名称
- `Tags`: 标签键值对
- `HpcClusterID`: HPC 集群 ID

### 计费参数
- `InstanceChargeType`: 计费方式（PostPaid, PrePaid）
- `InternetChargeType`: 网络计费方式（PayByBandwidth, PayByTraffic）
- `IoOptimized`: 是否为 I/O 优化实例
- `LockReason`: 锁定原因

### 分页参数
- `PageSize`: 每页数量（最大 100，默认 10）
- `PageNum`: 页码（默认 1）

### 高级选项
- `DryRun`: 是否只预检，不实际查询

## 参考文档

- [阿里云 ECS DescribeInstances API 文档](https://help.aliyun.com/zh/ecs/developer-reference/api-ecs-2014-05-26-describeinstances)
- [阿里云 Go SDK 文档](https://next.api.aliyun.com/)
