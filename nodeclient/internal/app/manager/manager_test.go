package manager

import (
	"net/http"
	"net/http/httptest"
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

func TestManagerStartOfflineWithCacheSyncsCurrentConfig(t *testing.T) {
	cfg := &config.Config{
		HubURL:              "http://invalid-url-that-will-fail:99999",
		NodeID:              "test-node",
		NodeToken:           "test-token",
		CachePath:           t.TempDir() + "/cache.json",
		HeartbeatInterval:   30,
		ConfigCheckInterval: 60,
	}

	cache := config.NewConfigCache(cfg.CachePath)
	cached := &domainconfig.NodeConfig{
		ConfigVersion: 7,
		Rules:         []domainconfig.RuleConfig{},
	}
	if err := cache.Save(cached); err != nil {
		t.Fatalf("Failed to save cached config: %v", err)
	}

	mgrIface := NewManager(cfg, "1.0.0-test")
	mgrImpl, ok := mgrIface.(*manager)
	if !ok {
		t.Fatalf("unexpected manager type: %T", mgrIface)
	}

	done := make(chan error, 1)
	go func() {
		done <- mgrIface.Start()
	}()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		current := mgrImpl.configManager.GetCurrent()
		if current != nil {
			if current.ConfigVersion != 7 {
				t.Fatalf("expected current config version 7, got %d", current.ConfigVersion)
			}
			if got := mgrImpl.configManager.GetVersion(); got != 7 {
				t.Fatalf("expected manager version 7, got %d", got)
			}
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	if mgrImpl.configManager.GetCurrent() == nil {
		t.Fatal("expected current config to be set after applying cached config")
	}

	mgrIface.Shutdown()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Start() returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Start() did not return after Shutdown()")
	}
}

func TestManagerStartOnlineBootstrapSyncsCurrentConfig(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/node-instances/heartbeat" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":{"config_updated":false,"new_config_version":0},"message":"ok","timestamp":"2026-03-10T00:00:00Z"}`))
	}))
	defer server.Close()

	cfg := &config.Config{
		HubURL:              server.URL,
		NodeID:              "test-node",
		NodeToken:           "test-token",
		CachePath:           t.TempDir() + "/cache.json",
		HeartbeatInterval:   30,
		ConfigCheckInterval: 60,
	}

	mgrIface := NewManager(cfg, "1.0.0-test")
	mgrImpl, ok := mgrIface.(*manager)
	if !ok {
		t.Fatalf("unexpected manager type: %T", mgrIface)
	}

	done := make(chan error, 1)
	go func() {
		done <- mgrIface.Start()
	}()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		current := mgrImpl.configManager.GetCurrent()
		if current != nil {
			if current.ConfigVersion != 0 {
				t.Fatalf("expected bootstrap config version 0, got %d", current.ConfigVersion)
			}
			if got := mgrImpl.configManager.GetVersion(); got != 0 {
				t.Fatalf("expected manager version 0, got %d", got)
			}
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	if mgrImpl.configManager.GetCurrent() == nil {
		t.Fatal("expected bootstrap config to be tracked in config manager")
	}

	mgrIface.Shutdown()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Start() returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Start() did not return after Shutdown()")
	}
}
