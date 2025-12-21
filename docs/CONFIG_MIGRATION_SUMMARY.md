# ZenOps 配置数据库化改造 - 实现总结

## 改造目标

将 ZenOps 的配置管理从 YAML 文件迁移到 SQLite 数据库,实现配置的分类管理和 Web 页面配置。

## 已完成的工作

### 1. 数据库架构设计 ✅

创建了 6 张表,按配置类型分类存储:

| 表名 | 用途 | 文件 |
|------|------|------|
| llm_config | LLM 配置 | [model/config_llm.go](../internal/model/config_llm.go) |
| provider_accounts | 云厂商账号 | [model/config_provider.go](../internal/model/config_provider.go) |
| im_config | IM 平台配置 | [model/config_im.go](../internal/model/config_im.go) |
| cicd_config | CICD 配置 | [model/config_cicd.go](../internal/model/config_cicd.go) |
| mcp_servers | MCP 服务器 | [model/config_mcp.go](../internal/model/config_mcp.go) |
| system_config | 系统配置 | [model/config_system.go](../internal/model/config_system.go) |

### 2. 数据库连接和模型 ✅

**实现文件:**
- [internal/database/db.go](../internal/database/db.go) - 数据库连接管理(单例模式)
- [internal/database/migrate.go](../internal/database/migrate.go) - 自动迁移表结构

**特性:**
- 使用 `github.com/glebarez/sqlite` (纯 Go 实现)
- 单例模式管理数据库连接
- 支持环境变量配置数据库路径
- 自动创建数据目录和表结构

### 3. 配置服务层 ✅

**实现文件:**
- [internal/service/config_service.go](../internal/service/config_service.go) - 配置 CRUD 操作
- [internal/service/config_migration.go](../internal/service/config_migration.go) - YAML 到数据库迁移
- [internal/service/mcp_migration.go](../internal/service/mcp_migration.go) - MCP 配置迁移

**功能:**
- LLM 配置管理
- 云厂商账号 CRUD
- IM 配置管理
- CICD 配置管理
- MCP Server 配置管理
- 系统配置管理
- YAML 自动迁移
- MCP JSON 配置迁移

### 4. 配置加载逻辑 ✅

**实现文件:**
- [internal/config/loader.go](../internal/config/loader.go) - 配置加载入口
- [internal/config/db_loader.go](../internal/config/db_loader.go) - 数据库配置加载器

**策略:**
1. 优先从数据库加载
2. 数据库为空时从 YAML 加载
3. 自动将 YAML 迁移到数据库
4. 支持 MCP servers.json 迁移

### 5. HTTP API 接口 ✅

**实现文件:**
- [internal/server/config_handler.go](../internal/server/config_handler.go) - 配置管理 API handler
- [internal/server/http.go](../internal/server/http.go) - 路由注册(已修改)

**API 端点:**

```
/api/v1/config/
├── /llm                [GET, PUT]
├── /providers          [GET, POST, PUT, DELETE]
├── /im                 [GET]
├── /im/:platform       [GET, PUT]
├── /cicd               [GET]
├── /cicd/:platform     [GET, PUT]
├── /mcp                [GET, POST, PUT, DELETE]
└── /system             [GET, POST]
```

### 6. 依赖管理 ✅

**修改文件:** [go.mod](../go.mod)

**新增依赖:**
```go
github.com/glebarez/sqlite v1.11.0
gorm.io/gorm v1.25.5
```

### 7. 文档 ✅

创建了完整的文档:
- [CONFIG_DATABASE_MIGRATION.md](./CONFIG_DATABASE_MIGRATION.md) - 迁移指南
- [INTEGRATION_EXAMPLE.go](./INTEGRATION_EXAMPLE.go) - 集成示例代码

## 核心代码结构

```
ZenOps/
├── internal/
│   ├── database/              # 数据库管理
│   │   ├── db.go             # 数据库连接(单例)
│   │   └── migrate.go        # 表结构迁移
│   ├── model/                 # 数据模型
│   │   ├── config_llm.go     # LLM 配置模型
│   │   ├── config_provider.go # 云厂商配置模型
│   │   ├── config_im.go      # IM 配置模型
│   │   ├── config_cicd.go    # CICD 配置模型
│   │   ├── config_mcp.go     # MCP 配置模型
│   │   └── config_system.go  # 系统配置模型
│   ├── service/               # 业务逻辑层
│   │   ├── config_service.go    # 配置 CRUD
│   │   ├── config_migration.go  # 配置迁移
│   │   └── mcp_migration.go     # MCP 迁移
│   ├── config/                # 配置加载
│   │   ├── config.go          # 配置结构定义
│   │   ├── loader.go          # 配置加载器
│   │   └── db_loader.go       # DB 加载器
│   └── server/                # HTTP 服务
│       ├── http.go            # HTTP 服务器(已修改)
│       └── config_handler.go  # 配置 API (新增)
├── docs/                      # 文档
│   ├── CONFIG_DATABASE_MIGRATION.md
│   ├── INTEGRATION_EXAMPLE.go
│   └── CONFIG_MIGRATION_SUMMARY.md (本文件)
└── go.mod                     # 依赖管理(已修改)
```

## 使用流程

### 1. 首次启动

```bash
# 1. 安装依赖
go mod tidy

# 2. 启动服务(会自动迁移配置)
go run main.go run --config config.yaml
```

**执行过程:**
1. 初始化数据库 (创建 `./data/zenops.db`)
2. 尝试从数据库加载配置
3. 数据库为空,从 `config.yaml` 加载
4. 自动迁移配置到数据库
5. 如果有 `mcp_servers.json`,也会迁移
6. 服务启动,后续使用数据库配置

