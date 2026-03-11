package agent_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"nodepass-pro/nodeclient/internal/agent"
	"nodepass-pro/nodeclient/internal/infra/config"
)

func TestVersion(t *testing.T) {
	version := agent.Version()
	if version == "" {
		t.Fatal("Version() should not return empty string")
	}
	if version != "1.0.1" {
		t.Errorf("Expected version 1.0.1, got %s", version)
	}
}

func TestNewAgent(t *testing.T) {
	cfg := &config.Config{
		HubURL:                "http://localhost:8080",
		NodeID:                "test-node",
		GroupID:               1,
		ServiceName:           "test-service",
		NodeToken:             "test-token",
		HeartbeatInterval:     30,
		ConfigCheckInterval:   60,
		TrafficReportInterval: 60,
		CachePath:             "/tmp/test_cache.json",
	}

	agent := agent.NewAgent(cfg)
	if agent == nil {
		t.Fatal("NewAgent() should not return nil")
	}
}

func TestNewAgentNilConfig(t *testing.T) {
	agent := agent.NewAgent(nil)
	if agent == nil {
		t.Fatal("NewAgent() should handle nil config gracefully")
	}
}

func TestAgentStartFailsWhenLicenseVerificationFails(t *testing.T) {
	verifyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":false,"data":{"verified":false,"status":"invalid"},"message":"license invalid"}`))
	}))
	defer verifyServer.Close()

	cfg := &config.Config{
		HubURL:                "http://localhost:65534",
		NodeID:                "test-node",
		GroupID:               1,
		ServiceName:           "test-service",
		NodeToken:             "test-token",
		HeartbeatInterval:     30,
		ConfigCheckInterval:   60,
		TrafficReportInterval: 60,
		CachePath:             t.TempDir() + "/cache.json",
		LicenseEnabled:        true,
		LicenseVerifyURL:      verifyServer.URL,
		LicenseKey:            "invalid-key",
		LicenseTimeout:        5,
	}

	clientAgent := agent.NewAgent(cfg)
	err := clientAgent.Start()
	if err == nil {
		t.Fatal("expected Start() to fail when license verify fails")
	}
	if !strings.Contains(err.Error(), "授权校验失败") {
		t.Fatalf("expected license verify error, got: %v", err)
	}
}
