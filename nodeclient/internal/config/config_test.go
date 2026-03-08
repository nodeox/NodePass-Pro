package config

import (
	"bytes"
	"log"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad_AllowsMissingConfigFileWithCLIOverrides(t *testing.T) {
	missingPath := filepath.Join(t.TempDir(), "missing.yaml")

	cfg, err := Load(missingPath, CLIOverrides{
		HubURL:      "https://panel.example.com",
		NodeID:      "node-1",
		GroupID:     1,
		ServiceName: "nodeclient-test",
		Token:       "token-1",
	})
	if err != nil {
		t.Fatalf("期望配置加载成功，实际失败: %v", err)
	}

	if cfg.HubURL != "https://panel.example.com" {
		t.Fatalf("hub_url 不匹配: %s", cfg.HubURL)
	}
	if cfg.NodeID != "node-1" {
		t.Fatalf("node_id 不匹配: %s", cfg.NodeID)
	}
	if cfg.NodeToken != "token-1" {
		t.Fatalf("node_token 不匹配: %s", cfg.NodeToken)
	}
	if cfg.CachePath == "" {
		t.Fatal("cache_path 不应为空")
	}
}

func TestLoad_MissingConfigFileStillRunsValidation(t *testing.T) {
	missingPath := filepath.Join(t.TempDir(), "missing.yaml")

	_, err := Load(missingPath, CLIOverrides{
		HubURL:      "https://panel.example.com",
		NodeID:      "node-1",
		GroupID:     1,
		ServiceName: "nodeclient-test",
	})
	if err == nil {
		t.Fatal("缺少 token 时期望校验失败")
	}
	if !strings.Contains(err.Error(), "node_token") {
		t.Fatalf("期望返回 node_token 校验错误，实际: %v", err)
	}
}

func TestValidate_GroupIDWarningUsesLogger(t *testing.T) {
	buf := &bytes.Buffer{}
	SetWarnLogger(log.New(buf, "", 0))

	cfg := &Config{
		HubURL:                "https://panel.example.com",
		NodeID:                "node-1",
		GroupID:               0,
		ServiceName:           "nodeclient-test",
		NodeToken:             "token-1",
		HeartbeatInterval:     30,
		ConfigCheckInterval:   60,
		TrafficReportInterval: 60,
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("期望校验通过，实际失败: %v", err)
	}

	logText := buf.String()
	if !strings.Contains(logText, "group_id 未配置或为 0") {
		t.Fatalf("期望输出 group_id 警告日志，实际: %q", logText)
	}
}
