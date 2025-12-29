package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ServiceHandler 服务管理处理器
type ServiceHandler struct {
	serviceManager *ServiceManager
}

// NewServiceHandler 创建服务处理器
func NewServiceHandler(sm *ServiceManager) *ServiceHandler {
	return &ServiceHandler{
		serviceManager: sm,
	}
}

// ToggleIMService 切换 IM 服务状态
func (h *ServiceHandler) ToggleIMService(c *gin.Context) {
	platform := c.Param("platform")

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: err.Error(),
		})
		return
	}

	// 验证平台名称
	validPlatforms := map[string]bool{
		"dingtalk": true,
		"feishu":   true,
		"wecom":    true,
	}

	if !validPlatforms[platform] {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "invalid platform: must be dingtalk, feishu, or wecom",
		})
		return
	}

	// 切换服务状态
	if err := h.serviceManager.ToggleService(c.Request.Context(), platform, req.Enabled); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	status := "stopped"
	if req.Enabled {
		status = "started"
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: platform + " service " + status + " successfully",
		Data: gin.H{
			"platform": platform,
			"enabled":  req.Enabled,
			"status":   status,
		},
	})
}

// GetServiceStatus 获取服务状态
func (h *ServiceHandler) GetServiceStatus(c *gin.Context) {
	status := h.serviceManager.GetServiceStatus()

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "success",
		Data:    status,
	})
}

// GetPlatformStatus 获取指定平台的服务状态
func (h *ServiceHandler) GetPlatformStatus(c *gin.Context) {
	platform := c.Param("platform")

	status := h.serviceManager.GetServiceStatus()
	running, ok := status[platform]
	if !ok {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "invalid platform: must be dingtalk, feishu, or wecom",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "success",
		Data: gin.H{
			"platform": platform,
			"running":  running,
		},
	})
}
