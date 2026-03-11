package config

import (
	"os"
	"path/filepath"
	"testing"

	domainconfig "nodepass-pro/nodeclient/internal/domain/config"
	"nodepass-pro/nodeclient/internal/infra/config"
	"nodepass-pro/nodeclient/internal/infra/logger"
)

func TestNewManager(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	cache := config.NewConfigCache(cachePath)
	log, _ := logger.New(logger.Config{Level: "info", Prefix: "test"})

	mgr := NewManager(cache, log)
	if mgr == nil {
		t.Fatal("NewManager returned nil")
	}

	if mgr.GetVersion() != -1 {
		t.Errorf("Expected initial version -1, got %d", mgr.GetVersion())
	}

	if mgr.GetCurrent() != nil {
		t.Error("Expected initial config to be nil")
	}
}

func TestManagerSetGetVersion(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	cache := config.NewConfigCache(cachePath)
	log, _ := logger.New(logger.Config{Level: "info", Prefix: "test"})

	mgr := NewManager(cache, log)

	mgr.SetVersion(42)
	if got := mgr.GetVersion(); got != 42 {
		t.Errorf("Expected version 42, got %d", got)
	}

	mgr.SetVersion(100)
	if got := mgr.GetVersion(); got != 100 {
		t.Errorf("Expected version 100, got %d", got)
	}
}

func TestManagerHandleUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	cache := config.NewConfigCache(cachePath)
	log, _ := logger.New(logger.Config{Level: "info", Prefix: "test"})

	mgr := NewManager(cache, log)

	testConfig := &domainconfig.NodeConfig{
		ConfigVersion: 5,
		Rules: []domainconfig.RuleConfig{
			{RuleID: 1, Mode: "single"},
		},
	}

	applyCalled := false
	applyFunc := func(cfg *domainconfig.NodeConfig) error {
		applyCalled = true
		if cfg.ConfigVersion != 10 {
			t.Errorf("Expected config version 10, got %d", cfg.ConfigVersion)
		}
		return nil
	}

	err := mgr.HandleUpdate(testConfig, 10, applyFunc)
	if err != nil {
		t.Fatalf("HandleUpdate failed: %v", err)
	}

	if !applyCalled {
		t.Error("applyFunc was not called")
	}

	if mgr.GetVersion() != 10 {
		t.Errorf("Expected version 10, got %d", mgr.GetVersion())
	}

	current := mgr.GetCurrent()
	if current == nil {
		t.Fatal("GetCurrent returned nil")
	}
	if current.ConfigVersion != 10 {
		t.Errorf("Expected current config version 10, got %d", current.ConfigVersion)
	}
}

func TestManagerHandleUpdateNilConfig(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	cache := config.NewConfigCache(cachePath)
	log, _ := logger.New(logger.Config{Level: "info", Prefix: "test"})

	mgr := NewManager(cache, log)

	applyFunc := func(cfg *domainconfig.NodeConfig) error {
		return nil
	}

	err := mgr.HandleUpdate(nil, 1, applyFunc)
	if err == nil {
		t.Error("Expected error for nil config, got nil")
	}
}

func TestManagerHandleUpdateApplyError(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	cache := config.NewConfigCache(cachePath)
	log, _ := logger.New(logger.Config{Level: "info", Prefix: "test"})

	mgr := NewManager(cache, log)

	testConfig := &domainconfig.NodeConfig{
		ConfigVersion: 5,
		Rules:         []domainconfig.RuleConfig{},
	}

	applyFunc := func(cfg *domainconfig.NodeConfig) error {
		return os.ErrInvalid
	}

	err := mgr.HandleUpdate(testConfig, 10, applyFunc)
	if err == nil {
		t.Error("Expected error from applyFunc, got nil")
	}

	// Version should not be updated on error
	if mgr.GetVersion() != -1 {
		t.Errorf("Expected version to remain -1, got %d", mgr.GetVersion())
	}
}

func TestManagerSaveFinalState(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	cache := config.NewConfigCache(cachePath)
	log, _ := logger.New(logger.Config{Level: "info", Prefix: "test"})

	mgr := NewManager(cache, log)

	testConfig := &domainconfig.NodeConfig{
		ConfigVersion: 5,
		Rules: []domainconfig.RuleConfig{
			{RuleID: 1, Mode: "single"},
		},
	}

	applyFunc := func(cfg *domainconfig.NodeConfig) error {
		return nil
	}

	err := mgr.HandleUpdate(testConfig, 5, applyFunc)
	if err != nil {
		t.Fatalf("HandleUpdate failed: %v", err)
	}

	err = mgr.SaveFinalState()
	if err != nil {
		t.Fatalf("SaveFinalState failed: %v", err)
	}

	// Verify cache was saved
	if !cache.Exists() {
		t.Error("Cache file was not created")
	}

	loaded, err := cache.Load()
	if err != nil {
		t.Fatalf("Failed to load cache: %v", err)
	}

	if loaded.ConfigVersion != 5 {
		t.Errorf("Expected cached version 5, got %d", loaded.ConfigVersion)
	}
}

func TestManagerSaveFinalStateNilConfig(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	cache := config.NewConfigCache(cachePath)
	log, _ := logger.New(logger.Config{Level: "info", Prefix: "test"})

	mgr := NewManager(cache, log)

	// Should not error, just log warning
	err := mgr.SaveFinalState()
	if err != nil {
		t.Errorf("SaveFinalState with nil config should not error, got: %v", err)
	}
}

func TestManagerGetCurrentReturnsDeepCopy(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	cache := config.NewConfigCache(cachePath)
	log, _ := logger.New(logger.Config{Level: "info", Prefix: "test"})

	mgr := NewManager(cache, log)

	testConfig := &domainconfig.NodeConfig{
		ConfigVersion: 5,
		Rules: []domainconfig.RuleConfig{
			{RuleID: 1, Mode: "single"},
		},
	}

	applyFunc := func(cfg *domainconfig.NodeConfig) error {
		return nil
	}

	err := mgr.HandleUpdate(testConfig, 5, applyFunc)
	if err != nil {
		t.Fatalf("HandleUpdate failed: %v", err)
	}

	current1 := mgr.GetCurrent()
	current2 := mgr.GetCurrent()

	// Should be different pointers (deep copy)
	if current1 == current2 {
		t.Error("GetCurrent should return different pointers (deep copy)")
	}

	// But same content
	if current1.ConfigVersion != current2.ConfigVersion {
		t.Error("GetCurrent copies should have same content")
	}
}
