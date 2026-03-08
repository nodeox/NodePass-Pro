package metrics

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestNewCollector(t *testing.T) {
	collector := NewCollector()
	if collector == nil {
		t.Fatal("NewCollector() returned nil")
	}
	if collector.registry == nil {
		t.Error("registry should not be nil")
	}
}

func TestCollectorStartStop(t *testing.T) {
	collector := NewCollector()

	// 启动服务
	err := collector.Start(":19100")
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	// 等待服务启动
	time.Sleep(100 * time.Millisecond)

	// 测试健康检查端点
	resp, err := http.Get("http://localhost:19100/health")
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "OK" {
		t.Errorf("Expected body 'OK', got '%s'", string(body))
	}

	// 测试 metrics 端点
	resp, err = http.Get("http://localhost:19100/metrics")
	if err != nil {
		t.Fatalf("Metrics endpoint failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// 停止服务
	collector.Stop()

	// 验证服务已停止
	time.Sleep(100 * time.Millisecond)
	_, err = http.Get("http://localhost:19100/health")
	if err == nil {
		t.Error("Expected error after Stop(), but got none")
	}
}

func TestRecordHeartbeat(t *testing.T) {
	collector := NewCollector()

	// 记录心跳尝试
	collector.RecordHeartbeatAttempt()
	collector.RecordHeartbeatAttempt()

	// 记录心跳成功
	timestamp := float64(time.Now().Unix())
	collector.RecordHeartbeatSuccess(timestamp)

	// 记录心跳失败
	collector.RecordHeartbeatFailure()

	// 启动服务以验证指标
	err := collector.Start(":19101")
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer collector.Stop()

	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get("http://localhost:19101/metrics")
	if err != nil {
		t.Fatalf("Failed to get metrics: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	metrics := string(body)

	// 验证指标存在
	if !strings.Contains(metrics, "nodeclient_heartbeat_total") {
		t.Error("Missing nodeclient_heartbeat_total metric")
	}
	if !strings.Contains(metrics, "nodeclient_heartbeat_status") {
		t.Error("Missing nodeclient_heartbeat_status metric")
	}
	if !strings.Contains(metrics, "nodeclient_heartbeat_failures_total") {
		t.Error("Missing nodeclient_heartbeat_failures_total metric")
	}
}

func TestSetConfigVersion(t *testing.T) {
	collector := NewCollector()

	collector.SetConfigVersion(42)

	err := collector.Start(":19102")
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer collector.Stop()

	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get("http://localhost:19102/metrics")
	if err != nil {
		t.Fatalf("Failed to get metrics: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	metrics := string(body)

	if !strings.Contains(metrics, "nodeclient_config_version 42") {
		t.Error("Config version not set correctly")
	}
}

func TestSetRuleStats(t *testing.T) {
	collector := NewCollector()

	collector.SetRuleStats(10, 7, 2, 1)

	err := collector.Start(":19103")
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer collector.Stop()

	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get("http://localhost:19103/metrics")
	if err != nil {
		t.Fatalf("Failed to get metrics: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	metrics := string(body)

	if !strings.Contains(metrics, "nodeclient_rules_total 10") {
		t.Error("Rules total not set correctly")
	}
	if !strings.Contains(metrics, "nodeclient_rules_running 7") {
		t.Error("Rules running not set correctly")
	}
}

func TestSetSystemStats(t *testing.T) {
	collector := NewCollector()

	collector.SetSystemStats(45.5, 60.2, 75.8, 1024000, 2048000)

	err := collector.Start(":19104")
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer collector.Stop()

	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get("http://localhost:19104/metrics")
	if err != nil {
		t.Fatalf("Failed to get metrics: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	metrics := string(body)

	if !strings.Contains(metrics, "nodeclient_cpu_usage_percent 45.5") {
		t.Error("CPU usage not set correctly")
	}
	if !strings.Contains(metrics, "nodeclient_memory_usage_percent 60.2") {
		t.Error("Memory usage not set correctly")
	}
}

func TestNilCollector(t *testing.T) {
	var collector *Collector

	// 所有方法应该安全处理 nil
	collector.Start(":9999")
	collector.Stop()
	collector.RecordHeartbeatAttempt()
	collector.RecordHeartbeatSuccess(0)
	collector.RecordHeartbeatFailure()
	collector.SetConfigVersion(0)
	collector.SetRuleStats(0, 0, 0, 0)
	collector.SetSystemStats(0, 0, 0, 0, 0)

	// 如果没有 panic，测试通过
}
