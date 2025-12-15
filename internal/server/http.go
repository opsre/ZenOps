package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"cnb.cool/zhiqiangwang/pkg/logx"
	"github.com/eryajf/zenops/internal/config"
	"github.com/eryajf/zenops/internal/imcp"
	"github.com/eryajf/zenops/internal/model"
	"github.com/eryajf/zenops/internal/provider"
	aliyunprovider "github.com/eryajf/zenops/internal/provider/aliyun"
	"github.com/gin-gonic/gin"
)

// HTTPGinServer 基于 Gin 的 HTTP 服务器
type HTTPGinServer struct {
	config    *config.Config
	engine    *gin.Engine
	server    *http.Server
	mcpServer *imcp.MCPServer
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
	// API v1 路由组
	v1 := s.engine.Group("/api/v1")
	{
		// 健康检查
		v1.GET("/health", s.handleHealth)

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
	}
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
	var ossClient interface{ ListOSSBuckets(context.Context, int, int, map[string]string) ([]*model.OSSBucket, error) }
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
	var ossClient interface{ GetOSSBucket(context.Context, string) (*model.OSSBucket, error) }
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
