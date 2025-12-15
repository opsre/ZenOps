package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"cnb.cool/zhiqiangwang/pkg/logx"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/eryajf/zenops/internal/config"
	"github.com/eryajf/zenops/internal/model"
	"github.com/eryajf/zenops/internal/provider"
	aliyunprovider "github.com/eryajf/zenops/internal/provider/aliyun"
	"github.com/spf13/cobra"
)

var (
	aliyunRegion     string
	aliyunPageSize   int
	aliyunPageNum    int
	aliyunOutputType string
	aliyunAccount    string // 账号名称
	aliyunFetchAll   bool   // 是否获取所有资源
)

// aliyunCmd 阿里云查询命令组
var aliyunCmd = &cobra.Command{
	Use:   "aliyun",
	Short: "查询阿里云资源",
	Long:  `查询阿里云的 ECS 实例、RDS 数据库、OSS 存储桶等资源信息。`,
}

// aliyunECSCmd 阿里云 ECS 命令组
var aliyunECSCmd = &cobra.Command{
	Use:   "ecs",
	Short: "查询阿里云 ECS 实例",
	Long:  `查询阿里云 ECS 实例列表和详情。`,
}

// aliyunECSListCmd 列出 ECS 实例
var aliyunECSListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出 ECS 实例",
	Long:  `列出阿里云 ECS 实例列表。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		// 获取指定账号的配置
		aliyunConfig, err := getAliyunConfig(aliyunAccount)
		if err != nil {
			return err
		}

		// 获取 Aliyun Provider
		p, err := provider.GetProvider("aliyun")
		if err != nil {
			return fmt.Errorf("failed to get aliyun provider: %w", err)
		}

		// 初始化 Provider
		providerConfig := map[string]any{
			"access_key_id":     aliyunConfig.AK,
			"access_key_secret": aliyunConfig.SK,
			"regions":           interfaceSlice(aliyunConfig.Regions),
		}

		if err := p.Initialize(providerConfig); err != nil {
			return fmt.Errorf("failed to initialize aliyun provider: %w", err)
		}

		var instances []*model.Instance

		// 判断是否获取所有资源
		if aliyunFetchAll {
			// 分页获取所有实例
			pageNum := 1
			pageSize := aliyunPageSize
			if pageSize <= 0 {
				pageSize = 100 // 使用更大的分页大小提高效率
			}

			logx.Info("Fetching all instances, account %s", aliyunConfig.Name)

			for {
				opts := &provider.QueryOptions{
					Region:   aliyunRegion,
					PageSize: pageSize,
					PageNum:  pageNum,
				}

				pageInstances, err := p.ListInstances(ctx, opts)
				if err != nil {
					return fmt.Errorf("failed to list instances (page %d): %w", pageNum, err)
				}

				instances = append(instances, pageInstances...)

				// 如果返回的实例数少于分页大小,说明已经是最后一页
				if len(pageInstances) < pageSize {
					break
				}

				pageNum++
				logx.Debug("Fetching next page, page: %d , current_total: %d", pageNum, len(instances))
			}
		} else {
			// 单页查询
			opts := &provider.QueryOptions{
				Region:   aliyunRegion,
				PageSize: aliyunPageSize,
				PageNum:  aliyunPageNum,
			}

			instances, err = p.ListInstances(ctx, opts)
			if err != nil {
				return fmt.Errorf("failed to list instances: %w", err)
			}
		}

		// 输出结果
		if aliyunOutputType == "json" {
			data, _ := json.MarshalIndent(instances, "", "  ")
			fmt.Println(string(data))
		} else {
			// 使用 lipgloss/table 表格输出
			rows := [][]string{}

			for _, inst := range instances {
				privateIP := ""
				if len(inst.PrivateIP) > 0 {
					privateIP = inst.PrivateIP[0]
				}
				publicIP := ""
				if len(inst.PublicIP) > 0 {
					publicIP = inst.PublicIP[0]
				}
				rows = append(rows, []string{
					inst.ID, inst.Name, inst.Region, inst.Status,
					inst.InstanceType, privateIP, publicIP,
				})
			}

			t := table.New().
				Border(lipgloss.NormalBorder()).
				BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("99"))).
				Headers("ID", "Name", "Region", "Status", "Instance Type", "Private IP", "Public IP").
				Rows(rows...)

			fmt.Println(t)
			fmt.Println()
			logx.Info("Query completed, count %d, account %s", len(instances), aliyunConfig.Name)
		}

		return nil
	},
}

// aliyunECSGetCmd 获取 ECS 实例详情
var aliyunECSGetCmd = &cobra.Command{
	Use:   "get <instance-id>",
	Short: "获取 ECS 实例详情",
	Long:  `获取指定 ECS 实例的详细信息。`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		instanceID := args[0]
		ctx := context.Background()

		// 获取指定账号的配置
		aliyunConfig, err := getAliyunConfig(aliyunAccount)
		if err != nil {
			return err
		}

		// 获取 Aliyun Provider
		p, err := provider.GetProvider("aliyun")
		if err != nil {
			return fmt.Errorf("failed to get aliyun provider: %w", err)
		}

		// 初始化 Provider
		providerConfig := map[string]any{
			"access_key_id":     aliyunConfig.AK,
			"access_key_secret": aliyunConfig.SK,
			"regions":           interfaceSlice(aliyunConfig.Regions),
		}

		if err := p.Initialize(providerConfig); err != nil {
			return fmt.Errorf("failed to initialize aliyun provider: %w", err)
		}

		// 获取实例详情
		instance, err := p.GetInstance(ctx, instanceID)
		if err != nil {
			return fmt.Errorf("failed to get instance: %w", err)
		}

		// 输出结果
		data, _ := json.MarshalIndent(instance, "", "  ")
		fmt.Println(string(data))

		return nil
	},
}

// aliyunRDSCmd 阿里云 RDS 命令组
var aliyunRDSCmd = &cobra.Command{
	Use:   "rds",
	Short: "查询阿里云 RDS 数据库",
	Long:  `查询阿里云 RDS 数据库列表和详情。`,
}

// aliyunRDSListCmd 列出 RDS 实例
var aliyunRDSListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出 RDS 实例",
	Long:  `列出阿里云 RDS 数据库实例列表。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		// 获取指定账号的配置
		aliyunConfig, err := getAliyunConfig(aliyunAccount)
		if err != nil {
			return err
		}

		// 获取 Aliyun Provider
		p, err := provider.GetProvider("aliyun")
		if err != nil {
			return fmt.Errorf("failed to get aliyun provider: %w", err)
		}

		// 初始化 Provider
		providerConfig := map[string]any{
			"access_key_id":     aliyunConfig.AK,
			"access_key_secret": aliyunConfig.SK,
			"regions":           interfaceSlice(aliyunConfig.Regions),
		}

		if err := p.Initialize(providerConfig); err != nil {
			return fmt.Errorf("failed to initialize aliyun provider: %w", err)
		}

		var databases []*model.Database

		// 判断是否获取所有资源
		if aliyunFetchAll {
			// 分页获取所有数据库
			pageNum := 1
			pageSize := aliyunPageSize
			if pageSize <= 0 {
				pageSize = 100 // 使用更大的分页大小提高效率
			}

			logx.Info("Fetching all databases, account %s", aliyunConfig.Name)

			for {
				opts := &provider.QueryOptions{
					Region:   aliyunRegion,
					PageSize: pageSize,
					PageNum:  pageNum,
				}

				pageDatabases, err := p.ListDatabases(ctx, opts)
				if err != nil {
					return fmt.Errorf("failed to list databases (page %d): %w", pageNum, err)
				}

				databases = append(databases, pageDatabases...)

				// 如果返回的数据库数少于分页大小,说明已经是最后一页
				if len(pageDatabases) < pageSize {
					break
				}

				pageNum++
				logx.Debug("Fetching next page, page %d, current_total %d", pageNum, len(databases))
			}
		} else {
			// 单页查询
			opts := &provider.QueryOptions{
				Region:   aliyunRegion,
				PageSize: aliyunPageSize,
				PageNum:  aliyunPageNum,
			}

			databases, err = p.ListDatabases(ctx, opts)
			if err != nil {
				return fmt.Errorf("failed to list databases: %w", err)
			}
		}

		// 输出结果
		if aliyunOutputType == "json" {
			data, _ := json.MarshalIndent(databases, "", "  ")
			fmt.Println(string(data))
		} else {
			// 使用 lipgloss/table 表格输出
			rows := [][]string{}

			for _, db := range databases {
				rows = append(rows, []string{
					db.ID, db.Name, db.Region, db.Engine,
					db.EngineVersion, db.Status, db.Endpoint,
				})
			}

			t := table.New().
				Border(lipgloss.NormalBorder()).
				BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("99"))).
				Headers("ID", "Name", "Region", "Engine", "Version", "Status", "Endpoint").
				Rows(rows...)

			fmt.Println(t)
			fmt.Println()
			logx.Info("Query completed, count %d, account %s", len(databases), aliyunConfig.Name)
		}

		return nil
	},
}

