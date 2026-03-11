package nodepass_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"nodepass-pro/nodeclient/internal/domain/config"
	"nodepass-pro/nodeclient/internal/infra/nodepass"
)

func TestRestartBackoff(t *testing.T) {
	cases := []struct {
		attempt int
		expect  time.Duration
	}{
		{attempt: 0, expect: time.Second},
		{attempt: 1, expect: time.Second},
		{attempt: 2, expect: 2 * time.Second},
		{attempt: 3, expect: 4 * time.Second},
		{attempt: 4, expect: 8 * time.Second},
		{attempt: 5, expect: 16 * time.Second},
		{attempt: 6, expect: 32 * time.Second},
		{attempt: 10, expect: 32 * time.Second},
	}

	for _, tc := range cases {
		if got := nodepass.RestartBackoffForTest(tc.attempt); got != tc.expect {
			t.Fatalf("attempt=%d expect=%s got=%s", tc.attempt, tc.expect, got)
		}
	}
}

func TestHandleRestartFailure_RespectsLimit(t *testing.T) {
	integration := nodepass.NewIntegration(nil)
	instance := nodepass.NewInstanceForTest(
		1001,
		configForTest(1001),
		nodepass.WithStatus(nodepass.StatusRunning),
		nodepass.WithChannels(make(chan struct{}), make(chan struct{})),
	)

	if keepRunning := nodepass.HandleRestartFailureForTest(integration, instance, nodepass.MaxAutoRestartAttempts()-1, errors.New("temporary")); !keepRunning {
		t.Fatal("未达到上限时，期望继续自动恢复")
	}
	select {
	case <-instance.TestDoneCh:
		t.Fatal("未达到上限时，不应关闭 doneCh")
	default:
	}

	if keepRunning := nodepass.HandleRestartFailureForTest(integration, instance, nodepass.MaxAutoRestartAttempts(), errors.New("terminal")); keepRunning {
		t.Fatal("达到上限时，期望停止自动恢复")
	}
	select {
	case <-instance.TestDoneCh:
	default:
		t.Fatal("达到上限时，应关闭 doneCh")
	}
}

func TestStopRule_CleansUpAlreadyStoppingInstance(t *testing.T) {
	integration := nodepass.NewIntegration(nil)
	done := make(chan struct{})
	close(done)

	instance := nodepass.NewInstanceForTest(
		2002,
		configForTest(2002),
		nodepass.WithStatus(nodepass.StatusStopped),
		nodepass.WithDoneChannel(done, true),
		nodepass.WithStopChannel(make(chan struct{}), true),
	)
	integration.InstanceForTest(instance)

	if err := integration.StopRule(instance.RuleID); err != nil {
		t.Fatalf("期望清理成功，实际失败: %v", err)
	}
	if integration.InstanceExistsForTest(instance.RuleID) {
		t.Fatal("停止后实例应从 map 中移除")
	}
}

func TestWatchProcessExit_RemovesStoppedInstanceWithoutCmd(t *testing.T) {
	integration := nodepass.NewIntegration(nil)
	instance := nodepass.NewInstanceForTest(
		3003,
		configForTest(3003),
		nodepass.WithStatus(nodepass.StatusStopped),
		nodepass.WithChannels(make(chan struct{}), make(chan struct{})),
		nodepass.WithStopChannel(make(chan struct{}), true),
	)
	integration.InstanceForTest(instance)

	go nodepass.WatchProcessExitForTest(integration, instance)

	select {
	case <-instance.TestDoneCh:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("watchProcessExit 未在预期时间内退出")
	}

	if integration.InstanceExistsForTest(instance.RuleID) {
		t.Fatal("退出监听后实例应从 map 中移除")
	}
}

func configForTest(ruleID int) config.RuleConfig {
	return config.RuleConfig{
		RuleID:   ruleID,
		Mode:     "single",
		Protocol: "tcp",
		Listen: config.HostPort{
			Host: "127.0.0.1",
			Port: 12000 + ruleID%1000,
		},
		Target: config.HostPort{
			Host: "127.0.0.1",
			Port: 22000 + ruleID%1000,
		},
	}
}

func TestBuildCommand_RejectsRelativeNodepassBin(t *testing.T) {
	t.Setenv("NODEPASS_BIN", "nodepass-custom")

	_, err := nodepass.BuildCommandForTest(configForTest(4004))
	if err == nil {
		t.Fatal("期望相对路径被拒绝")
	}
	if !strings.Contains(err.Error(), "必须为绝对路径") {
		t.Fatalf("期望返回绝对路径错误，实际: %v", err)
	}
}

func TestBuildCommand_RejectsNonExecutableNodepassBin(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "nodepass.bin")
	if err := os.WriteFile(filePath, []byte("#!/bin/sh\necho test\n"), 0o644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}
	t.Setenv("NODEPASS_BIN", filePath)

	_, err := nodepass.BuildCommandForTest(configForTest(5005))
	if err == nil {
		t.Fatal("期望不可执行文件被拒绝")
	}
	if !strings.Contains(err.Error(), "不可执行") {
		t.Fatalf("期望返回不可执行错误，实际: %v", err)
	}
}
