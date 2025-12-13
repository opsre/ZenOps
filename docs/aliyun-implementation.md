# 阿里云 Provider 实现总结

## 实现概览

本文档总结了阿里云 Provider 的实现细节和架构。

## 项目结构

```
internal/provider/aliyun/
├── init.go         # Provider 注册
├── provider.go     # Provider 接口实现
├── client.go       # 阿里云客户端封装
├── ecs.go          # ECS 服务查询
└── rds.go          # RDS 服务查询

cmd/
└── query_aliyun.go # CLI 命令实现

docs/
└── aliyun-provider.md  # 使用文档
```

## 核心组件

### 1. Client 封装 (client.go)

**功能**: 封装阿里云 SDK 客户端,支持多区域管理

**关键实现**:
- 为每个区域维护独立的 ECS 和 RDS 客户端
- 懒加载客户端,按需创建
- 统一的错误处理

```go
type Client struct {
    AccessKeyID     string
    AccessKeySecret string
    Region          string
    ecsClient       *ecs.Client
    rdsClient       *rds.Client
}
```

**注意事项**:
- ECS 使用 `darabonba-openapi/v2` 配置
- RDS 使用 `darabonba-openapi/v1` 配置 (SDK 版本差异)

### 2. ECS 查询 (ecs.go)

**功能**: 查询 ECS 实例列表和详情

**主要方法**:
- `ListECSInstances()`: 分页查询 ECS 实例
- `GetECSInstance()`: 获取单个实例详情
- `convertECSToInstance()`: 数据模型转换

**数据转换**:
```go
// 阿里云 ECS 实例 → 统一实例模型
ecs.Instance → model.Instance
```

**转换内容**:
- 基本信息: ID, 名称, 区域, 状态, 规格
- 网络信息: 私网 IP, 公网 IP, EIP
- 资源信息: CPU, 内存, OS 类型
- 时间信息: 创建时间, 过期时间
- 标签和元数据

### 3. RDS 查询 (rds.go)

**功能**: 查询 RDS 数据库实例列表和详情

**主要方法**:
- `ListRDSInstances()`: 分页查询 RDS 实例
- `GetRDSInstance()`: 获取单个数据库详情
- `convertRDSToDatabase()`: 数据模型转换

**端口处理**:
由于 RDS SDK 不直接返回端口,根据引擎类型设置默认端口:
- MySQL: 3306
- PostgreSQL: 5432
- SQL Server: 1433
- Redis: 6379

### 4. Provider 实现 (provider.go)

**功能**: 实现统一的 Provider 接口

**关键特性**:
- **多区域支持**: 自动为配置的每个区域创建客户端
- **智能查询**:
  - 指定区域时只查询该区域
  - 未指定区域时聚合所有区域结果
- **容错机制**: 单个区域失败不影响其他区域查询
- **健康检查**: 检查至少一个区域客户端可用

**初始化流程**:
```
1. 读取配置 (AccessKey, Regions)
2. 验证必需参数
3. 为每个区域创建 Client
4. 缓存到 clients map
```

### 5. CLI 命令 (query_aliyun.go)

**功能**: 提供命令行查询接口

**命令结构**:
```
zenops query aliyun
├── ecs
│   ├── list     # 列出 ECS 实例
│   └── get      # 获取 ECS 详情
└── rds
    ├── list     # 列出 RDS 实例
    └── get      # 获取 RDS 详情
```

**参数**:
- `--region`: 指定区域
- `--page-size`: 分页大小
- `--page-num`: 页码
- `--output`: 输出格式 (table/json)

**输出格式**:
- **table**: 表格形式,适合人类阅读
- **json**: JSON 格式,适合程序处理

### 6. 注册机制 (init.go)

**功能**: 自动注册 Provider 到全局注册表

```go
func init() {
    provider.Register("aliyun", NewProvider())
}
```

**触发方式**: 在 cmd/root.go 中 blank import

```go
import _ "github.com/eryajf/zenops/internal/provider/aliyun"
```

## 数据流

### 查询流程

```
用户命令
  ↓
CLI 命令解析
  ↓
获取 Provider
  ↓
初始化 Provider (配置)
  ↓
调用 Provider 方法
  ↓
获取/创建 区域 Client
  ↓
调用阿里云 SDK
  ↓
转换数据模型
  ↓
返回统一格式结果
  ↓
格式化输出 (table/json)
```

### 多区域聚合查询

```
Provider.ListInstances(opts)
  ↓
遍历所有区域客户端
  ├─→ 区域1: client1.ListECSInstances()
  ├─→ 区域2: client2.ListECSInstances()
  └─→ 区域3: client3.ListECSInstances()
  ↓
合并所有结果
  ↓
返回聚合列表
```

