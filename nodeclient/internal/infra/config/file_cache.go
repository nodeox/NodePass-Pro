package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	domainconfig "nodepass-pro/nodeclient/internal/domain/config"
)

// ErrCacheNotFound 表示缓存文件不存在。
var ErrCacheNotFound = errors.New("cache file not found")

// ConfigCache 定义本地配置缓存管理器。
type ConfigCache struct {
	cachePath string
	mu        sync.RWMutex
}

// NewConfigCache 创建缓存管理器并确保缓存目录存在。
func NewConfigCache(path string) *ConfigCache {
	cache := &ConfigCache{
		cachePath: path,
	}

	if path != "" {
		parentDir := filepath.Dir(path)
		_ = os.MkdirAll(parentDir, 0o755)
	}

	return cache
}

// Save 原子写入本地配置缓存。
func (c *ConfigCache) Save(cfg *domainconfig.NodeConfig) error {
	if c == nil {
		return fmt.Errorf("配置缓存实例不能为空")
	}
	if cfg == nil {
		return fmt.Errorf("缓存配置不能为空")
	}
	if c.cachePath == "" {
		return fmt.Errorf("缓存路径不能为空")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	parentDir := filepath.Dir(c.cachePath)
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		return fmt.Errorf("创建缓存目录失败: %w", err)
	}

	content, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("序列化缓存配置失败: %w", err)
	}

	tempFile, err := os.CreateTemp(parentDir, ".node-config-*.tmp")
	if err != nil {
		return fmt.Errorf("创建临时缓存文件失败: %w", err)
	}
	tempPath := tempFile.Name()

	cleanupTemp := func() {
		_ = tempFile.Close()
		_ = os.Remove(tempPath)
	}

	if _, err := tempFile.Write(content); err != nil {
		cleanupTemp()
		return fmt.Errorf("写入临时缓存文件失败: %w", err)
	}

	if err := tempFile.Sync(); err != nil {
		cleanupTemp()
		return fmt.Errorf("刷新临时缓存文件失败: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("关闭临时缓存文件失败: %w", err)
	}

	if err := os.Rename(tempPath, c.cachePath); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("原子替换缓存文件失败: %w", err)
	}

	return nil
}

// Load 加载本地缓存并反序列化为 NodeConfig。
func (c *ConfigCache) Load() (*domainconfig.NodeConfig, error) {
	if c == nil {
		return nil, fmt.Errorf("配置缓存实例不能为空")
	}
	if c.cachePath == "" {
		return nil, fmt.Errorf("缓存路径不能为空")
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	content, err := os.ReadFile(c.cachePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrCacheNotFound
		}
		return nil, fmt.Errorf("读取缓存文件失败: %w", err)
	}
	if len(content) == 0 {
		return nil, fmt.Errorf("缓存文件为空")
	}

	cfg := &domainconfig.NodeConfig{}
	if err := json.Unmarshal(content, cfg); err != nil {
		return nil, fmt.Errorf("解析缓存文件失败: %w", err)
	}
	return cfg, nil
}

// Exists 判断缓存文件是否存在且非空。
func (c *ConfigCache) Exists() bool {
	if c == nil || c.cachePath == "" {
		return false
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	info, err := os.Stat(c.cachePath)
	if err != nil {
		return false
	}
	if info.IsDir() {
		return false
	}
	return info.Size() > 0
}

// GetVersion 快速读取缓存中的配置版本号。
func (c *ConfigCache) GetVersion() int {
	if c == nil || c.cachePath == "" {
		return 0
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	content, err := os.ReadFile(c.cachePath)
	if err != nil || len(content) == 0 {
		return 0
	}

	var versionOnly struct {
		ConfigVersion int `json:"config_version"`
	}
	if err := json.Unmarshal(content, &versionOnly); err != nil {
		return 0
	}
	return versionOnly.ConfigVersion
}