// aliyunRDSGetCmd 获取 RDS 实例详情
var aliyunRDSGetCmd = &cobra.Command{
	Use:   "get <instance-id>",
	Short: "获取 RDS 实例详情",
	Long:  `获取指定 RDS 实例的详细信息。`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		instanceID := args[0]
		ctx := context.Background()

		// 获取指定账号的配置
		aliyunConfig, err := getAliyunConfig(aliyunAccount)
		if err != nil {
			return err
		}

		// 获取 Aliyun Provider
		p, err := provider.GetProvider("aliyun")
		if err != nil {
			return fmt.Errorf("failed to get aliyun provider: %w", err)
		}

		// 初始化 Provider
		providerConfig := map[string]any{
			"access_key_id":     aliyunConfig.AK,
			"access_key_secret": aliyunConfig.SK,
			"regions":           interfaceSlice(aliyunConfig.Regions),
		}

		if err := p.Initialize(providerConfig); err != nil {
			return fmt.Errorf("failed to initialize aliyun provider: %w", err)
		}

		// 获取数据库详情
		database, err := p.GetDatabase(ctx, instanceID)
		if err != nil {
			return fmt.Errorf("failed to get database: %w", err)
		}

		// 输出结果
		data, _ := json.MarshalIndent(database, "", "  ")
		fmt.Println(string(data))

		return nil
	},
}

