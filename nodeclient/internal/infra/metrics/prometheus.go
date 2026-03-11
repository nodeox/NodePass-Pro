package metrics

import (
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Collector 定义 Prometheus 指标收集器。
type Collector struct {
	// 心跳相关指标
	heartbeatStatus    prometheus.Gauge
	heartbeatTotal     prometheus.Counter
	heartbeatFailures  prometheus.Counter
	configVersion      prometheus.Gauge
	lastHeartbeatTime  prometheus.Gauge

	// 规则相关指标
	rulesTotal         prometheus.Gauge
	rulesRunning       prometheus.Gauge
	rulesStopped       prometheus.Gauge
	rulesError         prometheus.Gauge
	ruleRestarts       *prometheus.CounterVec

	// 流量相关指标
	trafficInBytes     *prometheus.CounterVec
	trafficOutBytes    *prometheus.CounterVec
	activeConnections  *prometheus.GaugeVec

	// 系统相关指标
	cpuUsage           prometheus.Gauge
	memoryUsage        prometheus.Gauge
	diskUsage          prometheus.Gauge
	bandwidthInBytes   prometheus.Gauge
	bandwidthOutBytes  prometheus.Gauge

	registry *prometheus.Registry
	server   *http.Server
	logger   *log.Logger
	mu       sync.Mutex
	started  bool
}

// NewCollector 创建 Prometheus 指标收集器。
func NewCollector() *Collector {
	registry := prometheus.NewRegistry()

	c := &Collector{
		heartbeatStatus: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "nodeclient_heartbeat_status",
			Help: "Heartbeat status (1=online, 0=offline)",
		}),
		heartbeatTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "nodeclient_heartbeat_total",
			Help: "Total number of heartbeat attempts",
		}),
		heartbeatFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "nodeclient_heartbeat_failures_total",
			Help: "Total number of heartbeat failures",
		}),
		configVersion: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "nodeclient_config_version",
			Help: "Current configuration version",
		}),
		lastHeartbeatTime: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "nodeclient_last_heartbeat_timestamp_seconds",
			Help: "Timestamp of last successful heartbeat",
		}),

		rulesTotal: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "nodeclient_rules_total",
			Help: "Total number of rules",
		}),
		rulesRunning: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "nodeclient_rules_running",
			Help: "Number of running rules",
		}),
		rulesStopped: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "nodeclient_rules_stopped",
			Help: "Number of stopped rules",
		}),
		rulesError: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "nodeclient_rules_error",
			Help: "Number of rules in error state",
		}),
		ruleRestarts: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "nodeclient_rule_restarts_total",
				Help: "Total number of rule restarts",
			},
			[]string{"rule_id", "mode"},
		),

		trafficInBytes: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "nodeclient_traffic_in_bytes_total",
				Help: "Total inbound traffic in bytes",
			},
			[]string{"rule_id"},
		),
		trafficOutBytes: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "nodeclient_traffic_out_bytes_total",
				Help: "Total outbound traffic in bytes",
			},
			[]string{"rule_id"},
		),
		activeConnections: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "nodeclient_active_connections",
				Help: "Number of active connections",
			},
			[]string{"rule_id"},
		),

		cpuUsage: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "nodeclient_cpu_usage_percent",
			Help: "CPU usage percentage",
		}),
		memoryUsage: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "nodeclient_memory_usage_percent",
			Help: "Memory usage percentage",
		}),
		diskUsage: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "nodeclient_disk_usage_percent",
			Help: "Disk usage percentage",
		}),
		bandwidthInBytes: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "nodeclient_bandwidth_in_bytes_per_second",
			Help: "Inbound bandwidth in bytes per second",
		}),
		bandwidthOutBytes: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "nodeclient_bandwidth_out_bytes_per_second",
			Help: "Outbound bandwidth in bytes per second",
		}),

		registry: registry,
		logger:   log.New(os.Stdout, "[metrics] ", log.LstdFlags),
	}

	// 注册所有指标
	registry.MustRegister(
		c.heartbeatStatus,
		c.heartbeatTotal,
		c.heartbeatFailures,
		c.configVersion,
		c.lastHeartbeatTime,
		c.rulesTotal,
		c.rulesRunning,
		c.rulesStopped,
		c.rulesError,
		c.ruleRestarts,
		c.trafficInBytes,
		c.trafficOutBytes,
		c.activeConnections,
		c.cpuUsage,
		c.memoryUsage,
		c.diskUsage,
		c.bandwidthInBytes,
		c.bandwidthOutBytes,
	)

	return c
}

