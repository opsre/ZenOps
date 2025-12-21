# 配置数据库化迁移指南

## 概述

ZenOps 已完成配置管理从 YAML 文件到 SQLite 数据库的迁移。本文档介绍如何集成和使用新的配置管理系统。

## 架构设计

### 数据库表结构

配置按类型分表存储:

1. **llm_config** - LLM 大模型配置
2. **provider_accounts** - 云厂商账号配置(阿里云/腾讯云)
3. **im_config** - IM 平台配置(钉钉/飞书/企微)
4. **cicd_config** - CICD 工具配置(Jenkins)
5. **mcp_servers** - MCP 服务器配置
6. **system_config** - 系统配置(服务器/认证/缓存等)

### 配置加载策略

1. **优先从数据库加载**: 首次启动时优先尝试从数据库加载配置
2. **YAML 作为后备**: 如果数据库为空,从 YAML 文件加载
3. **自动迁移**: YAML 配置会自动迁移到数据库(仅首次)
4. **页面管理**: 后续通过 Web 页面管理配置

## 集成步骤

### 1. 修改 cmd/root.go

在启动命令中集成配置迁移逻辑:

```go
package cmd

import (
	"log"
	"os"

	"github.com/eryajf/ZenOps/internal/config"
	"github.com/eryajf/ZenOps/internal/database"
	"github.com/eryajf/ZenOps/internal/service"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "zenops",
	Short: "ZenOps - 运维智能化工具",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// 1. 初始化数据库
		db := database.GetDB()
		if db == nil {
			return fmt.Errorf("failed to initialize database")
		}

		// 2. 设置配置加载器
		configService := service.NewConfigService()

		// 设置数据库配置加载函数
		config.SetDBLoader(func() (*config.Config, error) {
			return configService.LoadConfigFromDB()
		})

		// 设置配置迁移函数
		config.SetDBMigrator(func(cfg *config.Config) error {
			// 迁移 YAML 配置到数据库
			if err := configService.MigrateFromYAML(cfg); err != nil {
				return err
			}

			// 迁移 MCP Servers 配置
			if cfg.MCPServersConfig != "" {
				if err := configService.MigrateMCPServersFromJSON(cfg.MCPServersConfig); err != nil {
					log.Printf("Warning: failed to migrate MCP servers: %v", err)
				}
			}

			return nil
		})

		// 3. 加载配置(优先从数据库)
		cfg, err := config.LoadConfigWithDB(configFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// 4. 设置全局配置
		config.SetGlobalConfig(cfg)

		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file (default: ./config.yaml)")
}
```

### 2. 数据库文件位置

默认数据库文件位置: `./data/zenops.db`

可通过环境变量自定义:
```bash
export ZENOPS_DB_PATH=/path/to/your/zenops.db
```

### 3. 安装依赖

```bash
go mod tidy
```

这会安装以下新依赖:
- `github.com/glebarez/sqlite v1.11.0`
- `gorm.io/gorm v1.25.5`

## API 接口

### LLM 配置

```bash
# 获取 LLM 配置
GET /api/v1/config/llm

# 保存 LLM 配置
PUT /api/v1/config/llm
Content-Type: application/json

{
  "enabled": true,
  "model": "DeepSeek-V3",
  "api_key": "your-api-key",
  "base_url": ""
}
```

### 云厂商账号配置

```bash
# 列出所有云厂商账号
GET /api/v1/config/providers?provider=aliyun

# 创建云厂商账号
POST /api/v1/config/providers
Content-Type: application/json

{
  "provider": "aliyun",
  "name": "production",
  "enabled": true,
  "access_key": "YOUR_AK",
  "secret_key": "YOUR_SK",
  "regions": ["cn-hangzhou", "cn-shanghai"]
}

# 更新云厂商账号
PUT /api/v1/config/providers/:id

# 删除云厂商账号
DELETE /api/v1/config/providers/:id
```

### IM 配置

