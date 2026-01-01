package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// VersionHandler 版本信息处理器
type VersionHandler struct {
	version   string
	gitCommit string
	buildTime string
}

// VersionInfo 版本信息响应
type VersionInfo struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	BuildTime string `json:"build_time"`
}

var globalVersionHandler *VersionHandler

// InitVersionHandler 初始化全局版本处理器
func InitVersionHandler(version, gitCommit, buildTime string) {
	globalVersionHandler = &VersionHandler{
		version:   version,
		gitCommit: gitCommit,
		buildTime: buildTime,
	}
}

// GetVersionInfo 获取版本信息
func (h *VersionHandler) GetVersionInfo(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": VersionInfo{
			Version:   h.version,
			GitCommit: h.gitCommit,
			BuildTime: h.buildTime,
		},
	})
}

// GetVersionInfo 获取版本信息的快捷函数
func GetVersionInfo(c *gin.Context) {
	if globalVersionHandler == nil {
		globalVersionHandler = &VersionHandler{
			version:   "dev",
			gitCommit: "unknown",
			buildTime: "unknown",
		}
	}
	globalVersionHandler.GetVersionInfo(c)
}
