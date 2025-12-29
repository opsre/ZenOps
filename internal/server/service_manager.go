package server

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cnb.cool/zhiqiangwang/pkg/logx"
	"github.com/eryajf/zenops/internal/config"
	"github.com/eryajf/zenops/internal/imcp"
	"github.com/eryajf/zenops/internal/model"
	"github.com/eryajf/zenops/internal/service"
	"github.com/eryajf/zenops/internal/wecom"
)

// ServiceManager 服务管理器，用于管理 IM 服务的生命周期
type ServiceManager struct {
	mu sync.RWMutex

	// 配置
	config *config.Config

	// MCP Server (共享)
	mcpServer *imcp.MCPServer

	// 各服务实例
	dingtalkService *DingTalkService
	feishuService   *FeishuStreamServer
	wecomHandler    *wecom.MessageHandler

	// 服务运行状态
	dingtalkRunning bool
	feishuRunning   bool
	wecomRunning    bool

	// 取消函数
	dingtalkCancel context.CancelFunc
	feishuCancel   context.CancelFunc

	// 配置服务
	configService *service.ConfigService
}

// NewServiceManager 创建服务管理器
func NewServiceManager(cfg *config.Config, mcpServer *imcp.MCPServer) *ServiceManager {
	return &ServiceManager{
		config:        cfg,
		mcpServer:     mcpServer,
		configService: service.NewConfigService(),
	}
}

// StartDingTalk 启动钉钉服务
func (sm *ServiceManager) StartDingTalk(ctx context.Context) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.dingtalkRunning {
		return fmt.Errorf("dingtalk service is already running")
	}

	// 从数据库获取最新配置
	imConfig, err := sm.configService.GetIMConfig("dingtalk")
	if err != nil {
		return fmt.Errorf("failed to get dingtalk config: %w", err)
	}

	if imConfig == nil {
		return fmt.Errorf("dingtalk config not found")
	}

	// 验证配置
	if imConfig.AppID == "" || imConfig.AppKey == "" {
		return fmt.Errorf("dingtalk config is incomplete: app_id and app_key are required")
	}

	// 更新内存中的配置
	sm.config.DingTalk.Enabled = true
	sm.config.DingTalk.AppKey = imConfig.AppID
	sm.config.DingTalk.AppSecret = imConfig.AppKey
	sm.config.DingTalk.CardTemplateID = imConfig.TemplateID

	// 创建钉钉服务
	dingtalkService, err := NewDingTalkService(sm.config, sm.mcpServer)
	if err != nil {
		return fmt.Errorf("failed to create dingtalk service: %w", err)
	}

	// 创建独立的 context（不依赖 HTTP 请求的 context）
	serviceCtx, cancel := context.WithCancel(context.Background())
	sm.dingtalkCancel = cancel
	sm.dingtalkService = dingtalkService

	// 启动服务
	go func() {
		if err := dingtalkService.Start(serviceCtx); err != nil {
			logx.Error("DingTalk service error: %v", err)
			sm.mu.Lock()
			sm.dingtalkRunning = false
			sm.mu.Unlock()
		}
	}()

	sm.dingtalkRunning = true
	logx.Info("🤖 DingTalk service started successfully")
	return nil
}

// StopDingTalk 停止钉钉服务
func (sm *ServiceManager) StopDingTalk(ctx context.Context) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.dingtalkRunning {
		// 服务未运行时，静默返回成功（不是错误）
		logx.Debug("DingTalk service is not running, nothing to stop")
		return nil
	}

	// 取消 context
	if sm.dingtalkCancel != nil {
		sm.dingtalkCancel()
	}

	// 停止服务
	if sm.dingtalkService != nil {
		if err := sm.dingtalkService.Stop(ctx); err != nil {
			logx.Warn("Failed to stop dingtalk service gracefully: %v", err)
		}
	}

	sm.dingtalkRunning = false
	sm.dingtalkService = nil
	sm.dingtalkCancel = nil
	sm.config.DingTalk.Enabled = false

	logx.Info("🤖 DingTalk service stopped successfully")
	return nil
}

// StartFeishu 启动飞书服务
func (sm *ServiceManager) StartFeishu(ctx context.Context) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.feishuRunning {
		return fmt.Errorf("feishu service is already running")
	}

	// 从数据库获取最新配置
	imConfig, err := sm.configService.GetIMConfig("feishu")
	if err != nil {
		return fmt.Errorf("failed to get feishu config: %w", err)
	}

	if imConfig == nil {
		return fmt.Errorf("feishu config not found")
	}

	// 验证配置
	if imConfig.AppID == "" || imConfig.AppKey == "" {
		return fmt.Errorf("feishu config is incomplete: app_id and app_secret are required")
	}

	// 更新内存中的配置
	sm.config.Feishu.Enabled = true
	sm.config.Feishu.AppID = imConfig.AppID
	sm.config.Feishu.AppSecret = imConfig.AppKey

	// 创建飞书服务
	feishuService, err := NewFeishuStreamServer(sm.config, sm.mcpServer)
	if err != nil {
		return fmt.Errorf("failed to create feishu service: %w", err)
	}

	if feishuService == nil {
		return fmt.Errorf("feishu service creation returned nil")
	}

	sm.feishuService = feishuService

	// 启动服务
	go func() {
		if err := feishuService.Start(); err != nil {
			logx.Error("Feishu service error: %v", err)
			sm.mu.Lock()
			sm.feishuRunning = false
			sm.mu.Unlock()
		}
	}()

	sm.feishuRunning = true
	logx.Info("🐦 Feishu service started successfully")
	return nil
}

