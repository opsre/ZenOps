package server

import (
	"net/http"
	"strings"

	"github.com/eryajf/zenops/internal/middleware"
	"github.com/eryajf/zenops/internal/model"
	"github.com/eryajf/zenops/internal/service"
	"github.com/gin-gonic/gin"
)

// AuthHandler 认证处理器
type AuthHandler struct {
	configService *service.ConfigService
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler() *AuthHandler {
	return &AuthHandler{
		configService: service.NewConfigService(),
	}
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	AccessToken string       `json:"accessToken"`
	UserInfo    UserInfoData `json:"userInfo"`
}

// UserInfoData 用户信息
type UserInfoData struct {
	ID       uint     `json:"id"`
	Username string   `json:"username"`
	Nickname string   `json:"nickname"`
	Email    string   `json:"email"`
	Avatar   string   `json:"avatar,omitempty"`
	Roles    []string `json:"roles"`
}

// Login 用户登录
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	db := h.configService.GetDB()

	// 查询用户
	var user model.User
	if err := db.Where("username = ?", req.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusOK, Response{
			Code:    401,
			Message: "用户名或密码错误",
		})
		return
	}

	// 检查用户是否启用
	if !user.Enabled {
		c.JSON(http.StatusOK, Response{
			Code:    403,
			Message: "用户已被禁用",
		})
		return
	}

	// 验证密码
	if !user.CheckPassword(req.Password) {
		c.JSON(http.StatusOK, Response{
			Code:    401,
			Message: "用户名或密码错误",
		})
		return
	}

	// 生成 JWT Token
	token, err := middleware.GenerateToken(user.ID, user.Username, user.Roles)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "生成令牌失败: " + err.Error(),
		})
		return
	}

	// 解析角色列表
	var roles []string
	if user.Roles != "" {
		roles = strings.Split(user.Roles, ",")
	} else {
		roles = []string{"user"}
	}

	// 返回登录成功响应
	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "登录成功",
		Data: LoginResponse{
			AccessToken: token,
			UserInfo: UserInfoData{
				ID:       user.ID,
				Username: user.Username,
				Nickname: user.Nickname,
				Email:    user.Email,
				Avatar:   user.Avatar,
				Roles:    roles,
			},
		},
	})
}

// Logout 用户登出
func (h *AuthHandler) Logout(c *gin.Context) {
	// JWT 是无状态的，登出只需要客户端删除 token 即可
	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "登出成功",
	})
}

// GetUserInfo 获取当前用户信息
func (h *AuthHandler) GetUserInfo(c *gin.Context) {
	// 从中间件中获取用户信息
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, Response{
			Code:    401,
			Message: "未认证",
		})
		return
	}

	db := h.configService.GetDB()

	// 查询用户详细信息
	var user model.User
	if err := db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, Response{
			Code:    404,
			Message: "用户不存在",
		})
		return
	}

	// 解析角色列表
	var roles []string
	if user.Roles != "" {
		roles = strings.Split(user.Roles, ",")
	} else {
		roles = []string{"user"}
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "success",
		Data: UserInfoData{
			ID:       user.ID,
			Username: user.Username,
			Nickname: user.Nickname,
			Email:    user.Email,
			Avatar:   user.Avatar,
			Roles:    roles,
		},
	})
}

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=6"`
}

// ChangePassword 修改密码
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	// 从中间件中获取用户信息
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, Response{
			Code:    401,
			Message: "未认证",
		})
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	db := h.configService.GetDB()

	// 查询用户
	var user model.User
	if err := db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, Response{
			Code:    404,
			Message: "用户不存在",
		})
		return
	}

	// 验证旧密码
	if !user.CheckPassword(req.OldPassword) {
		c.JSON(http.StatusOK, Response{
			Code:    400,
			Message: "原密码错误",
		})
		return
	}

	// 设置新密码
	if err := user.SetPassword(req.NewPassword); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "设置新密码失败: " + err.Error(),
		})
		return
	}

	// 更新数据库
	if err := db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "保存密码失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "密码修改成功",
	})
}
