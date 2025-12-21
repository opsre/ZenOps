package server

import (
	"net/http"
	"strconv"

	"github.com/eryajf/zenops/internal/model"
	"github.com/eryajf/zenops/internal/service"
	"github.com/gin-gonic/gin"
)

// ConfigHandler 配置管理处理器
type ConfigHandler struct {
	configService *service.ConfigService
}

// NewConfigHandler 创建配置处理器
func NewConfigHandler() *ConfigHandler {
	return &ConfigHandler{
		configService: service.NewConfigService(),
	}
}

// ========== LLM 配置 ==========

// GetLLMConfig 获取 LLM 配置
func (h *ConfigHandler) GetLLMConfig(c *gin.Context) {
	config, err := h.configService.GetLLMConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "success",
		Data:    config,
	})
}

// SaveLLMConfig 保存 LLM 配置
func (h *ConfigHandler) SaveLLMConfig(c *gin.Context) {
	var config model.LLMConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: err.Error(),
		})
		return
	}

	if err := h.configService.SaveLLMConfig(&config); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "LLM configuration saved successfully",
		Data:    config,
	})
}

// ========== 云厂商账号配置 ==========

// ListProviderAccounts 列出云厂商账号
func (h *ConfigHandler) ListProviderAccounts(c *gin.Context) {
	provider := c.Query("provider")

	accounts, err := h.configService.ListProviderAccounts(provider)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "success",
		Data:    accounts,
	})
}

// GetProviderAccount 获取云厂商账号详情
func (h *ConfigHandler) GetProviderAccount(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "invalid id",
		})
		return
	}

	account, err := h.configService.GetProviderAccount(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "success",
		Data:    account,
	})
}

// CreateProviderAccount 创建云厂商账号
func (h *ConfigHandler) CreateProviderAccount(c *gin.Context) {
	var account model.ProviderAccount
	if err := c.ShouldBindJSON(&account); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: err.Error(),
		})
		return
	}

	if err := h.configService.CreateProviderAccount(&account); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "Provider account created successfully",
		Data:    account,
	})
}

// UpdateProviderAccount 更新云厂商账号
func (h *ConfigHandler) UpdateProviderAccount(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "invalid id",
		})
		return
	}

	var account model.ProviderAccount
	if err := c.ShouldBindJSON(&account); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: err.Error(),
		})
		return
	}

	account.ID = uint(id)
	if err := h.configService.UpdateProviderAccount(&account); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "Provider account updated successfully",
		Data:    account,
	})
}

// DeleteProviderAccount 删除云厂商账号
func (h *ConfigHandler) DeleteProviderAccount(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "invalid id",
		})
		return
	}

	if err := h.configService.DeleteProviderAccount(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "Provider account deleted successfully",
	})
}

// ========== IM 配置 ==========

// GetIMConfig 获取 IM 配置
func (h *ConfigHandler) GetIMConfig(c *gin.Context) {
	platform := c.Param("platform")

	config, err := h.configService.GetIMConfig(platform)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "success",
		Data:    config,
	})
}

// SaveIMConfig 保存 IM 配置
func (h *ConfigHandler) SaveIMConfig(c *gin.Context) {
	platform := c.Param("platform")

	var config model.IMConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: err.Error(),
		})
		return
	}

	config.Platform = platform
	if err := h.configService.SaveIMConfig(&config); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "IM configuration saved successfully",
		Data:    config,
	})
}

// ListIMConfigs 列出所有 IM 配置
func (h *ConfigHandler) ListIMConfigs(c *gin.Context) {
	configs, err := h.configService.ListIMConfigs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "success",
		Data:    configs,
	})
}

// ========== CICD 配置 ==========

// GetCICDConfig 获取 CICD 配置
func (h *ConfigHandler) GetCICDConfig(c *gin.Context) {
	platform := c.Param("platform")

	config, err := h.configService.GetCICDConfig(platform)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "success",
		Data:    config,
	})
}

// SaveCICDConfig 保存 CICD 配置
func (h *ConfigHandler) SaveCICDConfig(c *gin.Context) {
	platform := c.Param("platform")

	var config model.CICDConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: err.Error(),
		})
		return
	}

	config.Platform = platform
	if err := h.configService.SaveCICDConfig(&config); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "CICD configuration saved successfully",
		Data:    config,
	})
}

// ListCICDConfigs 列出所有 CICD 配置
func (h *ConfigHandler) ListCICDConfigs(c *gin.Context) {
	configs, err := h.configService.ListCICDConfigs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "success",
		Data:    configs,
	})
}

// ========== MCP Server 配置 ==========

// ListMCPServers 列出 MCP 服务器
func (h *ConfigHandler) ListMCPServers(c *gin.Context) {
	servers, err := h.configService.ListMCPServers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "success",
		Data:    servers,
	})
}

// GetMCPServer 获取 MCP 服务器详情
func (h *ConfigHandler) GetMCPServer(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "invalid id",
		})
		return
	}

	server, err := h.configService.GetMCPServer(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "success",
		Data:    server,
	})
}

// CreateMCPServer 创建 MCP 服务器
func (h *ConfigHandler) CreateMCPServer(c *gin.Context) {
	var server model.MCPServer
	if err := c.ShouldBindJSON(&server); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: err.Error(),
		})
		return
	}

	if err := h.configService.CreateMCPServer(&server); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "MCP server created successfully",
		Data:    server,
	})
}

// UpdateMCPServer 更新 MCP 服务器
func (h *ConfigHandler) UpdateMCPServer(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "invalid id",
		})
		return
	}

	var server model.MCPServer
	if err := c.ShouldBindJSON(&server); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: err.Error(),
		})
		return
	}

	server.ID = uint(id)
	if err := h.configService.UpdateMCPServer(&server); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "MCP server updated successfully",
		Data:    server,
	})
}

// DeleteMCPServer 删除 MCP 服务器
func (h *ConfigHandler) DeleteMCPServer(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "invalid id",
		})
		return
	}

	if err := h.configService.DeleteMCPServer(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "MCP server deleted successfully",
	})
}

// ========== 系统配置 ==========

// ListSystemConfigs 列出所有系统配置
func (h *ConfigHandler) ListSystemConfigs(c *gin.Context) {
	configs, err := h.configService.ListSystemConfigs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "success",
		Data:    configs,
	})
}

// GetSystemConfig 获取系统配置
func (h *ConfigHandler) GetSystemConfig(c *gin.Context) {
	key := c.Param("key")

	config, err := h.configService.GetSystemConfig(key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "success",
		Data:    config,
	})
}

// SetSystemConfig 设置系统配置
func (h *ConfigHandler) SetSystemConfig(c *gin.Context) {
	var req struct {
		Key         string `json:"key" binding:"required"`
		Value       string `json:"value" binding:"required"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: err.Error(),
		})
		return
	}

	if err := h.configService.SetSystemConfig(req.Key, req.Value, req.Description); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "System configuration saved successfully",
	})
}
