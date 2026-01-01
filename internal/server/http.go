package server

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"cnb.cool/zhiqiangwang/pkg/logx"
	"github.com/eryajf/zenops/internal/config"
	"github.com/eryajf/zenops/internal/imcp"
	"github.com/eryajf/zenops/internal/middleware"
	"github.com/eryajf/zenops/internal/model"
	"github.com/eryajf/zenops/internal/provider"
	aliyunprovider "github.com/eryajf/zenops/internal/provider/aliyun"
	"github.com/eryajf/zenops/internal/wecom"
	"github.com/eryajf/zenops/web"
	"github.com/gin-gonic/gin"
)

// HTTPGinServer 基于 Gin 的 HTTP 服务器
type HTTPGinServer struct {
	config         *config.Config
	engine         *gin.Engine
	server         *http.Server
	mcpServer      *imcp.MCPServer
	wecomHandler   *wecom.MessageHandler
	serviceManager *ServiceManager
	chatHandler    *ChatHandler
}

// NewHTTPGinServer 创建基于 Gin 的 HTTP 服务器
func NewHTTPGinServer(cfg *config.Config) *HTTPGinServer {
	// 设置 Gin 模式
	if cfg.Server.HTTP.Debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()

	s := &HTTPGinServer{
		config:    cfg,
		engine:    engine,
		mcpServer: nil,
	}

	// 注册中间件
	s.registerMiddlewares()

	// 注册路由
	s.registerRoutes()

	return s
}

// SetMCPServer 设置 MCP Server
func (s *HTTPGinServer) SetMCPServer(mcpServer *imcp.MCPServer) {
	s.mcpServer = mcpServer

	// 创建服务管理器
	s.serviceManager = NewServiceManager(s.config, mcpServer)

	// 创建 ChatHandler（需要 mcpServer）
	s.chatHandler = NewChatHandler(s.config, mcpServer)

	// 注册 AI 对话路由（需要在 mcpServer 设置后）
	s.registerChatRoutes()

	// 如果启用了企业微信,初始化消息处理器
	if s.config.Wecom.Enabled {
		handler, err := wecom.NewMessageHandler(s.config, mcpServer)
		if err != nil {
			logx.Error("Failed to create Wecom message handler: %v", err)
		} else {
			s.wecomHandler = handler
			logx.Info("Wecom message handler initialized")
		}
	}

	// 注册服务管理路由
	s.registerServiceRoutes()

	// 从数据库同步并启动已启用的 IM 服务
	go func() {
		if err := s.serviceManager.SyncWithDatabase(context.Background()); err != nil {
			logx.Error("Failed to sync IM services from database: %v", err)
		}
	}()
}

// GetServiceManager 获取服务管理器
func (s *HTTPGinServer) GetServiceManager() *ServiceManager {
	return s.serviceManager
}

// registerMiddlewares 注册中间件
func (s *HTTPGinServer) registerMiddlewares() {
	// 恢复中间件 - 从 panic 恢复
	s.engine.Use(gin.Recovery())

	// 自定义日志中间件
	s.engine.Use(s.loggingMiddleware())

	// CORS 中间件(如果需要)
	s.engine.Use(s.corsMiddleware())
}

// loggingMiddleware 自定义日志中间件
func (s *HTTPGinServer) loggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		logx.Info("HTTP request, method %s, path %s, remote_addr %s", method, path, c.ClientIP())

		c.Next()

		duration := time.Since(start)
		status := c.Writer.Status()

		logx.Info("HTTP response, method %s, path %s, status %d, duration %s",
			method, path, status, duration)
	}
}

