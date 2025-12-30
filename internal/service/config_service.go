package service

import (
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/eryajf/zenops/internal/database"
	"github.com/eryajf/zenops/internal/model"
)

// ConfigService 配置服务
type ConfigService struct {
	db *gorm.DB
}

// NewConfigService 创建配置服务实例
func NewConfigService() *ConfigService {
	return &ConfigService{
		db: database.GetDB(),
	}
}

// GetDB 获取数据库连接
func (s *ConfigService) GetDB() *gorm.DB {
	return s.db
}

// ========== LLM 配置管理 ==========

// ListLLMConfigs 列出所有LLM配置
func (s *ConfigService) ListLLMConfigs() ([]model.LLMConfig, error) {
	var configs []model.LLMConfig
	err := s.db.Order("id").Find(&configs).Error
	return configs, err
}

// GetLLMConfig 获取指定ID的LLM配置
func (s *ConfigService) GetLLMConfig(id uint) (*model.LLMConfig, error) {
	var config model.LLMConfig
	err := s.db.First(&config, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &config, nil
}

// GetLLMConfigByName 根据名称获取LLM配置
func (s *ConfigService) GetLLMConfigByName(name string) (*model.LLMConfig, error) {
	var config model.LLMConfig
	err := s.db.Where("name = ?", name).First(&config).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &config, nil
}

// CreateLLMConfig 创建LLM配置
func (s *ConfigService) CreateLLMConfig(config *model.LLMConfig) error {
	// 检查是否已存在同名配置
	existing, err := s.GetLLMConfigByName(config.Name)
	if err != nil {
		return err
	}
	if existing != nil {
		return fmt.Errorf("LLM config already exists: %s", config.Name)
	}
	return s.db.Create(config).Error
}

// UpdateLLMConfig 更新LLM配置
func (s *ConfigService) UpdateLLMConfig(config *model.LLMConfig) error {
	return s.db.Save(config).Error
}

// DeleteLLMConfig 删除LLM配置
func (s *ConfigService) DeleteLLMConfig(id uint) error {
	return s.db.Delete(&model.LLMConfig{}, id).Error
}

// GetEnabledLLMConfigs 获取所有启用的LLM配置
func (s *ConfigService) GetEnabledLLMConfigs() ([]model.LLMConfig, error) {
	var configs []model.LLMConfig
	err := s.db.Where("enabled = ?", true).Order("id").Find(&configs).Error
	return configs, err
}

// GetDefaultLLMConfig 获取默认LLM配置（第一个启用的配置）
func (s *ConfigService) GetDefaultLLMConfig() (*model.LLMConfig, error) {
	var config model.LLMConfig
	err := s.db.Where("enabled = ?", true).Order("id").First(&config).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &config, nil
}

// ========== 云厂商账号配置管理 ==========

// ListProviderAccounts 列出云厂商账号
func (s *ConfigService) ListProviderAccounts(provider string) ([]model.ProviderAccount, error) {
	var accounts []model.ProviderAccount
	query := s.db.Order("provider, name")
	if provider != "" {
		query = query.Where("provider = ?", provider)
	}
	err := query.Find(&accounts).Error
	return accounts, err
}

// GetProviderAccount 获取指定云厂商账号
func (s *ConfigService) GetProviderAccount(id uint) (*model.ProviderAccount, error) {
	var account model.ProviderAccount
	err := s.db.First(&account, id).Error
	if err != nil {
		return nil, err
	}
	return &account, nil
}

// CreateProviderAccount 创建云厂商账号
func (s *ConfigService) CreateProviderAccount(account *model.ProviderAccount) error {
	// 检查是否已存在同名账号
	var existing model.ProviderAccount
	err := s.db.Where("provider = ? AND name = ?", account.Provider, account.Name).First(&existing).Error
	if err == nil {
		return fmt.Errorf("provider account already exists: %s/%s", account.Provider, account.Name)
	}
	if err != gorm.ErrRecordNotFound {
		return err
	}
	return s.db.Create(account).Error
}

// UpdateProviderAccount 更新云厂商账号
func (s *ConfigService) UpdateProviderAccount(account *model.ProviderAccount) error {
	return s.db.Save(account).Error
}

// DeleteProviderAccount 删除云厂商账号
func (s *ConfigService) DeleteProviderAccount(id uint) error {
	return s.db.Delete(&model.ProviderAccount{}, id).Error
}

// ========== IM 配置管理 ==========

// GetIMConfig 获取IM配置
func (s *ConfigService) GetIMConfig(platform string) (*model.IMConfig, error) {
	var config model.IMConfig
	err := s.db.Where("platform = ?", platform).First(&config).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &config, nil
}

// SaveIMConfig 保存IM配置
func (s *ConfigService) SaveIMConfig(config *model.IMConfig) error {
	// 检查是否已存在配置
	var existing model.IMConfig
	err := s.db.Where("platform = ?", config.Platform).First(&existing).Error
	if err == nil {
		// 存在则更新
		config.ID = existing.ID
		return s.db.Save(config).Error
	}
	if err == gorm.ErrRecordNotFound {
		// 不存在则创建
		return s.db.Create(config).Error
	}
	return err
}

// ListIMConfigs 列出所有IM配置
func (s *ConfigService) ListIMConfigs() ([]model.IMConfig, error) {
	var configs []model.IMConfig
	err := s.db.Find(&configs).Error
	return configs, err
}

// ========== CICD 配置管理 ==========

// GetCICDConfig 获取CICD配置
func (s *ConfigService) GetCICDConfig(platform string) (*model.CICDConfig, error) {
	var config model.CICDConfig
	err := s.db.Where("platform = ?", platform).First(&config).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &config, nil
}

// SaveCICDConfig 保存CICD配置
func (s *ConfigService) SaveCICDConfig(config *model.CICDConfig) error {
	// 检查是否已存在配置
	var existing model.CICDConfig
	err := s.db.Where("platform = ?", config.Platform).First(&existing).Error
	if err == nil {
		// 存在则更新
		config.ID = existing.ID
		return s.db.Save(config).Error
	}
	if err == gorm.ErrRecordNotFound {
		// 不存在则创建
		return s.db.Create(config).Error
	}
	return err
}

// ListCICDConfigs 列出所有CICD配置
func (s *ConfigService) ListCICDConfigs() ([]model.CICDConfig, error) {
	var configs []model.CICDConfig
	err := s.db.Find(&configs).Error
	return configs, err
}

// ========== MCP Server 配置管理 ==========

// ListMCPServers 列出MCP服务器
func (s *ConfigService) ListMCPServers() ([]model.MCPServer, error) {
	var servers []model.MCPServer
	err := s.db.Preload("Tools").Order("name").Find(&servers).Error
	return servers, err
}

// GetMCPServer 获取指定MCP服务器
func (s *ConfigService) GetMCPServer(id uint) (*model.MCPServer, error) {
	var server model.MCPServer
	err := s.db.First(&server, id).Error
	if err != nil {
		return nil, err
	}
	return &server, nil
}

// GetMCPServerByName 根据名称获取MCP服务器
func (s *ConfigService) GetMCPServerByName(name string) (*model.MCPServer, error) {
	var server model.MCPServer
	err := s.db.Preload("Tools").Where("name = ?", name).First(&server).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &server, nil
}

// CreateMCPServer 创建MCP服务器
func (s *ConfigService) CreateMCPServer(server *model.MCPServer) error {
	// 检查是否已存在同名服务器
	existing, err := s.GetMCPServerByName(server.Name)
	if err != nil {
		return err
	}
	if existing != nil {
		return fmt.Errorf("MCP server already exists: %s", server.Name)
	}
	return s.db.Create(server).Error
}

// UpdateMCPServer 更新MCP服务器
func (s *ConfigService) UpdateMCPServer(server *model.MCPServer) error {
	return s.db.Save(server).Error
}

// DeleteMCPServer 删除MCP服务器
func (s *ConfigService) DeleteMCPServer(id uint) error {
	return s.db.Delete(&model.MCPServer{}, id).Error
}

// ========== MCP Tool 配置管理 ==========

// CreateMCPTool 创建MCP工具
func (s *ConfigService) CreateMCPTool(tool *model.MCPTool) error {
	return s.db.Create(tool).Error
}

// UpsertMCPTool 创建或更新MCP工具（基于server_id + name唯一性）
func (s *ConfigService) UpsertMCPTool(tool *model.MCPTool) error {
	// 使用原生SQL实现真正的Upsert
	// SQLite使用 INSERT ... ON CONFLICT ... DO UPDATE
	inputSchemaJSON, err := json.Marshal(tool.InputSchema)
	if err != nil {
		return err
	}

	now := time.Now()
	if tool.CreatedAt.IsZero() {
		tool.CreatedAt = now
	}
	tool.UpdatedAt = now

	return s.db.Exec(`
		INSERT INTO mcp_tools (server_id, name, description, is_enabled, input_schema, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(server_id, name)
		DO UPDATE SET
			description = excluded.description,
			is_enabled = excluded.is_enabled,
			input_schema = excluded.input_schema,
			updated_at = excluded.updated_at
	`, tool.ServerID, tool.Name, tool.Description, tool.IsEnabled,
		string(inputSchemaJSON), tool.CreatedAt, tool.UpdatedAt).Error
}

// DeleteMCPToolsByServerID 删除指定服务器的所有工具
func (s *ConfigService) DeleteMCPToolsByServerID(serverID uint) error {
	return s.db.Where("server_id = ?", serverID).Delete(&model.MCPTool{}).Error
}

// GetMCPToolsByServerID 获取指定服务器的所有工具
func (s *ConfigService) GetMCPToolsByServerID(serverID uint) ([]model.MCPTool, error) {
	var tools []model.MCPTool
	err := s.db.Where("server_id = ?", serverID).Find(&tools).Error
	return tools, err
}

// UpdateMCPTool 更新MCP工具
func (s *ConfigService) UpdateMCPTool(tool *model.MCPTool) error {
	return s.db.Save(tool).Error
}

// ========== 系统配置管理 ==========

// GetSystemConfig 获取系统配置
func (s *ConfigService) GetSystemConfig(key string) (*model.SystemConfig, error) {
	var config model.SystemConfig
	err := s.db.Where("config_key = ?", key).First(&config).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &config, nil
}

// SetSystemConfig 设置系统配置
func (s *ConfigService) SetSystemConfig(key, value, description string) error {
	config, err := s.GetSystemConfig(key)
	if err != nil {
		return err
	}

	if config == nil {
		// 创建新配置
		config = &model.SystemConfig{
			ConfigKey:   key,
			ConfigValue: value,
			Description: description,
		}
		return s.db.Create(config).Error
	}

	// 更新配置
	config.ConfigValue = value
	if description != "" {
		config.Description = description
	}
	return s.db.Save(config).Error
}

// ListSystemConfigs 列出所有系统配置
func (s *ConfigService) ListSystemConfigs() ([]model.SystemConfig, error) {
	var configs []model.SystemConfig
	err := s.db.Order("config_key").Find(&configs).Error
	return configs, err
}

// GetIMConfigByID 根据ID获取IM配置
func (s *ConfigService) GetIMConfigByID(id uint) (*model.IMConfig, error) {
	var config model.IMConfig
	err := s.db.First(&config, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &config, nil
}

// DeleteIMConfig 删除IM配置
func (s *ConfigService) DeleteIMConfig(id uint) error {
	return s.db.Delete(&model.IMConfig{}, id).Error
}
