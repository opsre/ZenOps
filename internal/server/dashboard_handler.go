package server

import (
	"net/http"

	"github.com/eryajf/zenops/internal/model"
	"github.com/eryajf/zenops/internal/service"
	"github.com/gin-gonic/gin"
)

// DashboardHandler 仪表盘处理器
type DashboardHandler struct {
	configService *service.ConfigService
}

// NewDashboardHandler 创建仪表盘处理器
func NewDashboardHandler() *DashboardHandler {
	return &DashboardHandler{
		configService: service.NewConfigService(),
	}
}

// GetStats 获取仪表盘统计数据
func (h *DashboardHandler) GetStats(c *gin.Context) {
	db := h.configService.GetDB()

	// 统计活跃 IM 机器人数量
	var activeBots int64
	db.Model(&model.IMConfig{}).Where("enabled = ?", true).Count(&activeBots)

	// 统计 MCP 服务器数量
	var totalServers int64
	db.Model(&model.MCPServer{}).Count(&totalServers)

	// 统计 MCP 工具总数（只统计属于有效服务器的工具）
	var totalTools int64
	db.Model(&model.MCPTool{}).
		Joins("INNER JOIN mcp_servers ON mcp_tools.server_id = mcp_servers.id").
		Count(&totalTools)

	// 计算成功率(基于最近的日志)
	var totalLogs int64
	var successLogs int64
	db.Model(&model.MCPLog{}).Count(&totalLogs)
	db.Model(&model.MCPLog{}).Where("status = ?", "success").Count(&successLogs)

	successRate := 0.0
	if totalLogs > 0 {
		successRate = float64(successLogs) / float64(totalLogs) * 100
	}

	// 计算平均延迟
	var avgLatency float64
	db.Model(&model.MCPLog{}).Select("AVG(latency)").Row().Scan(&avgLatency)

	// 统计对话总次数
	var totalChats int64
	db.Model(&model.ChatLog{}).Count(&totalChats)

	// MCP 调用总次数就是 totalLogs
	totalMCPCalls := totalLogs

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data: gin.H{
			"activeBots":    activeBots,
			"totalServers":  totalServers,
			"totalTools":    totalTools,
			"successRate":   successRate,
			"avgLatency":    int(avgLatency),
			"totalMCPCalls": totalMCPCalls,
			"totalChats":    totalChats,
		},
	})
}

// GetHealth 获取基础设施健康状态
func (h *DashboardHandler) GetHealth(c *gin.Context) {
	db := h.configService.GetDB()
	components := []gin.H{}

	// 检查 IM 配置状态
	var imConfigs []model.IMConfig
	db.Find(&imConfigs)

	for _, im := range imConfigs {
		status := "offline"
		uptime := "0%"
		if im.Enabled {
			status = "online"
			uptime = "99.9%" // TODO: 实际应该从监控数据计算
		}

		components = append(components, gin.H{
			"label":  im.Platform + " Gateway",
			"status": status,
			"uptime": uptime,
			"detail": "",
		})
	}

	// 检查 LLM Provider 状态
	var llmConfigs []model.LLMConfig
	if err := db.Find(&llmConfigs).Error; err == nil {
		for _, llm := range llmConfigs {
			status := "offline"
			uptime := "0%"
			if llm.Enabled {
				status = "online"
				uptime = "98.5%" // TODO: 实际应该从监控数据计算
			}

			components = append(components, gin.H{
				"label":  llm.Name + " Provider",
				"status": status,
				"uptime": uptime,
			})
		}
	}

	// 添加 MCP Grid 状态
	var mcpCount int64
	db.Model(&model.MCPServer{}).Where("is_active = ?", true).Count(&mcpCount)
	mcpStatus := "offline"
	if mcpCount > 0 {
		mcpStatus = "online"
	}
	components = append(components, gin.H{
		"label":  "MCP Grid",
		"status": mcpStatus,
		"uptime": "100%",
	})

	// 添加数据库状态
	components = append(components, gin.H{
		"label":  "SQLite Database",
		"status": "online",
		"uptime": "100%",
	})

	// 检查云厂商状态
	var aliyunAccounts []model.ProviderAccount
	db.Where("provider = ? AND enabled = ?", "aliyun", true).Find(&aliyunAccounts)
	if len(aliyunAccounts) > 0 {
		components = append(components, gin.H{
			"label":  "Aliyun API",
			"status": "online",
			"uptime": "99.2%",
			"detail": "",
		})
	}

	var tencentAccounts []model.ProviderAccount
	db.Where("provider = ? AND enabled = ?", "tencent", true).Find(&tencentAccounts)
	if len(tencentAccounts) > 0 {
		components = append(components, gin.H{
			"label":  "Tencent Cloud",
			"status": "online",
			"uptime": "99.8%",
		})
	}

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data: gin.H{
			"components": components,
		},
	})
}
