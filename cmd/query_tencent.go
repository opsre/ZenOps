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
	"github.com/spf13/cobra"
)

var (
	tencentRegion     string
	tencentPageSize   int
	tencentPageNum    int
	tencentOutputType string
	tencentAccount    string
	tencentFetchAll   bool
)

// tencentCmd 腾讯云查询命令组
var tencentCmd = &cobra.Command{
	Use:   "tencent",
	Short: "查询腾讯云资源",
	Long:  `查询腾讯云的 CVM 实例、CDB 数据库等资源信息。`,
}

// tencentCVMCmd CVM 命令组
var tencentCVMCmd = &cobra.Command{
	Use:   "cvm",
	Short: "查询 CVM 实例",
	Long:  `查询腾讯云 CVM (云服务器) 实例。`,
}

// tencentCVMListCmd 列出 CVM 实例
var tencentCVMListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出 CVM 实例",
	Long:  `列出腾讯云 CVM 实例列表。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		// 获取指定账号的配置
		tencentConfig, err := getTencentConfig(tencentAccount)
		if err != nil {
			return err
		}

		// 获取 Tencent Provider
		p, err := provider.GetProvider("tencent")
		if err != nil {
			return fmt.Errorf("failed to get tencent provider: %w", err)
		}

		// 初始化 Provider
		providerConfig := map[string]any{
			"secret_id":  tencentConfig.AK,
			"secret_key": tencentConfig.SK,
			"regions":    interfaceSlice(tencentConfig.Regions),
		}

		if err := p.Initialize(providerConfig); err != nil {
			return fmt.Errorf("failed to initialize tencent provider: %w", err)
		}

		var instances []*model.Instance

		// 判断是否获取所有资源
		if tencentFetchAll {
			// 分页获取所有实例
			pageNum := 1
			pageSize := tencentPageSize
			if pageSize <= 0 {
				pageSize = 100
			}

			logx.Info("Fetching all instances, account %s", tencentConfig.Name)

			for {
				opts := &provider.QueryOptions{
					Region:   tencentRegion,
					PageSize: pageSize,
					PageNum:  pageNum,
				}

				pageInstances, err := p.ListInstances(ctx, opts)
				if err != nil {
					return fmt.Errorf("failed to list instances (page %d): %w", pageNum, err)
				}

				instances = append(instances, pageInstances...)

				if len(pageInstances) < pageSize {
					break
				}

				pageNum++
				logx.Debug("Fetching next page, page %d, current_total %d", pageNum, len(instances))
			}
		} else {
			// 单页查询
			opts := &provider.QueryOptions{
				Region:   tencentRegion,
				PageSize: tencentPageSize,
				PageNum:  tencentPageNum,
			}

			instances, err = p.ListInstances(ctx, opts)
			if err != nil {
				return fmt.Errorf("failed to list instances: %w", err)
			}
		}

		// 输出结果
		if tencentOutputType == "json" {
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
			logx.Info("Query completed, count %d, account %s", len(instances), tencentConfig.Name)
		}

		return nil
	},
}

// tencentCVMGetCmd 获取 CVM 实例详情
var tencentCVMGetCmd = &cobra.Command{
	Use:   "get <instance-id>",
	Short: "获取 CVM 实例详情",
	Long:  `获取指定 CVM 实例的详细信息。`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		instanceID := args[0]
		ctx := context.Background()

		// 获取指定账号的配置
		tencentConfig, err := getTencentConfig(tencentAccount)
		if err != nil {
			return err
		}

		// 获取 Tencent Provider
		p, err := provider.GetProvider("tencent")
		if err != nil {
			return fmt.Errorf("failed to get tencent provider: %w", err)
		}

		// 初始化 Provider
		providerConfig := map[string]any{
			"secret_id":  tencentConfig.AK,
			"secret_key": tencentConfig.SK,
			"regions":    interfaceSlice(tencentConfig.Regions),
		}

		if err := p.Initialize(providerConfig); err != nil {
			return fmt.Errorf("failed to initialize tencent provider: %w", err)
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

// tencentCDBCmd CDB 命令组
var tencentCDBCmd = &cobra.Command{
	Use:   "cdb",
	Short: "查询 CDB 数据库",
	Long:  `查询腾讯云 CDB (云数据库) 实例。`,
}

// tencentCDBListCmd 列出 CDB 实例
var tencentCDBListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出 CDB 实例",
	Long:  `列出腾讯云 CDB 数据库实例列表。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		// 获取指定账号的配置
		tencentConfig, err := getTencentConfig(tencentAccount)
		if err != nil {
			return err
		}

		// 获取 Tencent Provider
		p, err := provider.GetProvider("tencent")
		if err != nil {
			return fmt.Errorf("failed to get tencent provider: %w", err)
		}

		// 初始化 Provider
		providerConfig := map[string]any{
			"secret_id":  tencentConfig.AK,
			"secret_key": tencentConfig.SK,
			"regions":    interfaceSlice(tencentConfig.Regions),
		}

		if err := p.Initialize(providerConfig); err != nil {
			return fmt.Errorf("failed to initialize tencent provider: %w", err)
		}

		var databases []*model.Database

		// 判断是否获取所有资源
		if tencentFetchAll {
			// 分页获取所有数据库
			pageNum := 1
			pageSize := tencentPageSize
			if pageSize <= 0 {
				pageSize = 100
			}

			logx.Info("Fetching all databases, account %s", tencentConfig.Name)

			for {
				opts := &provider.QueryOptions{
					Region:   tencentRegion,
					PageSize: pageSize,
					PageNum:  pageNum,
				}

				pageDatabases, err := p.ListDatabases(ctx, opts)
				if err != nil {
					return fmt.Errorf("failed to list databases (page %d): %w", pageNum, err)
				}

				databases = append(databases, pageDatabases...)

				if len(pageDatabases) < pageSize {
					break
				}

				pageNum++
				logx.Debug("Fetching next page, page %d, current_total %d", pageNum, len(databases))
			}
		} else {
			// 单页查询
			opts := &provider.QueryOptions{
				Region:   tencentRegion,
				PageSize: tencentPageSize,
				PageNum:  tencentPageNum,
			}

			databases, err = p.ListDatabases(ctx, opts)
			if err != nil {
				return fmt.Errorf("failed to list databases: %w", err)
			}
		}

		// 输出结果
		if tencentOutputType == "json" {
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
			logx.Info("Query completed, count %d, account %s", len(databases), tencentConfig.Name)
		}

		return nil
	},
}

