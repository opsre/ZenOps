package database

import (
	"encoding/json"
	"fmt"
	"time"

	"cnb.cool/zhiqiangwang/pkg/logx"
	"gorm.io/gorm"

	"github.com/eryajf/zenops/internal/model"
)

// OldLLMConfig 旧的LLM配置结构(用于迁移)
type OldLLMConfig struct {
	ID        uint            `gorm:"primaryKey"`
	Providers json.RawMessage `gorm:"type:text"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// OldLLMProviderInstance 旧的LLM提供商实例结构
type OldLLMProviderInstance struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Enabled  bool   `json:"enabled"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
	APIKey   string `json:"api_key"`
	BaseURL  string `json:"base_url"`
}

func (OldLLMConfig) TableName() string {
	return "llm_config"
}

// AutoMigrate 自动迁移数据库表结构
func AutoMigrate(db *gorm.DB) error {
	// 先检查是否需要迁移旧的 llm_config 数据
	if err := migrateLLMConfig(db); err != nil {
		logx.Error("Failed to migrate LLM config: %v", err)
		// 不返回错误，继续其他迁移
	}

	// 迁移所有配置表
	err := db.AutoMigrate(
		&model.User{},
		&model.LLMConfig{},
		&model.ProviderAccount{},
		&model.IMConfig{},
		&model.CICDConfig{},
		&model.MCPServer{},
		&model.MCPTool{},
		&model.MCPLog{},
		&model.ChatLog{},
		&model.Conversation{},
		&model.SystemConfig{},
	)
	if err != nil {
		return fmt.Errorf("failed to migrate tables: %w", err)
	}

	// 创建默认用户
	if err := createDefaultUser(db); err != nil {
		logx.Error("Failed to create default user: %v", err)
		// 不返回错误，继续启动
	}

	return nil
}

// createDefaultUser 创建默认管理员用户
func createDefaultUser(db *gorm.DB) error {
	// 检查是否已存在用户
	var count int64
	if err := db.Model(&model.User{}).Count(&count).Error; err != nil {
		return err
	}

	// 如果已有用户，不创建
	if count > 0 {
		logx.Info("Users already exist, skipping default user creation")
		return nil
	}

	// 创建默认管理员用户
	defaultUser := &model.User{
		Username: "admin",
		Nickname: "管理员",
		Email:    "admin@zenops.local",
		Roles:    "admin,user",
		Enabled:  true,
	}

	// 设置默认密码: admin123
	if err := defaultUser.SetPassword("admin123"); err != nil {
		return fmt.Errorf("failed to set default password: %w", err)
	}

	// 创建用户
	if err := db.Create(defaultUser).Error; err != nil {
		return fmt.Errorf("failed to create default user: %w", err)
	}

	logx.Info("✅ Default admin user created successfully (username: admin, password: admin123)")
	return nil
}

// migrateLLMConfig 迁移旧的LLM配置到新结构
func migrateLLMConfig(db *gorm.DB) error {
	// 检查是否存在 providers 列（旧结构）
	var hasProvidersColumn bool
	rows, err := db.Raw("PRAGMA table_info(llm_config)").Rows()
	if err != nil {
		return nil // 表不存在，不需要迁移
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dfltValue interface{}
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
			continue
		}
		if name == "providers" {
			hasProvidersColumn = true
			break
		}
	}

	if !hasProvidersColumn {
		logx.Info("LLM config already in new format, skipping migration")
		return nil // 已经是新结构
	}

	logx.Info("Migrating LLM config from old format to new format...")

	// 读取旧数据
	var oldConfigs []OldLLMConfig
	if err := db.Find(&oldConfigs).Error; err != nil {
		return fmt.Errorf("failed to read old LLM configs: %w", err)
	}

	if len(oldConfigs) == 0 {
		logx.Info("No old LLM configs to migrate")
		// 删除旧表，创建新表
		if err := db.Exec("DROP TABLE IF EXISTS llm_config").Error; err != nil {
			return fmt.Errorf("failed to drop old table: %w", err)
		}
		return nil
	}

	// 解析并收集所有provider实例
	var newConfigs []model.LLMConfig
	for _, oldConfig := range oldConfigs {
		var providers []OldLLMProviderInstance
		if err := json.Unmarshal(oldConfig.Providers, &providers); err != nil {
			logx.Error("Failed to parse providers JSON: %v", err)
			continue
		}

		for _, p := range providers {
			// 如果旧数据的 created_at 是零值，使用当前时间
			createdAt := oldConfig.CreatedAt
			if createdAt.IsZero() {
				createdAt = time.Now()
			}
			updatedAt := oldConfig.UpdatedAt
			if updatedAt.IsZero() {
				updatedAt = time.Now()
			}

			newConfigs = append(newConfigs, model.LLMConfig{
				Name:      p.Name,
				Enabled:   p.Enabled,
				Provider:  p.Provider,
				Model:     p.Model,
				APIKey:    p.APIKey,
				BaseURL:   p.BaseURL,
				CreatedAt: createdAt,
				UpdatedAt: updatedAt,
			})
		}
	}

	// 删除旧表
	if err := db.Exec("DROP TABLE IF EXISTS llm_config").Error; err != nil {
		return fmt.Errorf("failed to drop old table: %w", err)
	}

	// 创建新表
	if err := db.AutoMigrate(&model.LLMConfig{}); err != nil {
		return fmt.Errorf("failed to create new table: %w", err)
	}

	// 插入新数据
	if len(newConfigs) > 0 {
		if err := db.Create(&newConfigs).Error; err != nil {
			return fmt.Errorf("failed to insert new configs: %w", err)
		}
		logx.Info("Successfully migrated %d LLM configs", len(newConfigs))
	}

	return nil
}
