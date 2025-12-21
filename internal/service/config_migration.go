package service

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/eryajf/zenops/internal/config"
	"github.com/eryajf/zenops/internal/model"
)

// MigrateFromYAML 从YAML配置迁移到数据库
func (s *ConfigService) MigrateFromYAML(cfg *config.Config) error {
	log.Println("Starting configuration migration from YAML to database...")

	// 1. 迁移 LLM 配置
	if err := s.migrateLLMConfig(cfg.LLM); err != nil {
		return fmt.Errorf("failed to migrate LLM config: %w", err)
	}

	// 2. 迁移云厂商配置
	if err := s.migrateProviders(cfg.Providers); err != nil {
		return fmt.Errorf("failed to migrate provider config: %w", err)
	}

	// 3. 迁移 IM 配置
	if err := s.migrateIMConfigs(cfg); err != nil {
		return fmt.Errorf("failed to migrate IM config: %w", err)
	}

	// 4. 迁移 CICD 配置
	if err := s.migrateCICDConfigs(cfg.CICD); err != nil {
		return fmt.Errorf("failed to migrate CICD config: %w", err)
	}

	// 5. 迁移系统配置
	if err := s.migrateSystemConfigs(cfg); err != nil {
		return fmt.Errorf("failed to migrate system config: %w", err)
	}

	log.Println("Configuration migration completed successfully")
	return nil
}

// migrateLLMConfig 迁移LLM配置
func (s *ConfigService) migrateLLMConfig(llmCfg config.LLMConfig) error {
	existing, err := s.GetLLMConfig()
	if err != nil {
		return err
	}
	if existing != nil {
		log.Println("LLM config already exists in database, skipping migration")
		return nil
	}

	llm := &model.LLMConfig{
		Enabled: llmCfg.Enabled,
		Model:   llmCfg.Model,
		APIKey:  llmCfg.APIKey,
		BaseURL: llmCfg.BaseURL,
	}

	return s.SaveLLMConfig(llm)
}

// migrateProviders 迁移云厂商配置
func (s *ConfigService) migrateProviders(providers config.ProvidersConfig) error {
	// 迁移阿里云账号
	for _, p := range providers.Aliyun {
		account := &model.ProviderAccount{
			Provider:  "aliyun",
			Name:      p.Name,
			Enabled:   p.Enabled,
			AccessKey: p.AK,
			SecretKey: p.SK,
			Regions:   p.Regions,
		}

		// 检查是否已存在
		existing, err := s.GetProviderAccountByName("aliyun", p.Name)
		if err != nil {
			return err
		}
		if existing != nil {
			log.Printf("Provider account already exists: aliyun/%s, skipping", p.Name)
			continue
		}

		if err := s.CreateProviderAccount(account); err != nil {
			return err
		}
		log.Printf("Migrated provider account: aliyun/%s", p.Name)
	}

	// 迁移腾讯云账号
	for _, p := range providers.Tencent {
		account := &model.ProviderAccount{
			Provider:  "tencent",
			Name:      p.Name,
			Enabled:   p.Enabled,
			AccessKey: p.AK,
			SecretKey: p.SK,
			Regions:   p.Regions,
		}

		// 检查是否已存在
		existing, err := s.GetProviderAccountByName("tencent", p.Name)
		if err != nil {
			return err
		}
		if existing != nil {
			log.Printf("Provider account already exists: tencent/%s, skipping", p.Name)
			continue
		}

		if err := s.CreateProviderAccount(account); err != nil {
			return err
		}
		log.Printf("Migrated provider account: tencent/%s", p.Name)
	}

	return nil
}

// GetProviderAccountByName 根据provider和name获取账号
func (s *ConfigService) GetProviderAccountByName(provider, name string) (*model.ProviderAccount, error) {
	var account model.ProviderAccount
	err := s.db.Where("provider = ? AND name = ?", provider, name).First(&account).Error
	if err != nil {
		if err.Error() == "record not found" {
			return nil, nil
		}
		return nil, err
	}
	return &account, nil
}

