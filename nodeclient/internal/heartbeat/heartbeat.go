package heartbeat

import (
	"bytes"
	cryptorand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"nodepass-pro/nodeclient/internal/config"
)

// SystemInfoData 表示节点系统信息。
type SystemInfoData struct {
	CPUUsage     float64 `json:"cpu_usage"`
	MemoryUsage  float64 `json:"memory_usage"`
	DiskUsage    float64 `json:"disk_usage"`
	BandwidthIn  int64   `json:"bandwidth_in"`
	BandwidthOut int64   `json:"bandwidth_out"`
	Connections  int64   `json:"connections"`
}

// TrafficData 表示节点流量信息。
type TrafficData struct {
	TrafficIn         int64 `json:"traffic_in"`
	TrafficOut        int64 `json:"traffic_out"`
	ActiveConnections int64 `json:"active_connections"`
}

// HeartbeatPayload 表示心跳上报载荷。
type HeartbeatPayload struct {
	NodeID               string         `json:"node_id"`
	Token                string         `json:"token"`
	NodeRole             string         `json:"node_role,omitempty"`
	CurrentConfigVersion int            `json:"current_config_version"`
	ConnectionAddress    string         `json:"connection_address,omitempty"`
	SystemInfo           SystemInfoData `json:"system_info"`
	TrafficStats         TrafficData    `json:"traffic_stats"`
}

// HeartbeatResponseData 表示心跳返回数据。
type HeartbeatResponseData struct {
	ConfigUpdated    bool               `json:"config_updated"`
	NewConfigVersion int                `json:"new_config_version"`
	Config           *config.NodeConfig `json:"config,omitempty"`
}

type heartbeatErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type heartbeatResponseEnvelope struct {
	Success   bool                  `json:"success"`
	Data      HeartbeatResponseData `json:"data"`
	Message   string                `json:"message"`
	Timestamp string                `json:"timestamp"`
	Error     *heartbeatErrorBody   `json:"error,omitempty"`
}

// RuntimeMetricsProvider 允许外部注入 NodePass 统计，用于覆盖心跳中的连接与流量。
type RuntimeMetricsProvider interface {
	SnapshotHeartbeatMetrics() (trafficIn int64, trafficOut int64, activeConnections int64)
}

// ConfigUpdateHandler 用于配置热更新回调。
type ConfigUpdateHandler func(cfg *config.NodeConfig, version int)

// HeartbeatService 定义节点心跳上报服务。
type HeartbeatService struct {
	config   *config.Config
	client   *http.Client
	interval time.Duration
	stopCh   chan struct{}
	isOnline bool
	mu       sync.RWMutex

	doneCh              chan struct{}
	startOnce           sync.Once
	stopOnce            sync.Once
	failureCount        int
	lastCPUTotal        uint64
	lastCPUIdle         uint64
	lastNetIn           uint64
	lastNetOut          uint64
	lastNetSnapshotTime time.Time
	logger              *log.Logger
	metricsProvider     RuntimeMetricsProvider
	metricsCollector    MetricsCollector
	configUpdateHandler ConfigUpdateHandler
	configVersion       int
	started             bool
	resolvedAddress     string
}

// MetricsCollector 定义 metrics 收集器接口。
type MetricsCollector interface {
	RecordHeartbeatAttempt()
	RecordHeartbeatSuccess(timestamp float64)
	RecordHeartbeatFailure()
	SetSystemStats(cpuUsage, memoryUsage, diskUsage float64, bandwidthIn, bandwidthOut int64)
}

// NewHeartbeatService 创建心跳服务。
// 默认上报间隔 30 秒，HTTP 超时 10 秒。
func NewHeartbeatService(cfg *config.Config) *HeartbeatService {
	interval := 30 * time.Second
	if cfg != nil && cfg.HeartbeatInterval > 0 {
		interval = time.Duration(cfg.HeartbeatInterval) * time.Second
	}

	return &HeartbeatService{
		config:        cfg,
		client:        &http.Client{Timeout: 10 * time.Second},
		interval:      interval,
		stopCh:        make(chan struct{}),
		doneCh:        make(chan struct{}),
		logger:        log.New(os.Stdout, "[heartbeat] ", log.LstdFlags),
		configVersion: -1,
	}
}

