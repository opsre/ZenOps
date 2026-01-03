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
	"github.com/eryajf/zenops/internal/service"
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
	memoryMgr, embeddingService, err := initializeMemoryManager(ctx, db, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize memory manager: %w", err)
	}
	logx.Info("✅ Memory Manager initialized")

	// 2. 初始化 Knowledge Retriever
	knowledgeRet := knowledge.NewRetriever(db, false, 3)
	// 如果有 embedding service，启用向量检索
	if embeddingService != nil {
		knowledgeRet.SetEmbeddingService(embeddingService)
	} else {
		logx.Info("✅ Knowledge Retriever initialized (FTS5 mode only, max_results=3)")
	}

	// 3. 初始化 Agent Orchestrator
	orchestrator := NewOrchestrator(memoryMgr, knowledgeRet, mcpServer)
	logx.Info("✅ Agent Orchestrator initialized (max_iterations=10)")

	// 4. 初始化 Stream Handler
	streamHandler, err := initializeStreamHandler(ctx, db, orchestrator, cfg)
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
func initializeMemoryManager(ctx context.Context, db *gorm.DB, cfg *config.Config) (*memory.Manager, *memory.EmbeddingService, error) {
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

	// 初始化 Embedding 服务（如果启用语义缓存）
	var embeddingService *memory.EmbeddingService
	var semanticConfig *memory.SemanticCacheConfig

	if cfg.SemanticCache.Enabled {
		logx.Info("📦 Initializing Semantic Cache...")

		// 从数据库获取 Embedding 模型配置
		configService := service.NewConfigService()
		embConfig, err := configService.GetDefaultEmbeddingConfig()

		if err != nil || embConfig == nil {
			logx.Warn("⚠️ No embedding model configured, semantic cache disabled")
		} else {
			embeddingService, err = memory.NewEmbeddingService(&memory.EmbeddingConfig{
				APIKey:  embConfig.APIKey,
				BaseURL: embConfig.BaseURL,
				Model:   embConfig.Model,
			}, redisCache)

			if err != nil {
				logx.Warn("⚠️ Failed to init embedding service: %v, semantic cache disabled", err)
				embeddingService = nil
			} else {
				logx.Info("✅ Embedding service initialized: model=%s", embConfig.Model)
			}
		}

		// 设置语义缓存配置
		threshold := cfg.SemanticCache.SimilarityThreshold
		if threshold <= 0 {
			threshold = 0.85 // 默认阈值
		}
		maxCandidates := cfg.SemanticCache.MaxCandidates
		if maxCandidates <= 0 {
			maxCandidates = 100 // 默认候选数
		}

		semanticConfig = &memory.SemanticCacheConfig{
			Enabled:             embeddingService != nil,
			SimilarityThreshold: threshold,
			MaxCandidates:       maxCandidates,
		}

		if semanticConfig.Enabled {
			logx.Info("✅ Semantic cache enabled: threshold=%.2f, max_candidates=%d",
				semanticConfig.SimilarityThreshold, semanticConfig.MaxCandidates)
		}
	}

	// 创建 Memory Manager
	memoryMgr := memory.NewManager(db, redisCache, embeddingService, semanticConfig)
	return memoryMgr, embeddingService, nil
}

// initializeStreamHandler 初始化流式处理器
func initializeStreamHandler(ctx context.Context, db *gorm.DB, orchestrator *Orchestrator, cfg *config.Config) (*StreamHandler, error) {
	// 使用 config.yaml 作为回退配置
	// StreamHandler 会在每次对话时动态读取数据库配置
	fallbackModelConfig := ModelConfig{
		Model:   cfg.LLM.Model,
		APIKey:  cfg.LLM.APIKey,
		BaseURL: cfg.LLM.BaseURL,
	}
	logx.Info("📦 LLM fallback config from config.yaml: model=%s, base_url=%s",
		cfg.LLM.Model, cfg.LLM.BaseURL)

	// 创建 Stream Handler（会在每次对话时动态读取最新配置）
	streamHandler, err := NewStreamHandler(orchestrator, fallbackModelConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create stream handler: %w", err)
	}

	return streamHandler, nil
}

// GetGlobalAgent 获取全局 Agent 实例
func GetGlobalAgent() *Agent {
	return globalAgent
}