// corsMiddleware CORS 中间件
func (s *HTTPGinServer) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// registerRoutes 注册路由
func (s *HTTPGinServer) registerRoutes() {
	// 企业微信机器人回调路由(不在 v1 组内)
	if s.config.Wecom.Enabled {
		s.engine.GET("/api/wecom/callback", s.handleWecomVerify)
		s.engine.POST("/api/wecom/callback", s.handleWecomMessage)
	}

	// API v1 路由组
	v1 := s.engine.Group("/api/v1")
	{
		// 健康检查
		v1.GET("/health", s.handleHealth)

		// 版本信息
		v1.GET("/version", GetVersionInfo)

		// 用户认证路由
		authHandler := NewAuthHandler()
		userHandler := NewUserHandler()

		// 公开路由 (不需要认证)
		auth := v1.Group("/auth")
		{
			auth.POST("/login", authHandler.Login)
			auth.POST("/logout", authHandler.Logout)
		}

		// 需要认证的用户路由
		user := v1.Group("/user")
		user.Use(middleware.AuthMiddleware())
		{
			user.GET("/info", authHandler.GetUserInfo)
			user.GET("/menu/list", userHandler.GetMenuList)
			user.POST("/change-password", authHandler.ChangePassword)
		}

		// 阿里云路由
		aliyun := v1.Group("/aliyun")
		{
			// ECS
			aliyun.GET("/ecs/list", s.handleAliyunECSList)
			aliyun.GET("/ecs/search", s.handleAliyunECSSearch)
			aliyun.GET("/ecs/get", s.handleAliyunECSGet)

			// RDS
			aliyun.GET("/rds/list", s.handleAliyunRDSList)
			aliyun.GET("/rds/search", s.handleAliyunRDSSearch)

			// OSS
			aliyun.GET("/oss/list", s.handleAliyunOSSList)
			aliyun.GET("/oss/get", s.handleAliyunOSSGet)
		}

		// 腾讯云路由
		tencent := v1.Group("/tencent")
		{
			// CVM
			tencent.GET("/cvm/list", s.handleTencentCVMList)
			tencent.GET("/cvm/search", s.handleTencentCVMSearch)
			tencent.GET("/cvm/get", s.handleTencentCVMGet)

			// CDB
			tencent.GET("/cdb/list", s.handleTencentCDBList)
			tencent.GET("/cdb/search", s.handleTencentCDBSearch)

			// COS
			tencent.GET("/cos/list", s.handleTencentCOSList)
			tencent.GET("/cos/get", s.handleTencentCOSGet)
		}

		// Jenkins 路由
		jenkins := v1.Group("/jenkins")
		{
			jenkins.GET("/job/list", s.handleJenkinsJobList)
			jenkins.GET("/job/get", s.handleJenkinsJobGet)
			jenkins.GET("/build/list", s.handleJenkinsBuildList)
		}

		// MCP Server 管理路由 (独立路由组)
		configHandler := NewConfigHandler()
		mcpHandler := NewMCPHandler()
		mcp := v1.Group("/mcp")
		{
			mcp.GET("/servers", configHandler.ListMCPServers)
			mcp.POST("/servers", configHandler.CreateMCPServer)
			mcp.GET("/servers/:name", configHandler.GetMCPServerByName)
			mcp.PUT("/servers/:name", configHandler.UpdateMCPServerByName)
			mcp.DELETE("/servers/:name", configHandler.DeleteMCPServerByName)
			mcp.PATCH("/servers/:name/toggle", configHandler.ToggleMCPServer)
			mcp.GET("/servers/:name/tools", configHandler.GetMCPTools)
			mcp.PATCH("/servers/:name/tools/:toolName/toggle", configHandler.ToggleMCPTool)
			mcp.POST("/servers/:name/tools/:toolName/test", configHandler.TestMCPTool)
			// MCP 调试接口
			mcp.POST("/debug/execute", mcpHandler.DebugExecute)
		}

		// 仪表盘路由
		dashboardHandler := NewDashboardHandler()
		dashboard := v1.Group("/dashboard")
		{
			dashboard.GET("/stats", dashboardHandler.GetStats)
			dashboard.GET("/health", dashboardHandler.GetHealth)
		}

		// 日志路由
		logHandler := NewLogHandler()
		logs := v1.Group("/logs")
		{
			logs.GET("/mcp", logHandler.GetMCPLogs)
			logs.GET("/mcp/stats", logHandler.GetMCPLogStats)
		}

		// AI 对话路由将在 SetMCPServer() 中注册（需要 mcpServer）

		// 对话历史路由
		historyHandler := NewHistoryHandler()
		history := v1.Group("/history")
		{
			history.GET("/chats", historyHandler.GetChatLogs)
			history.GET("/chats/:id/context", historyHandler.GetChatContext)
		}

		// 会话管理路由
		conversationHandler := NewConversationHandler()
		conversations := v1.Group("/conversations")
		{
			conversations.POST("", conversationHandler.CreateConversation)
			conversations.GET("", conversationHandler.ListConversations)
			conversations.GET("/:id", conversationHandler.GetConversation)
			conversations.PUT("/:id", conversationHandler.UpdateConversation)
			conversations.DELETE("/:id", conversationHandler.DeleteConversation)
		}

		// 配置管理路由
		config := v1.Group("/config")
		{
			// 全量配置
			config.GET("", configHandler.GetAllConfig)
			// LLM 配置 (RESTful 风格)
			config.GET("/llm", configHandler.ListLLMConfigs)
			config.POST("/llm", configHandler.CreateLLMConfig)
			config.GET("/llm/:id", configHandler.GetLLMConfig)
			config.PUT("/llm/:id", configHandler.UpdateLLMConfig)
			config.DELETE("/llm/:id", configHandler.DeleteLLMConfig)
			config.PATCH("/llm/:id/toggle", configHandler.ToggleLLMConfig)

			// 云厂商账号配置 (改为单数 provider)
			config.GET("/provider", configHandler.ListProviderAccounts)
			config.POST("/provider", configHandler.CreateProviderAccount)
			config.GET("/provider/:id", configHandler.GetProviderAccount)
			config.PUT("/provider/:id", configHandler.UpdateProviderAccount)
			config.DELETE("/provider/:id", configHandler.DeleteProviderAccount)

			// IM 配置 (添加 integration 别名)
			config.GET("/integration", configHandler.ListIntegrationConfigs)
			config.POST("/integration", configHandler.CreateIntegrationConfig)
			config.GET("/integration/:id", configHandler.GetIntegrationConfig)
			config.PUT("/integration/:id", configHandler.UpdateIntegrationConfig)
			config.DELETE("/integration/:id", configHandler.DeleteIntegrationConfig)

			// CICD 配置
			config.GET("/cicd", configHandler.ListCICDConfigs)
			config.GET("/cicd/:platform", configHandler.GetCICDConfig)
			config.PUT("/cicd/:platform", configHandler.SaveCICDConfig)

			// Jenkins 配置便捷路由
			config.GET("/jenkins", configHandler.GetJenkinsConfig)
			config.POST("/jenkins", configHandler.SaveJenkinsConfig)

			// 服务器配置
			config.GET("/server", configHandler.GetServerConfig)
			config.POST("/server", configHandler.SaveServerConfig)

			// 系统配置
			config.GET("/system", configHandler.ListSystemConfigs)
			config.GET("/system/:key", configHandler.GetSystemConfig)
			config.POST("/system", configHandler.SetSystemConfig)
		}
	}

	// 前端静态文件服务 (SPA 模式)
	s.registerStaticFiles()
}

