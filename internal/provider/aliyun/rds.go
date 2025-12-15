package aliyun

import (
	"context"
	"fmt"
	"time"

	"cnb.cool/zhiqiangwang/pkg/logx"
	rds "github.com/alibabacloud-go/rds-20140815/v14/client"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/eryajf/zenops/internal/model"
)

// ListRDSInstances 查询 RDS 实例列表
func (c *Client) ListRDSInstances(ctx context.Context, pageSize, pageNum int, filters map[string]string) ([]*model.Database, error) {
	rdsClient, err := c.GetRDSClient()
	if err != nil {
		return nil, err
	}

	if pageSize <= 0 {
		pageSize = 10
	}
	if pageNum <= 0 {
		pageNum = 1
	}

	request := &rds.DescribeDBInstancesRequest{
		RegionId:   tea.String(c.Region),
		PageSize:   tea.Int32(int32(pageSize)),
		PageNumber: tea.Int32(int32(pageNum)),
	}

	// 应用过滤条件
	if engine, ok := filters["engine"]; ok {
		request.Engine = tea.String(engine)
	}
	if instanceID, ok := filters["instance_id"]; ok {
		request.DBInstanceId = tea.String(instanceID)
	}

	logx.Debug("Querying Aliyun RDS instances, region %s, page_size %d, page_num %d",
		c.Region,
		pageSize,
		pageNum)

	response, err := rdsClient.DescribeDBInstances(request)
	if err != nil {
		return nil, fmt.Errorf("failed to describe RDS instances: %w", err)
	}

	if response.Body == nil || response.Body.Items == nil {
		return []*model.Database{}, nil
	}

	databases := make([]*model.Database, 0, len(response.Body.Items.DBInstance))
	for _, inst := range response.Body.Items.DBInstance {
		database := convertRDSToDatabase(inst, c.Region)
		databases = append(databases, database)
	}

	logx.Info("Successfully queried Aliyun RDS instances, count %d, region %s", len(databases), c.Region)

	return databases, nil
}

// GetRDSInstance 获取 RDS 实例详情
func (c *Client) GetRDSInstance(ctx context.Context, instanceID string) (*model.Database, error) {
	rdsClient, err := c.GetRDSClient()
	if err != nil {
		return nil, err
	}

	request := &rds.DescribeDBInstancesRequest{
		RegionId:     tea.String(c.Region),
		DBInstanceId: tea.String(instanceID),
	}

	logx.Debug("Querying Aliyun RDS instance, instance_id %s, region %s", instanceID, c.Region)

	response, err := rdsClient.DescribeDBInstances(request)
	if err != nil {
		return nil, fmt.Errorf("failed to describe RDS instance: %w", err)
	}

	if response.Body == nil || response.Body.Items == nil || len(response.Body.Items.DBInstance) == 0 {
		return nil, fmt.Errorf("RDS instance %s not found", instanceID)
	}

	database := convertRDSToDatabase(response.Body.Items.DBInstance[0], c.Region)

	logx.Info("Successfully queried Aliyun RDS instance, instance_id %s, region %s", instanceID, c.Region)

	return database, nil
}

// convertRDSToDatabase 将阿里云 RDS 实例转换为统一的数据库模型
func convertRDSToDatabase(inst *rds.DescribeDBInstancesResponseBodyItemsDBInstance, region string) *model.Database {
	database := &model.Database{
		ID:            tea.StringValue(inst.DBInstanceId),
		Name:          tea.StringValue(inst.DBInstanceDescription),
		Provider:      "aliyun",
		Region:        region,
		Engine:        tea.StringValue(inst.Engine),
		EngineVersion: tea.StringValue(inst.EngineVersion),
		Status:        tea.StringValue(inst.DBInstanceStatus),
		Tags:          make(map[string]string),
	}

	// 解析连接信息
	if inst.ConnectionString != nil {
		database.Endpoint = tea.StringValue(inst.ConnectionString)
	}
	// RDS 实例没有直接的 Port 字段,根据引擎类型设置默认端口
	switch tea.StringValue(inst.Engine) {
	case "MySQL":
		database.Port = 3306
	case "PostgreSQL":
		database.Port = 5432
	case "SQLServer":
		database.Port = 1433
	case "Redis":
		database.Port = 6379
	default:
		database.Port = 3306
	}

	// 解析创建时间
	if inst.CreateTime != nil {
		createTime := tea.StringValue(inst.CreateTime)
		// 阿里云时间格式可能是 "2006-01-02T15:04Z" 或 "2006-01-02T15:04:05Z"
		for _, layout := range []string{"2006-01-02T15:04:05Z", "2006-01-02T15:04Z"} {
			if t, err := time.Parse(layout, createTime); err == nil {
				database.CreatedAt = t
				break
			}
		}
	}

	// 如果名称为空,使用 ID 作为名称
	if database.Name == "" {
		database.Name = database.ID
	}

	// 生成控制台跳转URL
	database.ConsoleURL = fmt.Sprintf("https://rdsnext.console.aliyun.com/detail/%s/basicInfo",
		database.ID)

	return database
}
