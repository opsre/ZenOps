package config

// Config 应用配置
type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	Providers ProvidersConfig `mapstructure:"providers"`
	CICD      CICDConfig      `mapstructure:"cicd"`
	DingTalk  DingTalkConfig  `mapstructure:"dingtalk"`
	Feishu    FeishuConfig    `mapstructure:"feishu"`
	LLM       LLMConfig       `mapstructure:"llm"`
	Auth      AuthConfig      `mapstructure:"auth"`
	Cache     CacheConfig     `mapstructure:"cache"`
}

// ProvidersConfig 云服务提供商配置集合
type ProvidersConfig struct {
	Aliyun  []ProviderConfig `mapstructure:"aliyun"`
	Tencent []ProviderConfig `mapstructure:"tencent"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	HTTP HTTPConfig `mapstructure:"http"`
	MCP  MCPConfig  `mapstructure:"mcp"`
}

// HTTPConfig HTTP 服务配置
type HTTPConfig struct {
	Enabled bool `mapstructure:"enabled"`
	Port    int  `mapstructure:"port"`
	Debug   bool `mapstructure:"debug"` // Gin Debug 模式
}

// MCPConfig MCP 服务配置
type MCPConfig struct {
	Enabled bool `mapstructure:"enabled"`
	Port    int  `mapstructure:"port"`
}

// ProviderConfig 云服务提供商配置
type ProviderConfig struct {
	Name    string            `mapstructure:"name"` // 账号名称,用于区分多个账号
	Enabled bool              `mapstructure:"enabled"`
	AK      string            `mapstructure:"ak"`
	SK      string            `mapstructure:"sk"`
	Regions []string          `mapstructure:"regions"`
	Extra   map[string]string `mapstructure:"extra"`
}

// CICDConfig CI/CD 工具配置
type CICDConfig struct {
	Jenkins JenkinsConfig `mapstructure:"jenkins"`
}

// JenkinsConfig Jenkins 配置
type JenkinsConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	URL      string `mapstructure:"url"`
	Username string `mapstructure:"username"`
	Token    string `mapstructure:"token"`
}

// LLMConfig LLM 配置
type LLMConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Model   string `mapstructure:"model"`
	APIKey  string `mapstructure:"api_key"`
	BaseURL string `mapstructure:"base_url"` // 自定义 API 端点
}

// DingTalkConfig 钉钉配置
type DingTalkConfig struct {
	Enabled        bool   `mapstructure:"enabled"`
	AppKey         string `mapstructure:"app_key"`
	AppSecret      string `mapstructure:"app_secret"`
	AgentID        string `mapstructure:"agent_id"`
	CardTemplateID string `mapstructure:"card_template_id"` // AI 流式卡片模板 ID
}

// FeishuConfig 飞书配置
type FeishuConfig struct {
	Enabled   bool   `mapstructure:"enabled"`
	AppID     string `mapstructure:"app_id"`
	AppSecret string `mapstructure:"app_secret"`
}

// AuthConfig 认证配置
type AuthConfig struct {
	Enabled bool     `mapstructure:"enabled"`
	Type    string   `mapstructure:"type"` // token, basic, oauth2
	Tokens  []string `mapstructure:"tokens"`
}

// CacheConfig 缓存配置
type CacheConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Type    string `mapstructure:"type"` // memory, redis
	TTL     int    `mapstructure:"ttl"`  // 秒
}

var globalConfig *Config

// SetGlobalConfig 设置全局配置
func SetGlobalConfig(cfg *Config) {
	globalConfig = cfg
}

// GetGlobalConfig 获取全局配置
func GetGlobalConfig() *Config {
	return globalConfig
}
