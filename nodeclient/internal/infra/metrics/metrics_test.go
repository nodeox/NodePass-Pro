package metrics_test

import (
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"nodepass-pro/nodeclient/internal/infra/metrics"
)

func TestNewCollector(t *testing.T) {
	collector := metrics.NewCollector()
	if collector == nil {
		t.Fatal("NewCollector() returned nil")
	}
	// registry 为内部实现细节，这里只验证无 panic 即可。
}

func TestCollectorStartStop(t *testing.T) {
	collector := metrics.NewCollector()

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

func TestCollectorStartReturnsErrorWhenPortOccupied(t *testing.T) {
	reserved, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to reserve port: %v", err)
	}
	tcpAddr, ok := reserved.Addr().(*net.TCPAddr)
	if !ok {
		_ = reserved.Close()
		t.Fatalf("unexpected listener addr type: %T", reserved.Addr())
	}
	addr := ":" + strconv.Itoa(tcpAddr.Port)
	_ = reserved.Close()

	// 先启动一个 collector 占用动态端口
	collector1 := metrics.NewCollector()
	err = collector1.Start(addr)
	if err != nil {
		t.Fatalf("failed to start first collector: %v", err)
	}
	defer collector1.Stop()

	// 等待服务启动
	time.Sleep(100 * time.Millisecond)

	// 尝试在同一端口启动第二个 collector，应该失败
	collector2 := metrics.NewCollector()
	err = collector2.Start(addr)
	if err == nil {
		collector2.Stop()
		t.Fatal("expected Start() to return bind error when port is occupied")
	}
	if !strings.Contains(err.Error(), "监听") && !strings.Contains(err.Error(), "address already in use") {
		t.Errorf("expected bind error, got: %v", err)
	}
}

func TestRecordHeartbeat(t *testing.T) {
	collector := metrics.NewCollector()

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
	collector := metrics.NewCollector()

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
	collector := metrics.NewCollector()

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
	collector := metrics.NewCollector()

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
	var collector *metrics.Collector

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
