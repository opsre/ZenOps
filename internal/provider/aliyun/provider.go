package aliyun

import (
	"context"
	"fmt"

	"cnb.cool/zhiqiangwang/pkg/logx"
	"github.com/eryajf/zenops/internal/model"
	"github.com/eryajf/zenops/internal/provider"
)

// AliyunProvider 阿里云提供商实现
type AliyunProvider struct {
	clients map[string]*Client // region -> client
	config  map[string]any
}

// NewProvider 创建阿里云提供商
func NewProvider() provider.Provider {
	return &AliyunProvider{
		clients: make(map[string]*Client),
	}
}

// GetName 返回提供商名称
func (p *AliyunProvider) GetName() string {
	return "aliyun"
}

// Initialize 初始化阿里云提供商
func (p *AliyunProvider) Initialize(config map[string]any) error {
	p.config = config

	accessKeyID, ok := config["access_key_id"].(string)
	if !ok || accessKeyID == "" {
		return fmt.Errorf("access_key_id is required")
	}

	accessKeySecret, ok := config["access_key_secret"].(string)
	if !ok || accessKeySecret == "" {
		return fmt.Errorf("access_key_secret is required")
	}

	// 获取区域列表
	regions, ok := config["regions"].([]any)
	if !ok || len(regions) == 0 {
		regions = []any{"cn-hangzhou"} // 默认区域
	}

	// 为每个区域创建客户端
	for _, r := range regions {
		region, ok := r.(string)
		if !ok {
			continue
		}

		client, err := NewClient(accessKeyID, accessKeySecret, region)
		if err != nil {
			logx.Warn("Failed to create client for region " + region + ": " + err.Error())
			continue
		}

		p.clients[region] = client
		logx.Info("Initialized Aliyun client for region " + region)
	}

	if len(p.clients) == 0 {
		return fmt.Errorf("no valid region clients created")
	}

	logx.Info("Aliyun provider initialized successfully, regions " + fmt.Sprintf("%d", len(p.clients)))

	return nil
}

// ListInstances 列出所有实例
func (p *AliyunProvider) ListInstances(ctx context.Context, opts *provider.QueryOptions) ([]*model.Instance, error) {
	if opts == nil {
		opts = &provider.QueryOptions{}
	}

	// 如果指定了区域,只查询该区域
	if opts.Region != "" {
		client, ok := p.clients[opts.Region]
		if !ok {
			return nil, fmt.Errorf("region %s not configured", opts.Region)
		}

		return client.ListECSInstances(ctx, opts.PageSize, opts.PageNum, opts.Filters)
	}

	// 否则查询所有区域
	allInstances := make([]*model.Instance, 0)
	for region, client := range p.clients {
		instances, err := client.ListECSInstances(ctx, opts.PageSize, opts.PageNum, opts.Filters)
		if err != nil {
			logx.Warn("Failed to query instances in region, region %s, error %v", region, err)
			continue
		}
		allInstances = append(allInstances, instances...)
	}

	return allInstances, nil
}

// GetInstance 获取单个实例详情
func (p *AliyunProvider) GetInstance(ctx context.Context, instanceID string) (*model.Instance, error) {
	// 尝试在所有区域查找实例
	for region, client := range p.clients {
		instance, err := client.GetECSInstance(ctx, instanceID)
		if err == nil {
			return instance, nil
		}
		logx.Debug("Instance not found in region, instance_id %s, region %s", instanceID, region)
	}

	return nil, fmt.Errorf("instance %s not found in any region", instanceID)
}

// ListDatabases 列出数据库实例
func (p *AliyunProvider) ListDatabases(ctx context.Context, opts *provider.QueryOptions) ([]*model.Database, error) {
	if opts == nil {
		opts = &provider.QueryOptions{}
	}

	// 如果指定了区域,只查询该区域
	if opts.Region != "" {
		client, ok := p.clients[opts.Region]
		if !ok {
			return nil, fmt.Errorf("region %s not configured", opts.Region)
		}

		return client.ListRDSInstances(ctx, opts.PageSize, opts.PageNum, opts.Filters)
	}

	// 否则查询所有区域
	allDatabases := make([]*model.Database, 0)
	for region, client := range p.clients {
		databases, err := client.ListRDSInstances(ctx, opts.PageSize, opts.PageNum, opts.Filters)
		if err != nil {
			logx.Warn("Failed to query databases in region %s: %v", region, err)
			continue
		}
		allDatabases = append(allDatabases, databases...)
	}

	return allDatabases, nil
}

// GetDatabase 获取数据库详情
func (p *AliyunProvider) GetDatabase(ctx context.Context, dbID string) (*model.Database, error) {
	// 尝试在所有区域查找数据库
	for region, client := range p.clients {
		database, err := client.GetRDSInstance(ctx, dbID)
		if err == nil {
			return database, nil
		}
		logx.Debug("Database not found in region, db_id %s, region %s", dbID, region)
	}

	return nil, fmt.Errorf("database %s not found in any region", dbID)
}

// HealthCheck 健康检查
func (p *AliyunProvider) HealthCheck(ctx context.Context) error {
	if len(p.clients) == 0 {
		return fmt.Errorf("no clients initialized")
	}

	// 检查至少一个区域的客户端可用
	for region, client := range p.clients {
		// 尝试查询一个实例列表(限制为1条)
		_, err := client.ListECSInstances(ctx, 1, 1, nil)
		if err == nil {
			logx.Debug("Health check passed for region " + region)
			return nil
		}
		logx.Debug("Health check failed for region " + region + ", error " + err.Error())
	}

	return fmt.Errorf("all regions failed health check")
}
