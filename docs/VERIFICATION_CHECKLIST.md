# ZenOps 配置管理系统验证清单

## 后端验证

### 1. 编译验证
- [x] `go mod tidy` 执行成功
- [x] `go build -o zenops main.go` 编译成功
- [x] `./zenops --help` 运行正常

### 2. 数据库验证
```bash
# 启动服务，检查数据库初始化
./zenops run

# 应该看到以下日志:
# - Database initialized successfully
# - Auto-migration completed
# - Starting configuration migration from YAML to database...
# - MCP servers migration completed
```

检查项:
- [ ] 数据库文件创建成功 (`data/zenops.db`)
- [ ] 6 个配置表创建成功
- [ ] YAML 配置自动迁移成功
- [ ] MCP servers 自动迁移成功

### 3. API 端点验证

使用 curl 或 Postman 测试以下端点:

#### LLM 配置
```bash
# 获取 LLM 配置
curl http://localhost:8080/api/v1/config/llm

# 保存 LLM 配置
curl -X PUT http://localhost:8080/api/v1/config/llm \
  -H "Content-Type: application/json" \
  -d '{
    "enabled": true,
    "model": "DeepSeek-V3",
    "api_key": "your-api-key",
    "base_url": "https://api.deepseek.com"
  }'
```

#### 云厂商账号
```bash
# 列出账号
curl http://localhost:8080/api/v1/config/providers

# 创建账号
curl -X POST http://localhost:8080/api/v1/config/providers \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "aliyun",
    "name": "生产环境",
    "enabled": true,
    "access_key": "your-access-key",
    "secret_key": "your-secret-key",
    "regions": ["cn-hangzhou", "cn-shanghai"]
  }'

# 更新账号
curl -X PUT http://localhost:8080/api/v1/config/providers/1 \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "aliyun",
    "name": "生产环境",
    "enabled": false,
    "access_key": "your-access-key",
    "secret_key": "your-secret-key",
    "regions": ["cn-hangzhou"]
  }'

# 删除账号
curl -X DELETE http://localhost:8080/api/v1/config/providers/1
```

#### IM 配置
```bash
# 列出所有 IM 配置
curl http://localhost:8080/api/v1/config/im

# 获取钉钉配置
curl http://localhost:8080/api/v1/config/im/dingtalk

# 保存钉钉配置
curl -X PUT http://localhost:8080/api/v1/config/im/dingtalk \
  -H "Content-Type: application/json" \
  -d '{
    "platform": "dingtalk",
    "enabled": true,
    "config_data": {
      "app_key": "your-app-key",
      "app_secret": "your-app-secret",
      "agent_id": "your-agent-id"
    }
  }'
```

#### CICD 配置
```bash
# 列出所有 CICD 配置
curl http://localhost:8080/api/v1/config/cicd

# 获取 Jenkins 配置
curl http://localhost:8080/api/v1/config/cicd/jenkins

# 保存 Jenkins 配置
curl -X PUT http://localhost:8080/api/v1/config/cicd/jenkins \
  -H "Content-Type: application/json" \
  -d '{
    "platform": "jenkins",
    "enabled": true,
    "url": "https://jenkins.example.com",
    "username": "admin",
    "token": "your-api-token"
  }'
```

#### MCP Server
```bash
# 列出所有 MCP 服务器
curl http://localhost:8080/api/v1/config/mcp

# 创建 MCP 服务器
curl -X POST http://localhost:8080/api/v1/config/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "name": "weather-mcp",
    "is_active": true,
    "type": "sse",
    "description": "Weather data provider",
    "base_url": "https://weather.example.com/sse",
    "headers": {
      "Authorization": "Bearer token"
    }
  }'

# 更新 MCP 服务器
curl -X PUT http://localhost:8080/api/v1/config/mcp/1 \
  -H "Content-Type: application/json" \
  -d '{
    "name": "weather-mcp",
    "is_active": false,
    "type": "sse",
    "description": "Weather data provider (disabled)",
    "base_url": "https://weather.example.com/sse"
  }'

# 删除 MCP 服务器
curl -X DELETE http://localhost:8080/api/v1/config/mcp/1
```