// Start 启动 Prometheus HTTP 服务器。
func (c *Collector) Start(addr string) error {
	if c == nil {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.started {
		return nil
	}

	if addr == "" {
		addr = ":9100"
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(c.registry, promhttp.HandlerOpts{}))
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	c.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		c.logger.Printf("[INFO] Prometheus metrics 服务已启动: %s", addr)
		if err := c.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			c.logger.Printf("[ERROR] Prometheus metrics 服务异常: %v", err)
		}
	}()

	c.started = true
	return nil
}

// Stop 停止 Prometheus HTTP 服务器。
func (c *Collector) Stop() {
	if c == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.started || c.server == nil {
		return
	}

	if err := c.server.Close(); err != nil {
		c.logger.Printf("[WARN] 关闭 Prometheus metrics 服务失败: %v", err)
	}

	c.started = false
	c.logger.Printf("[INFO] Prometheus metrics 服务已停止")
}

// RecordHeartbeatAttempt 记录心跳尝试。
func (c *Collector) RecordHeartbeatAttempt() {
	if c == nil {
		return
	}
	c.heartbeatTotal.Inc()
}

// RecordHeartbeatSuccess 记录心跳成功。
func (c *Collector) RecordHeartbeatSuccess(timestamp float64) {
	if c == nil {
		return
	}
	c.heartbeatStatus.Set(1)
	c.lastHeartbeatTime.Set(timestamp)
}

// RecordHeartbeatFailure 记录心跳失败。
func (c *Collector) RecordHeartbeatFailure() {
	if c == nil {
		return
	}
	c.heartbeatStatus.Set(0)
	c.heartbeatFailures.Inc()
}

// SetConfigVersion 设置配置版本。
func (c *Collector) SetConfigVersion(version int) {
	if c == nil {
		return
	}
	c.configVersion.Set(float64(version))
}

// SetRuleStats 设置规则统计。
func (c *Collector) SetRuleStats(total, running, stopped, errored int) {
	if c == nil {
		return
	}
	c.rulesTotal.Set(float64(total))
	c.rulesRunning.Set(float64(running))
	c.rulesStopped.Set(float64(stopped))
	c.rulesError.Set(float64(errored))
}

// RecordRuleRestart 记录规则重启。
func (c *Collector) RecordRuleRestart(ruleID, mode string) {
	if c == nil {
		return
	}
	c.ruleRestarts.WithLabelValues(ruleID, mode).Inc()
}

// SetTrafficStats 设置流量统计。
func (c *Collector) SetTrafficStats(ruleID string, inBytes, outBytes int64) {
	if c == nil {
		return
	}
	c.trafficInBytes.WithLabelValues(ruleID).Add(float64(inBytes))
	c.trafficOutBytes.WithLabelValues(ruleID).Add(float64(outBytes))
}

// SetActiveConnections 设置活跃连接数。
func (c *Collector) SetActiveConnections(ruleID string, count int64) {
	if c == nil {
		return
	}
	c.activeConnections.WithLabelValues(ruleID).Set(float64(count))
}

// SetSystemStats 设置系统统计。
func (c *Collector) SetSystemStats(cpuUsage, memoryUsage, diskUsage float64, bandwidthIn, bandwidthOut int64) {
	if c == nil {
		return
	}
	c.cpuUsage.Set(cpuUsage)
	c.memoryUsage.Set(memoryUsage)
	c.diskUsage.Set(diskUsage)
	c.bandwidthInBytes.Set(float64(bandwidthIn))
	c.bandwidthOutBytes.Set(float64(bandwidthOut))
}
