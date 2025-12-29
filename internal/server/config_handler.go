package server

import (
	"fmt"
	"net/http"
	"strconv"

	"cnb.cool/zhiqiangwang/pkg/logx"
	"github.com/eryajf/zenops/internal/mcpclient"
	"github.com/eryajf/zenops/internal/model"
	"github.com/eryajf/zenops/internal/service"
	"github.com/gin-gonic/gin"
)

// 全局 MCP 客户端管理器
var globalMCPManager *mcpclient.Manager

// GetGlobalMCPManager 获取全局 MCP 客户端管理器
func GetGlobalMCPManager() *mcpclient.Manager {
	if globalMCPManager == nil {
		globalMCPManager = mcpclient.NewManager()
	}
	return globalMCPManager
}

// SetGlobalMCPManager 设置全局 MCP 客户端管理器
func SetGlobalMCPManager(m *mcpclient.Manager) {
	globalMCPManager = m
}

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

// ListLLMConfigs 列出所有 LLM 配置
func (h *ConfigHandler) ListLLMConfigs(c *gin.Context) {
	configs, err := h.configService.ListLLMConfigs()
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

// GetLLMConfig 获取指定 LLM 配置
func (h *ConfigHandler) GetLLMConfig(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "invalid id",
		})
		return
	}

	config, err := h.configService.GetLLMConfig(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	if config == nil {
		c.JSON(http.StatusNotFound, Response{
			Code:    404,
			Message: "LLM config not found",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "success",
		Data:    config,
	})
}

// CreateLLMConfig 创建 LLM 配置
func (h *ConfigHandler) CreateLLMConfig(c *gin.Context) {
	var config model.LLMConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: err.Error(),
		})
		return
	}

	if err := h.configService.CreateLLMConfig(&config); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "LLM configuration created successfully",
		Data:    config,
	})
}

// UpdateLLMConfig 更新 LLM 配置
func (h *ConfigHandler) UpdateLLMConfig(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "invalid id",
		})
		return
	}

	var config model.LLMConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: err.Error(),
		})
		return
	}

	config.ID = uint(id)
	if err := h.configService.UpdateLLMConfig(&config); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "LLM configuration updated successfully",
		Data:    config,
	})
}

// DeleteLLMConfig 删除 LLM 配置
func (h *ConfigHandler) DeleteLLMConfig(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "invalid id",
		})
		return
	}

	if err := h.configService.DeleteLLMConfig(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "LLM configuration deleted successfully",
	})
}

// ToggleLLMConfig 切换 LLM 配置启用状态
func (h *ConfigHandler) ToggleLLMConfig(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "invalid id",
		})
		return
	}

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

	config, err := h.configService.GetLLMConfig(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	if config == nil {
		c.JSON(http.StatusNotFound, Response{
			Code:    404,
			Message: "LLM config not found",
		})
		return
	}

	config.Enabled = req.Enabled
	if err := h.configService.UpdateLLMConfig(config); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "LLM configuration status updated successfully",
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

	// 返回前端期望的格式
	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "Provider account created successfully",
		Data: gin.H{
			"id":      account.ID,
			"name":    account.Name,
			"enabled": account.Enabled,
			"ak":      account.AccessKey,
			"sk":      account.SecretKey,
			"regions": account.Regions,
		},
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

	// 返回前端期望的格式
	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "Provider account updated successfully",
		Data: gin.H{
			"id":      account.ID,
			"name":    account.Name,
			"enabled": account.Enabled,
			"ak":      account.AccessKey,
			"sk":      account.SecretKey,
			"regions": account.Regions,
		},
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
		Data: gin.H{
			"servers": servers,
		},
	})
}