// registerChatRoutes 注册 AI 对话路由（需要在 SetMCPServer 之后调用）
func (s *HTTPGinServer) registerChatRoutes() {
	if s.chatHandler == nil {
		logx.Warn("ChatHandler is nil, skipping chat routes registration")
		return
	}

	// AI 对话路由
	v1 := s.engine.Group("/api/v1")
	chat := v1.Group("/chat")
	{
		chat.POST("/completions", s.chatHandler.Completions)
		chat.GET("/models", s.chatHandler.GetModels)
	}

	logx.Info("✅ Chat routes registered successfully")
}

// registerServiceRoutes 注册服务管理路由
func (s *HTTPGinServer) registerServiceRoutes() {
	if s.serviceManager == nil {
		logx.Warn("ServiceManager is nil, skipping service routes registration")
		return
	}

	serviceHandler := NewServiceHandler(s.serviceManager)

	// 服务管理路由
	services := s.engine.Group("/api/v1/services")
	{
		services.GET("/status", serviceHandler.GetServiceStatus)
		services.GET("/status/:platform", serviceHandler.GetPlatformStatus)
		services.POST("/toggle/:platform", serviceHandler.ToggleIMService)
	}
}

// registerStaticFiles 注册前端静态文件服务
func (s *HTTPGinServer) registerStaticFiles() {
	// 获取嵌入的前端文件系统
	distFS := web.GetFS()
	subFS, err := fs.Sub(distFS, "dist")
	if err != nil {
		logx.Error("Failed to get embedded frontend files: %v", err)
		return
	}

	// 静态文件服务器
	fileServer := http.FileServer(http.FS(subFS))

	// 处理所有非 API 请求
	s.engine.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		// 如果是 API 请求，返回 404
		if strings.HasPrefix(path, "/api/") {
			c.JSON(http.StatusNotFound, Response{
				Code:    404,
				Message: "API not found",
			})
			return
		}

		// 尝试直接提供静态文件
		// 检查文件是否存在
		f, err := subFS.Open(strings.TrimPrefix(path, "/"))
		if err == nil {
			f.Close()
			// 文件存在，直接提供
			fileServer.ServeHTTP(c.Writer, c.Request)
			return
		}

		// 文件不存在，返回 index.html (SPA 模式)
		c.Request.URL.Path = "/"
		fileServer.ServeHTTP(c.Writer, c.Request)
	})
}

// Start 启动 HTTP 服务器
func (s *HTTPGinServer) Start() error {
	addr := fmt.Sprintf("0.0.0.0:%d", s.config.Server.HTTP.Port)

	s.server = &http.Server{
		Addr:         addr,
		Handler:      s.engine,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	logx.Info("🛜 Starting HTTP Server (Gin), Addr %s", addr)
	return s.server.ListenAndServe()
}

// Stop 停止 HTTP 服务器
func (s *HTTPGinServer) Stop(ctx context.Context) error {
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

// Response 统一响应结构
type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// success 返回成功响应
func (s *HTTPGinServer) success(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "Success",
		Data:    data,
	})
}

// error 返回错误响应
func (s *HTTPGinServer) error(c *gin.Context, code int, message string) {
	c.JSON(code, Response{
		Code:    code,
		Message: message,
	})
}

// ==================== 健康检查 ====================

func (s *HTTPGinServer) handleHealth(c *gin.Context) {
	s.success(c, gin.H{
		"status": "healthy",
	})
}

// ==================== 阿里云 ECS API ====================

func (s *HTTPGinServer) handleAliyunECSList(c *gin.Context) {
	accountName := c.Query("account")
	region := c.Query("region")

	aliyunConfig, err := getAliyunConfigByName(s.config, accountName)
	if err != nil {
		s.error(c, http.StatusBadRequest, err.Error())
		return
	}

	p, err := provider.GetProvider("aliyun")
	if err != nil {
		s.error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to get provider: %v", err))
		return
	}

	providerConfig := map[string]any{
		"access_key_id":     aliyunConfig.AK,
		"access_key_secret": aliyunConfig.SK,
		"regions":           interfaceSlice(aliyunConfig.Regions),
	}

	if err := p.Initialize(providerConfig); err != nil {
		s.error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to initialize provider: %v", err))
		return
	}

	var allInstances []*model.Instance
	pageNum := 1
	pageSize := 100

	for {
		opts := &provider.QueryOptions{
			Region:   region,
			PageSize: pageSize,
			PageNum:  pageNum,
		}

		instances, err := p.ListInstances(c.Request.Context(), opts)
		if err != nil {
			s.error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to list instances: %v", err))
			return
		}

		allInstances = append(allInstances, instances...)

		if len(instances) < pageSize {
			break
		}
		pageNum++
	}

	s.success(c, gin.H{
		"total":     len(allInstances),
		"instances": allInstances,
		"account":   aliyunConfig.Name,
	})
}

