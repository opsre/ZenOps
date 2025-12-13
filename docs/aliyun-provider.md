# 阿里云 Provider 使用指南

本文档介绍如何使用 ZenOps 查询阿里云资源。

## 配置

### 1. 获取阿里云 AccessKey

登录阿里云控制台,创建 AccessKey:
- 访问 [RAM 访问控制](https://ram.console.aliyun.com/users)
- 创建 RAM 用户并授予相应权限 (ECS、RDS 读权限)
- 创建 AccessKey,保存 AccessKeyId 和 AccessKeySecret

### 2. 配置环境变量

```bash
# 设置阿里云凭证
export ALIYUN_ACCESS_KEY_ID="your-access-key-id"
export ALIYUN_ACCESS_KEY_SECRET="your-access-key-secret"
```

### 3. 配置文件

编辑 `configs/config.yaml`:

```yaml
providers:
  aliyun:
    enabled: true
    access_key_id: ${ALIYUN_ACCESS_KEY_ID}
    access_key_secret: ${ALIYUN_ACCESS_KEY_SECRET}
    regions:
      - cn-hangzhou
      - cn-shanghai
      - cn-beijing
```

## 使用方法

### 查询 ECS 实例

#### 列出所有 ECS 实例

```bash
# 查询所有区域的 ECS 实例
./bin/zenops query aliyun ecs list

# 指定区域查询
./bin/zenops query aliyun ecs list --region cn-hangzhou

# 设置分页参数
./bin/zenops query aliyun ecs list --page-size 20 --page-num 1

# JSON 格式输出
./bin/zenops query aliyun ecs list --output json
```

示例输出 (表格格式):

```
ID                     NAME           REGION        STATUS    INSTANCE_TYPE   PRIVATE_IP      PUBLIC_IP
i-bp1xxxxx             web-server-01  cn-hangzhou   Running   ecs.c6.large    172.16.1.10     47.xx.xx.xx
i-bp2xxxxx             db-server-01   cn-hangzhou   Running   ecs.g6.xlarge   172.16.1.20
i-bp3xxxxx             app-server-01  cn-shanghai   Running   ecs.c6.xlarge   172.17.1.10     47.yy.yy.yy
```

#### 获取 ECS 实例详情

```bash
# 获取指定实例的详细信息
./bin/zenops query aliyun ecs get i-bp1xxxxx
```

示例输出:

```json
{
  "id": "i-bp1xxxxx",
  "name": "web-server-01",
  "provider": "aliyun",
  "region": "cn-hangzhou",
  "zone": "cn-hangzhou-h",
  "instance_type": "ecs.c6.large",
  "status": "Running",
  "private_ip": ["172.16.1.10"],
  "public_ip": ["47.xx.xx.xx"],
  "cpu": 2,
  "memory": 4096,
  "os_type": "linux",
  "os_name": "CentOS 7.9 64位",
  "created_at": "2024-01-01T00:00:00Z",
  "tags": {
    "env": "production",
    "project": "web"
  },
  "metadata": {
    "description": "Web 服务器",
    "image_id": "centos_7_9_x64",
    "instance_charge_type": "PostPaid",
    "internet_charge_type": "PayByTraffic",
    "internet_max_bandwidth_out": 100
  }
}
```

### 查询 RDS 实例

#### 列出所有 RDS 实例

```bash
# 查询所有区域的 RDS 实例
./bin/zenops query aliyun rds list

# 指定区域查询
./bin/zenops query aliyun rds list --region cn-hangzhou

# JSON 格式输出
./bin/zenops query aliyun rds list --output json
```

示例输出 (表格格式):

```
ID                  NAME              REGION        ENGINE      VERSION  STATUS    ENDPOINT
rm-bp1xxxxx         mysql-prod-01     cn-hangzhou   MySQL       8.0      Running   rm-bp1xxxxx.mysql.rds.aliyuncs.com
rm-bp2xxxxx         postgresql-dev    cn-shanghai   PostgreSQL  13.0     Running   rm-bp2xxxxx.pg.rds.aliyuncs.com
```

#### 获取 RDS 实例详情

```bash
# 获取指定数据库实例的详细信息
./bin/zenops query aliyun rds get rm-bp1xxxxx
```

示例输出:

```json
{
  "id": "rm-bp1xxxxx",
  "name": "mysql-prod-01",
  "provider": "aliyun",
  "region": "cn-hangzhou",
  "engine": "MySQL",
  "engine_version": "8.0",
  "status": "Running",
  "endpoint": "rm-bp1xxxxx.mysql.rds.aliyuncs.com",
  "port": 3306,
  "created_at": "2024-01-01T00:00:00Z",
  "tags": {}
}
```

## 高级用法

### 组合使用配置和命令行参数

```bash
# 使用自定义配置文件
./bin/zenops query aliyun ecs list --config /path/to/custom-config.yaml

# 设置日志级别为 debug
./bin/zenops query aliyun ecs list --log-level debug

# 组合多个参数
./bin/zenops query aliyun ecs list \
  --region cn-hangzhou \
  --page-size 50 \
  --output json \
  --log-level debug
```

### 在脚本中使用

```bash
#!/bin/bash

# 查询所有 ECS 实例并保存为 JSON
./bin/zenops query aliyun ecs list --output json > instances.json

# 统计实例数量
instance_count=$(cat instances.json | jq 'length')
echo "Total instances: $instance_count"

# 筛选运行中的实例
cat instances.json | jq '.[] | select(.status == "Running") | {id, name, region}'
```

## 支持的资源类型

当前阿里云 Provider 支持以下资源:

- ✅ **ECS 实例**: 弹性计算服务器
- ✅ **RDS 实例**: 云数据库 (MySQL, PostgreSQL, SQL Server, Redis)
- 🚧 **SLB 负载均衡**: 计划中
- 🚧 **VPC 网络**: 计划中
- 🚧 **OSS 对象存储**: 计划中

## 权限要求

阿里云 RAM 用户需要以下权限:

### ECS 查询权限

```json
{
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ecs:DescribeInstances",
        "ecs:DescribeInstanceAttribute"
      ],
      "Resource": "*"
    }
  ],
  "Version": "1"
}
```

### RDS 查询权限

```json
{
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "rds:DescribeDBInstances",
        "rds:DescribeDBInstanceAttribute"
      ],
      "Resource": "*"
    }
  ],
  "Version": "1"
}
```

### 只读权限推荐

为安全起见,建议授予只读权限:

- `AliyunECSReadOnlyAccess` - ECS 只读权限
- `AliyunRDSReadOnlyAccess` - RDS 只读权限

## 常见问题

### Q: 如何查询多个区域的资源?

A: 在配置文件中配置多个区域,查询时不指定 `--region` 参数,会自动聚合所有区域的资源:

```yaml
providers:
  aliyun:
    regions:
      - cn-hangzhou
      - cn-shanghai
      - cn-beijing
      - cn-shenzhen
```

### Q: 查询速度慢怎么办?

A:
1. 指定具体区域而不是查询所有区域
2. 使用分页参数减少单次查询数量
3. 启用缓存功能 (在配置文件中设置 `cache.enabled: true`)

### Q: 如何处理 API 限流?

A: 阿里云 API 有速率限制,如遇到限流:
1. 增加请求间隔
2. 减少并发查询数
3. 使用 RAM 角色而非子账号 (更高的限额)

### Q: 支持跨账号查询吗?

A: 支持。配置多个 Provider 实例即可:

```yaml
providers:
  aliyun_account1:
    enabled: true
    access_key_id: ${ACCOUNT1_ACCESS_KEY_ID}
    access_key_secret: ${ACCOUNT1_ACCESS_KEY_SECRET}
    regions: [cn-hangzhou]

  aliyun_account2:
    enabled: true
    access_key_id: ${ACCOUNT2_ACCESS_KEY_ID}
    access_key_secret: ${ACCOUNT2_ACCESS_KEY_SECRET}
    regions: [cn-shanghai]
```

## 故障排查

### 认证失败

```
Error: failed to list instances: InvalidAccessKeyId.NotFound
```

**解决方法**: 检查 AccessKey 是否正确,环境变量是否已设置

### 权限不足

```
Error: failed to list instances: Forbidden.RAM
```

**解决方法**: 确认 RAM 用户拥有相应的读取权限

### 区域不存在

```
Error: region cn-invalid not configured
```

**解决方法**: 检查配置文件中是否配置了该区域

## 下一步

- 查看 [快速入门指南](getting-started.md)
- 了解 [Provider 开发指南](../DESIGN.md)
- 集成到 [MCP 协议](mcp.md)