## 技术细节

### SDK 版本兼容性

```go
// ECS v4 使用新版 openapi v2
import openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
import ecs "github.com/alibabacloud-go/ecs-20140526/v4/client"

// RDS v2 使用旧版 openapi v1
import openapiv1 "github.com/alibabacloud-go/darabonba-openapi/client"
import rds "github.com/alibabacloud-go/rds-20140815/v2/client"
```

### 时间解析

阿里云 API 返回 ISO 8601 格式时间:

```go
time.Parse("2006-01-02T15:04:05Z", timeString)
```

### 标签处理

```go
// 阿里云标签结构
Tags: {
    Tag: [
        {TagKey: "env", TagValue: "prod"},
        {TagKey: "app", TagValue: "web"}
    ]
}

// 转换为 map
tags := map[string]string{
    "env": "prod",
    "app": "web"
}
```

### 空值处理

使用 `tea.StringValue()` 安全获取指针值:

```go
name := tea.StringValue(inst.InstanceName)  // 自动处理 nil
```

## 错误处理

### 分层错误处理

1. **SDK 层**: 捕获阿里云 API 错误
2. **Client 层**: 添加上下文信息
3. **Provider 层**: 容错和降级
4. **CLI 层**: 用户友好的错误消息

### 常见错误

```go
// 认证错误
InvalidAccessKeyId.NotFound

// 权限错误
Forbidden.RAM

// 资源不存在
InvalidInstanceId.NotFound

// 区域错误
InvalidRegionId.NotFound
```

## 性能优化

### 1. 客户端复用

每个区域的客户端创建后缓存,避免重复创建

### 2. 并发查询 (未来优化)

```go
// 可以使用 goroutine 并发查询多个区域
var wg sync.WaitGroup
results := make(chan []*model.Instance)

for region, client := range p.clients {
    wg.Add(1)
    go func(r string, c *Client) {
        defer wg.Done()
        instances, _ := c.ListECSInstances(ctx, opts)
        results <- instances
    }(region, client)
}
```

### 3. 分页查询

支持分页参数,避免一次加载大量数据

## 扩展指南

### 添加新的阿里云服务

1. 创建服务文件,如 `slb.go`
2. 实现查询方法
3. 添加数据模型转换
4. 在 Provider 中实现接口方法
5. 添加 CLI 命令
6. 更新文档

示例 - 添加 SLB 支持:

```go
// slb.go
func (c *Client) ListSLBInstances(ctx context.Context, opts) ([]*model.LoadBalancer, error) {
    // 实现逻辑
}

// provider.go
func (p *AliyunProvider) ListLoadBalancers(ctx context.Context, opts) ([]*model.LoadBalancer, error) {
    // 聚合多区域结果
}

// query_aliyun.go
var aliyunSLBCmd = &cobra.Command{
    Use: "slb",
    // ...
}
```

## 测试建议

### 单元测试

```go
// client_test.go
func TestNewClient(t *testing.T) {
    client, err := NewClient("test-id", "test-secret", "cn-hangzhou")
    assert.NoError(t, err)
    assert.NotNil(t, client)
}

// ecs_test.go
func TestConvertECSToInstance(t *testing.T) {
    // 测试数据转换
}
```

### 集成测试

```bash
# 使用测试凭证
export ALIYUN_ACCESS_KEY_ID="test-key"
export ALIYUN_ACCESS_KEY_SECRET="test-secret"

# 运行测试
go test ./internal/provider/aliyun/...
```

## 已知限制

1. **API 限流**: 阿里云 API 有速率限制,需要注意控制请求频率
2. **权限依赖**: 需要 RAM 用户有相应的只读权限
3. **区域覆盖**: 仅支持配置文件中指定的区域
4. **RDS 端口**: 使用默认端口,无法获取自定义端口配置

## 未来改进

- [ ] 支持更多阿里云服务 (SLB, VPC, OSS 等)
- [ ] 实现并发查询提升性能
- [ ] 添加缓存机制减少 API 调用
- [ ] 支持更复杂的过滤条件
- [ ] 实现增量查询和变更检测
- [ ] 添加 metrics 和 tracing
- [ ] 支持批量操作

## 参考资源

- [阿里云 ECS Go SDK](https://github.com/alibabacloud-go/ecs-20140526)
- [阿里云 RDS Go SDK](https://github.com/alibabacloud-go/rds-20140815)
- [阿里云 OpenAPI](https://next.api.aliyun.com/)
- [RAM 权限策略](https://ram.console.aliyun.com/policies)