func (s *HTTPGinServer) handleAliyunECSSearch(c *gin.Context) {
	accountName := c.Query("account")
	ip := c.Query("ip")
	instanceName := c.Query("name")

	if ip == "" && instanceName == "" {
		s.error(c, http.StatusBadRequest, "Either 'ip' or 'name' parameter is required")
		return
	}

	aliyunConfig, err := getAliyunConfigByName(s.config, accountName)
	if err != nil {
		s.error(c, http.StatusBadRequest, err.Error())
		return
	}

	p, err := provider.GetProvider("aliyun")
	if err != nil {
		s.error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to get provider: %v", err))
		return
	}

	providerConfig := map[string]any{
		"access_key_id":     aliyunConfig.AK,
		"access_key_secret": aliyunConfig.SK,
		"regions":           interfaceSlice(aliyunConfig.Regions),
	}

	if err := p.Initialize(providerConfig); err != nil {
		s.error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to initialize provider: %v", err))
		return
	}

	var matchedInstances []*model.Instance
	pageNum := 1
	pageSize := 100

	for {
		opts := &provider.QueryOptions{
			PageSize: pageSize,
			PageNum:  pageNum,
		}

		instances, err := p.ListInstances(c.Request.Context(), opts)
		if err != nil {
			s.error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to list instances: %v", err))
			return
		}

		for _, inst := range instances {
			if ip != "" {
				for _, privateIP := range inst.PrivateIP {
					if privateIP == ip {
						matchedInstances = append(matchedInstances, inst)
						break
					}
				}
				for _, publicIP := range inst.PublicIP {
					if publicIP == ip {
						matchedInstances = append(matchedInstances, inst)
						break
					}
				}
			}
			if instanceName != "" && inst.Name == instanceName {
				matchedInstances = append(matchedInstances, inst)
			}
		}

		if len(instances) < pageSize {
			break
		}
		pageNum++
	}

	if len(matchedInstances) == 0 {
		s.error(c, http.StatusNotFound, "No matching instances found")
		return
	}

	s.success(c, gin.H{
		"total":     len(matchedInstances),
		"instances": matchedInstances,
		"account":   aliyunConfig.Name,
	})
}

func (s *HTTPGinServer) handleAliyunECSGet(c *gin.Context) {
	accountName := c.Query("account")
	instanceID := c.Query("instance_id")

	if instanceID == "" {
		s.error(c, http.StatusBadRequest, "instance_id is required")
		return
	}

	aliyunConfig, err := getAliyunConfigByName(s.config, accountName)
	if err != nil {
		s.error(c, http.StatusBadRequest, err.Error())
		return
	}

	p, err := provider.GetProvider("aliyun")
	if err != nil {
		s.error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to get provider: %v", err))
		return
	}

	providerConfig := map[string]any{
		"access_key_id":     aliyunConfig.AK,
		"access_key_secret": aliyunConfig.SK,
		"regions":           interfaceSlice(aliyunConfig.Regions),
	}

	if err := p.Initialize(providerConfig); err != nil {
		s.error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to initialize provider: %v", err))
		return
	}

	instance, err := p.GetInstance(c.Request.Context(), instanceID)
	if err != nil {
		s.error(c, http.StatusNotFound, fmt.Sprintf("Failed to get instance: %v", err))
		return
	}

	s.success(c, gin.H{
		"instance": instance,
		"account":  aliyunConfig.Name,
	})
}

// ==================== 阿里云 RDS API ====================

func (s *HTTPGinServer) handleAliyunRDSList(c *gin.Context) {
	accountName := c.Query("account")
	region := c.Query("region")

	aliyunConfig, err := getAliyunConfigByName(s.config, accountName)
	if err != nil {
		s.error(c, http.StatusBadRequest, err.Error())
		return
	}

	p, err := provider.GetProvider("aliyun")
	if err != nil {
		s.error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to get provider: %v", err))
		return
	}

	providerConfig := map[string]any{
		"access_key_id":     aliyunConfig.AK,
		"access_key_secret": aliyunConfig.SK,
		"regions":           interfaceSlice(aliyunConfig.Regions),
	}

	if err := p.Initialize(providerConfig); err != nil {
		s.error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to initialize provider: %v", err))
		return
	}

	var allDatabases []*model.Database
	pageNum := 1
	pageSize := 100

	for {
		opts := &provider.QueryOptions{
			Region:   region,
			PageSize: pageSize,
			PageNum:  pageNum,
		}

		databases, err := p.ListDatabases(c.Request.Context(), opts)
		if err != nil {
			s.error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to list databases: %v", err))
			return
		}

		allDatabases = append(allDatabases, databases...)

		if len(databases) < pageSize {
			break
		}
		pageNum++
	}

	s.success(c, gin.H{
		"total":     len(allDatabases),
		"databases": allDatabases,
		"account":   aliyunConfig.Name,
	})
}

func (s *HTTPGinServer) handleAliyunRDSSearch(c *gin.Context) {
	accountName := c.Query("account")
	name := c.Query("name")
	endpoint := c.Query("endpoint")

	if name == "" && endpoint == "" {
		s.error(c, http.StatusBadRequest, "Either 'name' or 'endpoint' parameter is required")
		return
	}

	aliyunConfig, err := getAliyunConfigByName(s.config, accountName)
	if err != nil {
		s.error(c, http.StatusBadRequest, err.Error())
		return
	}

	p, err := provider.GetProvider("aliyun")
	if err != nil {
		s.error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to get provider: %v", err))
		return
	}

	providerConfig := map[string]any{
		"access_key_id":     aliyunConfig.AK,
		"access_key_secret": aliyunConfig.SK,
		"regions":           interfaceSlice(aliyunConfig.Regions),
	}

	if err := p.Initialize(providerConfig); err != nil {
		s.error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to initialize provider: %v", err))
		return
	}

	var matchedDatabases []*model.Database
	pageNum := 1
	pageSize := 100

	for {
		opts := &provider.QueryOptions{
			PageSize: pageSize,
			PageNum:  pageNum,
		}

		databases, err := p.ListDatabases(c.Request.Context(), opts)
		if err != nil {
			s.error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to list databases: %v", err))
			return
		}

		for _, db := range databases {
			if (name != "" && db.Name == name) || (endpoint != "" && db.Endpoint == endpoint) {
				matchedDatabases = append(matchedDatabases, db)
			}
		}

		if len(databases) < pageSize {
			break
		}
		pageNum++
	}

	if len(matchedDatabases) == 0 {
		s.error(c, http.StatusNotFound, "No matching databases found")
		return
	}

	s.success(c, gin.H{
		"total":     len(matchedDatabases),
		"databases": matchedDatabases,
		"account":   aliyunConfig.Name,
	})
}