#### 系统配置
```bash
# 列出所有系统配置
curl http://localhost:8080/api/v1/config/system

# 获取系统配置
curl http://localhost:8080/api/v1/config/system/app_name

# 设置系统配置
curl -X POST http://localhost:8080/api/v1/config/system \
  -H "Content-Type: application/json" \
  -d '{
    "key": "app_name",
    "value": "ZenOps",
    "description": "应用名称"
  }'
```

检查项:
- [ ] 所有 GET 请求返回 200 状态码
- [ ] 所有 POST/PUT 请求成功创建/更新数据
- [ ] 所有 DELETE 请求成功删除数据
- [ ] 响应格式符合预期 (code, message, data)

## 前端验证

### 1. 项目结构验证
- [x] 删除了无用的 result 和 exception 页面
- [x] 更新了路由配置
- [x] 更新了国际化配置
- [x] 保留了所有核心配置页面

### 2. 依赖安装
```bash
cd web
npm install
```

检查项:
- [ ] 依赖安装成功，无错误

### 3. 开发服务器启动
```bash
cd web
npm run dev
```

检查项:
- [ ] 开发服务器启动成功
- [ ] 访问 `http://localhost:5173` 能够正常显示页面
- [ ] 无 console 错误

### 4. 页面访问验证

#### 登录
- [ ] 访问 `http://localhost:5173/#/auth/login`
- [ ] 页面正常显示
- [ ] 表单验证正常
- [ ] 登录功能正常

#### 仪表盘
- [ ] 访问 `http://localhost:5173/#/dashboard/console`
- [ ] 页面正常显示
- [ ] 数据加载正常

#### LLM 配置
- [ ] 访问 `http://localhost:5173/#/config/llm`
- [ ] 页面正常显示
- [ ] 能够获取当前配置
- [ ] 能够保存配置
- [ ] 启用/禁用开关工作正常
- [ ] 测试连接按钮正常

#### 云厂商账号
- [ ] 访问 `http://localhost:5173/#/config/provider`
- [ ] 页面正常显示
- [ ] 列表数据加载正常
- [ ] 能够创建新账号
- [ ] 能够编辑账号
- [ ] 能够删除账号
- [ ] 搜索和筛选功能正常
- [ ] 启用/禁用开关工作正常

#### 集成配置
- [ ] 访问 `http://localhost:5173/#/config/integration`
- [ ] IM 平台配置标签页正常显示
- [ ] CICD 工具配置标签页正常显示
- [ ] 钉钉配置能够保存
- [ ] 飞书配置能够保存
- [ ] 企业微信配置能够保存
- [ ] Jenkins 配置能够保存
- [ ] 测试连接功能正常

#### MCP Server
- [ ] 访问 `http://localhost:5173/#/config/mcp`
- [ ] 页面正常显示
- [ ] 列表数据加载正常
- [ ] 能够创建新 MCP 服务器
- [ ] 能够编辑 MCP 服务器
- [ ] 能够删除 MCP 服务器
- [ ] stdio 类型配置正常
- [ ] sse 类型配置正常
- [ ] streamableHttp 类型配置正常
- [ ] Headers 动态添加/删除正常
- [ ] 环境变量动态添加/删除正常
- [ ] 命令参数动态添加/删除正常

#### 系统管理
- [ ] 访问 `http://localhost:5173/#/system/user`
- [ ] 用户管理页面正常显示
- [ ] 访问 `http://localhost:5173/#/system/role`
- [ ] 角色管理页面正常显示
- [ ] 访问 `http://localhost:5173/#/system/menu`
- [ ] 菜单管理页面正常显示
- [ ] 访问 `http://localhost:5173/#/system/user-center`
- [ ] 用户中心页面正常显示

### 5. TypeScript 类型检查
```bash
cd web
npm run build
```

检查项:
- [ ] TypeScript 编译无错误
- [ ] Vite 构建成功
- [ ] 生成 dist 目录

### 6. 响应式布局验证
- [ ] 在桌面浏览器中布局正常
- [ ] 在平板视图中布局正常
- [ ] 在移动视图中布局正常

### 7. 国际化验证
- [ ] 切换到中文，菜单正确显示
- [ ] 切换到英文，菜单正确显示
- [ ] 配置页面标题正确翻译

## 集成测试

### 1. 完整流程测试

