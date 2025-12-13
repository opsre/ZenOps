package cmd

import (
	"github.com/spf13/cobra"
)

// queryCmd 查询命令组
var queryCmd = &cobra.Command{
	Use:   "query",
	Short: "查询云资源和 CI/CD 信息",
	Long:  `查询云服务商的资源信息(如实例、数据库)和 CI/CD 工具的任务信息。`,
}

func init() {
	rootCmd.AddCommand(queryCmd)
}
