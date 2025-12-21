package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	db   *gorm.DB
	once sync.Once
)

// GetDB 获取数据库连接实例(单例模式)
func GetDB() *gorm.DB {
	once.Do(func() {
		var err error
		db, err = initDB()
		if err != nil {
			log.Fatalf("Failed to initialize database: %v", err)
		}
	})
	return db
}

// initDB 初始化数据库连接
func initDB() (*gorm.DB, error) {
	// 获取数据库文件路径
	dbPath := getDBPath()

	// 确保数据目录存在
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// 连接数据库
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		// 禁用外键(指定外键时不会在sqlite创建真实的外键约束)
		DisableForeignKeyConstraintWhenMigrating: true,
		Logger:                                   logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect sqlite: %w", err)
	}

	// 获取底层数据库连接
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sqlite database object: %w", err)
	}

	// 参见： https://github.com/glebarez/sqlite/issues/52
	// SQLite 只支持单个写入连接
	sqlDB.SetMaxOpenConns(1)

	// 自动迁移数据库表结构
	if err := AutoMigrate(db); err != nil {
		return nil, fmt.Errorf("failed to auto migrate: %w", err)
	}

	log.Printf("Database initialized successfully at: %s", dbPath)
	return db, nil
}

// getDBPath 获取数据库文件路径
func getDBPath() string {
	// 优先使用环境变量
	if dbPath := os.Getenv("ZENOPS_DB_PATH"); dbPath != "" {
		return dbPath
	}

	// 默认使用当前目录下的 data/zenops.db
	return "./data/zenops.db"
}

// Close 关闭数据库连接
func Close() error {
	if db == nil {
		return nil
	}

	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	return sqlDB.Close()
}
