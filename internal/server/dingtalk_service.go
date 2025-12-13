package server

import (
	"context"
	"fmt"

	"cnb.cool/zhiqiangwang/pkg/logx"
	"github.com/eryajf/zenops/internal/config"
	"github.com/eryajf/zenops/internal/imcp"
)

// DingTalkService 钉钉服务
type DingTalkService struct {
	config        *config.Config
	mcpServer     *imcp.MCPServer
	streamHandler *DingTalkStreamHandler
}

// NewDingTalkService 创建钉钉服务
func NewDingTalkService(cfg *config.Config, mcpServer *imcp.MCPServer) (*DingTalkService, error) {
	if !cfg.DingTalk.Enabled {
		return nil, fmt.Errorf("dingtalk is not enabled")
	}

	return &DingTalkService{
		config:    cfg,
		mcpServer: mcpServer,
	}, nil
}

// Start 启动钉钉服务
func (s *DingTalkService) Start(ctx context.Context) error {
	return s.startStreamMode(ctx)
}

// Stop 停止钉钉服务
func (s *DingTalkService) Stop(ctx context.Context) error {
	if s.streamHandler != nil {
		return s.streamHandler.Stop()
	}
	return nil
}

// startStreamMode 启动Stream模式
func (s *DingTalkService) startStreamMode(ctx context.Context) error {
	// 创建卡片客户端
	cardClient, err := NewDingTalkStreamClient(
		s.config.DingTalk.AppKey,
		s.config.DingTalk.AppSecret,
		s.config.DingTalk.CardTemplateID,
	)
	if err != nil {
		return fmt.Errorf("failed to create card client: %w", err)
	}

	// 创建Stream处理器
	s.streamHandler = NewDingTalkStreamHandler(s.config, cardClient, s.mcpServer)

	logx.Info("🤖 DingTalk Stream Mode Started Successfully.")

	// 启动Stream客户端
	return s.streamHandler.Start(ctx)
}
