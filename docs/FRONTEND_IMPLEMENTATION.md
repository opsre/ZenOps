# ZenOps 前端配置管理实现文档

## 概述

基于 Art Design Pro (Vue 3 + TypeScript + Element Plus) 实现的配置管理前端页面。

## 项目结构

```
web/src/
├── api/
│   └── config.ts                    # 配置管理 API 封装
├── types/
│   └── api/
│       └── config.d.ts              # 配置管理类型定义
├── views/
│   └── config/                      # 配置管理页面
│       ├── llm/
│       │   └── index.vue            # LLM 配置页面
│       ├── provider/
│       │   └── index.vue            # 云厂商账号管理页面
│       ├── integration/
│       │   └── index.vue            # IM 和 CICD 配置页面
│       └── mcp/
│           └── index.vue            # MCP Server 管理页面
├── router/
│   └── modules/
│       └── config.ts                # 配置管理路由
└── locales/
    └── langs/
        └── zh.json                  # 中文国际化(已更新)
```

## 已实现功能

### 1. API 封装 (`api/config.ts`)

提供完整的配置管理 API 接口:

#### LLM 配置
- `fetchGetLLMConfig()` - 获取 LLM 配置
- `fetchSaveLLMConfig(data)` - 保存 LLM 配置

#### 云厂商账号
- `fetchGetProviderAccounts(provider?)` - 获取账号列表
- `fetchGetProviderAccount(id)` - 获取账号详情
- `fetchCreateProviderAccount(data)` - 创建账号
- `fetchUpdateProviderAccount(id, data)` - 更新账号
- `fetchDeleteProviderAccount(id)` - 删除账号

#### IM 配置
- `fetchGetIMConfigs()` - 获取所有 IM 配置
- `fetchGetIMConfig(platform)` - 获取指定平台配置
- `fetchSaveIMConfig(platform, data)` - 保存配置

#### CICD 配置
- `fetchGetCICDConfigs()` - 获取所有 CICD 配置
- `fetchGetCICDConfig(platform)` - 获取指定平台配置
- `fetchSaveCICDConfig(platform, data)` - 保存配置

#### MCP Server
- `fetchGetMCPServers()` - 获取 MCP Server 列表
- `fetchGetMCPServer(id)` - 获取详情
- `fetchCreateMCPServer(data)` - 创建
- `fetchUpdateMCPServer(id, data)` - 更新
- `fetchDeleteMCPServer(id)` - 删除

#### 系统配置
- `fetchGetSystemConfigs()` - 获取所有系统配置
- `fetchGetSystemConfig(key)` - 获取指定配置
- `fetchSetSystemConfig(data)` - 设置配置

### 2. 类型定义 (`types/api/config.d.ts`)

完整的 TypeScript 类型定义,包括:
- LLMConfig
- ProviderAccount
- IMConfig (DingTalkConfig, FeishuConfig, WecomConfig)
- CICDConfig
- MCPServer
- SystemConfig
- Response

### 3. 页面实现

#### LLM 配置页面 (`views/config/llm/index.vue`)

**功能:**
- 表单配置 LLM 参数
- 启用/禁用开关
- 模型名称、API Key、Base URL
- 表单验证
- 保存/重置/测试连接
- 配置说明卡片

**特点:**
- 响应式布局
- 实时验证
- 密码字段保护
- 友好的帮助提示

#### 云厂商账号管理页面 (`views/config/provider/index.vue`)

**功能:**
- 搜索栏(按云厂商筛选)
- 数据表格展示
  - 云厂商标签
  - 启用状态开关
  - 区域标签展示
- 添加/编辑对话框
  - 云厂商选择(阿里云/腾讯云)
  - 账号名称(仅字母数字下划线)
  - Access Key / Secret Key
  - 区域多选(预设+自定义)
  - 启用状态
- 删除确认

**特点:**
- 表格 CRUD 完整功能
- 智能区域选择(根据云厂商动态加载)
- 行内启用/禁用切换
- 表单验证(名称格式、必填项)

#### IM 和 CICD 配置页面 (`views/config/integration/index.vue`)

**功能:**
- 标签页切换(IM 平台 / CICD 工具)
- **IM 平台配置:**
  - 钉钉: App Key, App Secret, Agent ID, 卡片模板 ID
  - 飞书: App ID, App Secret
  - 企微: Token, EncodingAESKey
- **CICD 工具配置:**
  - Jenkins: URL, Username, API Token
  - 测试连接功能

**特点:**
- 卡片式布局
- 响应式列布局(桌面 2 列,移动端 1 列)
- 启用/禁用独立控制
- 密码字段保护
- 测试连接按钮(Jenkins)

#### MCP Server 管理页面 (`views/config/mcp/index.vue`)

**功能:**
- 数据表格展示
  - 类型标签(stdio/sse/streamableHttp)
  - 启用状态开关
  - 标签展示
