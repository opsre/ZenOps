package jenkins

import "github.com/eryajf/zenops/internal/provider"

func init() {
	provider.RegisterCICD("jenkins", NewJenkinsProvider())
}