// migrateIMConfigs 迁移IM配置
func (s *ConfigService) migrateIMConfigs(cfg *config.Config) error {
	// 迁移钉钉配置
	if cfg.DingTalk.AppKey != "" || cfg.DingTalk.AppSecret != "" {
		existing, err := s.GetIMConfig("dingtalk")
		if err != nil {
			return err
		}
		if existing != nil {
			log.Println("DingTalk config already exists, skipping")
		} else {
			configData := model.JSONMap{
				"app_key":          cfg.DingTalk.AppKey,
				"app_secret":       cfg.DingTalk.AppSecret,
				"agent_id":         cfg.DingTalk.AgentID,
				"card_template_id": cfg.DingTalk.CardTemplateID,
			}
			imConfig := &model.IMConfig{
				Platform:   "dingtalk",
				Enabled:    cfg.DingTalk.Enabled,
				ConfigData: configData,
			}
			if err := s.SaveIMConfig(imConfig); err != nil {
				return err
			}
			log.Println("Migrated DingTalk config")
		}
	}

	// 迁移飞书配置
	if cfg.Feishu.AppID != "" || cfg.Feishu.AppSecret != "" {
		existing, err := s.GetIMConfig("feishu")
		if err != nil {
			return err
		}
		if existing != nil {
			log.Println("Feishu config already exists, skipping")
		} else {
			configData := model.JSONMap{
				"app_id":     cfg.Feishu.AppID,
				"app_secret": cfg.Feishu.AppSecret,
			}
			imConfig := &model.IMConfig{
				Platform:   "feishu",
				Enabled:    cfg.Feishu.Enabled,
				ConfigData: configData,
			}
			if err := s.SaveIMConfig(imConfig); err != nil {
				return err
			}
			log.Println("Migrated Feishu config")
		}
	}

	// 迁移企业微信配置
	if cfg.Wecom.Token != "" || cfg.Wecom.EncodingAESKey != "" {
		existing, err := s.GetIMConfig("wecom")
		if err != nil {
			return err
		}
		if existing != nil {
			log.Println("Wecom config already exists, skipping")
		} else {
			configData := model.JSONMap{
				"token":            cfg.Wecom.Token,
				"encoding_aes_key": cfg.Wecom.EncodingAESKey,
			}
			imConfig := &model.IMConfig{
				Platform:   "wecom",
				Enabled:    cfg.Wecom.Enabled,
				ConfigData: configData,
			}
			if err := s.SaveIMConfig(imConfig); err != nil {
				return err
			}
			log.Println("Migrated Wecom config")
		}
	}

	return nil
}

// migrateCICDConfigs 迁移CICD配置
func (s *ConfigService) migrateCICDConfigs(cicdCfg config.CICDConfig) error {
	// 迁移 Jenkins 配置
	if cicdCfg.Jenkins.URL != "" {
		existing, err := s.GetCICDConfig("jenkins")
		if err != nil {
			return err
		}
		if existing != nil {
			log.Println("Jenkins config already exists, skipping")
		} else {
			jenkins := &model.CICDConfig{
				Platform: "jenkins",
				Enabled:  cicdCfg.Jenkins.Enabled,
				URL:      cicdCfg.Jenkins.URL,
				Username: cicdCfg.Jenkins.Username,
				Token:    cicdCfg.Jenkins.Token,
			}
			if err := s.SaveCICDConfig(jenkins); err != nil {
				return err
			}
			log.Println("Migrated Jenkins config")
		}
	}

	return nil
}

// migrateSystemConfigs 迁移系统配置
func (s *ConfigService) migrateSystemConfigs(cfg *config.Config) error {
	configs := map[string]interface{}{
		model.ConfigKeyServerHTTPEnabled:                  cfg.Server.HTTP.Enabled,
		model.ConfigKeyServerHTTPPort:                     cfg.Server.HTTP.Port,
		model.ConfigKeyServerMCPEnabled:                   cfg.Server.MCP.Enabled,
		model.ConfigKeyServerMCPPort:                      cfg.Server.MCP.Port,
		model.ConfigKeyServerMCPAutoRegisterExternalTools: cfg.Server.MCP.AutoRegisterExternalTools,
		model.ConfigKeyServerMCPToolNameFormat:            cfg.Server.MCP.ToolNameFormat,
		model.ConfigKeyAuthEnabled:                        cfg.Auth.Enabled,
		model.ConfigKeyAuthType:                           cfg.Auth.Type,
		model.ConfigKeyCacheEnabled:                       cfg.Cache.Enabled,
		model.ConfigKeyCacheType:                          cfg.Cache.Type,
		model.ConfigKeyCacheTTL:                           cfg.Cache.TTL,
	}

	// 特殊处理 tokens (数组转JSON)
	if len(cfg.Auth.Tokens) > 0 {
		tokensJSON, err := json.Marshal(cfg.Auth.Tokens)
		if err != nil {
			return err
		}
		configs[model.ConfigKeyAuthTokens] = string(tokensJSON)
	}

	for key, value := range configs {
		existing, err := s.GetSystemConfig(key)
		if err != nil {
			return err
		}
		if existing != nil {
			log.Printf("System config already exists: %s, skipping", key)
			continue
		}

		valueStr := fmt.Sprintf("%v", value)
		if err := s.SetSystemConfig(key, valueStr, ""); err != nil {
			return err
		}
		log.Printf("Migrated system config: %s", key)
	}

	return nil
}