### 2. 通过 API 管理配置

```bash
# 查看 LLM 配置
curl http://localhost:8080/api/v1/config/llm

# 更新 LLM 配置
curl -X PUT http://localhost:8080/api/v1/config/llm \
  -H "Content-Type: application/json" \
  -d '{
    "enabled": true,
    "model": "DeepSeek-V3",
    "api_key": "sk-xxxx",
    "base_url": ""
  }'

# 添加阿里云账号
curl -X POST http://localhost:8080/api/v1/config/providers \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "aliyun",
    "name": "test",
    "enabled": true,
    "access_key": "LTAI...",
    "secret_key": "xxx",
    "regions": ["cn-hangzhou"]
  }'

# 查看所有云厂商账号
curl http://localhost:8080/api/v1/config/providers

# 添加 MCP Server
curl -X POST http://localhost:8080/api/v1/config/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "name": "gitea",
    "is_active": true,
    "type": "streamableHttp",
    "base_url": "http://localhost:5566/mcp",
    "headers": {"Authorization": "Bearer token"},
    "auto_register": true
  }'
```

### 3. 配置生效

配置修改后需要**重启服务**才能生效。

## 集成到项目

### 方式一: 完整集成

参考 [INTEGRATION_EXAMPLE.go](./INTEGRATION_EXAMPLE.go),将代码整合到 `cmd/root.go`

### 方式二: 最小集成

在 `cmd/root.go` 的 `PersistentPreRunE` 中添加:

```go
import (
	"github.com/eryajf/ZenOps/internal/config"
	"github.com/eryajf/ZenOps/internal/database"
	"github.com/eryajf/ZenOps/internal/service"
)

// 在命令执行前初始化
PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
	// 1. 初始化数据库
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("failed to init database")
	}

	// 2. 设置配置加载器
	configService := service.NewConfigService()
	config.SetDBLoader(configService.LoadConfigFromDB)
	config.SetDBMigrator(func(cfg *config.Config) error {
		if err := configService.MigrateFromYAML(cfg); err != nil {
			return err
		}
		if cfg.MCPServersConfig != "" {
			configService.MigrateMCPServersFromJSON(cfg.MCPServersConfig)
		}
		return nil
	})

	// 3. 加载配置
	cfg, err := config.LoadConfigWithDB(configFile)
	if err != nil {
		return err
	}
	config.SetGlobalConfig(cfg)

	return nil
},
```

## 环境变量

```bash
# 自定义数据库路径
export ZENOPS_DB_PATH=/custom/path/zenops.db

# 其他配置环境变量仍然支持
export ZENOPS_LLM_API_KEY=sk-xxx
export ZENOPS_SERVER_HTTP_PORT=9090
```

## 数据库文件

- **默认路径**: `./data/zenops.db`
- **大小**: 初始约 20KB,随配置增加而增长
- **备份**: 建议定期备份 `zenops.db` 文件
- **权限**: 确保应用有读写权限

## 后续工作

### 待实现

1. **前端配置管理页面** (基于现有 Vue 3 项目)
   - LLM 配置页面
   - 云厂商账号管理页面
   - IM 配置页面
   - CICD 配置页面
   - MCP Server 管理页面

2. **配置热更新**
   - 监听配置变更
   - 动态重载配置
   - 无需重启服务

3. **配置版本控制**
   - 记录配置变更历史
   - 支持配置回滚

4. **配置导入导出**
   - 导出配置为 YAML/JSON
   - 批量导入配置

5. **配置校验**
   - 添加配置字段校验
   - 连接测试功能

## 测试建议

### 1. 单元测试

```bash
# 测试数据库连接
go test ./internal/database/...

# 测试配置服务
go test ./internal/service/...

# 测试配置加载
go test ./internal/config/...
```

### 2. 集成测试

1. 删除现有数据库
2. 启动服务,验证自动迁移
3. 通过 API 修改配置
4. 重启服务,验证配置持久化

### 3. API 测试

使用 Postman 或 curl 测试所有 API 端点

## 性能考虑

1. **SQLite 并发**: 设置 `MaxOpenConns(1)`,适合单实例部署
2. **配置缓存**: 建议在内存中缓存配置,减少数据库查询
3. **索引优化**: 已在 provider/name 等字段上创建索引

## 安全考虑

1. **敏感信息**: API Key、Token 存储在数据库中
2. **数据库权限**: 确保数据库文件权限设置正确 (600 或 644)
3. **API 认证**: 建议为配置 API 添加认证中间件
4. **配置审计**: 记录配置变更日志

## 兼容性

- **Go 版本**: Go 1.25.4+
- **SQLite**: 通过 glebarez/sqlite (纯 Go 实现),无需 CGO
- **操作系统**: Linux/macOS/Windows 全平台支持
- **YAML 配置**: 保持向后兼容,仍可使用 YAML

## 总结

本次改造完成了:
1. ✅ 数据库架构设计
2. ✅ 数据模型定义
3. ✅ 配置服务层实现
4. ✅ 配置迁移逻辑
5. ✅ HTTP API 接口
6. ✅ 依赖管理
7. ✅ 完整文档

**核心优势:**
- 配置分类清晰,易于管理
- 自动迁移,无需手动操作
- RESTful API,便于集成
- 完整文档,易于理解

**下一步:**
- 开发前端配置管理界面
- 实现配置热更新
- 添加配置校验和测试功能

---

**作者**: Claude Sonnet 4.5
**日期**: 2025-12-21
**版本**: v1.0
