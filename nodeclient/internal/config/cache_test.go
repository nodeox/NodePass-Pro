package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigCacheNewConfigCache(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "test_cache.json")

	cache := NewConfigCache(cachePath)
	if cache == nil {
		t.Fatal("NewConfigCache() returned nil")
	}
	if cache.cachePath != cachePath {
		t.Errorf("Expected cachePath %s, got %s", cachePath, cache.cachePath)
	}
}

func TestConfigCacheSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "test_cache.json")
	cache := NewConfigCache(cachePath)

	// 创建测试配置
	testConfig := &NodeConfig{
		ConfigVersion: 42,
		Rules: []RuleConfig{
			{
				RuleID:   1,
				Mode:     "single",
				Protocol: "tcp",
				Listen:   HostPort{Host: "0.0.0.0", Port: 8080},
				Target:   HostPort{Host: "127.0.0.1", Port: 9090},
			},
		},
		Settings: Settings{
			HeartbeatInterval:   30,
			ConfigCheckInterval: 60,
		},
	}

	// 保存配置
	err := cache.Save(testConfig)
	if err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// 验证文件存在
	if !cache.Exists() {
		t.Error("Cache file should exist after Save()")
	}

	// 加载配置
	loaded, err := cache.Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// 验证配置内容
	if loaded.ConfigVersion != testConfig.ConfigVersion {
		t.Errorf("Expected ConfigVersion %d, got %d", testConfig.ConfigVersion, loaded.ConfigVersion)
	}
	if len(loaded.Rules) != len(testConfig.Rules) {
		t.Errorf("Expected %d rules, got %d", len(testConfig.Rules), len(loaded.Rules))
	}
	if loaded.Rules[0].RuleID != testConfig.Rules[0].RuleID {
		t.Errorf("Expected RuleID %d, got %d", testConfig.Rules[0].RuleID, loaded.Rules[0].RuleID)
	}
}

func TestConfigCacheGetVersion(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "test_cache.json")
	cache := NewConfigCache(cachePath)

	testConfig := &NodeConfig{
		ConfigVersion: 123,
		Rules:         []RuleConfig{},
		Settings: Settings{
			HeartbeatInterval:   30,
			ConfigCheckInterval: 60,
		},
	}

	err := cache.Save(testConfig)
	if err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	version := cache.GetVersion()
	if version != 123 {
		t.Errorf("Expected version 123, got %d", version)
	}
}

func TestConfigCacheLoadNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "nonexistent.json")
	cache := NewConfigCache(cachePath)

	_, err := cache.Load()
	if err == nil {
		t.Error("Expected error when loading non-existent cache")
	}
	if err != ErrCacheNotFound {
		t.Errorf("Expected ErrCacheNotFound, got %v", err)
	}
}

func TestConfigCacheExistsEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "empty.json")
	cache := NewConfigCache(cachePath)

	// 创建空文件
	err := os.WriteFile(cachePath, []byte{}, 0644)
	if err != nil {
		t.Fatalf("Failed to create empty file: %v", err)
	}

	// 空文件应该返回 false
	if cache.Exists() {
		t.Error("Exists() should return false for empty file")
	}
}

func TestConfigCacheSaveNil(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "test_cache.json")
	cache := NewConfigCache(cachePath)

	err := cache.Save(nil)
	if err == nil {
		t.Error("Expected error when saving nil config")
	}
}

func TestConfigCacheAtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "test_cache.json")
	cache := NewConfigCache(cachePath)

	// 保存初始配置
	config1 := &NodeConfig{
		ConfigVersion: 1,
		Rules:         []RuleConfig{},
		Settings: Settings{
			HeartbeatInterval:   30,
			ConfigCheckInterval: 60,
		},
	}
	err := cache.Save(config1)
	if err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// 保存新配置（应该原子替换）
	config2 := &NodeConfig{
		ConfigVersion: 2,
		Rules:         []RuleConfig{},
		Settings: Settings{
			HeartbeatInterval:   30,
			ConfigCheckInterval: 60,
		},
	}
	err = cache.Save(config2)
	if err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// 验证只有新配置
	loaded, err := cache.Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	if loaded.ConfigVersion != 2 {
		t.Errorf("Expected ConfigVersion 2, got %d", loaded.ConfigVersion)
	}

	// 验证没有临时文件残留
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("ReadDir() failed: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("Expected 1 file, got %d", len(entries))
	}
}

func TestConfigCacheNilCache(t *testing.T) {
	var cache *ConfigCache

	// 所有方法应该安全处理 nil
	err := cache.Save(&NodeConfig{})
	if err == nil {
		t.Error("Expected error for nil cache Save()")
	}

	_, err = cache.Load()
	if err == nil {
		t.Error("Expected error for nil cache Load()")
	}

	if cache.Exists() {
		t.Error("Exists() should return false for nil cache")
	}

	version := cache.GetVersion()
	if version != 0 {
		t.Errorf("Expected version 0 for nil cache, got %d", version)
	}
}