```bash
# 获取钉钉配置
GET /api/v1/config/im/dingtalk

# 保存钉钉配置
PUT /api/v1/config/im/dingtalk
Content-Type: application/json

{
  "enabled": true,
  "config_data": {
    "app_key": "your-app-key",
    "app_secret": "your-app-secret",
    "agent_id": "your-agent-id",
    "card_template_id": ""
  }
}

# 列出所有 IM 配置
GET /api/v1/config/im
```

### CICD 配置

```bash
# 获取 Jenkins 配置
GET /api/v1/config/cicd/jenkins

# 保存 Jenkins 配置
PUT /api/v1/config/cicd/jenkins
Content-Type: application/json

{
  "enabled": true,
  "url": "https://jenkins.example.com",
  "username": "admin",
  "token": "your-token"
}
```

### MCP Server 配置

```bash
# 列出所有 MCP 服务器
GET /api/v1/config/mcp

# 创建 MCP 服务器
POST /api/v1/config/mcp
Content-Type: application/json

{
  "name": "gitea",
  "is_active": true,
  "type": "streamableHttp",
  "description": "Gitea MCP Server",
  "base_url": "http://localhost:5566/mcp",
  "headers": {
    "Authorization": "Bearer your-token"
  },
  "long_running": true,
  "timeout": 300,
  "tool_prefix": "",
  "auto_register": true
}

# 更新 MCP 服务器
PUT /api/v1/config/mcp/:id

# 删除 MCP 服务器
DELETE /api/v1/config/mcp/:id
```

### 系统配置

```bash
# 列出所有系统配置
GET /api/v1/config/system

# 设置系统配置
POST /api/v1/config/system
Content-Type: application/json

{
  "key": "server.http.port",
  "value": "8080",
  "description": "HTTP server port"
}
```

## 配置迁移流程

### 首次启动

1. 程序检查数据库中是否有配置
2. 如果数据库为空,从 `config.yaml` 加载配置
3. 自动将 YAML 配置迁移到数据库
4. 如果有 `mcp_servers.json`,也会自动迁移
5. 后续启动直接从数据库加载

### 配置优先级

```
数据库配置 > YAML 配置 > 环境变量 > 默认值
```

### 手动导出配置

如果需要将数据库配置导出为 JSON:

```go
configService := service.NewConfigService()
err := configService.ExportMCPServersToJSON("./mcp_servers_backup.json")
```

## 环境变量支持

依然支持通过环境变量覆盖配置:

```bash
export ZENOPS_DB_PATH=/custom/path/zenops.db
export ZENOPS_LLM_API_KEY=your-api-key
export ZENOPS_SERVER_HTTP_PORT=9090
```

## 数据库备份

建议定期备份数据库文件:

```bash
# 备份数据库
cp ./data/zenops.db ./data/zenops_backup_$(date +%Y%m%d).db

# 恢复数据库
cp ./data/zenops_backup_20250101.db ./data/zenops.db
```

## 注意事项

1. **SQLite 并发限制**: SQLite 设置了 `MaxOpenConns(1)`,适合单实例部署
2. **配置热更新**: 通过 API 修改配置后,需要重启服务生效
3. **敏感信息**: API Key、Token 等敏感信息存储在数据库中,请妥善保管数据库文件
4. **YAML 配置**: 迁移后,YAML 配置文件仍保留作为备份,但不再使用

## 故障排查

### 数据库初始化失败

检查目录权限:
```bash
mkdir -p ./data
chmod 755 ./data
```

### 配置迁移失败

查看日志输出,手动检查 YAML 配置文件格式是否正确

### API 调用失败

1. 检查 HTTP 服务是否启动
2. 检查端口是否正确
3. 使用 curl 测试接口

```bash
curl http://localhost:8080/api/v1/config/llm
```

## 下一步

1. 开发前端配置管理页面
2. 实现配置变更通知机制
3. 添加配置版本控制
4. 支持配置导入导出功能
