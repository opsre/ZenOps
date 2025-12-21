package config

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// LoadConfig 加载配置文件
// 优先从数据库加载配置,如果数据库为空则从YAML文件加载并自动迁移
func LoadConfig(configFile string) (*Config, error) {
	// 先尝试从数据库加载配置
	dbConfig, err := loadConfigFromDB()
	if err == nil && dbConfig != nil {
		log.Println("Configuration loaded from database")
		return dbConfig, nil
	}

	// 如果数据库加载失败,则从YAML加载
	log.Println("Loading configuration from YAML file...")
	yamlConfig, err := loadConfigFromYAML(configFile)
	if err != nil {
		return nil, err
	}

	// 尝试将YAML配置迁移到数据库
	if err := migrateConfigToDB(yamlConfig); err != nil {
		log.Printf("Warning: failed to migrate config to database: %v", err)
		// 迁移失败不影响启动,继续使用YAML配置
	}

	return yamlConfig, nil
}

// loadConfigFromDB 从数据库加载配置
func loadConfigFromDB() (*Config, error) {
	// 这个函数将在启动时从 service 层调用
	// 这里只是占位,实际加载逻辑在 cmd/root.go 中实现
	return nil, fmt.Errorf("database config loading is handled externally")
}

// migrateConfigToDB 迁移配置到数据库
func migrateConfigToDB(cfg *Config) error {
	// 这个函数将在启动时从 service 层调用
	// 这里只是占位,实际迁移逻辑在 cmd/root.go 中实现
	return fmt.Errorf("config migration is handled externally")
}

// loadConfigFromYAML 从YAML文件加载配置
func loadConfigFromYAML(configFile string) (*Config, error) {
	v := viper.New()

	// 设置配置文件
	if configFile != "" {
		v.SetConfigFile(configFile)
	} else {
		// 默认配置文件搜索路径
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("./configs")
		v.AddConfigPath("$HOME/.zenops")
		v.AddConfigPath("/etc/zenops")
	}

	// 支持环境变量
	v.SetEnvPrefix("ZENOPS")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// 设置默认值
	setDefaults(v)

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		// 如果是找不到配置文件，则使用默认配置
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// 解析配置
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// 替换环境变量
	expandEnvVars(&config)

	return &config, nil
}

// setDefaults 设置默认配置值
func setDefaults(v *viper.Viper) {
	// Server 默认配置
	v.SetDefault("server.http.enabled", true)
	v.SetDefault("server.http.port", 8080)
	v.SetDefault("server.mcp.enabled", false)
	v.SetDefault("server.mcp.port", 8081)

	// Auth 默认配置
	v.SetDefault("auth.enabled", false)
	v.SetDefault("auth.type", "token")

	// Cache 默认配置
	v.SetDefault("cache.enabled", false)
	v.SetDefault("cache.type", "memory")
	v.SetDefault("cache.ttl", 300)
}

// expandEnvVars 展开环境变量
func expandEnvVars(config *Config) {
	// 展开阿里云账号配置中的环境变量
	for i := range config.Providers.Aliyun {
		config.Providers.Aliyun[i].AK = os.ExpandEnv(config.Providers.Aliyun[i].AK)
		config.Providers.Aliyun[i].SK = os.ExpandEnv(config.Providers.Aliyun[i].SK)
	}

	// 展开腾讯云账号配置中的环境变量
	for i := range config.Providers.Tencent {
		config.Providers.Tencent[i].AK = os.ExpandEnv(config.Providers.Tencent[i].AK)
		config.Providers.Tencent[i].SK = os.ExpandEnv(config.Providers.Tencent[i].SK)
	}

	// 展开 CICD 配置中的环境变量
	config.CICD.Jenkins.Username = os.ExpandEnv(config.CICD.Jenkins.Username)
	config.CICD.Jenkins.Token = os.ExpandEnv(config.CICD.Jenkins.Token)

	// 展开 DingTalk 配置中的环境变量
	config.DingTalk.AppKey = os.ExpandEnv(config.DingTalk.AppKey)
	config.DingTalk.AppSecret = os.ExpandEnv(config.DingTalk.AppSecret)
	config.DingTalk.AgentID = os.ExpandEnv(config.DingTalk.AgentID)

	// 展开 Auth 配置中的环境变量
	for i, token := range config.Auth.Tokens {
		config.Auth.Tokens[i] = os.ExpandEnv(token)
	}
}
