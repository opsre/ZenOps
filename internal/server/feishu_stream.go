package server

import (
	"context"
	"sync"
	"time"

	"cnb.cool/zhiqiangwang/pkg/logx"
	"github.com/eryajf/zenops/internal/config"
	"github.com/eryajf/zenops/internal/feishu"
	"github.com/eryajf/zenops/internal/imcp"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"
)

// FeishuStreamServer 飞书 Stream 服务
type FeishuStreamServer struct {
	ctx       context.Context
	cancel    context.CancelFunc
	config    *config.Config
	handler   *feishu.MessageHandler
	msgMap    sync.Map // 消息去重
	wsClient  *larkws.Client
}

// NewFeishuStreamServer 创建飞书 Stream 服务
func NewFeishuStreamServer(cfg *config.Config, mcpServer *imcp.MCPServer) (*FeishuStreamServer, error) {
	if !cfg.Feishu.Enabled {
		logx.Info("Feishu is disabled, skipping initialization")
		return nil, nil
	}

	handler, err := feishu.NewMessageHandler(cfg, mcpServer)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	server := &FeishuStreamServer{
		ctx:     ctx,
		cancel:  cancel,
		config:  cfg,
		handler: handler,
	}

	// 启动消息清理协程
	go server.startMessageCleanup()

	return server, nil
}

// Start 启动飞书 Stream 服务
func (s *FeishuStreamServer) Start() error {
	if s == nil {
		return nil
	}

	logx.Info("Starting Feishu Stream server...")

	// 创建事件处理器
	eventHandler := dispatcher.NewEventDispatcher("", "").
		OnP2MessageReceiveV1(func(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
			return s.handleMessage(ctx, event)
		})

	// 创建 WebSocket 客户端
	s.wsClient = larkws.NewClient(
		s.config.Feishu.AppID,
		s.config.Feishu.AppSecret,
		larkws.WithEventHandler(eventHandler),
	)

	// 启动 WebSocket 连接
	if err := s.wsClient.Start(s.ctx); err != nil {
		return err
	}

	logx.Info("Feishu Stream server started successfully")
	return nil
}

// Stop 停止飞书 Stream 服务
func (s *FeishuStreamServer) Stop() error {
	if s == nil {
		return nil
	}

	logx.Info("Stopping Feishu Stream server...")
	s.cancel()
	logx.Info("Feishu Stream server stopped")
	return nil
}

// handleMessage 处理消息
func (s *FeishuStreamServer) handleMessage(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
	// 消息去重
	messageID := *event.Event.Message.MessageId
	if messageID == "" {
		return nil
	}

	if _, exists := s.msgMap.Load(messageID); exists {
		logx.Debug("Duplicate message ignored: %s", messageID)
		return nil
	}

	s.msgMap.Store(messageID, time.Now().Unix())
	logx.Info("Processing message: id %s, type %s, chat_type %s",
		messageID,
		*event.Event.Message.MessageType,
		*event.Event.Message.ChatType)

	// 只处理文本消息
	if *event.Event.Message.MessageType != "text" {
		logx.Debug("Ignoring non-text message: %s", *event.Event.Message.MessageType)
		return nil
	}

	// 异步处理消息,避免阻塞事件循环
	go func() {
		if err := s.handler.HandleTextMessage(ctx, event); err != nil {
			logx.Error("Failed to handle message: %v", err)
		}
	}()

	return nil
}

// startMessageCleanup 启动消息清理协程
func (s *FeishuStreamServer) startMessageCleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			// 清理 5 分钟前的消息记录
			now := time.Now().Unix()
			s.msgMap.Range(func(key, value any) bool {
				if now-value.(int64) > 5*60 {
					s.msgMap.Delete(key)
				}
				return true
			})
		}
	}
}
