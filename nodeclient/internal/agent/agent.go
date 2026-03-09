package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"nodepass-pro/nodeclient/internal/config"
	"nodepass-pro/nodeclient/internal/heartbeat"
	"nodepass-pro/nodeclient/internal/logger"
	"nodepass-pro/nodeclient/internal/metrics"
	"nodepass-pro/nodeclient/internal/nodepass"
)

var clientVersion = "1.0.0"

// Version 返回节点客户端版本号。
func Version() string {
	return clientVersion
}

// Agent 定义节点客户端核心控制器（纯新链路：仅 node-instances 心跳）。
type Agent struct {
	config      *config.Config
	configCache *config.ConfigCache
	nodePass    *nodepass.Integration
	heartbeat   *heartbeat.HeartbeatService
	metrics     *metrics.Collector
	isOnline    bool
	mu          sync.RWMutex
	applyMu     sync.Mutex

	logger    *logger.Logger
	stopCh    chan struct{}
	stopOnce  sync.Once
	waitGroup sync.WaitGroup

	configVersion int
	currentConfig *config.NodeConfig
}

// NewAgent 创建 Agent。
func NewAgent(cfg *config.Config) *Agent {
	if cfg == nil {
		cfg = &config.Config{}
	}

	log := newLogger(cfg)

	a := &Agent{
		config:        cfg,
		configCache:   config.NewConfigCache(cfg.CachePath),
		nodePass:      nodepass.NewIntegration(log.StdLogger()),
		metrics:       metrics.NewCollector(),
		isOnline:      false,
		logger:        log,
		stopCh:        make(chan struct{}),
		configVersion: -1,
	}

	a.heartbeat = heartbeat.NewHeartbeatService(cfg)
	a.heartbeat.SetClientVersion(Version())
	a.heartbeat.SetMetricsProvider(a)
	a.heartbeat.SetMetricsCollector(a.metrics)
	a.heartbeat.SetConfigUpdateHandler(a.handleHeartbeatConfigUpdate)

	if a.configCache.Exists() {
		cachedVersion := a.configCache.GetVersion()
		a.setConfigVersion(cachedVersion)
		a.heartbeat.SetCurrentConfigVersion(cachedVersion)
	} else {
		a.heartbeat.SetCurrentConfigVersion(-1)
	}

	return a
}

// Start 启动 Agent 并阻塞等待退出信号。
func (a *Agent) Start() error {
	if a == nil || a.config == nil {
		return fmt.Errorf("agent 配置不能为空")
	}

	var cachedConfig *config.NodeConfig
	if a.configCache.Exists() {
		cfg, err := a.configCache.Load()
		if err != nil {
			return fmt.Errorf("读取本地缓存配置失败: %w", err)
		}
		cachedConfig = cfg
		a.setConfigVersion(cfg.ConfigVersion)
		a.heartbeat.SetCurrentConfigVersion(cfg.ConfigVersion)
	} else {
		a.setConfigVersion(-1)
		a.heartbeat.SetCurrentConfigVersion(-1)
	}

	if err := a.heartbeat.Report(); err != nil {
		a.logger.Warn("无法连接面板", "error", err)
		if cachedConfig == nil {
			return fmt.Errorf("无缓存配置，无法启动")
		}
		if applyErr := a.applyConfig(cachedConfig); applyErr != nil {
			return fmt.Errorf("应用缓存配置失败: %w", applyErr)
		}
		a.setOnline(false)
		a.logger.Warn("离线模式: 使用本地缓存配置继续运行", "version", cachedConfig.ConfigVersion)
	} else {
		a.setOnline(true)
		current := a.getCurrentConfig()
		if current == nil {
			if cachedConfig != nil {
				if applyErr := a.applyConfig(cachedConfig); applyErr != nil {
					return fmt.Errorf("应用缓存配置失败: %w", applyErr)
				}
				a.logger.Info("在线模式: 面板可用，使用缓存配置启动", "version", cachedConfig.ConfigVersion)
			} else {
				bootstrap := &config.NodeConfig{
					ConfigVersion: 0,
					Rules:         []config.RuleConfig{},
					Settings: config.Settings{
						HeartbeatInterval:   a.config.HeartbeatInterval,
						ConfigCheckInterval: a.config.ConfigCheckInterval,
					},
				}
				if applyErr := a.applyConfig(bootstrap); applyErr != nil {
					return fmt.Errorf("应用启动空配置失败: %w", applyErr)
				}
				if saveErr := a.configCache.Save(bootstrap); saveErr != nil {
					a.logger.Warn("保存启动空配置失败", "error", saveErr)
				}
				a.logger.Info("在线模式: 启动空配置", "version", bootstrap.ConfigVersion)
			}
		} else {
			a.logger.Info("在线模式: 从面板获取配置成功", "version", current.ConfigVersion)
		}
	}

	a.waitGroup.Add(3)
	go func() {
		defer a.waitGroup.Done()
		a.heartbeat.Start()
	}()
	go func() {
		defer a.waitGroup.Done()
		a.watchHeartbeatState()
	}()
	go func() {
		defer a.waitGroup.Done()
		if err := a.metrics.Start(":9100"); err != nil {
			a.logger.Warn("启动 Prometheus metrics 服务失败", "error", err)
		}
	}()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signalCh)

	select {
	case sig := <-signalCh:
		a.logger.Info("收到退出信号", "signal", sig.String())
	case <-a.stopCh:
		a.logger.Info("收到停止信号")
	}

	a.Shutdown()
	return nil
}

