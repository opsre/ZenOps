package provider

import (
	"context"

	"github.com/eryajf/zenops/internal/model"
)

// Provider 定义了云服务提供商的统一接口
type Provider interface {
	// GetName 返回提供商名称 (如: aliyun, tencent, aws)
	GetName() string

	// Initialize 初始化提供商客户端
	Initialize(config map[string]any) error

	// ListInstances 列出所有实例 (ECS/CVM/EC2)
	ListInstances(ctx context.Context, opts *QueryOptions) ([]*model.Instance, error)

	// GetInstance 获取单个实例详情
	GetInstance(ctx context.Context, instanceID string) (*model.Instance, error)

	// ListDatabases 列出数据库实例
	ListDatabases(ctx context.Context, opts *QueryOptions) ([]*model.Database, error)

	// GetDatabase 获取数据库详情
	GetDatabase(ctx context.Context, dbID string) (*model.Database, error)

	// HealthCheck 健康检查
	HealthCheck(ctx context.Context) error
}

// CICDProvider 定义 CI/CD 工具的统一接口
type CICDProvider interface {
	// GetName 返回提供商名称 (如: jenkins, gitlab-ci)
	GetName() string

	// Initialize 初始化客户端
	Initialize(config map[string]any) error

	// ListJobs 列出所有任务
	ListJobs(ctx context.Context, opts *QueryOptions) ([]*model.Job, error)

	// GetJob 获取任务详情
	GetJob(ctx context.Context, jobName string) (*model.Job, error)

	// GetJobBuilds 获取任务的构建历史
	GetJobBuilds(ctx context.Context, jobName string, limit int) ([]*model.Build, error)

	// HealthCheck 健康检查
	HealthCheck(ctx context.Context) error
}

// QueryOptions 查询选项
type QueryOptions struct {
	Region   string            // 区域
	PageSize int               // 分页大小
	PageNum  int               // 页码
	Filters  map[string]string // 过滤条件
	Tags     map[string]string // 标签过滤
}