// getAliyunConfig 获取指定名称的阿里云账号配置
func getAliyunConfig(accountName string) (*config.ProviderConfig, error) {
	if len(cfg.Providers.Aliyun) == 0 {
		return nil, fmt.Errorf("no aliyun account configured")
	}

	// 如果未指定账号名称,使用第一个启用的账号
	if accountName == "" {
		for _, acc := range cfg.Providers.Aliyun {
			if acc.Enabled {
				return &acc, nil
			}
		}
		// 如果没有启用的账号,返回第一个
		return &cfg.Providers.Aliyun[0], nil
	}

	// 查找指定名称的账号
	for _, acc := range cfg.Providers.Aliyun {
		if acc.Name == accountName {
			return &acc, nil
		}
	}

	return nil, fmt.Errorf("aliyun account '%s' not found", accountName)
}

// aliyunOSSCmd 阿里云 OSS 命令组
var aliyunOSSCmd = &cobra.Command{
	Use:   "oss",
	Short: "查询阿里云 OSS 存储桶",
	Long:  `查询阿里云 OSS 存储桶列表和详情。`,
}

// aliyunOSSListCmd 列出 OSS 存储桶
var aliyunOSSListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出 OSS 存储桶",
	Long:  `列出阿里云 OSS 存储桶列表。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		// 获取指定账号的配置
		aliyunConfig, err := getAliyunConfig(aliyunAccount)
		if err != nil {
			return err
		}

		// 创建临时客户端
		var ossClient *aliyunprovider.Client
		for _, region := range aliyunConfig.Regions {
			c, err := aliyunprovider.NewClient(aliyunConfig.AK, aliyunConfig.SK, region)
			if err == nil {
				ossClient = c
				break
			}
		}
		if ossClient == nil {
			return fmt.Errorf("failed to create OSS client")
		}

		var buckets []*model.OSSBucket
		if aliyunFetchAll {
			pageNum := 1
			pageSize := aliyunPageSize
			if pageSize <= 0 {
				pageSize = 100
			}

			logx.Info("Fetching all OSS buckets, account %s", aliyunConfig.Name)

			for {
				pageBuckets, err := ossClient.ListOSSBuckets(ctx, pageSize, pageNum, nil)
				if err != nil {
					return fmt.Errorf("failed to list OSS buckets (page %d): %w", pageNum, err)
				}

				buckets = append(buckets, pageBuckets...)

				if len(pageBuckets) < pageSize {
					break
				}

				pageNum++
				logx.Debug("Fetching next page, page: %d, current_total: %d", pageNum, len(buckets))
			}
		} else {
			buckets, err = ossClient.ListOSSBuckets(ctx, aliyunPageSize, aliyunPageNum, nil)
			if err != nil {
				return fmt.Errorf("failed to list OSS buckets: %w", err)
			}
		}

		// 输出结果
		if aliyunOutputType == "json" {
			data, _ := json.MarshalIndent(buckets, "", "  ")
			fmt.Println(string(data))
		} else {
			// 使用 lipgloss/table 表格输出
			rows := [][]string{}

			for _, bucket := range buckets {
				rows = append(rows, []string{
					bucket.Name, bucket.Region, bucket.StorageClass,
					bucket.CreatedAt, bucket.ACL,
				})
			}

			t := table.New().
				Border(lipgloss.NormalBorder()).
				BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("99"))).
				Headers("Name", "Region", "Storage Class", "Created At", "ACL").
				Rows(rows...)

			fmt.Println(t)
			fmt.Println()
			logx.Info("Query completed, count %d, account %s", len(buckets), aliyunConfig.Name)
		}

		return nil
	},
}

// aliyunOSSGetCmd 获取 OSS 存储桶详情
var aliyunOSSGetCmd = &cobra.Command{
	Use:   "get <bucket-name>",
	Short: "获取 OSS 存储桶详情",
	Long:  `获取指定 OSS 存储桶的详细信息。`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		bucketName := args[0]
		ctx := context.Background()

		// 获取指定账号的配置
		aliyunConfig, err := getAliyunConfig(aliyunAccount)
		if err != nil {
			return err
		}

		// 创建临时客户端
		var ossClient *aliyunprovider.Client
		for _, region := range aliyunConfig.Regions {
			c, err := aliyunprovider.NewClient(aliyunConfig.AK, aliyunConfig.SK, region)
			if err == nil {
				ossClient = c
				break
			}
		}
		if ossClient == nil {
			return fmt.Errorf("failed to create OSS client")
		}

		// 获取存储桶详情
		bucket, err := ossClient.GetOSSBucket(ctx, bucketName)
		if err != nil {
			return fmt.Errorf("failed to get OSS bucket: %w", err)
		}

		// 输出结果
		data, _ := json.MarshalIndent(bucket, "", "  ")
		fmt.Println(string(data))

		return nil
	},
}

func init() {
	// 添加阿里云命令到查询命令组
	queryCmd.AddCommand(aliyunCmd)

	// 添加 ECS 命令
	aliyunCmd.AddCommand(aliyunECSCmd)
	aliyunECSCmd.AddCommand(aliyunECSListCmd)
	aliyunECSCmd.AddCommand(aliyunECSGetCmd)

	// 添加 RDS 命令
	aliyunCmd.AddCommand(aliyunRDSCmd)
	aliyunRDSCmd.AddCommand(aliyunRDSListCmd)
	aliyunRDSCmd.AddCommand(aliyunRDSGetCmd)

	// 添加 OSS 命令
	aliyunCmd.AddCommand(aliyunOSSCmd)
	aliyunOSSCmd.AddCommand(aliyunOSSListCmd)
	aliyunOSSCmd.AddCommand(aliyunOSSGetCmd)

	// 通用标志
	aliyunCmd.PersistentFlags().StringVarP(&aliyunAccount, "account", "a", "", "指定账号名称 (默认: 使用第一个启用的账号)")
	aliyunCmd.PersistentFlags().StringVarP(&aliyunRegion, "region", "r", "", "指定区域 (默认: 所有区域)")
	aliyunCmd.PersistentFlags().IntVar(&aliyunPageSize, "page-size", 10, "分页大小")
	aliyunCmd.PersistentFlags().IntVar(&aliyunPageNum, "page-num", 1, "页码")
	aliyunCmd.PersistentFlags().BoolVar(&aliyunFetchAll, "all", true, "获取所有资源 (分页循环获取)")
	aliyunCmd.PersistentFlags().StringVarP(&aliyunOutputType, "output", "o", "table", "输出格式 (table, json)")
}

// interfaceSlice 将 []string 转换为 []any
func interfaceSlice(s []string) []any {
	result := make([]any, len(s))
	for i, v := range s {
		result[i] = v
	}
	return result
}
