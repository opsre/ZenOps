package tencent

import (
	"context"
	"fmt"

	"cnb.cool/zhiqiangwang/pkg/logx"
	"github.com/eryajf/zenops/internal/model"
	"github.com/eryajf/zenops/internal/provider"
)

// TencentProvider 腾讯云 Provider
type TencentProvider struct {
	name      string
	secretID  string
	secretKey string
	regions   []string
	clients   map[string]*Client // region -> client
}

// NewTencentProvider 创建腾讯云 Provider
func NewTencentProvider() provider.Provider {
	return &TencentProvider{
		name:    "tencent",
		clients: make(map[string]*Client),
	}
}

// GetName 获取 Provider 名称
func (p *TencentProvider) GetName() string {
	return p.name
}

// Initialize 初始化 Provider
func (p *TencentProvider) Initialize(config map[string]any) error {
	// 解析配置
	secretID, ok := config["secret_id"].(string)
	if !ok {
		return fmt.Errorf("secret_id is required")
	}

	secretKey, ok := config["secret_key"].(string)
	if !ok {
		return fmt.Errorf("secret_key is required")
	}

	regions, ok := config["regions"].([]any)
	if !ok || len(regions) == 0 {
		return fmt.Errorf("regions are required")
	}

	p.secretID = secretID
	p.secretKey = secretKey

	// 初始化每个区域的客户端
	for _, r := range regions {
		region, ok := r.(string)
		if !ok {
			continue
		}

		p.regions = append(p.regions, region)
		p.clients[region] = NewClient(secretID, secretKey, region)

		logx.Debug("Initialized Tencent client for region " + region)
	}

	logx.Info("Tencent Provider initialized, regions count %d", len(p.regions))

	return nil
}

// ListInstances 列出实例
func (p *TencentProvider) ListInstances(ctx context.Context, opts *provider.QueryOptions) ([]*model.Instance, error) {
	return p.ListCVMInstances(ctx, opts)
}

// GetInstance 获取实例详情
func (p *TencentProvider) GetInstance(ctx context.Context, instanceID string) (*model.Instance, error) {
	return p.GetCVMInstance(ctx, instanceID)
}

// ListDatabases 列出数据库
func (p *TencentProvider) ListDatabases(ctx context.Context, opts *provider.QueryOptions) ([]*model.Database, error) {
	return p.ListCDBInstances(ctx, opts)
}

// GetDatabase 获取数据库详情
func (p *TencentProvider) GetDatabase(ctx context.Context, dbID string) (*model.Database, error) {
	return p.GetCDBInstance(ctx, dbID)
}

// HealthCheck 健康检查
func (p *TencentProvider) HealthCheck(ctx context.Context) error {
	if len(p.clients) == 0 {
		return fmt.Errorf("no clients initialized")
	}

	// 检查至少一个区域可用
	for region, client := range p.clients {
		_, err := client.GetCVMClient()
		if err == nil {
			logx.Debug("Health check passed, region %s", region)
			return nil
		}
	}

	return fmt.Errorf("all regions failed health check")
}
