package model

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// User 用户模型
type User struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	Username string `gorm:"uniqueIndex;not null;size:50" json:"username"`
	Password string `gorm:"not null;size:255" json:"-"` // 不在 JSON 中返回密码
	Nickname string `gorm:"size:100" json:"nickname"`
	Email    string `gorm:"size:100" json:"email"`
	Avatar   string `gorm:"size:255" json:"avatar,omitempty"`
	Roles    string `gorm:"size:255;default:'user'" json:"roles"` // 角色列表，逗号分隔
	Enabled  bool   `gorm:"default:true" json:"enabled"`
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}

// SetPassword 设置密码（加密）
func (u *User) SetPassword(password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashedPassword)
	return nil
}

// CheckPassword 验证密码
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}