// tencentCDBGetCmd 获取 CDB 实例详情
var tencentCDBGetCmd = &cobra.Command{
	Use:   "get <instance-id>",
	Short: "获取 CDB 实例详情",
	Long:  `获取指定 CDB 实例的详细信息。`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		instanceID := args[0]
		ctx := context.Background()

		// 获取指定账号的配置
		tencentConfig, err := getTencentConfig(tencentAccount)
		if err != nil {
			return err
		}

		// 获取 Tencent Provider
		p, err := provider.GetProvider("tencent")
		if err != nil {
			return fmt.Errorf("failed to get tencent provider: %w", err)
		}

		// 初始化 Provider
		providerConfig := map[string]any{
			"secret_id":  tencentConfig.AK,
			"secret_key": tencentConfig.SK,
			"regions":    interfaceSlice(tencentConfig.Regions),
		}

		if err := p.Initialize(providerConfig); err != nil {
			return fmt.Errorf("failed to initialize tencent provider: %w", err)
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

// getTencentConfig 获取指定名称的腾讯云账号配置
func getTencentConfig(accountName string) (*config.ProviderConfig, error) {
	if len(cfg.Providers.Tencent) == 0 {
		return nil, fmt.Errorf("no tencent account configured")
	}

	// 如果未指定账号名称,使用第一个启用的账号
	if accountName == "" {
		for _, acc := range cfg.Providers.Tencent {
			if acc.Enabled {
				return &acc, nil
			}
		}
		// 如果没有启用的账号,返回第一个
		return &cfg.Providers.Tencent[0], nil
	}

	// 查找指定名称的账号
	for _, acc := range cfg.Providers.Tencent {
		if acc.Name == accountName {
			return &acc, nil
		}
	}

	return nil, fmt.Errorf("tencent account '%s' not found", accountName)
}

// tencentCOSCmd COS 命令组
var tencentCOSCmd = &cobra.Command{
	Use:   "cos",
	Short: "查询腾讯云 COS 存储桶",
	Long:  `查询腾讯云 COS 存储桶列表和详情。`,
}

// tencentCOSListCmd 列出 COS 存储桶
var tencentCOSListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出 COS 存储桶",
	Long:  `列出腾讯云 COS 存储桶列表。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		// 获取指定账号的配置
		tencentConfig, err := getTencentConfig(tencentAccount)
		if err != nil {
			return err
		}

		// 获取 Tencent Provider
		p, err := provider.GetProvider("tencent")
		if err != nil {
			return fmt.Errorf("failed to get tencent provider: %w", err)
		}

		// 初始化 Provider
		providerConfig := map[string]any{
			"secret_id":  tencentConfig.AK,
			"secret_key": tencentConfig.SK,
			"regions":    interfaceSlice(tencentConfig.Regions),
		}

		if err := p.Initialize(providerConfig); err != nil {
			return fmt.Errorf("failed to initialize tencent provider: %w", err)
		}

		var buckets []*model.OSSBucket

		// 判断是否获取所有资源
		if tencentFetchAll {
			pageNum := 1
			pageSize := tencentPageSize
			if pageSize <= 0 {
				pageSize = 100
			}

			logx.Info("Fetching all COS buckets, account %s", tencentConfig.Name)

			for {
				opts := &provider.QueryOptions{
					Region:   tencentRegion,
					PageSize: pageSize,
					PageNum:  pageNum,
				}

				pageBuckets, err := p.ListOSSBuckets(ctx, opts)
				if err != nil {
					return fmt.Errorf("failed to list COS buckets (page %d): %w", pageNum, err)
				}

				buckets = append(buckets, pageBuckets...)

				if len(pageBuckets) < pageSize {
					break
				}

				pageNum++
				logx.Debug("Fetching next page, page: %d, current_total: %d", pageNum, len(buckets))
			}
		} else {
			opts := &provider.QueryOptions{
				Region:   tencentRegion,
				PageSize: tencentPageSize,
				PageNum:  tencentPageNum,
			}

			buckets, err = p.ListOSSBuckets(ctx, opts)
			if err != nil {
				return fmt.Errorf("failed to list COS buckets: %w", err)
			}
		}

		// 输出结果
		if tencentOutputType == "json" {
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
			logx.Info("Query completed, count %d, account %s", len(buckets), tencentConfig.Name)
		}

		return nil
	},
}

// tencentCOSGetCmd 获取 COS 存储桶详情
var tencentCOSGetCmd = &cobra.Command{
	Use:   "get <bucket-name>",
	Short: "获取 COS 存储桶详情",
	Long:  `获取指定 COS 存储桶的详细信息。`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		bucketName := args[0]
		ctx := context.Background()

		// 获取指定账号的配置
		tencentConfig, err := getTencentConfig(tencentAccount)
		if err != nil {
			return err
		}

		// 获取 Tencent Provider
		p, err := provider.GetProvider("tencent")
		if err != nil {
			return fmt.Errorf("failed to get tencent provider: %w", err)
		}

		// 初始化 Provider
		providerConfig := map[string]any{
			"secret_id":  tencentConfig.AK,
			"secret_key": tencentConfig.SK,
			"regions":    interfaceSlice(tencentConfig.Regions),
		}

		if err := p.Initialize(providerConfig); err != nil {
			return fmt.Errorf("failed to initialize tencent provider: %w", err)
		}

		// 获取存储桶详情
		bucket, err := p.GetOSSBucket(ctx, bucketName)
		if err != nil {
			return fmt.Errorf("failed to get COS bucket: %w", err)
		}

		// 输出结果
		data, _ := json.MarshalIndent(bucket, "", "  ")
		fmt.Println(string(data))

		return nil
	},
}

func init() {
	// 添加腾讯云命令到查询命令组
	queryCmd.AddCommand(tencentCmd)

	// 添加 CVM 命令
	tencentCmd.AddCommand(tencentCVMCmd)
	tencentCVMCmd.AddCommand(tencentCVMListCmd)
	tencentCVMCmd.AddCommand(tencentCVMGetCmd)

	// 添加 CDB 命令
	tencentCmd.AddCommand(tencentCDBCmd)
	tencentCDBCmd.AddCommand(tencentCDBListCmd)
	tencentCDBCmd.AddCommand(tencentCDBGetCmd)

	// 添加 COS 命令
	tencentCmd.AddCommand(tencentCOSCmd)
	tencentCOSCmd.AddCommand(tencentCOSListCmd)
	tencentCOSCmd.AddCommand(tencentCOSGetCmd)

	// 通用标志
	tencentCmd.PersistentFlags().StringVarP(&tencentAccount, "account", "a", "", "指定账号名称 (默认: 使用第一个启用的账号)")
	tencentCmd.PersistentFlags().StringVarP(&tencentRegion, "region", "r", "", "指定区域 (默认: 所有区域)")
	tencentCmd.PersistentFlags().IntVar(&tencentPageSize, "page-size", 10, "分页大小")
	tencentCmd.PersistentFlags().IntVar(&tencentPageNum, "page-num", 1, "页码")
	tencentCmd.PersistentFlags().BoolVar(&tencentFetchAll, "all", true, "获取所有资源 (分页循环获取)")
	tencentCmd.PersistentFlags().StringVarP(&tencentOutputType, "output", "o", "table", "输出格式 (table, json)")
}