func (a *Agent) watchHeartbeatState() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-a.stopCh:
			return
		case <-ticker.C:
			a.setOnline(a.heartbeat.IsOnline())
			a.updateMetrics()
		}
	}
}

func (a *Agent) handleHeartbeatConfigUpdate(cfg *config.NodeConfig, version int) {
	if cfg == nil {
		a.logger.Warn("心跳返回配置更新，但未携带配置内容")
		return
	}

	next := cloneNodeConfig(cfg)
	if next == nil {
		a.logger.Warn("配置更新内容为空，已忽略")
		return
	}
	if version >= 0 {
		next.ConfigVersion = version
	}

	prevVersion := a.getConfigVersion()
	if err := a.applyConfig(next); err != nil {
		a.logger.Warn("应用新配置失败", "error", err)
		return
	}
	if err := a.configCache.Save(next); err != nil {
		a.logger.Warn("保存配置缓存失败", "error", err)
	}
	a.heartbeat.SetCurrentConfigVersion(next.ConfigVersion)

	if next.ConfigVersion != prevVersion {
		a.logger.Info("配置已更新", "from", prevVersion, "to", next.ConfigVersion)
	}
}

func (a *Agent) applyConfig(nextConfig *config.NodeConfig) error {
	if nextConfig == nil {
		return fmt.Errorf("配置不能为空")
	}
	a.applyMu.Lock()
	defer a.applyMu.Unlock()

	currentRules := a.nodePass.SnapshotRules()
	nextRules := make(map[int]config.RuleConfig, len(nextConfig.Rules))
	for _, rule := range nextConfig.Rules {
		nextRules[rule.RuleID] = rule
	}

	for ruleID := range currentRules {
		if _, exists := nextRules[ruleID]; exists {
			continue
		}
		if err := a.nodePass.StopRule(ruleID); err != nil {
			return fmt.Errorf("停止规则 %d 失败: %w", ruleID, err)
		}
	}

	for ruleID, nextRule := range nextRules {
		currentRule, exists := currentRules[ruleID]
		if !exists {
			if err := a.nodePass.StartRule(nextRule); err != nil {
				return fmt.Errorf("启动规则 %d 失败: %w", ruleID, err)
			}
			continue
		}

		if nodepass.IsSameRuleConfig(currentRule, nextRule) {
			continue
		}
		if err := a.nodePass.RestartRule(ruleID, nextRule); err != nil {
			return fmt.Errorf("重启规则 %d 失败: %w", ruleID, err)
		}
	}

	a.setConfigVersion(nextConfig.ConfigVersion)
	a.mu.Lock()
	a.currentConfig = cloneNodeConfig(nextConfig)
	a.mu.Unlock()
	return nil
}

