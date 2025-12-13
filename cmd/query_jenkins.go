package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"cnb.cool/zhiqiangwang/pkg/logx"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/eryajf/zenops/internal/provider"
	"github.com/spf13/cobra"
)

var (
	jenkinsOutputType string
	jenkinsPageSize   int
	jenkinsPageNum    int
)

// jenkinsCmd Jenkins 查询命令组
var jenkinsCmd = &cobra.Command{
	Use:   "jenkins",
	Short: "查询 Jenkins 资源",
	Long:  `查询 Jenkins 的 Job、Build 等信息。`,
}

// jenkinsJobCmd Job 命令组
var jenkinsJobCmd = &cobra.Command{
	Use:   "job",
	Short: "查询 Jenkins Job",
	Long:  `查询 Jenkins Job 信息。`,
}

// jenkinsJobListCmd 列出所有 Job
var jenkinsJobListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出所有 Job",
	Long:  `列出 Jenkins 中的所有 Job。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		// 获取 Jenkins Provider
		p, err := provider.GetCICDProvider("jenkins")
		if err != nil {
			return fmt.Errorf("failed to get jenkins provider: %w", err)
		}

		// 初始化 Provider
		providerConfig := map[string]any{
			"url":      cfg.CICD.Jenkins.URL,
			"username": cfg.CICD.Jenkins.Username,
			"token":    cfg.CICD.Jenkins.Token,
		}

		if err := p.Initialize(providerConfig); err != nil {
			return fmt.Errorf("failed to initialize jenkins provider: %w", err)
		}

		// 查询 Jobs
		opts := &provider.QueryOptions{
			PageSize: jenkinsPageSize,
			PageNum:  jenkinsPageNum,
		}

		jobs, err := p.ListJobs(ctx, opts)
		if err != nil {
			return fmt.Errorf("failed to list jobs: %w", err)
		}

		// 输出结果
		if jenkinsOutputType == "json" {
			data, _ := json.MarshalIndent(jobs, "", "  ")
			fmt.Println(string(data))
		} else {
			// 使用 lipgloss/table 表格输出
			rows := [][]string{}

			for _, job := range jobs {
				buildable := "✓"
				if !job.Buildable {
					buildable = "✗"
				}

				lastBuild := "-"
				if job.LastBuild != nil {
					lastBuild = fmt.Sprintf("#%d", job.LastBuild.Number)
				}

				rows = append(rows, []string{
					job.Name,
					job.DisplayName,
					lastBuild,
					buildable,
				})
			}

			t := table.New().
				Border(lipgloss.NormalBorder()).
				BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("99"))).
				Headers("Name", "Display Name", "Last Build", "Buildable").
				Rows(rows...)

			fmt.Println(t)
			fmt.Println()
			logx.Info("Query completed, count %d", len(jobs))
		}

		return nil
	},
}

// jenkinsJobGetCmd 获取 Job 详情
var jenkinsJobGetCmd = &cobra.Command{
	Use:   "get <job-name>",
	Short: "获取 Job 详情",
	Long:  `获取指定 Job 的详细信息。支持文件夹路径,如 "folder/job"。`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		jobName := args[0]
		ctx := context.Background()

		// 获取 Jenkins Provider
		p, err := provider.GetCICDProvider("jenkins")
		if err != nil {
			return fmt.Errorf("failed to get jenkins provider: %w", err)
		}

		// 初始化 Provider
		providerConfig := map[string]any{
			"url":      cfg.CICD.Jenkins.URL,
			"username": cfg.CICD.Jenkins.Username,
			"token":    cfg.CICD.Jenkins.Token,
		}

		if err := p.Initialize(providerConfig); err != nil {
			return fmt.Errorf("failed to initialize jenkins provider: %w", err)
		}

		// 获取 Job 详情
		job, err := p.GetJob(ctx, jobName)
		if err != nil {
			return fmt.Errorf("failed to get job: %w", err)
		}

		// 输出结果
		data, _ := json.MarshalIndent(job, "", "  ")
		fmt.Println(string(data))

		return nil
	},
}

// jenkinsBuildCmd Build 命令组
var jenkinsBuildCmd = &cobra.Command{
	Use:   "build",
	Short: "查询 Build 构建",
	Long:  `查询 Jenkins Build 构建信息。`,
}

// jenkinsBuildListCmd 列出 Build 历史
var jenkinsBuildListCmd = &cobra.Command{
	Use:   "list <job-name>",
	Short: "列出 Build 历史",
	Long:  `列出指定 Job 的构建历史。`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		jobName := args[0]
		ctx := context.Background()

		// 获取 Jenkins Provider
		p, err := provider.GetCICDProvider("jenkins")
		if err != nil {
			return fmt.Errorf("failed to get jenkins provider: %w", err)
		}

		// 初始化 Provider
		providerConfig := map[string]any{
			"url":      cfg.CICD.Jenkins.URL,
			"username": cfg.CICD.Jenkins.Username,
			"token":    cfg.CICD.Jenkins.Token,
		}

		if err := p.Initialize(providerConfig); err != nil {
			return fmt.Errorf("failed to initialize jenkins provider: %w", err)
		}

		// 查询 Builds
		limit := jenkinsPageSize
		if limit <= 0 {
			limit = 10
		}

		builds, err := p.GetJobBuilds(ctx, jobName, limit)
		if err != nil {
			return fmt.Errorf("failed to list builds: %w", err)
		}

		// 输出结果
		if jenkinsOutputType == "json" {
			data, _ := json.MarshalIndent(builds, "", "  ")
			fmt.Println(string(data))
		} else {
			// 使用 lipgloss/table 表格输出
			rows := [][]string{}

			for _, build := range builds {
				timestamp := build.Timestamp.Format("2006-01-02 15:04:05")
				duration := fmt.Sprintf("%dms", build.Duration)

				rows = append(rows, []string{
					fmt.Sprintf("#%d", build.Number),
					build.Status,
					build.Result,
					timestamp,
					duration,
				})
			}

			t := table.New().
				Border(lipgloss.NormalBorder()).
				BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("99"))).
				Headers("Build", "Status", "Result", "Timestamp", "Duration").
				Rows(rows...)

			fmt.Println(t)
			fmt.Println()
			logx.Info("Query completed, job %s, count %d", jobName, len(builds))
		}

		return nil
	},
}

func init() {
	// 添加 Jenkins 命令到查询命令组
	queryCmd.AddCommand(jenkinsCmd)

	// 添加 Job 命令
	jenkinsCmd.AddCommand(jenkinsJobCmd)
	jenkinsJobCmd.AddCommand(jenkinsJobListCmd)
	jenkinsJobCmd.AddCommand(jenkinsJobGetCmd)

	// 添加 Build 命令
	jenkinsCmd.AddCommand(jenkinsBuildCmd)
	jenkinsBuildCmd.AddCommand(jenkinsBuildListCmd)

	// 通用标志
	jenkinsCmd.PersistentFlags().IntVar(&jenkinsPageSize, "page-size", 10, "分页大小")
	jenkinsCmd.PersistentFlags().IntVar(&jenkinsPageNum, "page-num", 1, "页码")
	jenkinsCmd.PersistentFlags().StringVarP(&jenkinsOutputType, "output", "o", "table", "输出格式 (table, json)")
}
