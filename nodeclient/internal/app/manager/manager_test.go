package manager

import (
	"testing"
	"time"

	domainconfig "nodepass-pro/nodeclient/internal/domain/config"
	"nodepass-pro/nodeclient/internal/infra/config"
)

func TestNewManager(t *testing.T) {
	cfg := &config.Config{
		HubURL:              "http://localhost:8080",
		NodeID:              "test-node",
		NodeToken:           "test-token",
		CachePath:           t.TempDir() + "/cache.json",
		HeartbeatInterval:   30,
		ConfigCheckInterval: 60,
	}

	mgr := NewManager(cfg, "1.0.0-test")
	if mgr == nil {
		t.Fatal("NewManager returned nil")
	}
}

func TestNewManagerNilConfig(t *testing.T) {
	mgr := NewManager(nil, "1.0.0-test")
	if mgr == nil {
		t.Fatal("NewManager should handle nil config")
	}
}

func TestManagerIsOnlineInitial(t *testing.T) {
	cfg := &config.Config{
		HubURL:              "http://localhost:8080",
		NodeID:              "test-node",
		NodeToken:           "test-token",
		CachePath:           t.TempDir() + "/cache.json",
		HeartbeatInterval:   30,
		ConfigCheckInterval: 60,
	}

	mgr := NewManager(cfg, "1.0.0-test")

	// Should be offline initially
	if mgr.IsOnline() {
		t.Error("Expected manager to be offline initially")
	}
}

func TestManagerShutdownIdempotent(t *testing.T) {
	cfg := &config.Config{
		HubURL:              "http://localhost:8080",
		NodeID:              "test-node",
		NodeToken:           "test-token",
		CachePath:           t.TempDir() + "/cache.json",
		HeartbeatInterval:   30,
		ConfigCheckInterval: 60,
	}

	mgr := NewManager(cfg, "1.0.0-test")

	// Multiple shutdowns should not panic
	mgr.Shutdown()
	mgr.Shutdown()
	mgr.Shutdown()
}

func TestManagerSnapshotHeartbeatMetrics(t *testing.T) {
	cfg := &config.Config{
		HubURL:              "http://localhost:8080",
		NodeID:              "test-node",
		NodeToken:           "test-token",
		CachePath:           t.TempDir() + "/cache.json",
		HeartbeatInterval:   30,
		ConfigCheckInterval: 60,
	}

	mgr := NewManager(cfg, "1.0.0-test")

	// SnapshotHeartbeatMetrics is internal to manager and used by heartbeat
	// We can't test it directly through the Manager interface
	// This test just verifies the manager was created successfully
	if mgr == nil {
		t.Fatal("NewManager returned nil")
	}
}

func TestManagerStartWithInvalidHub(t *testing.T) {
	cfg := &config.Config{
		HubURL:              "http://invalid-url-that-will-fail:99999",
		NodeID:              "test-node",
		NodeToken:           "test-token",
		CachePath:           t.TempDir() + "/cache.json",
		HeartbeatInterval:   30,
		ConfigCheckInterval: 60,
	}

	mgr := NewManager(cfg, "1.0.0-test")

	// Start in goroutine with timeout
	done := make(chan error, 1)
	go func() {
		done <- mgr.Start()
	}()

	// Give it a moment to try connecting
	time.Sleep(100 * time.Millisecond)

	// Shutdown to stop the Start() call
	mgr.Shutdown()

	// Wait for Start to return
	select {
	case err := <-done:
		// Should error because no cache and can't connect
		if err == nil {
			t.Error("Expected error when starting with invalid hub and no cache")
		}
	case <-time.After(2 * time.Second):
		t.Error("Start() did not return after Shutdown()")
	}
}

func TestManagerStartStopCycle(t *testing.T) {
	cfg := &config.Config{
		HubURL:              "http://localhost:8080",
		NodeID:              "test-node",
		NodeToken:           "test-token",
		CachePath:           t.TempDir() + "/cache.json",
		HeartbeatInterval:   30,
		ConfigCheckInterval: 60,
	}

	// Create a bootstrap cache so Start doesn't fail
	cache := config.NewConfigCache(cfg.CachePath)
	bootstrapConfig := &domainconfig.NodeConfig{
		ConfigVersion: 0,
		Rules:         []domainconfig.RuleConfig{},
	}
	if err := cache.Save(bootstrapConfig); err != nil {
		t.Fatalf("Failed to save bootstrap cache: %v", err)
	}

	mgr := NewManager(cfg, "1.0.0-test")

	// Start in goroutine
	done := make(chan error, 1)
	go func() {
		done <- mgr.Start()
	}()

	// Give it time to start
	time.Sleep(200 * time.Millisecond)

	// Should be running (though offline due to invalid hub)
	// Shutdown
	mgr.Shutdown()

	// Wait for Start to return
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Start returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("Start() did not return after Shutdown()")
	}
}