// SnapshotHeartbeatMetrics 返回心跳上报需要的运行统计。
func (a *Agent) SnapshotHeartbeatMetrics() (trafficIn int64, trafficOut int64, activeConnections int64) {
	stats := a.nodePass.GetTrafficStats()
	for _, item := range stats {
		trafficIn += item.TrafficIn
		trafficOut += item.TrafficOut
	}

	ruleStatus := a.nodePass.GetAllStatus()
	for _, item := range ruleStatus {
		if item.Status == nodepass.StatusRunning {
			activeConnections++
		}
	}
	return trafficIn, trafficOut, activeConnections
}

// Shutdown 停止所有服务并释放资源。
func (a *Agent) Shutdown() {
	a.stopOnce.Do(func() {
		close(a.stopCh)

		a.heartbeat.Stop()
		a.metrics.Stop()
		a.saveFinalState()
		a.nodePass.StopAll()

		a.waitGroup.Wait()

		if a.logger != nil {
			if err := a.logger.Close(); err != nil {
				a.logger.Warn("关闭日志器失败", "error", err)
			}
		}

		if a.logger != nil {
			a.logger.Info("Agent 已停止")
		}
	})
}

func (a *Agent) setOnline(online bool) {
	a.mu.Lock()
	prev := a.isOnline
	a.isOnline = online
	a.mu.Unlock()

	if prev == online {
		return
	}
	if online {
		a.logger.Info("运行模式切换: OFFLINE -> ONLINE")
	} else {
		a.logger.Info("运行模式切换: ONLINE -> OFFLINE")
	}
}

func (a *Agent) setConfigVersion(version int) {
	a.mu.Lock()
	a.configVersion = version
	a.mu.Unlock()
}

func (a *Agent) getConfigVersion() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.configVersion
}

func (a *Agent) getCurrentConfig() *config.NodeConfig {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return cloneNodeConfig(a.currentConfig)
}

func newLogger(cfg *config.Config) *logger.Logger {
	logLevel := "info"
	if cfg.DebugMode || cfg.Debug {
		logLevel = "debug"
	}

	logPath := ""
	if cfg.LogPath != "" {
		logPath = filepath.Join(cfg.LogPath, "nodeclient.log")
	}

	log, err := logger.New(logger.Config{
		Level:      logLevel,
		OutputPath: logPath,
		Prefix:     "nodeclient",
	})
	if err != nil {
		// 降级到标准输出
		log, _ = logger.New(logger.Config{
			Level:  logLevel,
			Prefix: "nodeclient",
		})
	}

	return log
}

func cloneNodeConfig(cfg *config.NodeConfig) *config.NodeConfig {
	if cfg == nil {
		return nil
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		return cfg
	}
	result := &config.NodeConfig{}
	if err := json.Unmarshal(raw, result); err != nil {
		return cfg
	}
	return result
}

func (a *Agent) saveFinalState() {
	cfg := a.getCurrentConfig()
	version := a.getConfigVersion()
	if cfg == nil {
		a.logger.Warn("无可保存的配置状态，跳过缓存落盘")
		return
	}
	if cfg.ConfigVersion == 0 && version > 0 {
		cfg.ConfigVersion = version
	}

	if err := a.configCache.Save(cfg); err != nil {
		a.logger.Warn("保存最终配置状态失败", "error", err)
		return
	}
	a.logger.Info("已保存最终配置状态到缓存", "version", cfg.ConfigVersion)
}

func (a *Agent) updateMetrics() {
	if a == nil || a.metrics == nil {
		return
	}

	// 更新配置版本
	a.metrics.SetConfigVersion(a.getConfigVersion())

	// 更新规则统计
	ruleStatus := a.nodePass.GetAllStatus()
	var running, stopped, errored int
	for _, status := range ruleStatus {
		switch status.Status {
		case nodepass.StatusRunning:
			running++
		case nodepass.StatusStopped:
			stopped++
		case nodepass.StatusError:
			errored++
		}
	}
	a.metrics.SetRuleStats(len(ruleStatus), running, stopped, errored)

	// 更新流量统计
	trafficStats := a.nodePass.GetTrafficStats()
	for _, stat := range trafficStats {
		ruleID := strconv.Itoa(stat.RuleID)
		a.metrics.SetTrafficStats(ruleID, stat.DeltaIn, stat.DeltaOut)
		// 注意：activeConnections 在 nodepass 中暂未实现，这里设为 0
		a.metrics.SetActiveConnections(ruleID, 0)
	}
}
