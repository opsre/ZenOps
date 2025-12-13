package jenkins

import (
	"context"
	"fmt"

	"cnb.cool/zhiqiangwang/pkg/logx"
	"github.com/eryajf/zenops/internal/model"
	"github.com/eryajf/zenops/internal/provider"
)

// JenkinsProvider Jenkins Provider
type JenkinsProvider struct {
	name   string
	client *Client
}

// NewJenkinsProvider 创建 Jenkins Provider
func NewJenkinsProvider() provider.CICDProvider {
	return &JenkinsProvider{
		name: "jenkins",
	}
}

// GetName 获取 Provider 名称
func (p *JenkinsProvider) GetName() string {
	return p.name
}

// Initialize 初始化 Provider
func (p *JenkinsProvider) Initialize(config map[string]any) error {
	// 解析配置
	url, ok := config["url"].(string)
	if !ok || url == "" {
		return fmt.Errorf("url is required")
	}

	username, ok := config["username"].(string)
	if !ok || username == "" {
		return fmt.Errorf("username is required")
	}

	token, ok := config["token"].(string)
	if !ok || token == "" {
		return fmt.Errorf("token is required")
	}

	// 创建客户端
	p.client = NewClient(url, username, token)

	logx.Info("Jenkins Provider initialized, url %s, username %s", url, username)

	return nil
}

// GetJobBuilds 实现 CICDProvider 接口
func (p *JenkinsProvider) GetJobBuilds(ctx context.Context, jobName string, limit int) ([]*model.Build, error) {
	opts := &provider.QueryOptions{
		PageSize: limit,
		PageNum:  1,
	}
	return p.ListBuilds(ctx, jobName, opts)
}

// HealthCheck 健康检查
func (p *JenkinsProvider) HealthCheck(ctx context.Context) error {
	if p.client == nil {
		return fmt.Errorf("client not initialized")
	}

	if err := p.client.Connect(ctx); err != nil {
		return err
	}

	jenkins := p.client.GetJenkins()
	if jenkins == nil {
		return fmt.Errorf("jenkins client is nil")
	}

	logx.Debug("Health check passed")
	return nil
}