// GetMCPServerByName 根据名称获取 MCP 服务器详情
func (h *ConfigHandler) GetMCPServerByName(c *gin.Context) {
	name := c.Param("name")

	server, err := h.configService.GetMCPServerByName(name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	if server == nil {
		c.JSON(http.StatusNotFound, Response{
			Code:    404,
			Message: "MCP server not found",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "success",
		Data: gin.H{
			"server": server,
		},
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
		Data: gin.H{
			"server": server,
		},
	})
}

// UpdateMCPServerByName 根据名称更新 MCP 服务器
func (h *ConfigHandler) UpdateMCPServerByName(c *gin.Context) {
	name := c.Param("name")

	// 先查找现有服务器
	existing, err := h.configService.GetMCPServerByName(name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	if existing == nil {
		c.JSON(http.StatusNotFound, Response{
			Code:    404,
			Message: "MCP server not found",
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

	// 保留原有ID和名称
	server.ID = existing.ID
	server.Name = name

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
		Data: gin.H{
			"server": server,
		},
	})
}

// DeleteMCPServerByName 根据名称删除 MCP 服务器
func (h *ConfigHandler) DeleteMCPServerByName(c *gin.Context) {
	name := c.Param("name")

	server, err := h.configService.GetMCPServerByName(name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	if server == nil {
		c.JSON(http.StatusNotFound, Response{
			Code:    404,
			Message: "MCP server not found",
		})
		return
	}

	if err := h.configService.DeleteMCPServer(server.ID); err != nil {
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

// ToggleMCPServer 切换 MCP 服务器启用状态
func (h *ConfigHandler) ToggleMCPServer(c *gin.Context) {
	name := c.Param("name")

	var req struct {
		IsActive bool `json:"isActive"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: err.Error(),
		})
		return
	}

	server, err := h.configService.GetMCPServerByName(name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	if server == nil {
		c.JSON(http.StatusNotFound, Response{
			Code:    404,
			Message: "MCP server not found",
		})
		return
	}

	// 获取全局 MCP 管理器
	mcpManager := GetGlobalMCPManager()

	// 根据启用/禁用状态执行连接/断开操作
	if req.IsActive {
		// 启用：尝试连接 MCP 服务器
		if !mcpManager.IsRegistered(name) {
			// 转换 env 和 headers
			env := make(map[string]string)
			if server.Env != nil {
				for k, v := range server.Env {
					// 尝试多种类型转换
					switch val := v.(type) {
					case string:
						env[k] = val
					case fmt.Stringer:
						env[k] = val.String()
					default:
						env[k] = fmt.Sprintf("%v", val)
					}
				}
			}
			headers := make(map[string]string)
			if server.Headers != nil {
				for k, v := range server.Headers {
					// 尝试多种类型转换
					switch val := v.(type) {
					case string:
						headers[k] = val
					case fmt.Stringer:
						headers[k] = val.String()
					default:
						headers[k] = fmt.Sprintf("%v", val)
					}
				}
			}

			// 注册并连接 MCP 客户端
			logx.Info("Attempting to register MCP server: %s (type: %s, command: %s, args: %v)",
				name, server.Type, server.Command, server.Args)

			if err := mcpManager.RegisterFromDB(
				name,
				server.Type,
				server.Command,
				server.Args,
				env,
				server.BaseURL,
				headers,
				server.Timeout,
			); err != nil {
				logx.Error("Failed to register MCP server %s: %v", name, err)
				c.JSON(http.StatusInternalServerError, Response{
					Code:    500,
					Message: fmt.Sprintf("Failed to connect MCP server: %v", err),
				})
				return
			}

			// 连接成功后，获取并保存工具列表到数据库
			mcpClient, err := mcpManager.Get(name)
			if err == nil && mcpClient != nil {
				// 先删除该服务器的旧工具
				if err := h.configService.DeleteMCPToolsByServerID(server.ID); err != nil {
					fmt.Printf("Warning: Failed to delete old tools for server %s: %v\n", name, err)
				}

				// 保存新的工具列表
				for _, tool := range mcpClient.Tools {
					// 转换 InputSchema
					inputSchema := make(map[string]interface{})
					if tool.InputSchema.Type != "" {
						inputSchema["type"] = tool.InputSchema.Type
					}
					if tool.InputSchema.Properties != nil {
						inputSchema["properties"] = tool.InputSchema.Properties
					}
					if tool.InputSchema.Required != nil {
						inputSchema["required"] = tool.InputSchema.Required
					}

					mcpTool := model.MCPTool{
						ServerID:    server.ID,
						Name:        tool.Name,
						Description: tool.Description,
						IsEnabled:   true,
						InputSchema: inputSchema,
					}
					if err := h.configService.CreateMCPTool(&mcpTool); err != nil {
						fmt.Printf("Warning: Failed to save tool %s for server %s: %v\n", tool.Name, name, err)
					}
				}
			}
		}
	} else {
		// 禁用：断开 MCP 服务器连接
		if mcpManager.IsRegistered(name) {
			if err := mcpManager.Unregister(name); err != nil {
				// 忽略断开连接的错误，继续更新状态
				fmt.Printf("Warning: Failed to disconnect MCP server %s: %v\n", name, err)
			}
		}
	}

	// 更新数据库状态
	server.IsActive = req.IsActive
	if err := h.configService.UpdateMCPServer(server); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	// 重新加载服务器信息（包括工具列表）
	updatedServer, err := h.configService.GetMCPServerByName(name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	status := "disconnected"
	if req.IsActive && mcpManager.IsRegistered(name) {
		status = "connected"
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "MCP server status updated successfully",
		Data: gin.H{
			"server": updatedServer,
			"status": status,
		},
	})
}

// GetMCPTools 获取 MCP 服务器的工具列表
func (h *ConfigHandler) GetMCPTools(c *gin.Context) {
	_ = c.Param("name") // serverName for future use

	// TODO: 实现获取MCP服务器工具列表的逻辑
	// 这需要与MCP服务器实际通信,暂时返回空列表
	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "success",
		Data: gin.H{
			"tools": []interface{}{},
		},
	})
}

// TestMCPTool 测试调用 MCP 工具
func (h *ConfigHandler) TestMCPTool(c *gin.Context) {
	serverName := c.Param("name")
	toolName := c.Param("toolName")

	var args interface{}
	if err := c.ShouldBindJSON(&args); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: err.Error(),
		})
		return
	}

	// TODO: 实现测试调用MCP工具的逻辑
	// 这需要与MCP服务器实际通信
	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "success",
		Data: gin.H{
			"result": gin.H{
				"server_name": serverName,
				"tool_name":   toolName,
				"args":        args,
				"content": []gin.H{
					{
						"type": "text",
						"text": "Tool test not implemented yet",
					},
				},
			},
		},
	})
}

// ========== Integration (IM) 配置 - 前端兼容接口 ==========

// ListIntegrationConfigs 列出所有集成应用配置
func (h *ConfigHandler) ListIntegrationConfigs(c *gin.Context) {
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

// GetIntegrationConfig 获取指定集成应用配置
func (h *ConfigHandler) GetIntegrationConfig(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "invalid id",
		})
		return
	}

	config, err := h.configService.GetIMConfigByID(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	if config == nil {
		c.JSON(http.StatusNotFound, Response{
			Code:    404,
			Message: "Integration config not found",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "success",
		Data:    config,
	})
}

// CreateIntegrationConfig 创建集成应用配置
func (h *ConfigHandler) CreateIntegrationConfig(c *gin.Context) {
	var config model.IMConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: err.Error(),
		})
		return
	}

	if err := h.configService.SaveIMConfig(&config); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "Integration configuration created successfully",
		Data:    config,
	})
}

// UpdateIntegrationConfig 更新集成应用配置
func (h *ConfigHandler) UpdateIntegrationConfig(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "invalid id",
		})
		return
	}

	var config model.IMConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: err.Error(),
		})
		return
	}

	config.ID = uint(id)
	if err := h.configService.SaveIMConfig(&config); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "Integration configuration updated successfully",
		Data:    config,
	})
}

// DeleteIntegrationConfig 删除集成应用配置
func (h *ConfigHandler) DeleteIntegrationConfig(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "invalid id",
		})
		return
	}

	if err := h.configService.DeleteIMConfig(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "Integration configuration deleted successfully",
	})
}

// ========== Jenkins 配置便捷接口 ==========

// GetJenkinsConfig 获取 Jenkins 配置
func (h *ConfigHandler) GetJenkinsConfig(c *gin.Context) {
	config, err := h.configService.GetCICDConfig("jenkins")
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

// SaveJenkinsConfig 保存 Jenkins 配置
func (h *ConfigHandler) SaveJenkinsConfig(c *gin.Context) {
	var config model.CICDConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: err.Error(),
		})
		return
	}

	config.Platform = "jenkins"
	if err := h.configService.SaveCICDConfig(&config); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "Jenkins configuration saved successfully",
		Data:    config,
	})
}

// ========== 服务器配置 ==========

// GetServerConfig 获取服务器配置
func (h *ConfigHandler) GetServerConfig(c *gin.Context) {
	// 从系统配置中读取服务器相关配置
	keys := []string{
		model.ConfigKeyServerHTTPEnabled,
		model.ConfigKeyServerHTTPPort,
		model.ConfigKeyServerMCPEnabled,
		model.ConfigKeyServerMCPPort,
		model.ConfigKeyServerMCPAutoRegisterExternalTools,
		model.ConfigKeyServerMCPToolNameFormat,
	}

	serverConfig := make(map[string]string)
	for _, key := range keys {
		config, err := h.configService.GetSystemConfig(key)
		if err != nil {
			c.JSON(http.StatusInternalServerError, Response{
				Code:    500,
				Message: err.Error(),
			})
			return
		}
		if config != nil {
			serverConfig[key] = config.ConfigValue
		}
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "success",
		Data:    serverConfig,
	})
}

// SaveServerConfig 保存服务器配置
func (h *ConfigHandler) SaveServerConfig(c *gin.Context) {
	var configMap map[string]interface{}
	if err := c.ShouldBindJSON(&configMap); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: err.Error(),
		})
		return
	}

	// 保存每个配置项
	configKeys := map[string]string{
		"http_enabled":                      model.ConfigKeyServerHTTPEnabled,
		"http_port":                         model.ConfigKeyServerHTTPPort,
		"mcp_enabled":                       model.ConfigKeyServerMCPEnabled,
		"mcp_port":                          model.ConfigKeyServerMCPPort,
		"auto_register_external_tools":      model.ConfigKeyServerMCPAutoRegisterExternalTools,
		"tool_name_format":                  model.ConfigKeyServerMCPToolNameFormat,
	}

	for jsonKey, dbKey := range configKeys {
		if value, ok := configMap[jsonKey]; ok {
			var strValue string
			switch v := value.(type) {
			case string:
				strValue = v
			case bool:
				if v {
					strValue = "true"
				} else {
					strValue = "false"
				}
			case float64:
				strValue = strconv.FormatFloat(v, 'f', 0, 64)
			default:
				strValue = fmt.Sprintf("%v", v)
			}

			if err := h.configService.SetSystemConfig(dbKey, strValue, ""); err != nil {
				c.JSON(http.StatusInternalServerError, Response{
					Code:    500,
					Message: err.Error(),
				})
				return
			}
		}
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "Server configuration saved successfully",
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

// ========== 全量配置 ==========

// GetAllConfig 获取全量配置
func (h *ConfigHandler) GetAllConfig(c *gin.Context) {
	// 获取 LLM 配置列表
	llmConfigs, err := h.configService.ListLLMConfigs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	// 获取 IM 配置
	imConfigs, err := h.configService.ListIMConfigs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	// 获取云厂商账号并转换为前端格式
	aliyunAccounts, _ := h.configService.ListProviderAccounts("aliyun")
	tencentAccounts, _ := h.configService.ListProviderAccounts("tencent")

	// 转换为前端期望的格式 (ak/sk)
	convertAccounts := func(accounts []model.ProviderAccount) []gin.H {
		result := make([]gin.H, len(accounts))
		for i, acc := range accounts {
			result[i] = gin.H{
				"id":      acc.ID,
				"name":    acc.Name,
				"enabled": acc.Enabled,
				"ak":      acc.AccessKey,
				"sk":      acc.SecretKey,
				"regions": acc.Regions,
			}
		}
		return result
	}

	// 获取系统配置
	serverConfigs, _ := h.configService.ListSystemConfigs()

	// 构建响应
	config := gin.H{
		"server": gin.H{
			"http": gin.H{
				"enabled": true,
				"port":    8080,
			},
			"mcp": gin.H{
				"enabled":                      true,
				"port":                         8081,
				"auto_register_external_tools": true,
				"tool_name_format":             "{prefix}{name}",
			},
		},
		"logger": gin.H{
			"level": "info",
			"file":  "./zenops.log",
		},
		"database": gin.H{
			"driver": "sqlite",
			"dsn":    "zenops.db",
		},
		"llm_providers": llmConfigs,
		"dingtalk":      gin.H{"enabled": false},
		"feishu":        gin.H{"enabled": false},
		"wecom":         gin.H{"enabled": false},
		"providers": gin.H{
			"aliyun":  convertAccounts(aliyunAccounts),
			"tencent": convertAccounts(tencentAccounts),
		},
		"auth": gin.H{
			"enabled": false,
			"type":    "token",
			"tokens":  []string{},
		},
		"cache": gin.H{
			"enabled": true,
			"type":    "memory",
			"ttl":     300,
		},
		"mcp_servers_config": "./mcp_servers.json",
	}

	// 填充 IM 配置
	for _, im := range imConfigs {
		switch im.Platform {
		case "dingtalk":
			config["dingtalk"] = im
		case "feishu":
			config["feishu"] = im
		case "wecom":
			config["wecom"] = im
		}
	}

	// 从系统配置中读取服务器配置
	for _, sc := range serverConfigs {
		switch sc.ConfigKey {
		case model.ConfigKeyServerHTTPPort:
			if port, err := strconv.Atoi(sc.ConfigValue); err == nil {
				serverConfig := config["server"].(gin.H)
				httpConfig := serverConfig["http"].(gin.H)
				httpConfig["port"] = port
			}
		case model.ConfigKeyServerMCPPort:
			if port, err := strconv.Atoi(sc.ConfigValue); err == nil {
				serverConfig := config["server"].(gin.H)
				mcpConfig := serverConfig["mcp"].(gin.H)
				mcpConfig["port"] = port
			}
		}
	}

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    config,
	})
}
