package imcp

import (
	"context"
	"fmt"
	"strings"

	"cnb.cool/zhiqiangwang/pkg/logx"
	"github.com/eryajf/zenops/internal/model"
	"github.com/eryajf/zenops/internal/provider"
	"github.com/mark3labs/mcp-go/mcp"
)

// ==================== Jenkins 处理函数 ====================

// handleListJenkinsJobs 处理列出所有 Jenkins Job 的请求
func (s *MCPServer) handleListJenkinsJobs(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	p, err := s.getJenkinsProvider()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var allJobs []*model.Job
	pageNum := 1
	pageSize := 100

	for {
		opts := &provider.QueryOptions{
			PageSize: pageSize,
			PageNum:  pageNum,
		}

		jobs, err := p.ListJobs(ctx, opts)
		if err != nil {
			logx.Error("Failed to list jobs: %v", err)
			break
		}

		allJobs = append(allJobs, jobs...)

		if len(jobs) < pageSize {
			break
		}
		pageNum++
	}

	result := formatJobs(allJobs)
	return mcp.NewToolResultText(result), nil
}

// handleGetJenkinsJob 处理获取 Jenkins Job 详情的请求
func (s *MCPServer) handleGetJenkinsJob(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]any)
	if !ok {
		return mcp.NewToolResultError("invalid arguments type"), nil
	}

	jobName, ok := args["job_name"].(string)
	if !ok || jobName == "" {
		return mcp.NewToolResultError("job_name parameter is required"), nil
	}

	p, err := s.getJenkinsProvider()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	job, err := p.GetJob(ctx, jobName)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("未找到 Job '%s': %v", jobName, err)), nil
	}

	result := formatJobs([]*model.Job{job})
	return mcp.NewToolResultText(result), nil
}

// handleListJenkinsBuilds 处理列出 Jenkins Build 历史的请求
func (s *MCPServer) handleListJenkinsBuilds(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]any)
	if !ok {
		return mcp.NewToolResultError("invalid arguments type"), nil
	}

	jobName, ok := args["job_name"].(string)
	if !ok || jobName == "" {
		return mcp.NewToolResultError("job_name parameter is required"), nil
	}

	// 获取可选的 limit 参数,默认为 10
	limit := 10
	if limitArg, ok := args["limit"].(float64); ok {
		limit = int(limitArg)
	}

	p, err := s.getJenkinsProvider()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	builds, err := p.GetJobBuilds(ctx, jobName, limit)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("获取 Job '%s' 的构建历史失败: %v", jobName, err)), nil
	}

	if len(builds) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("Job '%s' 没有构建历史", jobName)), nil
	}

	result := formatBuilds(builds, jobName)
	return mcp.NewToolResultText(result), nil
}

// ==================== 格式化函数 ====================

// formatJobs 格式化 Jenkins Job 列表为文本输出
func formatJobs(jobs []*model.Job) string {
	if len(jobs) == 0 {
		return "未找到任何 Jenkins Job"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("找到 %d 个 Jenkins Job:\n\n", len(jobs)))

	for i, job := range jobs {
		sb.WriteString(fmt.Sprintf("Job %d:\n", i+1))
		sb.WriteString(fmt.Sprintf("  名称: %s\n", job.Name))
		if job.DisplayName != "" && job.DisplayName != job.Name {
			sb.WriteString(fmt.Sprintf("  显示名称: %s\n", job.DisplayName))
		}
		if job.Description != "" {
			sb.WriteString(fmt.Sprintf("  描述: %s\n", job.Description))
		}
		sb.WriteString(fmt.Sprintf("  URL: %s\n", job.URL))

		buildable := "是"
		if !job.Buildable {
			buildable = "否"
		}
		sb.WriteString(fmt.Sprintf("  可构建: %s\n", buildable))

		if job.LastBuild != nil {
			sb.WriteString(fmt.Sprintf("  最后构建: #%d\n", job.LastBuild.Number))
			if job.LastBuild.URL != "" {
				sb.WriteString(fmt.Sprintf("  最后构建 URL: %s\n", job.LastBuild.URL))
			}
		} else {
			sb.WriteString("  最后构建: 无\n")
		}

		sb.WriteString("\n")
	}

	return sb.String()
}

// formatBuilds 格式化 Jenkins Build 列表为文本输出
func formatBuilds(builds []*model.Build, jobName string) string {
	if len(builds) == 0 {
		return fmt.Sprintf("Job '%s' 没有构建历史", jobName)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Job '%s' 的构建历史 (共 %d 个构建):\n\n", jobName, len(builds)))

	for i, build := range builds {
		sb.WriteString(fmt.Sprintf("Build %d:\n", i+1))
		sb.WriteString(fmt.Sprintf("  构建号: #%d\n", build.Number))
		sb.WriteString(fmt.Sprintf("  状态: %s\n", build.Status))
		if build.Result != "" {
			sb.WriteString(fmt.Sprintf("  结果: %s\n", build.Result))
		}
		if !build.Timestamp.IsZero() {
			sb.WriteString(fmt.Sprintf("  时间: %s\n", build.Timestamp.Format("2006-01-02 15:04:05")))
		}
		if build.Duration > 0 {
			sb.WriteString(fmt.Sprintf("  时长: %dms\n", build.Duration))
		}
		if build.URL != "" {
			sb.WriteString(fmt.Sprintf("  URL: %s\n", build.URL))
		}

		sb.WriteString("\n")
	}

	return sb.String()
}
