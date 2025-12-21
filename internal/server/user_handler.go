package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// UserHandler 用户处理器
type UserHandler struct{}

// NewUserHandler 创建用户处理器
func NewUserHandler() *UserHandler {
	return &UserHandler{}
}

// GetUserInfo 获取用户信息 (开发模式使用的模拟接口)
func (h *UserHandler) GetUserInfo(c *gin.Context) {
	// 返回模拟的管理员用户信息
	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "success",
		Data: gin.H{
			"id":       1,
			"username": "admin",
			"nickname": "管理员",
			"email":    "admin@zenops.local",
			"roles":    []string{"R_SUPER", "R_ADMIN"},
			"avatar":   "",
		},
	})
}

// Login 用户登录 (开发模式使用的模拟接口)
func (h *UserHandler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: err.Error(),
		})
		return
	}

	// 简单的模拟登录，任何用户名密码都可以登录
	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "登录成功",
		Data: gin.H{
			"accessToken": "mock-access-token",
			"userInfo": gin.H{
				"id":       1,
				"username": req.Username,
				"nickname": "管理员",
				"email":    "admin@zenops.local",
				"roles":    []string{"R_SUPER", "R_ADMIN"},
			},
		},
	})
}

// Logout 用户登出
func (h *UserHandler) Logout(c *gin.Context) {
	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "登出成功",
	})
}

// GetMenuList 获取菜单列表 (开发模式使用的模拟接口)
func (h *UserHandler) GetMenuList(c *gin.Context) {
	// 返回空菜单，让前端使用路由模块中定义的菜单
	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "success",
		Data:    []interface{}{},
	})
}
