package provider

import (
	"fmt"
	"sync"
)

var (
	// providers 存储所有已注册的 Provider
	providers = make(map[string]Provider)
	// cicdProviders 存储所有已注册的 CICD Provider
	cicdProviders = make(map[string]CICDProvider)
	mu            sync.RWMutex
)

// Register 注册一个 Provider
func Register(name string, provider Provider) {
	mu.Lock()
	defer mu.Unlock()
	if provider == nil {
		panic("provider: Register provider is nil")
	}
	if _, dup := providers[name]; dup {
		panic("provider: Register called twice for provider " + name)
	}
	providers[name] = provider
}

// RegisterCICD 注册一个 CICD Provider
func RegisterCICD(name string, provider CICDProvider) {
	mu.Lock()
	defer mu.Unlock()
	if provider == nil {
		panic("provider: Register CICD provider is nil")
	}
	if _, dup := cicdProviders[name]; dup {
		panic("provider: Register called twice for CICD provider " + name)
	}
	cicdProviders[name] = provider
}

// GetProvider 获取指定名称的 Provider
func GetProvider(name string) (Provider, error) {
	mu.RLock()
	defer mu.RUnlock()
	provider, ok := providers[name]
	if !ok {
		return nil, fmt.Errorf("provider %s not found", name)
	}
	return provider, nil
}

// GetCICDProvider 获取指定名称的 CICD Provider
func GetCICDProvider(name string) (CICDProvider, error) {
	mu.RLock()
	defer mu.RUnlock()
	provider, ok := cicdProviders[name]
	if !ok {
		return nil, fmt.Errorf("CICD provider %s not found", name)
	}
	return provider, nil
}

// ListProviders 列出所有已注册的 Provider 名称
func ListProviders() []string {
	mu.RLock()
	defer mu.RUnlock()
	names := make([]string, 0, len(providers))
	for name := range providers {
		names = append(names, name)
	}
	return names
}

// ListCICDProviders 列出所有已注册的 CICD Provider 名称
func ListCICDProviders() []string {
	mu.RLock()
	defer mu.RUnlock()
	names := make([]string, 0, len(cicdProviders))
	for name := range cicdProviders {
		names = append(names, name)
	}
	return names
}

// UnregisterAll 清空所有已注册的 Provider (用于测试)
func UnregisterAll() {
	mu.Lock()
	defer mu.Unlock()
	providers = make(map[string]Provider)
	cicdProviders = make(map[string]CICDProvider)
}
