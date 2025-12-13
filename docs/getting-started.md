# ZenOps 快速入门指南

本指南将帮助你快速上手 ZenOps 项目的开发。

## 前置要求

- Go 1.21 或更高版本
- Make (可选,用于构建管理)
- Git

## 项目设置

### 1. 克隆项目

```bash
git clone https://github.com/eryajf/zenops.git
cd zenops
```

### 2. 安装依赖

```bash
make deps
```

或使用 Go 命令:

```bash
go mod download
go mod tidy
```

### 3. 配置应用

复制配置文件模板:

```bash
cp configs/config.example.yaml configs/config.yaml
```

编辑 `configs/config.yaml` 并设置必要的环境变量,或直接导出环境变量:

```bash
# 阿里云配置
export ALIYUN_ACCESS_KEY_ID="your-key-id"
export ALIYUN_ACCESS_KEY_SECRET="your-key-secret"

# 腾讯云配置
export TENCENT_SECRET_ID="your-secret-id"
export TENCENT_SECRET_KEY="your-secret-key"

# Jenkins 配置
export JENKINS_USER="your-username"
export JENKINS_TOKEN="your-token"
```

## 构建项目

### 使用 Make

```bash
# 编译项目
make build

# 运行项目
make run

# 开发模式 (实时编译运行)
make dev
```

### 使用 Go 命令

```bash
# 编译
go build -o bin/zenops .

# 运行
./bin/zenops
```

## 基本使用

### 查看版本

```bash
./bin/zenops version
```

### 查看帮助

```bash
./bin/zenops --help
./bin/zenops serve --help
```

### 启动服务

```bash
# 启动 HTTP 和 MCP 服务 (根据配置文件)
./bin/zenops serve --config configs/config.yaml

# 仅启动 HTTP 服务
./bin/zenops serve --http-only

# 仅启动 MCP 服务
./bin/zenops serve --mcp-only
```

## 开发工作流

### 1. 创建新的 Provider

如果你想添加一个新的云服务提供商,按照以下步骤:

1. 在 `internal/provider/` 下创建新目录,如 `aws/`
2. 实现 `Provider` 接口 (参考 `internal/provider/interface.go`)
3. 在 `internal/provider/` 中注册新的 Provider
4. 在配置文件中添加相应配置
5. 编写单元测试

示例代码结构:

```go
// internal/provider/aws/aws.go
package aws

import (
    "context"
    "github.com/eryajf/zenops/internal/model"
    "github.com/eryajf/zenops/internal/provider"
)

type AWSProvider struct {
    // 客户端配置
}

func NewProvider() provider.Provider {
    return &AWSProvider{}
}

func (p *AWSProvider) GetName() string {
    return "aws"
}

func (p *AWSProvider) Initialize(config map[string]any) error {
    // 初始化逻辑
    return nil
}

// 实现其他接口方法...
```

注册 Provider:

```go
// internal/provider/aws/init.go
package aws

import "github.com/eryajf/zenops/internal/provider"

func init() {
    provider.Register("aws", NewProvider())
}
```

### 2. 添加新的 CLI 命令

在 `cmd/` 目录下创建新的命令文件:

```go
// cmd/your_command.go
package cmd

import (
    "github.com/spf13/cobra"
)

var yourCmd = &cobra.Command{
    Use:   "your-command",
    Short: "命令简短描述",
    Long:  `命令详细描述`,
    RunE: func(cmd *cobra.Command, args []string) error {
        // 命令实现
        return nil
    },
}

func init() {
    rootCmd.AddCommand(yourCmd)

    // 添加命令标志
    yourCmd.Flags().StringP("flag", "f", "", "标志描述")
}
```

### 3. 运行测试

```bash
# 运行所有测试
make test

# 运行测试并生成覆盖率报告
make test-coverage

# 运行特定包的测试
go test -v ./internal/provider/...
```

### 4. 代码格式化和检查

```bash
# 格式化代码
make fmt

# 代码检查 (需要安装 golangci-lint)
make lint
```

### 5. 构建不同平台的二进制文件

```bash
# 构建 Linux 版本
make build-linux

# 构建 macOS 版本
make build-darwin

# 构建 Windows 版本
make build-windows

# 构建所有平台版本
make build-all
```

## 项目结构说明

```
zenops/
├── cmd/                    # CLI 命令实现
│   ├── root.go            # 根命令
│   ├── serve.go           # 服务启动命令
│   ├── version.go         # 版本命令
│   └── query.go           # 查询命令组
│
├── internal/              # 私有应用代码
│   ├── provider/         # Provider 接口和实现
│   ├── model/            # 数据模型
│   ├── config/           # 配置管理
│   ├── service/          # 业务逻辑
│   ├── api/              # HTTP API
│   ├── mcp/              # MCP 协议
│   └── dingtalk/         # 钉钉集成
│
├── pkg/                   # 公共库代码
│   ├── utils/            # 工具函数
│   └── constants/        # 常量定义
│
├── configs/               # 配置文件
│   ├── config.yaml       # 默认配置
│   └── config.example.yaml  # 配置示例
│
└── docs/                  # 文档
```

## 调试技巧

### 使用日志

在代码中使用日志记录:

```go
import (
    "github.com/eryajf/zenops/pkg/utils"
    "go.uber.org/zap"
)

// 记录不同级别的日志
logx.Debug("调试信息", zap.String("key", "value"))
logx.Info("普通信息", zap.Int("count", 10))
logx.Warn("警告信息", zap.Error(err))
logx.Error("错误信息", zap.Error(err))
```

### 调整日志级别

在配置文件中修改日志级别:

```yaml
logging:
  level: debug  # debug, info, warn, error
  format: console
  output: stdout
```

或使用命令行标志:

```bash
./bin/zenops serve --log-level debug
```

## 常见问题

### Q: 编译失败,提示找不到依赖

A: 运行 `make deps` 或 `go mod tidy` 来下载依赖

### Q: 如何添加新的配置项?

A: 在 `internal/config/config.go` 中添加配置结构,然后在 `configs/config.yaml` 中添加默认值

### Q: 如何测试新添加的 Provider?

A: 在 `internal/provider/your_provider/` 目录下创建 `*_test.go` 文件,编写单元测试

## 下一步

- 阅读 [DESIGN.md](../DESIGN.md) 了解详细的架构设计
- 查看各个 Phase 的开发任务
- 开始实现具体的 Provider

## 获取帮助

如果遇到问题,可以:

1. 查看 [DESIGN.md](../DESIGN.md) 文档
2. 提交 Issue 到 GitHub
3. 联系项目维护者 @eryajf
