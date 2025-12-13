# 腾讯云 Provider 使用指南

## 概述

ZenOps 的腾讯云 Provider 提供了对腾讯云资源的统一查询能力,包括:

- **CVM (云服务器)**: 查询云服务器实例信息
- **CDB (云数据库)**: 查询 MySQL 数据库实例信息

## 配置

### 1. 获取腾讯云 API 密钥

登录 [腾讯云控制台](https://console.cloud.tencent.com/cam/capi),创建并获取:

- Secret ID
- Secret Key

### 2. 配置文件

编辑 `configs/config.yaml`:

```yaml
providers:
  tencent:
    - name: production  # 账号名称
      enabled: true
      secret_id: ${TENCENT_SECRET_ID}
      secret_key: ${TENCENT_SECRET_KEY}
      regions:
        - ap-guangzhou  # 广州
        - ap-shanghai   # 上海
        - ap-beijing    # 北京
```

### 3. 环境变量

```bash
export TENCENT_SECRET_ID="your-secret-id"
export TENCENT_SECRET_KEY="your-secret-key"
```

## CLI 命令使用

### CVM (云服务器)

#### 列出所有 CVM 实例

```bash
# 列出所有区域的实例
./bin/zenops query tencent cvm list

# 指定区域查询
./bin/zenops query tencent cvm list --region ap-guangzhou

# JSON 格式输出
./bin/zenops query tencent cvm list --output json

# 指定账号查询
./bin/zenops query tencent cvm list --account production

# 单页查询 (不自动分页)
./bin/zenops query tencent cvm list --all=false --page-size 20
```

#### 获取 CVM 实例详情

```bash
# 通过实例 ID 获取详情
./bin/zenops query tencent cvm get ins-xxxxxx
```

### CDB (云数据库)

#### 列出所有 CDB 实例

```bash
# 列出所有区域的数据库
./bin/zenops query tencent cdb list

# 指定区域查询
./bin/zenops query tencent cdb list --region ap-shanghai

# JSON 格式输出
./bin/zenops query tencent cdb list --output json
```

#### 获取 CDB 实例详情

```bash
# 通过实例 ID 获取详情
./bin/zenops query tencent cdb get cdb-xxxxxx
```

## 命令参数

### 通用参数

| 参数 | 简写 | 默认值 | 说明 |
|-----|------|--------|------|
| `--account` | `-a` | (第一个启用账号) | 指定账号名称 |
| `--region` | `-r` | (所有区域) | 指定区域 |
| `--output` | `-o` | `table` | 输出格式 (table, json) |
| `--page-size` | | `10` | 分页大小 |
| `--page-num` | | `1` | 页码 |
| `--all` | | `true` | 自动分页获取所有资源 |

## 支持的区域

### 中国大陆

- `ap-guangzhou` - 广州
- `ap-shanghai` - 上海
- `ap-nanjing` - 南京
- `ap-beijing` - 北京
- `ap-chengdu` - 成都
- `ap-chongqing` - 重庆

### 其他地区

- `ap-hongkong` - 香港
- `ap-singapore` - 新加坡
- `ap-tokyo` - 东京
- `ap-seoul` - 首尔
- `na-siliconvalley` - 硅谷
- `na-ashburn` - 弗吉尼亚

完整区域列表请参考: https://cloud.tencent.com/document/product/213/6091

## 数据模型

### CVM 实例

```json
{
  "id": "ins-xxxxxx",
  "name": "my-server",
  "provider": "tencent",
  "region": "ap-guangzhou",
  "zone": "ap-guangzhou-3",
  "instance_type": "S5.MEDIUM4",
  "status": "RUNNING",
  "private_ip": ["10.0.0.1"],
  "public_ip": ["1.2.3.4"],
  "cpu": 2,
  "memory": 4096,
  "os_type": "CentOS 7.6 64bit",
  "tags": {
    "env": "production"
  },
  "metadata": {
    "vpc_id": "vpc-xxxxxx",
    "subnet_id": "subnet-xxxxxx",
    "charge_type": "POSTPAID_BY_HOUR",
    "created_time": "2023-01-01T00:00:00Z"
  }
}
```

### CDB 实例

```json
{
  "id": "cdb-xxxxxx",
  "name": "my-database",
  "provider": "tencent",
  "region": "ap-shanghai",
  "engine": "MySQL",
  "engine_version": "5.7",
  "status": "Running",
  "endpoint": "10.0.0.100",
  "port": 3306,
  "tags": {
    "env": "production"
  }
}
```

## 多账号支持

ZenOps 支持配置多个腾讯云账号:

```yaml
providers:
  tencent:
    - name: production
      enabled: true
      secret_id: ${TENCENT_PROD_SECRET_ID}
      secret_key: ${TENCENT_PROD_SECRET_KEY}
      regions:
        - ap-guangzhou
        - ap-shanghai

    - name: development
      enabled: true
      secret_id: ${TENCENT_DEV_SECRET_ID}
      secret_key: ${TENCENT_DEV_SECRET_KEY}
      regions:
        - ap-guangzhou
```

使用指定账号查询:

```bash
./bin/zenops query tencent cvm list --account production
./bin/zenops query tencent cvm list --account development
```

## 权限要求

### 最小权限策略

腾讯云 RAM 用户至少需要以下只读权限:

```json
{
  "version": "2.0",
  "statement": [
    {
      "effect": "allow",
      "action": [
        "cvm:DescribeInstances",
        "cdb:DescribeDBInstances"
      ],
      "resource": "*"
    }
  ]
}
```

## 常见问题

### 1. 认证失败

**问题**: `AuthFailure` 错误

**解决方法**:
- 检查 Secret ID 和 Secret Key 是否正确
- 确认环境变量已正确设置
- 验证 RAM 用户权限

### 2. 区域不可用

**问题**: 某个区域无法查询

**解决方法**:
- 确认该区域在腾讯云账号中已开通
- 检查区域代码是否正确 (如 `ap-guangzhou`)

### 3. 查询超时

**问题**: 查询大量实例时超时

**解决方法**:
- 使用 `--region` 参数缩小查询范围
- 调整 `--page-size` 参数

## 性能优化

### 1. 区域过滤

只查询需要的区域:

```bash
./bin/zenops query tencent cvm list --region ap-guangzhou
```

### 2. 自动分页

利用自动分页功能获取完整数据:

```bash
# 默认启用,每页 100 条
./bin/zenops query tencent cvm list
```

### 3. 并发查询

多区域查询会串行执行,可以通过脚本并发查询:

```bash
#!/bin/bash
regions=("ap-guangzhou" "ap-shanghai" "ap-beijing")
for region in "${regions[@]}"; do
  ./bin/zenops query tencent cvm list --region "$region" --output json > "${region}.json" &
done
wait
```

## 示例脚本

### 统计各区域实例数量

```bash
#!/bin/bash

regions=("ap-guangzhou" "ap-shanghai" "ap-beijing")

echo "Region,Instance Count"
for region in "${regions[@]}"; do
  count=$(./bin/zenops query tencent cvm list --region "$region" --output json | jq '. | length')
  echo "$region,$count"
done
```

### 查找特定标签的实例

```bash
#!/bin/bash

# 查询所有实例并过滤 env=production 标签
./bin/zenops query tencent cvm list --output json | \
  jq '.[] | select(.tags.env == "production")'
```

## 相关链接

- [腾讯云 API 文档](https://cloud.tencent.com/document/api)
- [CVM API 参考](https://cloud.tencent.com/document/api/213)
- [CDB API 参考](https://cloud.tencent.com/document/api/236)
- [区域和可用区](https://cloud.tencent.com/document/product/213/6091)

## 技术支持

遇到问题请提交 Issue: https://github.com/eryajf/zenops/issues
