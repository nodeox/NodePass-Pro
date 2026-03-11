package rules

import (
	"testing"

	domainconfig "nodepass-pro/nodeclient/internal/domain/config"
	"nodepass-pro/nodeclient/internal/infra/logger"
	"nodepass-pro/nodeclient/internal/infra/nodepass"
)

func TestNewManager(t *testing.T) {
	log, _ := logger.New(logger.Config{Level: "info", Prefix: "test"})
	integration := nodepass.NewIntegration(log.StdLogger())

	mgr := NewManager(integration, log)
	if mgr == nil {
		t.Fatal("NewManager returned nil")
	}
}

func TestManagerApplyConfigNil(t *testing.T) {
	log, _ := logger.New(logger.Config{Level: "info", Prefix: "test"})
	integration := nodepass.NewIntegration(log.StdLogger())

	mgr := NewManager(integration, log)

	err := mgr.ApplyConfig(nil)
	if err == nil {
		t.Error("Expected error for nil config, got nil")
	}
}

func TestManagerApplyConfigEmpty(t *testing.T) {
	log, _ := logger.New(logger.Config{Level: "info", Prefix: "test"})
	integration := nodepass.NewIntegration(log.StdLogger())

	mgr := NewManager(integration, log)

	config := &domainconfig.NodeConfig{
		ConfigVersion: 1,
		Rules:         []domainconfig.RuleConfig{},
	}

	err := mgr.ApplyConfig(config)
	if err != nil {
		t.Errorf("ApplyConfig with empty rules failed: %v", err)
	}
}

func TestManagerGetStatus(t *testing.T) {
	log, _ := logger.New(logger.Config{Level: "info", Prefix: "test"})
	integration := nodepass.NewIntegration(log.StdLogger())

	mgr := NewManager(integration, log)

	status := mgr.GetStatus()
	if status == nil {
		t.Error("GetStatus returned nil")
	}
	if len(status) != 0 {
		t.Errorf("Expected empty status, got %d items", len(status))
	}
}

func TestManagerGetTrafficStats(t *testing.T) {
	log, _ := logger.New(logger.Config{Level: "info", Prefix: "test"})
	integration := nodepass.NewIntegration(log.StdLogger())

	mgr := NewManager(integration, log)

	stats := mgr.GetTrafficStats()
	if stats == nil {
		t.Error("GetTrafficStats returned nil")
	}
	if len(stats) != 0 {
		t.Errorf("Expected empty stats, got %d items", len(stats))
	}
}

func TestManagerSnapshotRules(t *testing.T) {
	log, _ := logger.New(logger.Config{Level: "info", Prefix: "test"})
	integration := nodepass.NewIntegration(log.StdLogger())

	mgr := NewManager(integration, log)

	snapshot := mgr.SnapshotRules()
	if snapshot == nil {
		t.Error("SnapshotRules returned nil")
	}
	if len(snapshot) != 0 {
		t.Errorf("Expected empty snapshot, got %d items", len(snapshot))
	}
}

func TestManagerStopAll(t *testing.T) {
	log, _ := logger.New(logger.Config{Level: "info", Prefix: "test"})
	integration := nodepass.NewIntegration(log.StdLogger())

	mgr := NewManager(integration, log)

	// Should not panic
	mgr.StopAll()
}

func TestManagerApplyConfigSerializes(t *testing.T) {
	log, _ := logger.New(logger.Config{Level: "info", Prefix: "test"})
	integration := nodepass.NewIntegration(log.StdLogger())

	mgr := NewManager(integration, log)

	config := &domainconfig.NodeConfig{
		ConfigVersion: 1,
		Rules:         []domainconfig.RuleConfig{},
	}

	// Multiple concurrent calls should be serialized
	done := make(chan bool, 3)
	for i := 0; i < 3; i++ {
		go func() {
			err := mgr.ApplyConfig(config)
			if err != nil {
				t.Errorf("ApplyConfig failed: %v", err)
			}
			done <- true
		}()
	}

	for i := 0; i < 3; i++ {
		<-done
	}
}