// ==================== 阿里云 OSS API ====================

func (s *HTTPGinServer) handleAliyunOSSList(c *gin.Context) {
	accountName := c.Query("account")

	aliyunConfig, err := getAliyunConfigByName(s.config, accountName)
	if err != nil {
		s.error(c, http.StatusBadRequest, err.Error())
		return
	}

	// 创建临时客户端
	var ossClient interface {
		ListOSSBuckets(context.Context, int, int, map[string]string) ([]*model.OSSBucket, error)
	}
	for _, region := range aliyunConfig.Regions {
		c, err := createAliyunClient(aliyunConfig.AK, aliyunConfig.SK, region)
		if err == nil {
			ossClient = c
			break
		}
	}
	if ossClient == nil {
		s.error(c, http.StatusInternalServerError, "Failed to create OSS client")
		return
	}

	var allBuckets []*model.OSSBucket
	pageNum := 1
	pageSize := 100

	for {
		buckets, err := ossClient.ListOSSBuckets(c.Request.Context(), pageSize, pageNum, nil)
		if err != nil {
			s.error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to list OSS buckets: %v", err))
			return
		}

		allBuckets = append(allBuckets, buckets...)

		if len(buckets) < pageSize {
			break
		}
		pageNum++
	}

	s.success(c, gin.H{
		"total":   len(allBuckets),
		"buckets": allBuckets,
		"account": aliyunConfig.Name,
	})
}

func (s *HTTPGinServer) handleAliyunOSSGet(c *gin.Context) {
	accountName := c.Query("account")
	bucketName := c.Query("bucket_name")

	if bucketName == "" {
		s.error(c, http.StatusBadRequest, "bucket_name is required")
		return
	}

	aliyunConfig, err := getAliyunConfigByName(s.config, accountName)
	if err != nil {
		s.error(c, http.StatusBadRequest, err.Error())
		return
	}

	// 创建临时客户端
	var ossClient interface {
		GetOSSBucket(context.Context, string) (*model.OSSBucket, error)
	}
	for _, region := range aliyunConfig.Regions {
		c, err := createAliyunClient(aliyunConfig.AK, aliyunConfig.SK, region)
		if err == nil {
			ossClient = c
			break
		}
	}
	if ossClient == nil {
		s.error(c, http.StatusInternalServerError, "Failed to create OSS client")
		return
	}

	bucket, err := ossClient.GetOSSBucket(c.Request.Context(), bucketName)
	if err != nil {
		s.error(c, http.StatusNotFound, fmt.Sprintf("Failed to get OSS bucket: %v", err))
		return
	}

	s.success(c, gin.H{
		"bucket":  bucket,
		"account": aliyunConfig.Name,
	})
}

// ==================== 腾讯云 CVM API ====================

func (s *HTTPGinServer) handleTencentCVMList(c *gin.Context) {
	accountName := c.Query("account")
	region := c.Query("region")

	tencentConfig, err := getTencentConfigByName(s.config, accountName)
	if err != nil {
		s.error(c, http.StatusBadRequest, err.Error())
		return
	}

	p, err := provider.GetProvider("tencent")
	if err != nil {
		s.error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to get provider: %v", err))
		return
	}

	providerConfig := map[string]any{
		"secret_id":  tencentConfig.AK,
		"secret_key": tencentConfig.SK,
		"regions":    interfaceSlice(tencentConfig.Regions),
	}

	if err := p.Initialize(providerConfig); err != nil {
		s.error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to initialize provider: %v", err))
		return
	}

	var allInstances []*model.Instance
	pageNum := 1
	pageSize := 100

	for {
		opts := &provider.QueryOptions{
			Region:   region,
			PageSize: pageSize,
			PageNum:  pageNum,
		}

		instances, err := p.ListInstances(c.Request.Context(), opts)
		if err != nil {
			s.error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to list instances: %v", err))
			return
		}

		allInstances = append(allInstances, instances...)

		if len(instances) < pageSize {
			break
		}
		pageNum++
	}

	s.success(c, gin.H{
		"total":     len(allInstances),
		"instances": allInstances,
		"account":   tencentConfig.Name,
	})
}

