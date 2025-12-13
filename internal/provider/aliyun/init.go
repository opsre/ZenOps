package aliyun

import (
	"github.com/eryajf/zenops/internal/provider"
)

func init() {
	// 注册阿里云 Provider
	provider.Register("aliyun", NewProvider())
}
