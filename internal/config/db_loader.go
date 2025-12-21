package config

import (
	"log"
)

// ConfigLoader 配置加载器接口
type ConfigLoader interface {
	LoadConfig(configFile string) (*Config, error)
	LoadConfigFromDB() (*Config, error)
	MigrateConfigToDB(cfg *Config) error
}

// DefaultConfigLoader 默认配置加载器
type DefaultConfigLoader struct {
	dbLoader   func() (*Config, error)
	dbMigrator func(*Config) error
}

var defaultLoader *DefaultConfigLoader

// SetDBLoader 设置数据库配置加载函数
func SetDBLoader(loader func() (*Config, error)) {
	if defaultLoader == nil {
		defaultLoader = &DefaultConfigLoader{}
	}
	defaultLoader.dbLoader = loader
}

// SetDBMigrator 设置数据库配置迁移函数
func SetDBMigrator(migrator func(*Config) error) {
	if defaultLoader == nil {
		defaultLoader = &DefaultConfigLoader{}
	}
	defaultLoader.dbMigrator = migrator
}

// LoadConfigWithDB 加载配置(支持数据库)
func LoadConfigWithDB(configFile string) (*Config, error) {
	// 先尝试从数据库加载
	if defaultLoader != nil && defaultLoader.dbLoader != nil {
		cfg, err := defaultLoader.dbLoader()
		if err == nil && cfg != nil {
			log.Println("✓ Configuration loaded from database")
			return cfg, nil
		}
	}

	// 从YAML加载
	log.Println("Loading configuration from YAML file...")
	cfg, err := loadConfigFromYAML(configFile)
	if err != nil {
		return nil, err
	}

	// 尝试迁移到数据库
	if defaultLoader != nil && defaultLoader.dbMigrator != nil {
		if err := defaultLoader.dbMigrator(cfg); err != nil {
			log.Printf("Warning: failed to migrate config to database: %v", err)
		} else {
			log.Println("✓ Configuration migrated to database successfully")
		}
	}

	return cfg, nil
}
