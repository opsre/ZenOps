package database

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/eryajf/zenops/internal/model"
)

// AutoMigrate 自动迁移数据库表结构
func AutoMigrate(db *gorm.DB) error {
	// 迁移所有配置表
	err := db.AutoMigrate(
		&model.LLMConfig{},
		&model.ProviderAccount{},
		&model.IMConfig{},
		&model.CICDConfig{},
		&model.MCPServer{},
		&model.MCPTool{},
		&model.MCPLog{},
		&model.ChatLog{},
		&model.SystemConfig{},
	)
	if err != nil {
		return fmt.Errorf("failed to migrate tables: %w", err)
	}

	return nil
}
