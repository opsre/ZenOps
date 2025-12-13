package tencent

import "github.com/eryajf/zenops/internal/provider"

func init() {
	provider.Register("tencent", NewTencentProvider())
}
