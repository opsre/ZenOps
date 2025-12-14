package jenkins

import (
	"context"
	"fmt"
	"strings"

	"cnb.cool/zhiqiangwang/pkg/logx"
	"github.com/bndr/gojenkins"
	"github.com/eryajf/zenops/internal/model"
	"github.com/eryajf/zenops/internal/provider"
)

// ListJobs 列出所有 Job
func (p *JenkinsProvider) ListJobs(ctx context.Context, opts *provider.QueryOptions) ([]*model.Job, error) {
	if err := p.client.Connect(ctx); err != nil {
		return nil, err
	}

	jenkins := p.client.GetJenkins()

	// 获取所有 Job
	jobs, err := jenkins.GetAllJobs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all jobs: %w", err)
	}

	logx.Debug("Fetched Jenkins jobs, count %d", len(jobs))

	var result []*model.Job
	for _, job := range jobs {
		// 跳过文件夹类型
		if job.Raw.Class == "com.cloudbees.hudson.plugins.folder.Folder" {
			continue
		}

		modelJob := convertJobToModel(job)
		result = append(result, modelJob)
	}

	// 应用分页
	if opts.PageSize > 0 && opts.PageNum > 0 {
		start := (opts.PageNum - 1) * opts.PageSize
		end := start + opts.PageSize

		if start >= len(result) {
			return []*model.Job{}, nil
		}
		if end > len(result) {
			end = len(result)
		}

		result = result[start:end]
	}

	return result, nil
}

// GetJob 获取 Job 详情
func (p *JenkinsProvider) GetJob(ctx context.Context, jobName string) (*model.Job, error) {
	if err := p.client.Connect(ctx); err != nil {
		return nil, err
	}

	jenkins := p.client.GetJenkins()

	// 支持文件夹路径,如 "folder/subfolder/job"
	job, err := jenkins.GetJob(ctx, jobName)
	if err != nil {
		return nil, fmt.Errorf("failed to get job '%s': %w", jobName, err)
	}

	logx.Info("Fetched Jenkins job, name %s", jobName)

	return convertJobToModel(job), nil
}

// SearchJobs 搜索 Job
func (p *JenkinsProvider) SearchJobs(ctx context.Context, keyword string) ([]*model.Job, error) {
	if err := p.client.Connect(ctx); err != nil {
		return nil, err
	}

	jenkins := p.client.GetJenkins()

	// 获取所有 Job
	jobs, err := jenkins.GetAllJobs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all jobs: %w", err)
	}

	var result []*model.Job
	keyword = strings.ToLower(keyword)

	for _, job := range jobs {
		// 跳过文件夹类型
		if job.Raw.Class == "com.cloudbees.hudson.plugins.folder.Folder" {
			continue
		}

		// 按名称或描述搜索
		if strings.Contains(strings.ToLower(job.GetName()), keyword) ||
			strings.Contains(strings.ToLower(job.GetDescription()), keyword) {
			modelJob := convertJobToModel(job)
			result = append(result, modelJob)
		}
	}

	logx.Info("Search completed, keyword %s, found %d", keyword, len(result))

	return result, nil
}

// convertJobToModel 将 Jenkins Job 转换为统一的 Job 模型
func convertJobToModel(job *gojenkins.Job) *model.Job {
	modelJob := &model.Job{
		Name:        job.GetName(),
		DisplayName: job.GetName(),
		Description: job.GetDescription(),
		URL:         job.Raw.URL,
		Buildable:   true, // 默认为 true
	}

	// 最后构建信息
	if job.Raw.LastBuild.Number > 0 {
		modelJob.LastBuild = &model.Build{
			Number: int(job.Raw.LastBuild.Number),
			URL:    job.Raw.LastBuild.URL,
		}
	}

	return modelJob
}

// These functions are intentionally kept for future use
// extractJobType 从 Java 类名提取任务类型
// func extractJobType(class string) string {
// 	// 例如: "hudson.model.FreeStyleProject" -> "FreeStyle"
// 	//      "org.jenkinsci.plugins.workflow.job.WorkflowJob" -> "Pipeline"
// 	if strings.Contains(class, "FreeStyleProject") {
// 		return "FreeStyle"
// 	}
// 	if strings.Contains(class, "WorkflowJob") {
// 		return "Pipeline"
// 	}
// 	if strings.Contains(class, "MatrixProject") {
// 		return "Matrix"
// 	}
// 	if strings.Contains(class, "MavenModuleSet") {
// 		return "Maven"
// 	}
//
// 	// 提取类名最后一部分
// 	parts := strings.Split(class, ".")
// 	if len(parts) > 0 {
// 		return parts[len(parts)-1]
// 	}
//
// 	return "Unknown"
// }

// convertJobColor 转换 Jenkins 颜色状态为友好的状态名
// func convertJobColor(color string) string {
// 	// 移除 _anime 后缀 (表示正在构建)
// 	color = strings.TrimSuffix(color, "_anime")
//
// 	switch color {
// 	case "blue":
// 		return "Success"
// 	case "red":
// 		return "Failed"
// 	case "yellow":
// 		return "Unstable"
// 	case "grey":
// 		return "NotBuilt"
// 	case "disabled":
// 		return "Disabled"
// 	case "aborted":
// 		return "Aborted"
// 	default:
// 		return color
// 	}
// }
