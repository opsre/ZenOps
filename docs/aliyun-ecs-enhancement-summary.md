# 阿里云 ECS 查询功能增强总结

## 改进概述

本次改进大幅增强了阿里云 ECS 实例的查询能力，使其更适合 AI 和 MCP（Model Context Protocol）场景的精确查询需求。

## 主要改进

### 1. 新增 `ECSQueryParams` 结构体

新增了一个功能丰富的查询参数结构体，支持以下参数类型：

#### 基础查询参数
- `InstanceIDs` ([]string): 实例 ID 列表，最多支持 100 个
- `InstanceName` (string): 实例名称，支持通配符 (*)

#### 网络相关参数
- `VpcID`: VPC ID
- `VSwitchID`: 交换机 ID
- `ZoneID`: 可用区 ID
- `InstanceNetworkType`: 网络类型（classic, vpc）
- `SecurityGroupID`: 安全组 ID
- `PrivateIPAddresses`: VPC 内网 IP 地址列表
- `PublicIPAddresses`: 公网 IP 地址列表
- `EipAddresses`: 弹性公网 IP 地址列表
- `InnerIPAddresses`: 经典网络内网 IP 地址列表

#### 实例状态与类型
- `Status`: 实例状态
- `ImageID`: 镜像 ID
- `InstanceType`: 实例规格
- `InstanceTypeFamily`: 实例规格族

#### 资源与配置
- `ResourceGroupID`: 资源组 ID
- `KeyPairName`: SSH 密钥对名称
- `Tags`: 标签键值对
- `HpcClusterID`: HPC 集群 ID

#### 计费相关
- `InstanceChargeType`: 计费方式（PostPaid, PrePaid）
- `InternetChargeType`: 网络计费方式
- `IoOptimized`: 是否为 I/O 优化实例
- `LockReason`: 锁定原因

#### 分页与高级选项
- `PageSize`: 每页数量
- `PageNum`: 页码
- `DryRun`: 是否只预检

### 2. 新增核心方法

#### `QueryECSInstances`
```go
func (c *Client) QueryECSInstances(ctx context.Context, params *ECSQueryParams) ([]*model.Instance, error)
```
- 使用增强的参数结构进行查询
- 支持所有阿里云 DescribeInstances API 的查询参数
- 适合复杂的组合查询场景

#### `GetECSInstanceByQuery`
```go
func (c *Client) GetECSInstanceByQuery(ctx context.Context, params *ECSQueryParams) (*model.Instance, error)
```
- 根据自定义参数查询单个实例
- 内部使用 `QueryECSInstances`，自动限制返回一个结果

#### `GetECSInstanceByName`
```go
func (c *Client) GetECSInstanceByName(ctx context.Context, instanceName string) (*model.Instance, error)
```
- 根据实例名称直接查询
- 简化常见的按名称查询场景

#### `GetECSInstanceByIP`
```go
func (c *Client) GetECSInstanceByIP(ctx context.Context, ipAddress string, ipType string) (*model.Instance, error)
```
- 根据 IP 地址查询实例
- 支持指定 IP 类型（private, public, eip）
- 如果不指定类型，会自动尝试所有类型

### 3. 辅助函数

#### `buildDescribeInstancesRequest`
- 封装了参数到阿里云 API 请求的转换逻辑
- 处理 JSON 数组格式、标签等复杂参数
- 统一的参数验证和默认值设置

#### `formatIPArray`
- 格式化 IP 地址数组为阿里云 API 要求的 JSON 字符串格式

### 4. 向后兼容性

原有的方法保持不变，确保现有代码继续工作：

- `ListECSInstances`: 内部重构为使用新的 `QueryECSInstances`
- `GetECSInstance`: 内部重构为使用新的 `GetECSInstanceByQuery`
- provider 层的接口保持不变

## 使用场景

### AI/MCP 场景示例

#### 自然语言查询映射
```
用户: "查询生产环境在杭州可用区 I 的所有运行中的实例"
↓
params := &aliyun.ECSQueryParams{
    ZoneID: "cn-hangzhou-i",
    Status: "Running",
    Tags: map[string]string{
        "Environment": "production",
    },
}
```