func (s *HTTPGinServer) handleTencentCVMSearch(c *gin.Context) {
	accountName := c.Query("account")
	ip := c.Query("ip")
	instanceName := c.Query("name")

	if ip == "" && instanceName == "" {
		s.error(c, http.StatusBadRequest, "Either 'ip' or 'name' parameter is required")
		return
	}

	tencentConfig, err := getTencentConfigByName(s.config, accountName)
	if err != nil {
		s.error(c, http.StatusBadRequest, err.Error())
		return
	}

	p, err := provider.GetProvider("tencent")
	if err != nil {
		s.error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to get provider: %v", err))
		return
	}

	providerConfig := map[string]any{
		"secret_id":  tencentConfig.AK,
		"secret_key": tencentConfig.SK,
		"regions":    interfaceSlice(tencentConfig.Regions),
	}

	if err := p.Initialize(providerConfig); err != nil {
		s.error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to initialize provider: %v", err))
		return
	}

	var matchedInstances []*model.Instance
	pageNum := 1
	pageSize := 100

	for {
		opts := &provider.QueryOptions{
			PageSize: pageSize,
			PageNum:  pageNum,
		}

		instances, err := p.ListInstances(c.Request.Context(), opts)
		if err != nil {
			s.error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to list instances: %v", err))
			return
		}

		for _, inst := range instances {
			if ip != "" {
				for _, privateIP := range inst.PrivateIP {
					if privateIP == ip {
						matchedInstances = append(matchedInstances, inst)
						break
					}
				}
				for _, publicIP := range inst.PublicIP {
					if publicIP == ip {
						matchedInstances = append(matchedInstances, inst)
						break
					}
				}
			}
			if instanceName != "" && inst.Name == instanceName {
				matchedInstances = append(matchedInstances, inst)
			}
		}

		if len(instances) < pageSize {
			break
		}
		pageNum++
	}

	if len(matchedInstances) == 0 {
		s.error(c, http.StatusNotFound, "No matching instances found")
		return
	}

	s.success(c, gin.H{
		"total":     len(matchedInstances),
		"instances": matchedInstances,
		"account":   tencentConfig.Name,
	})
}

func (s *HTTPGinServer) handleTencentCVMGet(c *gin.Context) {
	accountName := c.Query("account")
	instanceID := c.Query("instance_id")

	if instanceID == "" {
		s.error(c, http.StatusBadRequest, "instance_id is required")
		return
	}

	tencentConfig, err := getTencentConfigByName(s.config, accountName)
	if err != nil {
		s.error(c, http.StatusBadRequest, err.Error())
		return
	}

	p, err := provider.GetProvider("tencent")
	if err != nil {
		s.error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to get provider: %v", err))
		return
	}

	providerConfig := map[string]any{
		"secret_id":  tencentConfig.AK,
		"secret_key": tencentConfig.SK,
		"regions":    interfaceSlice(tencentConfig.Regions),
	}

	if err := p.Initialize(providerConfig); err != nil {
		s.error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to initialize provider: %v", err))
		return
	}

	instance, err := p.GetInstance(c.Request.Context(), instanceID)
	if err != nil {
		s.error(c, http.StatusNotFound, fmt.Sprintf("Failed to get instance: %v", err))
		return
	}

	s.success(c, gin.H{
		"instance": instance,
		"account":  tencentConfig.Name,
	})
}

// ==================== 腾讯云 CDB API ====================

func (s *HTTPGinServer) handleTencentCDBList(c *gin.Context) {
	accountName := c.Query("account")
	region := c.Query("region")

	tencentConfig, err := getTencentConfigByName(s.config, accountName)
	if err != nil {
		s.error(c, http.StatusBadRequest, err.Error())
		return
	}

	p, err := provider.GetProvider("tencent")
	if err != nil {
		s.error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to get provider: %v", err))
		return
	}

	providerConfig := map[string]any{
		"secret_id":  tencentConfig.AK,
		"secret_key": tencentConfig.SK,
		"regions":    interfaceSlice(tencentConfig.Regions),
	}

	if err := p.Initialize(providerConfig); err != nil {
		s.error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to initialize provider: %v", err))
		return
	}

	var allDatabases []*model.Database
	pageNum := 1
	pageSize := 100

	for {
		opts := &provider.QueryOptions{
			Region:   region,
			PageSize: pageSize,
			PageNum:  pageNum,
		}

		databases, err := p.ListDatabases(c.Request.Context(), opts)
		if err != nil {
			s.error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to list databases: %v", err))
			return
		}

		allDatabases = append(allDatabases, databases...)

		if len(databases) < pageSize {
			break
		}
		pageNum++
	}

	s.success(c, gin.H{
		"total":     len(allDatabases),
		"databases": allDatabases,
		"account":   tencentConfig.Name,
	})
}

func (s *HTTPGinServer) handleTencentCDBSearch(c *gin.Context) {
	accountName := c.Query("account")
	name := c.Query("name")
	endpoint := c.Query("endpoint")

	if name == "" && endpoint == "" {
		s.error(c, http.StatusBadRequest, "Either 'name' or 'endpoint' parameter is required")
		return
	}

	tencentConfig, err := getTencentConfigByName(s.config, accountName)
	if err != nil {
		s.error(c, http.StatusBadRequest, err.Error())
		return
	}

	p, err := provider.GetProvider("tencent")
	if err != nil {
		s.error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to get provider: %v", err))
		return
	}

	providerConfig := map[string]any{
		"secret_id":  tencentConfig.AK,
		"secret_key": tencentConfig.SK,
		"regions":    interfaceSlice(tencentConfig.Regions),
	}

	if err := p.Initialize(providerConfig); err != nil {
		s.error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to initialize provider: %v", err))
		return
	}

	var matchedDatabases []*model.Database
	pageNum := 1
	pageSize := 100

	for {
		opts := &provider.QueryOptions{
			PageSize: pageSize,
			PageNum:  pageNum,
		}

		databases, err := p.ListDatabases(c.Request.Context(), opts)
		if err != nil {
			s.error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to list databases: %v", err))
			return
		}

		for _, db := range databases {
			if (name != "" && db.Name == name) || (endpoint != "" && db.Endpoint == endpoint) {
				matchedDatabases = append(matchedDatabases, db)
			}
		}

		if len(databases) < pageSize {
			break
		}
		pageNum++
	}

	if len(matchedDatabases) == 0 {
		s.error(c, http.StatusNotFound, "No matching databases found")
		return
	}

	s.success(c, gin.H{
		"total":     len(matchedDatabases),
		"databases": matchedDatabases,
		"account":   tencentConfig.Name,
	})
}