// StopFeishu 停止飞书服务
func (sm *ServiceManager) StopFeishu(ctx context.Context) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.feishuRunning {
		// 服务未运行时，静默返回成功（不是错误）
		logx.Debug("Feishu service is not running, nothing to stop")
		return nil
	}

	// 停止服务
	if sm.feishuService != nil {
		if err := sm.feishuService.Stop(); err != nil {
			logx.Warn("Failed to stop feishu service gracefully: %v", err)
		}
	}

	sm.feishuRunning = false
	sm.feishuService = nil
	sm.config.Feishu.Enabled = false

	logx.Info("🐦 Feishu service stopped successfully")
	return nil
}

// StartWecom 启动企业微信服务
func (sm *ServiceManager) StartWecom(ctx context.Context) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.wecomRunning {
		return fmt.Errorf("wecom service is already running")
	}

	// 从数据库获取最新配置
	imConfig, err := sm.configService.GetIMConfig("wecom")
	if err != nil {
		return fmt.Errorf("failed to get wecom config: %w", err)
	}

	if imConfig == nil {
		return fmt.Errorf("wecom config not found")
	}

	// 验证配置
	if imConfig.AppID == "" || imConfig.AppKey == "" {
		return fmt.Errorf("wecom config is incomplete: bot_token and aes_key are required")
	}

	// 更新内存中的配置
	sm.config.Wecom.Enabled = true
	sm.config.Wecom.Token = imConfig.AppID
	sm.config.Wecom.EncodingAESKey = imConfig.AppKey

	// 创建企业微信消息处理器
	handler, err := wecom.NewMessageHandler(sm.config, sm.mcpServer)
	if err != nil {
		return fmt.Errorf("failed to create wecom handler: %w", err)
	}

	sm.wecomHandler = handler
	sm.wecomRunning = true

	logx.Info("💬 Wecom service started successfully")
	return nil
}

// StopWecom 停止企业微信服务
func (sm *ServiceManager) StopWecom(ctx context.Context) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.wecomRunning {
		// 服务未运行时，静默返回成功（不是错误）
		logx.Debug("Wecom service is not running, nothing to stop")
		return nil
	}

	sm.wecomHandler = nil
	sm.wecomRunning = false
	sm.config.Wecom.Enabled = false

	logx.Info("💬 Wecom service stopped successfully")
	return nil
}

// GetWecomHandler 获取企业微信处理器
func (sm *ServiceManager) GetWecomHandler() *wecom.MessageHandler {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.wecomHandler
}

// IsWecomRunning 检查企业微信服务是否运行
func (sm *ServiceManager) IsWecomRunning() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.wecomRunning
}

// GetServiceStatus 获取服务状态
func (sm *ServiceManager) GetServiceStatus() map[string]bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return map[string]bool{
		"dingtalk": sm.dingtalkRunning,
		"feishu":   sm.feishuRunning,
		"wecom":    sm.wecomRunning,
	}
}

// ToggleService 切换服务状态
func (sm *ServiceManager) ToggleService(ctx context.Context, platform string, enabled bool) error {
	switch platform {
	case "dingtalk":
		if enabled {
			return sm.StartDingTalk(ctx)
		}
		return sm.StopDingTalk(ctx)
	case "feishu":
		if enabled {
			return sm.StartFeishu(ctx)
		}
		return sm.StopFeishu(ctx)
	case "wecom":
		if enabled {
			return sm.StartWecom(ctx)
		}
		return sm.StopWecom(ctx)
	default:
		return fmt.Errorf("unknown platform: %s", platform)
	}
}

// SyncWithDatabase 从数据库同步服务状态
func (sm *ServiceManager) SyncWithDatabase(ctx context.Context) error {
	platforms := []string{"dingtalk", "feishu", "wecom"}

	for _, platform := range platforms {
		imConfig, err := sm.configService.GetIMConfig(platform)
		if err != nil {
			logx.Warn("Failed to get %s config: %v", platform, err)
			continue
		}

		if imConfig != nil && imConfig.Enabled {
			// 如果数据库中启用了，尝试启动服务
			if err := sm.ToggleService(ctx, platform, true); err != nil {
				logx.Warn("Failed to start %s service: %v", platform, err)
			}
		}
	}

	return nil
}

// UpdateAndToggle 更新配置并切换服务状态
func (sm *ServiceManager) UpdateAndToggle(ctx context.Context, imConfig *model.IMConfig) error {
	// 先保存配置到数据库
	if err := sm.configService.SaveIMConfig(imConfig); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// 给服务一点时间初始化
	time.Sleep(100 * time.Millisecond)

	// 切换服务状态
	return sm.ToggleService(ctx, imConfig.Platform, imConfig.Enabled)
}