// LoadConfigFromDB 从数据库加载配置到内存结构
func (s *ConfigService) LoadConfigFromDB() (*config.Config, error) {
	cfg := &config.Config{}

	// 1. 加载 LLM 配置
	llmConfig, err := s.GetLLMConfig()
	if err != nil {
		return nil, err
	}
	if llmConfig != nil {
		cfg.LLM = config.LLMConfig{
			Enabled: llmConfig.Enabled,
			Model:   llmConfig.Model,
			APIKey:  llmConfig.APIKey,
			BaseURL: llmConfig.BaseURL,
		}
	}

	// 2. 加载云厂商配置
	providers, err := s.LoadProvidersFromDB()
	if err != nil {
		return nil, err
	}
	cfg.Providers = providers

	// 3. 加载 IM 配置
	if err := s.LoadIMConfigsFromDB(cfg); err != nil {
		return nil, err
	}

	// 4. 加载 CICD 配置
	if err := s.LoadCICDConfigsFromDB(cfg); err != nil {
		return nil, err
	}

	// 5. 加载系统配置
	if err := s.LoadSystemConfigsFromDB(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// LoadProvidersFromDB 从数据库加载云厂商配置
func (s *ConfigService) LoadProvidersFromDB() (config.ProvidersConfig, error) {
	providers := config.ProvidersConfig{}

	// 加载阿里云账号
	aliyunAccounts, err := s.ListProviderAccounts("aliyun")
	if err != nil {
		return providers, err
	}
	for _, acc := range aliyunAccounts {
		providers.Aliyun = append(providers.Aliyun, config.ProviderConfig{
			Name:    acc.Name,
			Enabled: acc.Enabled,
			AK:      acc.AccessKey,
			SK:      acc.SecretKey,
			Regions: acc.Regions,
		})
	}

	// 加载腾讯云账号
	tencentAccounts, err := s.ListProviderAccounts("tencent")
	if err != nil {
		return providers, err
	}
	for _, acc := range tencentAccounts {
		providers.Tencent = append(providers.Tencent, config.ProviderConfig{
			Name:    acc.Name,
			Enabled: acc.Enabled,
			AK:      acc.AccessKey,
			SK:      acc.SecretKey,
			Regions: acc.Regions,
		})
	}

	return providers, nil
}

// LoadIMConfigsFromDB 从数据库加载IM配置
func (s *ConfigService) LoadIMConfigsFromDB(cfg *config.Config) error {
	// 加载钉钉配置
	dingtalk, err := s.GetIMConfig("dingtalk")
	if err != nil {
		return err
	}
	if dingtalk != nil {
		cfg.DingTalk = config.DingTalkConfig{
			Enabled:        dingtalk.Enabled,
			AppKey:         getStringFromJSONMap(dingtalk.ConfigData, "app_key"),
			AppSecret:      getStringFromJSONMap(dingtalk.ConfigData, "app_secret"),
			AgentID:        getStringFromJSONMap(dingtalk.ConfigData, "agent_id"),
			CardTemplateID: getStringFromJSONMap(dingtalk.ConfigData, "card_template_id"),
		}
	}

	// 加载飞书配置
	feishu, err := s.GetIMConfig("feishu")
	if err != nil {
		return err
	}
	if feishu != nil {
		cfg.Feishu = config.FeishuConfig{
			Enabled:   feishu.Enabled,
			AppID:     getStringFromJSONMap(feishu.ConfigData, "app_id"),
			AppSecret: getStringFromJSONMap(feishu.ConfigData, "app_secret"),
		}
	}

	// 加载企业微信配置
	wecom, err := s.GetIMConfig("wecom")
	if err != nil {
		return err
	}
	if wecom != nil {
		cfg.Wecom = config.WecomConfig{
			Enabled:        wecom.Enabled,
			Token:          getStringFromJSONMap(wecom.ConfigData, "token"),
			EncodingAESKey: getStringFromJSONMap(wecom.ConfigData, "encoding_aes_key"),
		}
	}

	return nil
}

// LoadCICDConfigsFromDB 从数据库加载CICD配置
func (s *ConfigService) LoadCICDConfigsFromDB(cfg *config.Config) error {
	jenkins, err := s.GetCICDConfig("jenkins")
	if err != nil {
		return err
	}
	if jenkins != nil {
		cfg.CICD.Jenkins = config.JenkinsConfig{
			Enabled:  jenkins.Enabled,
			URL:      jenkins.URL,
			Username: jenkins.Username,
			Token:    jenkins.Token,
		}
	}

	return nil
}

// LoadSystemConfigsFromDB 从数据库加载系统配置
func (s *ConfigService) LoadSystemConfigsFromDB(cfg *config.Config) error {
	// HTTP 配置
	if val, err := s.getSystemConfigBool(model.ConfigKeyServerHTTPEnabled); err == nil {
		cfg.Server.HTTP.Enabled = val
	}
	if val, err := s.getSystemConfigInt(model.ConfigKeyServerHTTPPort); err == nil {
		cfg.Server.HTTP.Port = val
	}

	// MCP 配置
	if val, err := s.getSystemConfigBool(model.ConfigKeyServerMCPEnabled); err == nil {
		cfg.Server.MCP.Enabled = val
	}
	if val, err := s.getSystemConfigInt(model.ConfigKeyServerMCPPort); err == nil {
		cfg.Server.MCP.Port = val
	}
	if val, err := s.getSystemConfigBool(model.ConfigKeyServerMCPAutoRegisterExternalTools); err == nil {
		cfg.Server.MCP.AutoRegisterExternalTools = val
	}
	if val, err := s.getSystemConfigString(model.ConfigKeyServerMCPToolNameFormat); err == nil {
		cfg.Server.MCP.ToolNameFormat = val
	}

	// Auth 配置
	if val, err := s.getSystemConfigBool(model.ConfigKeyAuthEnabled); err == nil {
		cfg.Auth.Enabled = val
	}
	if val, err := s.getSystemConfigString(model.ConfigKeyAuthType); err == nil {
		cfg.Auth.Type = val
	}
	if val, err := s.getSystemConfigString(model.ConfigKeyAuthTokens); err == nil && val != "" {
		var tokens []string
		if err := json.Unmarshal([]byte(val), &tokens); err == nil {
			cfg.Auth.Tokens = tokens
		}
	}

	// Cache 配置
	if val, err := s.getSystemConfigBool(model.ConfigKeyCacheEnabled); err == nil {
		cfg.Cache.Enabled = val
	}
	if val, err := s.getSystemConfigString(model.ConfigKeyCacheType); err == nil {
		cfg.Cache.Type = val
	}
	if val, err := s.getSystemConfigInt(model.ConfigKeyCacheTTL); err == nil {
		cfg.Cache.TTL = val
	}

	return nil
}

// 辅助函数

func getStringFromJSONMap(m model.JSONMap, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func (s *ConfigService) getSystemConfigString(key string) (string, error) {
	cfg, err := s.GetSystemConfig(key)
	if err != nil || cfg == nil {
		return "", err
	}
	return cfg.ConfigValue, nil
}

func (s *ConfigService) getSystemConfigInt(key string) (int, error) {
	str, err := s.getSystemConfigString(key)
	if err != nil {
		return 0, err
	}
	var val int
	_, err = fmt.Sscanf(str, "%d", &val)
	return val, err
}

func (s *ConfigService) getSystemConfigBool(key string) (bool, error) {
	str, err := s.getSystemConfigString(key)
	if err != nil {
		return false, err
	}
	return str == "true", nil
}
