package imcp

import (
	"fmt"
	"strings"

	"cnb.cool/zhiqiangwang/pkg/logx"
	"github.com/eryajf/zenops/internal/config"
	"github.com/eryajf/zenops/internal/model"
	"github.com/eryajf/zenops/internal/provider"
	"github.com/eryajf/zenops/internal/provider/aliyun"
	"github.com/eryajf/zenops/internal/service"
)

// ==================== Provider 辅助函数 ====================

// getAliyunProvider 获取阿里云 Provider
func (s *MCPServer) getAliyunProvider(accountName string) (provider.Provider, *config.ProviderConfig, error) {
	// 从数据库或配置获取账号配置
	aliyunConfig, err := s.getAliyunConfigByNameFromDB(accountName)
	if err != nil {
		return nil, nil, err
	}

	// 创建 Provider
	p, err := provider.GetProvider("aliyun")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get provider: %w", err)
	}

	// 初始化 Provider
	providerConfig := map[string]any{
		"access_key_id":     aliyunConfig.AK,
		"access_key_secret": aliyunConfig.SK,
		"regions":           interfaceSlice(aliyunConfig.Regions),
	}

	if err := p.Initialize(providerConfig); err != nil {
		return nil, nil, fmt.Errorf("failed to initialize provider for account %s: %w", accountName, err)
	}

	return p, aliyunConfig, nil
}

// getAliyunClient 直接获取阿里云客户端（用于高级查询）
func (s *MCPServer) getAliyunClient(accountName string, region string) (*aliyun.Client, *config.ProviderConfig, error) {
	// 从数据库或配置获取账号配置
	aliyunConfig, err := s.getAliyunConfigByNameFromDB(accountName)
	if err != nil {
		return nil, nil, err
	}

	// 如果没有指定 region，使用第一个配置的 region
	if region == "" && len(aliyunConfig.Regions) > 0 {
		region = aliyunConfig.Regions[0]
	}

	// 创建阿里云客户端
	aliyunClient, err := aliyun.NewClient(aliyunConfig.AK, aliyunConfig.SK, region)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create aliyun client: %w", err)
	}

	return aliyunClient, aliyunConfig, nil
}

// getTencentProvider 获取腾讯云 Provider
func (s *MCPServer) getTencentProvider(accountName string) (provider.Provider, *config.ProviderConfig, error) {
	// 从数据库或配置获取账号配置
	tencentConfig, err := s.getTencentConfigByNameFromDB(accountName)
	if err != nil {
		return nil, nil, err
	}

	// 创建 Provider
	p, err := provider.GetProvider("tencent")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get provider: %w", err)
	}

	// 初始化 Provider
	providerConfig := map[string]any{
		"secret_id":  tencentConfig.AK,
		"secret_key": tencentConfig.SK,
		"regions":    interfaceSlice(tencentConfig.Regions),
	}

	if err := p.Initialize(providerConfig); err != nil {
		return nil, nil, fmt.Errorf("failed to initialize provider for account %s: %w", accountName, err)
	}

	return p, tencentConfig, nil
}

// getJenkinsProvider 获取 Jenkins Provider
func (s *MCPServer) getJenkinsProvider() (provider.CICDProvider, error) {
	// 创建 Provider
	p, err := provider.GetCICDProvider("jenkins")
	if err != nil {
		return nil, fmt.Errorf("failed to get jenkins provider: %w", err)
	}

	// 初始化 Provider
	providerConfig := map[string]any{
		"url":      s.config.CICD.Jenkins.URL,
		"username": s.config.CICD.Jenkins.Username,
		"token":    s.config.CICD.Jenkins.Token,
	}

	if err := p.Initialize(providerConfig); err != nil {
		return nil, fmt.Errorf("failed to initialize jenkins provider: %w", err)
	}

	return p, nil
}

