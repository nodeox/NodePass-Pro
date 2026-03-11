package config

import (
	"fmt"
	"sync"

	domainconfig "nodepass-pro/nodeclient/internal/domain/config"
	"nodepass-pro/nodeclient/internal/infra/config"
	"nodepass-pro/nodeclient/internal/infra/logger"
)

// Manager 负责配置版本管理和持久化。
type Manager interface {
	GetCurrent() *domainconfig.NodeConfig
	GetVersion() int
	SetVersion(version int)
	HandleUpdate(cfg *domainconfig.NodeConfig, version int, applyFunc func(*domainconfig.NodeConfig) error) error
	SaveFinalState() error
}

type manager struct {
	configCache   *config.ConfigCache
	logger        *logger.Logger
	mu            sync.RWMutex
	configVersion int
	currentConfig *domainconfig.NodeConfig
}

// NewManager 创建配置管理器。
func NewManager(configCache *config.ConfigCache, logger *logger.Logger) Manager {
	return &manager{
		configCache:   configCache,
		logger:        logger,
		configVersion: -1,
	}
}

// GetCurrent 返回当前配置的深拷贝。
func (m *manager) GetCurrent() *domainconfig.NodeConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return cloneNodeConfig(m.currentConfig)
}

// GetVersion 返回当前配置版本。
func (m *manager) GetVersion() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.configVersion
}

// SetVersion 设置配置版本。
func (m *manager) SetVersion(version int) {
	m.mu.Lock()
	m.configVersion = version
	m.mu.Unlock()
}

// HandleUpdate 处理配置更新：应用配置、更新版本、保存缓存。
func (m *manager) HandleUpdate(cfg *domainconfig.NodeConfig, version int, applyFunc func(*domainconfig.NodeConfig) error) error {
	if cfg == nil {
		return fmt.Errorf("配置不能为空")
	}

	next := cloneNodeConfig(cfg)
	if next == nil {
		return fmt.Errorf("配置克隆失败")
	}
	if version >= 0 {
		next.ConfigVersion = version
	}

	if err := applyFunc(next); err != nil {
		return fmt.Errorf("应用配置失败: %w", err)
	}

	if err := m.configCache.Save(next); err != nil {
		m.logger.Warn("保存配置缓存失败", "error", err)
	}

	m.mu.Lock()
	m.configVersion = next.ConfigVersion
	m.currentConfig = cloneNodeConfig(next)
	m.mu.Unlock()

	return nil
}

// SaveFinalState 保存最终配置状态到缓存。
func (m *manager) SaveFinalState() error {
	cfg := m.GetCurrent()
	version := m.GetVersion()
	if cfg == nil {
		m.logger.Warn("无可保存的配置状态，跳过缓存落盘")
		return nil
	}
	if cfg.ConfigVersion == 0 && version > 0 {
		cfg.ConfigVersion = version
	}

	if err := m.configCache.Save(cfg); err != nil {
		m.logger.Warn("保存最终配置状态失败", "error", err)
		return err
	}
	m.logger.Info("已保存最终配置状态到缓存", "version", cfg.ConfigVersion)
	return nil
}
