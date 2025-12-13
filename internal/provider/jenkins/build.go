package jenkins

import (
	"context"
	"fmt"
	"time"

	"cnb.cool/zhiqiangwang/pkg/logx"
	"github.com/bndr/gojenkins"
	"github.com/eryajf/zenops/internal/model"
	"github.com/eryajf/zenops/internal/provider"
)

// ListBuilds 列出 Job 的构建历史
func (p *JenkinsProvider) ListBuilds(ctx context.Context, jobName string, opts *provider.QueryOptions) ([]*model.Build, error) {
	if err := p.client.Connect(ctx); err != nil {
		return nil, err
	}

	jenkins := p.client.GetJenkins()

	// 获取 Job
	job, err := jenkins.GetJob(ctx, jobName)
	if err != nil {
		return nil, fmt.Errorf("failed to get job '%s': %w", jobName, err)
	}

	// 获取所有构建 ID
	buildIds, err := job.GetAllBuildIds(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get build ids: %w", err)
	}

	logx.Debug("Fetched build IDs, job %s, count %d", jobName, len(buildIds))

	var result []*model.Build

	// 默认只获取最近的构建
	limit := 10
	if opts.PageSize > 0 {
		limit = opts.PageSize
	}

	// 应用分页
	start := 0
	if opts.PageNum > 1 {
		start = (opts.PageNum - 1) * limit
	}

	count := 0
	for i := start; i < len(buildIds) && count < limit; i++ {
		buildId := buildIds[i]

		build, err := job.GetBuild(ctx, buildId.Number)
		if err != nil {
			logx.Warn("Failed to get build, job %s, build %d, error %v", jobName, buildId.Number, err)
			continue
		}

		modelBuild := convertBuildToModel(build, jobName)
		result = append(result, modelBuild)
		count++
	}

	return result, nil
}

// GetBuild 获取指定构建详情
func (p *JenkinsProvider) GetBuild(ctx context.Context, jobName string, buildNumber int) (*model.Build, error) {
	if err := p.client.Connect(ctx); err != nil {
		return nil, err
	}

	jenkins := p.client.GetJenkins()

	// 获取 Job
	job, err := jenkins.GetJob(ctx, jobName)
	if err != nil {
		return nil, fmt.Errorf("failed to get job '%s': %w", jobName, err)
	}

	// 获取构建
	build, err := job.GetBuild(ctx, int64(buildNumber))
	if err != nil {
		return nil, fmt.Errorf("failed to get build #%d: %w", buildNumber, err)
	}

	logx.Info("Fetched build, job %s, build %d", jobName, buildNumber)

	return convertBuildToModel(build, jobName), nil
}

// GetLastBuild 获取最后一次构建
func (p *JenkinsProvider) GetLastBuild(ctx context.Context, jobName string) (*model.Build, error) {
	if err := p.client.Connect(ctx); err != nil {
		return nil, err
	}

	jenkins := p.client.GetJenkins()

	// 获取 Job
	job, err := jenkins.GetJob(ctx, jobName)
	if err != nil {
		return nil, fmt.Errorf("failed to get job '%s': %w", jobName, err)
	}

	// 获取最后一次构建
	build, err := job.GetLastBuild(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get last build: %w", err)
	}

	if build == nil {
		return nil, fmt.Errorf("no builds found for job '%s'", jobName)
	}

	return convertBuildToModel(build, jobName), nil
}

// convertBuildToModel 将 Jenkins Build 转换为统一的 Build 模型
func convertBuildToModel(build *gojenkins.Build, jobName string) *model.Build {
	modelBuild := &model.Build{
		Number:   int(build.Raw.Number),
		URL:      build.Raw.URL,
		Duration: int64(build.Raw.Duration), // 毫秒
	}

	// 状态和结果
	if build.Raw.Result != "" {
		modelBuild.Status = build.Raw.Result
		modelBuild.Result = build.Raw.Result
	} else if build.Raw.Building {
		modelBuild.Status = "BUILDING"
	} else {
		modelBuild.Status = "UNKNOWN"
	}

	// 时间戳
	if build.Raw.Timestamp > 0 {
		modelBuild.Timestamp = time.Unix(build.Raw.Timestamp/1000, 0)
	}

	return modelBuild
}