// getAliyunConfigByNameFromDB 从数据库或配置获取阿里云账号配置
func (s *MCPServer) getAliyunConfigByNameFromDB(accountName string) (*config.ProviderConfig, error) {
	// 先尝试从数据库加载
	configService := service.NewConfigService()
	accounts, err := configService.ListProviderAccounts("aliyun")
	if err == nil && len(accounts) > 0 {
		logx.Debug("Loading aliyun config from database, account count %d", len(accounts))

		// 如果没有指定账号名,返回第一个启用的账号
		if accountName == "" {
			for _, acc := range accounts {
				if acc.Enabled {
					return &config.ProviderConfig{
						Name:    acc.Name,
						Enabled: acc.Enabled,
						AK:      acc.AccessKey,
						SK:      acc.SecretKey,
						Regions: acc.Regions,
					}, nil
				}
			}
			// 如果没有启用的,返回第一个
			if len(accounts) > 0 {
				acc := accounts[0]
				return &config.ProviderConfig{
					Name:    acc.Name,
					Enabled: acc.Enabled,
					AK:      acc.AccessKey,
					SK:      acc.SecretKey,
					Regions: acc.Regions,
				}, nil
			}
		}

		// 根据名称查找
		for _, acc := range accounts {
			if acc.Name == accountName {
				return &config.ProviderConfig{
					Name:    acc.Name,
					Enabled: acc.Enabled,
					AK:      acc.AccessKey,
					SK:      acc.SecretKey,
					Regions: acc.Regions,
				}, nil
			}
		}

		return nil, fmt.Errorf("aliyun account '%s' not found in database", accountName)
	}

	// 如果数据库没有配置,回退到 YAML 配置
	logx.Debug("No aliyun config in database, falling back to YAML config")
	return getAliyunConfigByName(s.config, accountName)
}

// getAliyunConfigByName 根据名称获取阿里云账号配置(从YAML)
func getAliyunConfigByName(cfg *config.Config, accountName string) (*config.ProviderConfig, error) {
	if len(cfg.Providers.Aliyun) == 0 {
		return nil, fmt.Errorf("no aliyun account configured")
	}

	if accountName == "" {
		for _, acc := range cfg.Providers.Aliyun {
			if acc.Enabled {
				return &acc, nil
			}
		}
		return &cfg.Providers.Aliyun[0], nil
	}

	for _, acc := range cfg.Providers.Aliyun {
		if acc.Name == accountName {
			return &acc, nil
		}
	}

	return nil, fmt.Errorf("aliyun account '%s' not found", accountName)
}

// getTencentConfigByNameFromDB 从数据库或配置获取腾讯云账号配置
func (s *MCPServer) getTencentConfigByNameFromDB(accountName string) (*config.ProviderConfig, error) {
	// 先尝试从数据库加载
	configService := service.NewConfigService()
	accounts, err := configService.ListProviderAccounts("tencent")
	if err == nil && len(accounts) > 0 {
		logx.Debug("Loading tencent config from database, account count %d", len(accounts))

		// 如果没有指定账号名,返回第一个启用的账号
		if accountName == "" {
			for _, acc := range accounts {
				if acc.Enabled {
					return &config.ProviderConfig{
						Name:    acc.Name,
						Enabled: acc.Enabled,
						AK:      acc.AccessKey,
						SK:      acc.SecretKey,
						Regions: acc.Regions,
					}, nil
				}
			}
			// 如果没有启用的,返回第一个
			if len(accounts) > 0 {
				acc := accounts[0]
				return &config.ProviderConfig{
					Name:    acc.Name,
					Enabled: acc.Enabled,
					AK:      acc.AccessKey,
					SK:      acc.SecretKey,
					Regions: acc.Regions,
				}, nil
			}
		}

		// 根据名称查找
		for _, acc := range accounts {
			if acc.Name == accountName {
				return &config.ProviderConfig{
					Name:    acc.Name,
					Enabled: acc.Enabled,
					AK:      acc.AccessKey,
					SK:      acc.SecretKey,
					Regions: acc.Regions,
				}, nil
			}
		}

		return nil, fmt.Errorf("tencent account '%s' not found in database", accountName)
	}

	// 如果数据库没有配置,回退到 YAML 配置
	logx.Debug("No tencent config in database, falling back to YAML config")
	return getTencentConfigByName(s.config, accountName)
}