- 添加/编辑对话框
  - 基础信息(名称、类型、描述)
  - **HTTP 类型配置:**
    - Base URL
    - Headers (动态添加/删除键值对)
  - **stdio 类型配置:**
    - Command
    - Arguments (动态列表)
    - 环境变量(动态键值对)
  - 高级配置
    - Tool Prefix
    - 超时时间
    - 自动注册/长期运行/启用复选框
- 删除确认

**特点:**
- 根据类型动态显示配置项
- 动态键值对管理(Headers, Env)
- 动态数组管理(Args)
- 表单验证(名称格式)
- 类型颜色区分

### 4. 路由配置 (`router/modules/config.ts`)

路由结构:
```
/config                    # 配置管理
├── /llm                  # LLM 配置
├── /provider             # 云厂商账号
├── /integration          # 集成配置(IM + CICD)
└── /mcp                  # MCP Server
```

**权限控制:**
- 角色: `R_SUPER`, `R_ADMIN`
- 菜单图标: `ri:settings-3-line`
- 不启用 KeepAlive 缓存

### 5. 国际化 (`locales/langs/zh.json`)

已添加配置管理菜单翻译:
```json
"config": {
  "title": "配置管理",
  "llm": "LLM 配置",
  "provider": "云厂商账号",
  "integration": "集成配置",
  "mcp": "MCP Server"
}
```

## 技术栈

- **框架**: Vue 3 Composition API
- **UI 库**: Element Plus
- **样式**: Tailwind CSS + SCSS
- **类型**: TypeScript
- **状态**: Reactive
- **图标**: Remix Icon
- **HTTP**: Axios (封装)

## 设计特点

### 1. 响应式设计
- 移动端友好
- 桌面端多列布局
- 自适应卡片大小

### 2. 用户体验
- 表单实时验证
- 操作成功/失败提示
- 加载状态反馈
- 删除二次确认
- 密码字段保护

### 3. 代码规范
- TypeScript 类型安全
- Composition API
- 组件化思维
- API 统一封装
- 错误处理完善

### 4. 性能优化
- 按需加载组件
- 懒加载路由
- v-loading 优化

## 使用说明

### 1. 安装依赖

```bash
cd web
pnpm install
```

### 2. 开发运行

```bash
pnpm dev
```

前端会运行在 `http://localhost:3006`

### 3. 环境配置

编辑 `.env.development`:
```bash
# API 地址
VITE_API_URL = http://localhost:8080
```

### 4. 构建生产

```bash
pnpm build
```

## 访问路径

启动后访问:
- http://localhost:3006/config/llm - LLM 配置
- http://localhost:3006/config/provider - 云厂商账号
- http://localhost:3006/config/integration - 集成配置
- http://localhost:3006/config/mcp - MCP Server

## API 对接

前端 API 请求会自动代理到后端:
```
前端: http://localhost:3006/api/v1/config/*
 ↓
后端: http://localhost:8080/api/v1/config/*
```

确保后端服务运行在 `http://localhost:8080`

## 扩展开发

### 添加新的配置页面

1. **创建 Vue 组件**: `views/config/xxx/index.vue`
2. **添加路由**: 在 `router/modules/config.ts` 中添加子路由
3. **添加翻译**: 在 `locales/langs/zh.json` 中添加菜单文本
4. **实现 API**: 在 `api/config.ts` 中添加接口方法
5. **定义类型**: 在 `types/api/config.d.ts` 中添加类型

### 组件复用

项目使用了 Art Design Pro 的组件库:
- `<art-page-content>` - 页面容器
- `<art-search-bar>` - 搜索栏
- `<art-table>` - 数据表格

可以直接复用这些组件快速开发。

## 注意事项

1. **CORS 配置**: 确保后端允许前端域名的跨域请求
2. **认证**: 所有 API 请求会自动添加 Authorization header
3. **权限**: 配置管理菜单仅对 SUPER 和 ADMIN 角色可见
4. **表单验证**: 必填字段都有前端验证,但后端也应做验证
5. **敏感信息**: API Key、Token 等使用密码输入框,传输时确保 HTTPS

## 待优化项

1. **配置热更新**: 修改配置后自动重载,无需重启服务
2. **配置导入导出**: 支持批量导入导出配置
3. **配置历史**: 记录配置变更历史,支持回滚
4. **连接测试**: 完善各配置项的连接测试功能
5. **英文翻译**: 添加英文国际化支持
6. **配置校验**: 更详细的配置字段校验规则

## 总结

前端配置管理模块已完全实现,提供了:
- ✅ 完整的 CRUD 功能
- ✅ 友好的用户界面
- ✅ 响应式设计
- ✅ 类型安全
- ✅ 表单验证
- ✅ 国际化支持

可以直接投入使用,后续可根据需求继续优化和扩展功能。