// SetMetricsProvider 设置运行时统计提供者（可选）。
func (h *HeartbeatService) SetMetricsProvider(provider RuntimeMetricsProvider) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.metricsProvider = provider
}

// SetMetricsCollector 设置 Prometheus metrics 收集器（可选）。
func (h *HeartbeatService) SetMetricsCollector(collector MetricsCollector) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.metricsCollector = collector
}

// SetConfigUpdateHandler 设置配置更新回调（可选）。
func (h *HeartbeatService) SetConfigUpdateHandler(handler ConfigUpdateHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.configUpdateHandler = handler
}

// Start 启动后台心跳循环。
func (h *HeartbeatService) Start() {
	if h == nil {
		return
	}

	h.startOnce.Do(func() {
		h.mu.Lock()
		h.started = true
		h.mu.Unlock()

		go func() {
			defer close(h.doneCh)

			h.logger.Printf("[INFO] 心跳服务已启动，间隔=%s", h.interval)
			for {
				wait := h.nextReportInterval()
				timer := time.NewTimer(wait)
				select {
				case <-timer.C:
					_ = h.Report()
				case <-h.stopCh:
					timer.Stop()
					h.logger.Printf("[INFO] 心跳服务已停止")
					return
				}
			}
		}()
	})
}

// Stop 发送停止信号并等待退出。
func (h *HeartbeatService) Stop() {
	if h == nil {
		return
	}

	h.mu.RLock()
	started := h.started
	h.mu.RUnlock()
	if !started {
		return
	}

	h.stopOnce.Do(func() {
		close(h.stopCh)
	})

	<-h.doneCh
}

// SetCurrentConfigVersion 设置当前已应用配置版本。
func (h *HeartbeatService) SetCurrentConfigVersion(version int) {
	if h == nil {
		return
	}
	h.mu.Lock()
	h.configVersion = version
	h.mu.Unlock()
}

