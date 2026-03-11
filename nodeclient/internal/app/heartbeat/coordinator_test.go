package heartbeat

import (
	"testing"

	domainconfig "nodepass-pro/nodeclient/internal/domain/config"
	"nodepass-pro/nodeclient/internal/infra/config"
	"nodepass-pro/nodeclient/internal/infra/heartbeat"
	"nodepass-pro/nodeclient/internal/infra/logger"
)

func TestNewCoordinator(t *testing.T) {
	cfg := &config.Config{
		HubURL:    "http://localhost:8080",
		NodeID:    "test-node",
		NodeToken: "test-token",
	}
	hbService := heartbeat.NewHeartbeatService(cfg)
	log, _ := logger.New(logger.Config{Level: "info", Prefix: "test"})

	coordinator := NewCoordinator(hbService, log)
	if coordinator == nil {
		t.Fatal("NewCoordinator returned nil")
	}
}

func TestCoordinatorIsOnline(t *testing.T) {
	cfg := &config.Config{
		HubURL:    "http://localhost:8080",
		NodeID:    "test-node",
		NodeToken: "test-token",
	}
	hbService := heartbeat.NewHeartbeatService(cfg)
	log, _ := logger.New(logger.Config{Level: "info", Prefix: "test"})

	coordinator := NewCoordinator(hbService, log)

	// Initially should be offline
	if coordinator.IsOnline() {
		t.Error("Expected coordinator to be offline initially")
	}
}

func TestCoordinatorSetConfigUpdateHandler(t *testing.T) {
	cfg := &config.Config{
		HubURL:    "http://localhost:8080",
		NodeID:    "test-node",
		NodeToken: "test-token",
	}
	hbService := heartbeat.NewHeartbeatService(cfg)
	log, _ := logger.New(logger.Config{Level: "info", Prefix: "test"})

	coordinator := NewCoordinator(hbService, log)

	handlerCalled := false
	handler := func(cfg *domainconfig.NodeConfig, version int) {
		handlerCalled = true
	}

	// Should not panic
	coordinator.SetConfigUpdateHandler(handler)

	// We can't easily test if the handler is called without a real heartbeat,
	// but we can verify it doesn't panic
	if handlerCalled {
		t.Error("Handler should not be called during SetConfigUpdateHandler")
	}
}

func TestCoordinatorSetCurrentConfigVersion(t *testing.T) {
	cfg := &config.Config{
		HubURL:    "http://localhost:8080",
		NodeID:    "test-node",
		NodeToken: "test-token",
	}
	hbService := heartbeat.NewHeartbeatService(cfg)
	log, _ := logger.New(logger.Config{Level: "info", Prefix: "test"})

	coordinator := NewCoordinator(hbService, log)

	// Should not panic
	coordinator.SetCurrentConfigVersion(42)
}

func TestCoordinatorSetMetricsProvider(t *testing.T) {
	cfg := &config.Config{
		HubURL:    "http://localhost:8080",
		NodeID:    "test-node",
		NodeToken: "test-token",
	}
	hbService := heartbeat.NewHeartbeatService(cfg)
	log, _ := logger.New(logger.Config{Level: "info", Prefix: "test"})

	coordinator := NewCoordinator(hbService, log)

	// Mock metrics provider
	provider := &mockMetricsProvider{}

	// Should not panic
	coordinator.SetMetricsProvider(provider)
}

func TestCoordinatorStartStop(t *testing.T) {
	cfg := &config.Config{
		HubURL:    "http://localhost:8080",
		NodeID:    "test-node",
		NodeToken: "test-token",
	}
	hbService := heartbeat.NewHeartbeatService(cfg)
	log, _ := logger.New(logger.Config{Level: "info", Prefix: "test"})

	coordinator := NewCoordinator(hbService, log)

	// Start in goroutine since it blocks
	go coordinator.Start()

	// Stop should not panic
	coordinator.Stop()
}

func TestCoordinatorReport(t *testing.T) {
	cfg := &config.Config{
		HubURL:    "http://invalid-url-that-will-fail:99999",
		NodeID:    "test-node",
		NodeToken: "test-token",
	}
	hbService := heartbeat.NewHeartbeatService(cfg)
	log, _ := logger.New(logger.Config{Level: "info", Prefix: "test"})

	coordinator := NewCoordinator(hbService, log)

	// Should return error for invalid URL
	err := coordinator.Report()
	if err == nil {
		t.Error("Expected error for invalid hub URL, got nil")
	}
}

// Mock metrics provider for testing
type mockMetricsProvider struct{}

func (m *mockMetricsProvider) SnapshotHeartbeatMetrics() (trafficIn int64, trafficOut int64, activeConnections int64) {
	return 100, 200, 5
}