// ==================== 腾讯云 COS API ====================

func (s *HTTPGinServer) handleTencentCOSList(c *gin.Context) {
	accountName := c.Query("account")

	tencentConfig, err := getTencentConfigByName(s.config, accountName)
	if err != nil {
		s.error(c, http.StatusBadRequest, err.Error())
		return
	}

	p, err := provider.GetProvider("tencent")
	if err != nil {
		s.error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to get provider: %v", err))
		return
	}

	providerConfig := map[string]any{
		"secret_id":  tencentConfig.AK,
		"secret_key": tencentConfig.SK,
		"regions":    interfaceSlice(tencentConfig.Regions),
	}

	if err := p.Initialize(providerConfig); err != nil {
		s.error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to initialize provider: %v", err))
		return
	}

	var allBuckets []*model.OSSBucket
	pageNum := 1
	pageSize := 100

	for {
		opts := &provider.QueryOptions{
			PageSize: pageSize,
			PageNum:  pageNum,
		}

		buckets, err := p.ListOSSBuckets(c.Request.Context(), opts)
		if err != nil {
			s.error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to list COS buckets: %v", err))
			return
		}

		allBuckets = append(allBuckets, buckets...)

		if len(buckets) < pageSize {
			break
		}
		pageNum++
	}

	s.success(c, gin.H{
		"total":   len(allBuckets),
		"buckets": allBuckets,
		"account": tencentConfig.Name,
	})
}

func (s *HTTPGinServer) handleTencentCOSGet(c *gin.Context) {
	accountName := c.Query("account")
	bucketName := c.Query("bucket_name")

	if bucketName == "" {
		s.error(c, http.StatusBadRequest, "bucket_name is required")
		return
	}

	tencentConfig, err := getTencentConfigByName(s.config, accountName)
	if err != nil {
		s.error(c, http.StatusBadRequest, err.Error())
		return
	}

	p, err := provider.GetProvider("tencent")
	if err != nil {
		s.error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to get provider: %v", err))
		return
	}

	providerConfig := map[string]any{
		"secret_id":  tencentConfig.AK,
		"secret_key": tencentConfig.SK,
		"regions":    interfaceSlice(tencentConfig.Regions),
	}

	if err := p.Initialize(providerConfig); err != nil {
		s.error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to initialize provider: %v", err))
		return
	}

	bucket, err := p.GetOSSBucket(c.Request.Context(), bucketName)
	if err != nil {
		s.error(c, http.StatusNotFound, fmt.Sprintf("Failed to get COS bucket: %v", err))
		return
	}

	s.success(c, gin.H{
		"bucket":  bucket,
		"account": tencentConfig.Name,
	})
}

// ==================== Jenkins API ====================

func (s *HTTPGinServer) handleJenkinsJobList(c *gin.Context) {
	p, err := provider.GetCICDProvider("jenkins")
	if err != nil {
		s.error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to get provider: %v", err))
		return
	}

	providerConfig := map[string]any{
		"url":      s.config.CICD.Jenkins.URL,
		"username": s.config.CICD.Jenkins.Username,
		"token":    s.config.CICD.Jenkins.Token,
	}

	if err := p.Initialize(providerConfig); err != nil {
		s.error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to initialize provider: %v", err))
		return
	}

	opts := &provider.QueryOptions{
		PageSize: 100,
		PageNum:  1,
	}

	jobs, err := p.ListJobs(c.Request.Context(), opts)
	if err != nil {
		s.error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to list jobs: %v", err))
		return
	}

	s.success(c, gin.H{
		"total": len(jobs),
		"jobs":  jobs,
	})
}

func (s *HTTPGinServer) handleJenkinsJobGet(c *gin.Context) {
	jobName := c.Query("job_name")
	if jobName == "" {
		s.error(c, http.StatusBadRequest, "job_name is required")
		return
	}

	p, err := provider.GetCICDProvider("jenkins")
	if err != nil {
		s.error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to get provider: %v", err))
		return
	}

	providerConfig := map[string]any{
		"url":      s.config.CICD.Jenkins.URL,
		"username": s.config.CICD.Jenkins.Username,
		"token":    s.config.CICD.Jenkins.Token,
	}

	if err := p.Initialize(providerConfig); err != nil {
		s.error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to initialize provider: %v", err))
		return
	}

	job, err := p.GetJob(c.Request.Context(), jobName)
	if err != nil {
		s.error(c, http.StatusNotFound, fmt.Sprintf("Failed to get job: %v", err))
		return
	}

	s.success(c, gin.H{
		"job": job,
	})
}

func (s *HTTPGinServer) handleJenkinsBuildList(c *gin.Context) {
	jobName := c.Query("job_name")
	if jobName == "" {
		s.error(c, http.StatusBadRequest, "job_name is required")
		return
	}

	p, err := provider.GetCICDProvider("jenkins")
	if err != nil {
		s.error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to get provider: %v", err))
		return
	}

	providerConfig := map[string]any{
		"url":      s.config.CICD.Jenkins.URL,
		"username": s.config.CICD.Jenkins.Username,
		"token":    s.config.CICD.Jenkins.Token,
	}

	if err := p.Initialize(providerConfig); err != nil {
		s.error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to initialize provider: %v", err))
		return
	}

	limit := 20
	builds, err := p.GetJobBuilds(c.Request.Context(), jobName, limit)
	if err != nil {
		s.error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to list builds: %v", err))
		return
	}

	s.success(c, gin.H{
		"total":    len(builds),
		"builds":   builds,
		"job_name": jobName,
	})
}