// GetCurrentConfigVersion 获取当前已应用配置版本。
func (h *HeartbeatService) GetCurrentConfigVersion() int {
	if h == nil {
		return -1
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.configVersion
}

// IsOnline 线程安全返回当前在线状态。
func (h *HeartbeatService) IsOnline() bool {
	if h == nil {
		return false
	}

	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.isOnline
}

// Report 执行一次心跳上报。
func (h *HeartbeatService) Report() error {
	if h == nil {
		return fmt.Errorf("heartbeat 服务不能为空")
	}
	if h.config == nil {
		return h.reportError(fmt.Errorf("配置不能为空"))
	}

	// 记录心跳尝试
	h.mu.RLock()
	collector := h.metricsCollector
	h.mu.RUnlock()
	if collector != nil {
		collector.RecordHeartbeatAttempt()
	}

	hubURL := strings.TrimRight(strings.TrimSpace(h.config.HubURL), "/")
	if hubURL == "" {
		return h.reportError(fmt.Errorf("hub_url 不能为空"))
	}
	if strings.TrimSpace(h.config.NodeID) == "" {
		return h.reportError(fmt.Errorf("node_id 不能为空"))
	}
	if strings.TrimSpace(h.config.NodeToken) == "" {
		return h.reportError(fmt.Errorf("node_token 不能为空"))
	}

	sysInfo, traffic := h.collectHeartbeatData()
	connectionAddress := h.resolveConnectionAddress()

	payload := HeartbeatPayload{
		NodeID:               h.config.NodeID,
		Token:                h.config.NodeToken,
		NodeRole:             strings.TrimSpace(h.config.NodeRole),
		CurrentConfigVersion: h.GetCurrentConfigVersion(),
		ConnectionAddress:    connectionAddress,
		SystemInfo:           sysInfo,
		TrafficStats:         traffic,
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return h.reportError(fmt.Errorf("序列化心跳请求失败: %w", err))
	}

	req, err := http.NewRequest(http.MethodPost, hubURL+"/api/v1/node-instances/heartbeat", bytes.NewReader(raw))
	if err != nil {
		return h.reportError(fmt.Errorf("创建心跳请求失败: %w", err))
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Timestamp", strconv.FormatInt(time.Now().Unix(), 10))
	nonce, nonceErr := generateRequestNonce(16)
	if nonceErr != nil {
		return h.reportError(fmt.Errorf("生成心跳 nonce 失败: %w", nonceErr))
	}
	req.Header.Set("X-Nonce", nonce)

	resp, err := h.client.Do(req)
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			h.logger.Printf("[WARN] 心跳网络超时: %v", err)
		}
		return h.reportError(fmt.Errorf("发送心跳失败: %w", err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return h.reportError(fmt.Errorf("读取心跳响应失败: %w", err))
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return h.reportError(fmt.Errorf("心跳响应异常(%d): %s", resp.StatusCode, strings.TrimSpace(string(body))))
	}

	data, err := parseHeartbeatResponse(body)
	if err != nil {
		return h.reportError(err)
	}

	h.markOnlineSuccess()

	// 记录心跳成功和系统统计
	if collector != nil {
		collector.RecordHeartbeatSuccess(float64(time.Now().Unix()))
		collector.SetSystemStats(
			sysInfo.CPUUsage,
			sysInfo.MemoryUsage,
			sysInfo.DiskUsage,
			sysInfo.BandwidthIn,
			sysInfo.BandwidthOut,
		)
	}

	if data.ConfigUpdated {
		h.logger.Printf("[INFO] 收到新配置: version=%d", data.NewConfigVersion)
		h.mu.RLock()
		handler := h.configUpdateHandler
		h.mu.RUnlock()
		if handler != nil {
			handler(data.Config, data.NewConfigVersion)
		} else {
			h.logger.Printf("[WARN] 未设置配置更新回调，已忽略新配置")
		}
	} else {
		h.applyReportedConfigVersion(data.NewConfigVersion)
	}

	return nil
}

func (h *HeartbeatService) applyReportedConfigVersion(version int) {
	if h == nil {
		return
	}
	if version < 0 {
		return
	}

	current := h.GetCurrentConfigVersion()
	if current >= 0 && version < current {
		h.logger.Printf("[WARN] 忽略回退配置版本: current=%d reported=%d", current, version)
		return
	}

	h.SetCurrentConfigVersion(version)
}

func (h *HeartbeatService) resolveConnectionAddress() string {
	if h == nil || h.config == nil {
		return ""
	}

	candidates := []string{
		strings.TrimSpace(h.config.ConnectionAddress),
		strings.TrimSpace(h.config.ConnectHost),
	}
	for _, candidate := range candidates {
		if candidate == "" || strings.EqualFold(candidate, "auto") {
			continue
		}
		return candidate
	}

	h.mu.RLock()
	cached := strings.TrimSpace(h.resolvedAddress)
	h.mu.RUnlock()
	if cached != "" {
		return cached
	}

	detected := strings.TrimSpace(detectOutboundAddress())
	if detected == "" {
		if host, err := os.Hostname(); err == nil {
			detected = strings.TrimSpace(host)
		}
	}
	if detected == "" {
		return ""
	}

	h.mu.Lock()
	if strings.TrimSpace(h.resolvedAddress) == "" {
		h.resolvedAddress = detected
	}
	h.mu.Unlock()
	return detected
}

func detectOutboundAddress() string {
	targets := []string{
		"1.1.1.1:53",
		"8.8.8.8:53",
		"[2606:4700:4700::1111]:53",
		"[2001:4860:4860::8888]:53",
	}

	for _, target := range targets {
		conn, err := net.DialTimeout("udp", target, 2*time.Second)
		if err != nil {
			continue
		}

		localAddr := conn.LocalAddr().String()
		_ = conn.Close()

		host, _, splitErr := net.SplitHostPort(localAddr)
		if splitErr != nil {
			host = localAddr
		}
		host = strings.Trim(strings.TrimSpace(host), "[]")
		if host == "" || host == "127.0.0.1" || host == "::1" {
			continue
		}
		return host
	}

	return ""
}

func generateRequestNonce(size int) (string, error) {
	if size <= 0 {
		size = 16
	}
	buf := make([]byte, size)
	if _, err := cryptorand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func (h *HeartbeatService) nextReportInterval() time.Duration {
	h.mu.RLock()
	online := h.isOnline
	interval := h.interval
	h.mu.RUnlock()

	if interval <= 0 {
		interval = 30 * time.Second
	}
	if online {
		return interval
	}
	if interval < 60*time.Second {
		return 60 * time.Second
	}
	return interval
}

func (h *HeartbeatService) reportError(err error) error {
	h.mu.Lock()
	h.isOnline = false
	h.failureCount++
	failures := h.failureCount
	collector := h.metricsCollector
	h.mu.Unlock()

	// 记录心跳失败
	if collector != nil {
		collector.RecordHeartbeatFailure()
	}

	h.logger.Printf("[WARN] 心跳发送失败: %v", err)
	if failures >= 3 {
		h.logger.Printf("[ERROR] 心跳连续失败 %d 次: %v", failures, err)
	}

	return err
}

func (h *HeartbeatService) markOnlineSuccess() {
	h.mu.Lock()
	h.isOnline = true
	h.failureCount = 0
	h.mu.Unlock()
}

func (h *HeartbeatService) collectHeartbeatData() (SystemInfoData, TrafficData) {
	connections := readConnectionCount()
	netIn, netOut := readNetworkBytes()
	now := time.Now()

	bandwidthIn, bandwidthOut := h.calcBandwidth(netIn, netOut, now)

	trafficIn := saturatingToInt64(netIn)
	trafficOut := saturatingToInt64(netOut)
	activeConnections := connections

	h.mu.RLock()
	provider := h.metricsProvider
	h.mu.RUnlock()
	if provider != nil {
		in, out, active := provider.SnapshotHeartbeatMetrics()
		if in >= 0 {
			trafficIn = in
		}
		if out >= 0 {
			trafficOut = out
		}
		if active >= 0 {
			activeConnections = active
		}
	}

	sysInfo := SystemInfoData{
		CPUUsage:     h.readCPUUsagePercent(),
		MemoryUsage:  readMemoryUsagePercent(),
		DiskUsage:    readDiskUsagePercent("/"),
		BandwidthIn:  bandwidthIn,
		BandwidthOut: bandwidthOut,
		Connections:  connections,
	}

	traffic := TrafficData{
		TrafficIn:         trafficIn,
		TrafficOut:        trafficOut,
		ActiveConnections: activeConnections,
	}

	return sysInfo, traffic
}

func (h *HeartbeatService) calcBandwidth(totalIn uint64, totalOut uint64, now time.Time) (int64, int64) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.lastNetSnapshotTime.IsZero() || now.Before(h.lastNetSnapshotTime) {
		h.lastNetIn = totalIn
		h.lastNetOut = totalOut
		h.lastNetSnapshotTime = now
		return 0, 0
	}

	elapsed := now.Sub(h.lastNetSnapshotTime).Seconds()
	if elapsed <= 0 {
		return 0, 0
	}

	var inRate, outRate int64
	if totalIn >= h.lastNetIn {
		inRate = int64(float64(totalIn-h.lastNetIn) / elapsed)
	}
	if totalOut >= h.lastNetOut {
		outRate = int64(float64(totalOut-h.lastNetOut) / elapsed)
	}

	h.lastNetIn = totalIn
	h.lastNetOut = totalOut
	h.lastNetSnapshotTime = now

	return inRate, outRate
}

func (h *HeartbeatService) readCPUUsagePercent() float64 {
	total, idle, err := readCPUJiffies()
	if err != nil {
		return 0
	}

	h.mu.Lock()
	prevTotal := h.lastCPUTotal
	prevIdle := h.lastCPUIdle
	h.lastCPUTotal = total
	h.lastCPUIdle = idle
	h.mu.Unlock()

	if prevTotal == 0 || total <= prevTotal {
		return 0
	}

	deltaTotal := total - prevTotal
	deltaIdle := idle - prevIdle
	if deltaTotal == 0 {
		return 0
	}

	usage := (1 - float64(deltaIdle)/float64(deltaTotal)) * 100
	return clampPercent(usage)
}

func parseHeartbeatResponse(body []byte) (HeartbeatResponseData, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err == nil {
		_, hasSuccess := raw["success"]
		_, hasData := raw["data"]
		_, hasMessage := raw["message"]
		_, hasError := raw["error"]

		if hasSuccess || hasData || hasMessage || hasError {
			var envelope heartbeatResponseEnvelope
			if err := json.Unmarshal(body, &envelope); err != nil {
				return HeartbeatResponseData{}, fmt.Errorf("解析心跳响应失败: %w", err)
			}
			if !envelope.Success {
				if envelope.Error != nil && envelope.Error.Message != "" {
					return HeartbeatResponseData{}, fmt.Errorf("心跳接口失败: %s (%s)", envelope.Error.Message, envelope.Error.Code)
				}
				if strings.TrimSpace(envelope.Message) != "" {
					return HeartbeatResponseData{}, fmt.Errorf("心跳接口失败: %s", strings.TrimSpace(envelope.Message))
				}
				return HeartbeatResponseData{}, fmt.Errorf("心跳接口失败: success=false")
			}
			return envelope.Data, nil
		}
	}

	var direct HeartbeatResponseData
	if err := json.Unmarshal(body, &direct); err != nil {
		return HeartbeatResponseData{}, fmt.Errorf("解析心跳响应失败: %w", err)
	}
	return direct, nil
}

func readCPUJiffies() (total uint64, idle uint64, err error) {
	content, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0, 0, err
	}

	for _, line := range strings.Split(string(content), "\n") {
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 5 {
			return 0, 0, fmt.Errorf("/proc/stat 格式异常")
		}

		values := make([]uint64, 0, len(fields)-1)
		for _, raw := range fields[1:] {
			v, parseErr := strconv.ParseUint(raw, 10, 64)
			if parseErr != nil {
				return 0, 0, parseErr
			}
			values = append(values, v)
			total += v
		}

		idle = values[3]
		if len(values) > 4 {
			idle += values[4] // iowait
		}
		return total, idle, nil
	}

	return 0, 0, fmt.Errorf("未找到 CPU 统计信息")
}

func readMemoryUsagePercent() float64 {
	content, err := os.ReadFile("/proc/meminfo")
	if err == nil {
		var total float64
		var available float64

		for _, line := range strings.Split(string(content), "\n") {
			if strings.HasPrefix(line, "MemTotal:") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					if v, parseErr := strconv.ParseFloat(parts[1], 64); parseErr == nil {
						total = v
					}
				}
			}
			if strings.HasPrefix(line, "MemAvailable:") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					if v, parseErr := strconv.ParseFloat(parts[1], 64); parseErr == nil {
						available = v
					}
				}
			}
		}

		if total > 0 {
			used := total - available
			if used < 0 {
				used = total
			}
			return clampPercent((used / total) * 100)
		}
	}

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	if mem.Sys == 0 {
		return 0
	}
	return clampPercent((float64(mem.Alloc) / float64(mem.Sys)) * 100)
}

