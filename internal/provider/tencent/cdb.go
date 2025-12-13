package tencent

import (
	"context"
	"fmt"

	"cnb.cool/zhiqiangwang/pkg/logx"
	"github.com/eryajf/zenops/internal/model"
	"github.com/eryajf/zenops/internal/provider"
	cdb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cdb/v20170320"
)

// ListCDBInstances 列出 CDB 实例
func (p *TencentProvider) ListCDBInstances(ctx context.Context, opts *provider.QueryOptions) ([]*model.Database, error) {
	var allDatabases []*model.Database

	// 如果指定了区域,只查询该区域
	if opts.Region != "" {
		client, exists := p.clients[opts.Region]
		if !exists {
			return nil, fmt.Errorf("region %s not configured", opts.Region)
		}

		databases, err := p.listCDBInstancesInRegion(ctx, client, opts)
		if err != nil {
			return nil, err
		}
		allDatabases = append(allDatabases, databases...)
	} else {
		// 查询所有区域
		for region, client := range p.clients {
			logx.Debug("Querying CDB instances in region %s", region)

			databases, err := p.listCDBInstancesInRegion(ctx, client, opts)
			if err != nil {
				logx.Warn("Failed to query region %s, error %v", region, err)
				continue
			}

			allDatabases = append(allDatabases, databases...)
		}
	}

	return allDatabases, nil
}

// listCDBInstancesInRegion 查询单个区域的 CDB 实例
func (p *TencentProvider) listCDBInstancesInRegion(ctx context.Context, client *Client, opts *provider.QueryOptions) ([]*model.Database, error) {
	cdbClient, err := client.GetCDBClient()
	if err != nil {
		return nil, err
	}

	request := cdb.NewDescribeDBInstancesRequest()

	// 设置分页参数
	if opts.PageSize > 0 {
		limit := uint64(opts.PageSize)
		request.Limit = &limit
	}

	if opts.PageNum > 1 {
		offset := uint64((opts.PageNum - 1) * opts.PageSize)
		request.Offset = &offset
	}

	response, err := cdbClient.DescribeDBInstances(request)
	if err != nil {
		return nil, fmt.Errorf("failed to describe database instances: %w", err)
	}

	var databases []*model.Database
	for _, inst := range response.Response.Items {
		databases = append(databases, convertCDBToDatabase(inst, client.Region))
	}

	return databases, nil
}

// GetCDBInstance 获取 CDB 实例详情
func (p *TencentProvider) GetCDBInstance(ctx context.Context, instanceID string) (*model.Database, error) {
	// 遍历所有区域查找实例
	for region, client := range p.clients {
		logx.Debug("Searching database in region %s, instance_id %s", region, instanceID)

		cdbClient, err := client.GetCDBClient()
		if err != nil {
			logx.Warn("Failed to get CDB client, region %s, error %v", region, err)
			continue
		}

		request := cdb.NewDescribeDBInstancesRequest()
		request.InstanceIds = []*string{&instanceID}

		response, err := cdbClient.DescribeDBInstances(request)
		if err != nil {
			logx.Warn("Failed to describe database, instance_id %s, region %s, error %v", instanceID, region, err)
			continue
		}

		if len(response.Response.Items) > 0 {
			return convertCDBToDatabase(response.Response.Items[0], region), nil
		}
	}

	return nil, fmt.Errorf("database instance %s not found in any region", instanceID)
}

// convertCDBToDatabase 将腾讯云 CDB 实例转换为统一的 Database 模型
func convertCDBToDatabase(inst *cdb.InstanceInfo, region string) *model.Database {
	database := &model.Database{
		Provider: "tencent",
		Region:   region,
		Tags:     make(map[string]string),
	}

	// 基本信息
	if inst.InstanceId != nil {
		database.ID = *inst.InstanceId
	}
	if inst.InstanceName != nil {
		database.Name = *inst.InstanceName
	}
	if inst.EngineVersion != nil {
		database.EngineVersion = *inst.EngineVersion
	}
	if inst.Status != nil {
		database.Status = convertCDBStatus(*inst.Status)
	}

	// 数据库引擎 (腾讯云 CDB 主要是 MySQL)
	database.Engine = "MySQL"

	// 端口
	if inst.Vport != nil {
		database.Port = int(*inst.Vport)
	} else {
		// MySQL 默认端口
		database.Port = 3306
	}

	// 内网地址
	if inst.Vip != nil {
		database.Endpoint = *inst.Vip
	}

	// 如果名称为空,使用 ID 作为名称
	if database.Name == "" {
		database.Name = database.ID
	}

	// 生成控制台跳转URL
	database.ConsoleURL = fmt.Sprintf("https://console.cloud.tencent.com/cdb/%s", database.ID)

	return database
}

// convertCDBStatus 转换 CDB 实例状态
func convertCDBStatus(status int64) string {
	// 腾讯云 CDB 状态码
	// 0-创建中 1-运行中 4-隔离中 5-隔离中
	switch status {
	case 0:
		return "Creating"
	case 1:
		return "Running"
	case 4, 5:
		return "Isolated"
	default:
		return fmt.Sprintf("Unknown(%d)", status)
	}
}