// ==================== 辅助函数 ====================

// getAliyunConfigByName 根据名称获取阿里云账号配置
func getAliyunConfigByName(cfg *config.Config, accountName string) (*config.ProviderConfig, error) {
	if len(cfg.Providers.Aliyun) == 0 {
		return nil, fmt.Errorf("no aliyun account configured")
	}

	if accountName == "" {
		for _, acc := range cfg.Providers.Aliyun {
			if acc.Enabled {
				return &acc, nil
			}
		}
		return &cfg.Providers.Aliyun[0], nil
	}

	for _, acc := range cfg.Providers.Aliyun {
		if acc.Name == accountName {
			return &acc, nil
		}
	}

	return nil, fmt.Errorf("aliyun account '%s' not found", accountName)
}

// getTencentConfigByName 根据名称获取腾讯云账号配置
func getTencentConfigByName(cfg *config.Config, accountName string) (*config.ProviderConfig, error) {
	if len(cfg.Providers.Tencent) == 0 {
		return nil, fmt.Errorf("no tencent account configured")
	}

	if accountName == "" {
		for _, acc := range cfg.Providers.Tencent {
			if acc.Enabled {
				return &acc, nil
			}
		}
		return &cfg.Providers.Tencent[0], nil
	}

	for _, acc := range cfg.Providers.Tencent {
		if acc.Name == accountName {
			return &acc, nil
		}
	}

	return nil, fmt.Errorf("tencent account '%s' not found", accountName)
}

// interfaceSlice 将 []string 转换为 []any
func interfaceSlice(s []string) []any {
	result := make([]any, len(s))
	for i, v := range s {
		result[i] = v
	}
	return result
}

// createAliyunClient 创建阿里云客户端
func createAliyunClient(ak, sk, region string) (*aliyunprovider.Client, error) {
	return aliyunprovider.NewClient(ak, sk, region)
}

// ==================== 企业微信机器人 API ====================

// handleWecomVerify 处理企业微信URL验证
func (s *HTTPGinServer) handleWecomVerify(c *gin.Context) {
	// 优先从 ServiceManager 获取 handler
	handler := s.getWecomHandler()
	if handler == nil {
		s.error(c, http.StatusServiceUnavailable, "Wecom handler not initialized")
		return
	}

	signature := c.Query("msg_signature")
	timestamp := c.Query("timestamp")
	nonce := c.Query("nonce")
	echoStr := c.Query("echostr")

	logx.Info("Wecom verify request: signature=%s, timestamp=%s, nonce=%s", signature, timestamp, nonce)

	replyEchoStr, err := handler.Client.VerifyURL(signature, timestamp, nonce, echoStr)
	if err != nil {
		logx.Error("Failed to verify Wecom URL: %v", err)
		s.error(c, http.StatusBadRequest, fmt.Sprintf("Verification failed: %v", err))
		return
	}

	c.String(http.StatusOK, replyEchoStr)
}

// handleWecomMessage 处理企业微信消息回调
func (s *HTTPGinServer) handleWecomMessage(c *gin.Context) {
	// 优先从 ServiceManager 获取 handler
	handler := s.getWecomHandler()
	if handler == nil {
		s.error(c, http.StatusServiceUnavailable, "Wecom handler not initialized")
		return
	}

	signature := c.Query("msg_signature")
	timestamp := c.Query("timestamp")
	nonce := c.Query("nonce")

	// 读取请求体
	body, err := c.GetRawData()
	if err != nil {
		logx.Error("Failed to read request body: %v", err)
		s.error(c, http.StatusBadRequest, "Failed to read request body")
		return
	}

	logx.Debug("Wecom message request: signature=%s, timestamp=%s, nonce=%s, body=%s",
		signature, timestamp, nonce, string(body))

	// 解密消息
	req, err := handler.Client.DecryptUserReq(signature, timestamp, nonce, string(body))
	if err != nil {
		logx.Error("Failed to decrypt Wecom message: %v", err)
		c.String(http.StatusOK, "") // 企业微信要求返回200
		return
	}

	var response string
	ctx := c.Request.Context()

	// 根据消息类型处理
	switch req.Msgtype {
	case "text":
		// 处理文本消息
		response, err = handler.HandleTextMessage(ctx, req)
		if err != nil {
			logx.Error("Failed to handle text message: %v", err)
			c.String(http.StatusOK, "")
			return
		}

	case "stream":
		// 处理流式轮询请求
		response, err = handler.HandleStreamRequest(ctx, req)
		if err != nil {
			logx.Error("Failed to handle stream request: %v", err)
			c.String(http.StatusOK, "")
			return
		}

	default:
		logx.Warn("Unsupported message type: %s", req.Msgtype)
		c.String(http.StatusOK, "")
		return
	}

	// 返回加密响应
	c.Header("Content-Type", "application/json")
	c.String(http.StatusOK, response)
}

// getWecomHandler 获取企业微信处理器（优先从 ServiceManager 获取）
func (s *HTTPGinServer) getWecomHandler() *wecom.MessageHandler {
	// 优先从 ServiceManager 获取
	if s.serviceManager != nil {
		if handler := s.serviceManager.GetWecomHandler(); handler != nil {
			return handler
		}
	}
	// 回退到本地 handler
	return s.wecomHandler
}