func readDiskUsagePercent(path string) float64 {
	if strings.TrimSpace(path) == "" {
		path = "/"
	}

	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0
	}
	if stat.Blocks == 0 {
		return 0
	}

	total := float64(stat.Blocks) * float64(stat.Bsize)
	free := float64(stat.Bavail) * float64(stat.Bsize)
	used := total - free
	if total <= 0 {
		return 0
	}

	return clampPercent((used / total) * 100)
}

func readNetworkBytes() (uint64, uint64) {
	content, err := os.ReadFile("/proc/net/dev")
	if err != nil {
		return 0, 0
	}

	var totalIn uint64
	var totalOut uint64

	lines := strings.Split(string(content), "\n")
	for _, line := range lines[2:] {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, ":")
		if len(parts) != 2 {
			continue
		}

		iface := strings.TrimSpace(parts[0])
		if iface == "lo" {
			continue
		}

		fields := strings.Fields(parts[1])
		if len(fields) < 9 {
			continue
		}

		recv, recvErr := strconv.ParseUint(fields[0], 10, 64)
		sent, sentErr := strconv.ParseUint(fields[8], 10, 64)
		if recvErr != nil || sentErr != nil {
			continue
		}

		totalIn += recv
		totalOut += sent
	}

	return totalIn, totalOut
}

func readConnectionCount() int64 {
	files := []string{"/proc/net/tcp", "/proc/net/tcp6", "/proc/net/udp", "/proc/net/udp6"}
	var total int64

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		lines := strings.Split(strings.TrimSpace(string(content)), "\n")
		if len(lines) <= 1 {
			continue
		}

		total += int64(len(lines) - 1)
	}

	return total
}

func clampPercent(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

func saturatingToInt64(value uint64) int64 {
	if value > math.MaxInt64 {
		return math.MaxInt64
	}
	return int64(value)
}