// getTencentConfigByName 根据名称获取腾讯云账号配置(从YAML)
func getTencentConfigByName(cfg *config.Config, accountName string) (*config.ProviderConfig, error) {
	if len(cfg.Providers.Tencent) == 0 {
		return nil, fmt.Errorf("no tencent account configured")
	}

	if accountName == "" {
		for _, acc := range cfg.Providers.Tencent {
			if acc.Enabled {
				return &acc, nil
			}
		}
		return &cfg.Providers.Tencent[0], nil
	}

	for _, acc := range cfg.Providers.Tencent {
		if acc.Name == accountName {
			return &acc, nil
		}
	}

	return nil, fmt.Errorf("tencent account '%s' not found", accountName)
}

// interfaceSlice 将 []string 转换为 []any
func interfaceSlice(s []string) []any {
	result := make([]any, len(s))
	for i, v := range s {
		result[i] = v
	}
	return result
}

// ==================== 格式化函数 ====================

// formatInstances 格式化 ECS/CVM 实例信息
func formatInstances(instances []*model.Instance, accountName string) string {
	if len(instances) == 0 {
		return "未找到任何实例"
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("找到 %d 个实例 (账号: %s):\n\n", len(instances), accountName))

	for i, inst := range instances {
		result.WriteString(fmt.Sprintf("【实例 %d】\n", i+1))
		result.WriteString(fmt.Sprintf("  实例 ID: %s\n", inst.ID))
		result.WriteString(fmt.Sprintf("  实例名称: %s\n", inst.Name))
		result.WriteString(fmt.Sprintf("  区域: %s\n", inst.Region))
		result.WriteString(fmt.Sprintf("  可用区: %s\n", inst.Zone))
		result.WriteString(fmt.Sprintf("  实例规格: %s\n", inst.InstanceType))
		result.WriteString(fmt.Sprintf("  状态: %s\n", inst.Status))
		result.WriteString(fmt.Sprintf("  CPU: %d 核\n", inst.CPU))
		result.WriteString(fmt.Sprintf("  内存: %d MB\n", inst.Memory))
		result.WriteString(fmt.Sprintf("  操作系统: %s\n", inst.OSName))

		if len(inst.PrivateIP) > 0 {
			result.WriteString(fmt.Sprintf("  私网 IP: %v\n", inst.PrivateIP))
		}
		if len(inst.PublicIP) > 0 {
			result.WriteString(fmt.Sprintf("  公网 IP: %v\n", inst.PublicIP))
		}

		result.WriteString(fmt.Sprintf("  创建时间: %s\n", inst.CreatedAt.Format("2006-01-02 15:04:05")))

		if inst.ConsoleURL != "" {
			result.WriteString(fmt.Sprintf("  控制台地址: %s\n", inst.ConsoleURL))
		}

		result.WriteString("\n")
	}

	return result.String()
}

// formatDatabases 格式化 RDS/CDB 实例信息
func formatDatabases(databases []*model.Database, accountName string) string {
	if len(databases) == 0 {
		return "未找到任何数据库实例"
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("找到 %d 个数据库实例 (账号: %s):\n\n", len(databases), accountName))

	for i, db := range databases {
		result.WriteString(fmt.Sprintf("【实例 %d】\n", i+1))
		result.WriteString(fmt.Sprintf("  实例 ID: %s\n", db.ID))
		result.WriteString(fmt.Sprintf("  实例名称: %s\n", db.Name))
		result.WriteString(fmt.Sprintf("  区域: %s\n", db.Region))
		result.WriteString(fmt.Sprintf("  引擎: %s %s\n", db.Engine, db.EngineVersion))
		result.WriteString(fmt.Sprintf("  状态: %s\n", db.Status))

		if db.Endpoint != "" {
			result.WriteString(fmt.Sprintf("  连接地址: %s:%d\n", db.Endpoint, db.Port))
		}

		result.WriteString(fmt.Sprintf("  创建时间: %s\n", db.CreatedAt.Format("2006-01-02 15:04:05")))

		if db.ConsoleURL != "" {
			result.WriteString(fmt.Sprintf("  控制台地址: %s\n", db.ConsoleURL))
		}

		result.WriteString("\n")
	}

	return result.String()
}