#### 智能 IP 查询
```
用户: "找出使用 10.0.1.100 这个 IP 的机器"
↓
instance, err := client.GetECSInstanceByIP(ctx, "10.0.1.100", "")
// 自动检测 IP 类型
```

#### 复杂组合查询
```
用户: "列出 web-app VPC 下所有的 g7 规格族按量付费机器"
↓
params := &aliyun.ECSQueryParams{
    VpcID:              "vpc-web-app",
    InstanceTypeFamily: "ecs.g7",
    InstanceChargeType: "PostPaid",
}
```

## 技术细节

### API 对齐
所有参数都严格对齐阿里云官方 API 文档：
- [DescribeInstances API 文档](https://help.aliyun.com/zh/ecs/developer-reference/api-ecs-2014-05-26-describeinstances)

### JSON 数组处理
对于需要 JSON 数组格式的参数（如 InstanceIds、IP 地址列表），提供了统一的格式化函数：
```go
["id1", "id2", "id3"]
```

### 标签支持
标签参数转换为阿里云 SDK 要求的结构：
```go
[]*ecs.DescribeInstancesRequestTag{
    {Key: "Environment", Value: "production"},
    {Key: "Project", Value: "web-app"},
}
```

## 测试

### 编译验证
- ✅ 单模块编译通过: `go build ./internal/provider/aliyun/...`
- ✅ 全项目编译通过: `go build ./...`

### 功能验证建议
1. 基础查询：实例 ID、实例名称
2. 网络查询：VPC、交换机、IP 地址
3. 状态查询：运行中、已停止等状态
4. 标签查询：单标签、多标签组合
5. 分页查询：大数据量场景

## 文档

新增以下文档：
1. [aliyun-ecs-query-examples.md](aliyun-ecs-query-examples.md) - 详细的使用示例
2. `/tmp/test_ecs_query.go` - 可运行的测试代码示例

## 优势

### 对 AI 的优势
1. **参数化更清晰**：结构体参数比 map[string]string 更易于理解和生成
2. **类型安全**：编译时检查，减少运行时错误
3. **功能完整**：支持所有官方 API 参数，无需后续扩展

### 对 MCP 的优势
1. **精确查询**：支持多维度组合查询，可精确定位资源
2. **灵活性高**：可根据各种条件（ID、名称、IP、标签等）查询
3. **易于集成**：清晰的接口定义，便于 MCP server 集成

### 对用户的优势
1. **向后兼容**：现有代码无需修改
2. **渐进增强**：可选择使用简单方法或增强方法
3. **文档完善**：提供丰富的示例和说明

## 后续建议

### 可能的扩展方向
1. 添加批量操作支持（启动、停止、重启等）
2. 添加实例监控数据查询
3. 添加实例变更历史查询
4. 支持自动分页（迭代器模式）
5. 添加查询结果缓存机制

### MCP Server 集成
建议在 MCP server 中提供以下工具：
- `query_ecs_instances`: 使用 ECSQueryParams 进行查询
- `get_instance_by_id`: 快速查询单个实例
- `get_instance_by_name`: 按名称查询
- `get_instance_by_ip`: 按 IP 查询
- `search_instances`: 自然语言搜索（映射到参数）

## 相关文件

### 修改的文件
- [internal/provider/aliyun/ecs.go](../../internal/provider/aliyun/ecs.go)

### 新增的文件
- [docs/aliyun-ecs-query-examples.md](aliyun-ecs-query-examples.md)
- [docs/aliyun-ecs-enhancement-summary.md](aliyun-ecs-enhancement-summary.md)
- `/tmp/test_ecs_query.go` (测试示例)

### 未修改的文件
- [internal/provider/aliyun/provider.go](../../internal/provider/aliyun/provider.go) (继续使用兼容接口)

## 版本信息

- 修改日期: 2025-12-15
- 相关 SDK: github.com/alibabacloud-go/ecs-20140526/v4
- API 版本: 2014-05-26