#### LLM 配置流程
1. [ ] 启动后端服务
2. [ ] 启动前端服务
3. [ ] 登录系统
4. [ ] 进入 LLM 配置页面
5. [ ] 修改配置并保存
6. [ ] 刷新页面，验证配置已保存
7. [ ] 重启后端服务
8. [ ] 再次访问配置页面，验证配置持久化

#### 云厂商账号流程
1. [ ] 进入云厂商账号页面
2. [ ] 创建阿里云账号
3. [ ] 编辑账号信息
4. [ ] 禁用账号
5. [ ] 启用账号
6. [ ] 删除账号
7. [ ] 验证所有操作成功

#### IM 配置流程
1. [ ] 进入集成配置页面
2. [ ] 配置钉钉
3. [ ] 配置飞书
4. [ ] 配置企业微信
5. [ ] 保存所有配置
6. [ ] 验证配置生效

#### MCP Server 流程
1. [ ] 进入 MCP Server 页面
2. [ ] 创建 stdio 类型服务器
3. [ ] 创建 sse 类型服务器
4. [ ] 编辑服务器配置
5. [ ] 禁用服务器
6. [ ] 删除服务器
7. [ ] 验证所有操作成功

### 2. 数据一致性验证
```bash
# 查看数据库内容
sqlite3 data/zenops.db "SELECT * FROM llm_config;"
sqlite3 data/zenops.db "SELECT * FROM provider_accounts;"
sqlite3 data/zenops.db "SELECT * FROM im_config;"
sqlite3 data/zenops.db "SELECT * FROM cicd_config;"
sqlite3 data/zenops.db "SELECT * FROM mcp_servers;"
sqlite3 data/zenops.db "SELECT * FROM system_config;"
```

检查项:
- [ ] 数据库中的数据与前端显示一致
- [ ] JSON 字段正确存储和解析
- [ ] 日期时间正确记录

### 3. 错误处理验证
- [ ] 提交空表单，显示验证错误
- [ ] 提交无效数据，显示错误提示
- [ ] 网络错误时显示友好提示
- [ ] 删除操作有确认提示

## 性能验证

### 1. 页面加载性能
- [ ] 首页加载时间 < 2s
- [ ] 配置页面切换流畅
- [ ] 列表数据加载快速

### 2. 数据库性能
- [ ] 查询响应时间 < 100ms
- [ ] 插入/更新响应时间 < 200ms
- [ ] 并发操作无冲突

## 安全验证

### 1. 敏感数据保护
- [ ] API Key 显示为密码框
- [ ] Secret Key 显示为密码框
- [ ] 数据库中密钥未加密（需要后续添加加密）

### 2. 权限控制
- [ ] 配置页面需要 R_SUPER 或 R_ADMIN 权限
- [ ] 未授权访问返回 403

## 文档验证

### 已完成的文档
- [x] `docs/CONFIG_DATABASE_MIGRATION.md` - 后端迁移指南
- [x] `docs/FRONTEND_IMPLEMENTATION.md` - 前端实现文档
- [x] `docs/FINAL_IMPLEMENTATION_SUMMARY.md` - 完整实现总结
- [x] `docs/FRONTEND_CLEANUP_SUMMARY.md` - 前端清理总结
- [x] `docs/VERIFICATION_CHECKLIST.md` - 验证清单

### 文档完整性
- [ ] 所有 API 端点有文档说明
- [ ] 所有配置项有说明
- [ ] 有使用示例
- [ ] 有故障排查指南

## 问题记录

### 已解决的问题
1. ✅ Go 模块路径大小写不一致 - 已修复为小写 `github.com/eryajf/zenops`
2. ✅ Response 类型重复声明 - 已删除重复定义
3. ✅ 前端无用页面过多 - 已删除 result 和 exception 页面

### 待解决的问题
- [ ] (在此记录测试中发现的问题)

## 验证结论

### 后端
- [ ] ✅ 全部通过
- [ ] ⚠️ 部分问题需要修复
- [ ] ❌ 存在严重问题

### 前端
- [ ] ✅ 全部通过
- [ ] ⚠️ 部分问题需要修复
- [ ] ❌ 存在严重问题

### 集成
- [ ] ✅ 全部通过
- [ ] ⚠️ 部分问题需要修复
- [ ] ❌ 存在严重问题

## 备注

测试时间: ___________
测试人员: ___________
环境信息:
- OS: ___________
- Go 版本: ___________
- Node 版本: ___________
- 浏览器: ___________
