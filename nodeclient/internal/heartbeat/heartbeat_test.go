package heartbeat

import (
	"strings"
	"testing"

	"nodepass-pro/nodeclient/internal/config"
)

func TestApplyReportedConfigVersion_PreventRollback(t *testing.T) {
	hb := NewHeartbeatService(&config.Config{
		HeartbeatInterval: 30,
	})
	hb.SetCurrentConfigVersion(10)

	hb.applyReportedConfigVersion(0)
	if got := hb.GetCurrentConfigVersion(); got != 10 {
		t.Fatalf("期望忽略回退版本，got=%d", got)
	}

	hb.applyReportedConfigVersion(-1)
	if got := hb.GetCurrentConfigVersion(); got != 10 {
		t.Fatalf("期望忽略负数版本，got=%d", got)
	}

	hb.applyReportedConfigVersion(11)
	if got := hb.GetCurrentConfigVersion(); got != 11 {
		t.Fatalf("期望更新到新版本，got=%d", got)
	}
}

func TestParseHeartbeatResponse_EnvelopeFailureWithoutMessageReturnsError(t *testing.T) {
	body := []byte(`{"success":false,"data":{"config_updated":false,"new_config_version":12}}`)

	_, err := parseHeartbeatResponse(body)
	if err == nil {
		t.Fatal("期望返回错误，实际为 nil")
	}
	if !strings.Contains(err.Error(), "success=false") {
		t.Fatalf("期望错误包含 success=false，实际: %v", err)
	}
}

func TestParseHeartbeatResponse_DirectPayloadStillSupported(t *testing.T) {
	body := []byte(`{"config_updated":true,"new_config_version":15}`)

	data, err := parseHeartbeatResponse(body)
	if err != nil {
		t.Fatalf("期望解析成功，实际失败: %v", err)
	}
	if !data.ConfigUpdated {
		t.Fatal("期望 config_updated=true")
	}
	if data.NewConfigVersion != 15 {
		t.Fatalf("期望 version=15，实际=%d", data.NewConfigVersion)
	}
}
