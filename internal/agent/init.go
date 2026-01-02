package agent

import (
	"context"
	"fmt"
	"time"

	"cnb.cool/zhiqiangwang/pkg/logx"
	"github.com/eryajf/zenops/internal/config"
	"github.com/eryajf/zenops/internal/imcp"
	"github.com/eryajf/zenops/internal/knowledge"
	"github.com/eryajf/zenops/internal/memory"
	"gorm.io/gorm"
)

// Agent 全局 Agent 实例
type Agent struct {
	Orchestrator  *Orchestrator
	StreamHandler *StreamHandler
}

var globalAgent *Agent

// Initialize 初始化 Agent 系统
// 包括: Memory Manager, Knowledge Retriever, Agent Orchestrator, Stream Handler
func Initialize(ctx context.Context, db *gorm.DB, mcpServer *imcp.MCPServer, cfg *config.Config) (*Agent, error) {
	logx.Info("🤖 Initializing Agent System...")

	// 1. 初始化 Memory Manager
	memoryMgr, err := initializeMemoryManager(ctx, db, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize memory manager: %w", err)
	}
	logx.Info("✅ Memory Manager initialized")

	// 2. 初始化 Knowledge Retriever
	knowledgeRet := knowledge.NewRetriever(db, false, 3)
	logx.Info("✅ Knowledge Retriever initialized (FTS5 mode, max_results=3)")

	// 3. 初始化 Agent Orchestrator
	orchestrator := NewOrchestrator(memoryMgr, knowledgeRet, mcpServer)
	logx.Info("✅ Agent Orchestrator initialized (max_iterations=10)")

	// 4. 初始化 Stream Handler
	streamHandler, err := initializeStreamHandler(orchestrator, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize stream handler: %w", err)
	}
	logx.Info("✅ Stream Handler initialized")

	agent := &Agent{
		Orchestrator:  orchestrator,
		StreamHandler: streamHandler,
	}

	globalAgent = agent
	logx.Info("🎉 Agent System initialization completed!")

	return agent, nil
}

// initializeMemoryManager 初始化内存管理器
func initializeMemoryManager(ctx context.Context, db *gorm.DB, cfg *config.Config) (*memory.Manager, error) {
	var redisCache *memory.RedisCache

	// 检查是否启用 Redis
	if cfg.Cache.Enabled && cfg.Cache.Type == "redis" {
		logx.Info("📦 Initializing Redis cache...")

		// 创建 Redis 缓存
		addr := fmt.Sprintf("%s:%d", cfg.Cache.Redis.Host, cfg.Cache.Redis.Port)
		ttl := time.Duration(cfg.Cache.TTL) * time.Second

		var err error
		redisCache, err = memory.NewRedisCache(addr, cfg.Cache.Redis.Password, cfg.Cache.Redis.DB, ttl)
		if err != nil {
			logx.Warn("⚠️  Redis connection failed: %v, falling back to SQLite-only mode", err)
			redisCache = nil
		} else {
			logx.Info("✅ Redis cache connected: %s (DB: %d, TTL: %ds)",
				addr, cfg.Cache.Redis.DB, cfg.Cache.TTL)
		}
	}

	// 创建 Memory Manager
	memoryMgr := memory.NewManager(db, redisCache)
	return memoryMgr, nil
}

// initializeStreamHandler 初始化流式处理器
func initializeStreamHandler(orchestrator *Orchestrator, cfg *config.Config) (*StreamHandler, error) {
	// 构建 Model Config
	modelConfig := ModelConfig{
		Model:   cfg.LLM.Model,
		APIKey:  cfg.LLM.APIKey,
		BaseURL: cfg.LLM.BaseURL,
	}

	// 创建 Stream Handler
	streamHandler, err := NewStreamHandler(orchestrator, modelConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create stream handler: %w", err)
	}

	return streamHandler, nil
}

// GetGlobalAgent 获取全局 Agent 实例
func GetGlobalAgent() *Agent {
	return globalAgent
}
