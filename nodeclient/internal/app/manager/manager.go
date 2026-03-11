package manager

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	appconfig "nodepass-pro/nodeclient/internal/app/config"
	appheartbeat "nodepass-pro/nodeclient/internal/app/heartbeat"
	apprules "nodepass-pro/nodeclient/internal/app/rules"
	domainconfig "nodepass-pro/nodeclient/internal/domain/config"
	"nodepass-pro/nodeclient/internal/infra/config"
	"nodepass-pro/nodeclient/internal/infra/heartbeat"
	"nodepass-pro/nodeclient/internal/infra/logger"
	"nodepass-pro/nodeclient/internal/infra/metrics"
	"nodepass-pro/nodeclient/internal/infra/nodepass"
)

// Manager 负责整体生命周期管理和服务协调。
type Manager interface {
	Start() error
	Shutdown()
	IsOnline() bool
}

type manager struct {
	config               *config.Config
	logger               *logger.Logger
	metrics              *metrics.Collector
	configManager        appconfig.Manager
	rulesManager         apprules.Manager
	heartbeatCoordinator appheartbeat.Coordinator
	isOnline             bool
	mu                   sync.RWMutex
	stopCh               chan struct{}
	stopOnce             sync.Once
	waitGroup            sync.WaitGroup
}

// NewManager 创建整体管理器。
func NewManager(cfg *config.Config, clientVersion string) Manager {
	if cfg == nil {
		cfg = &config.Config{}
	}

	log := newLogger(cfg)
	configCache := config.NewConfigCache(cfg.CachePath)
	nodePassIntegration := nodepass.NewIntegration(log.StdLogger())
	metricsCollector := metrics.NewCollector()

	configMgr := appconfig.NewManager(configCache, log)
	rulesMgr := apprules.NewManager(nodePassIntegration, log)
	hbService := heartbeat.NewHeartbeatService(cfg)
	hbService.SetClientVersion(clientVersion)
	hbService.SetMetricsCollector(metricsCollector)
	hbCoordinator := appheartbeat.NewCoordinator(hbService, log)

	m := &manager{
		config:               cfg,
		logger:               log,
		metrics:              metricsCollector,
		configManager:        configMgr,
		rulesManager:         rulesMgr,
		heartbeatCoordinator: hbCoordinator,
		isOnline:             false,
		stopCh:               make(chan struct{}),
	}

	// 设置心跳回调
	hbCoordinator.SetMetricsProvider(m)
	hbCoordinator.SetConfigUpdateHandler(m.handleConfigUpdate)

	// 初始化配置版本
	if configCache.Exists() {
		cachedVersion := configCache.GetVersion()
		configMgr.SetVersion(cachedVersion)
		hbCoordinator.SetCurrentConfigVersion(cachedVersion)
	} else {
		configMgr.SetVersion(-1)
		hbCoordinator.SetCurrentConfigVersion(-1)
	}

	return m
}

// Start 启动 Manager 并阻塞等待退出信号。
func (m *manager) Start() error {
	if m == nil || m.config == nil {
		return fmt.Errorf("manager 配置不能为空")
	}

	// 加载缓存配置
	var cachedConfig *domainconfig.NodeConfig
	if m.config.CachePath != "" {
		configCache := config.NewConfigCache(m.config.CachePath)
		if configCache.Exists() {
			cfg, err := configCache.Load()
			if err != nil {
				return fmt.Errorf("读取本地缓存配置失败: %w", err)
			}
			cachedConfig = cfg
			m.configManager.SetVersion(cfg.ConfigVersion)
			m.heartbeatCoordinator.SetCurrentConfigVersion(cfg.ConfigVersion)
		}
	}

	// 执行初始心跳上报
	if err := m.heartbeatCoordinator.Report(); err != nil {
		m.logger.Warn("无法连接面板", "error", err)
		if cachedConfig == nil {
			return fmt.Errorf("无缓存配置，无法启动")
		}
		if applyErr := m.applyConfigAndTrackState(cachedConfig, cachedConfig.ConfigVersion); applyErr != nil {
			return fmt.Errorf("应用缓存配置失败: %w", applyErr)
		}
		m.setOnline(false)
		m.logger.Warn("离线模式: 使用本地缓存配置继续运行", "version", cachedConfig.ConfigVersion)
	} else {
		m.setOnline(true)
		current := m.configManager.GetCurrent()
		if current == nil {
			if cachedConfig != nil {
				if applyErr := m.applyConfigAndTrackState(cachedConfig, cachedConfig.ConfigVersion); applyErr != nil {
					return fmt.Errorf("应用缓存配置失败: %w", applyErr)
				}
				m.logger.Info("在线模式: 面板可用，使用缓存配置启动", "version", cachedConfig.ConfigVersion)
			} else {
				bootstrap := &domainconfig.NodeConfig{
					ConfigVersion: 0,
					Rules:         []domainconfig.RuleConfig{},
					Settings: domainconfig.Settings{
						HeartbeatInterval:   m.config.HeartbeatInterval,
						ConfigCheckInterval: m.config.ConfigCheckInterval,
					},
				}
				if applyErr := m.applyConfigAndTrackState(bootstrap, bootstrap.ConfigVersion); applyErr != nil {
					return fmt.Errorf("应用启动空配置失败: %w", applyErr)
				}
				m.logger.Info("在线模式: 启动空配置", "version", bootstrap.ConfigVersion)
			}
		} else {
			m.logger.Info("在线模式: 从面板获取配置成功", "version", current.ConfigVersion)
		}
	}

	// 启动 3 个 goroutine
	m.waitGroup.Add(3)
	go func() {
		defer m.waitGroup.Done()
		m.heartbeatCoordinator.Start()
	}()
	go func() {
		defer m.waitGroup.Done()
		m.watchHeartbeatState()
	}()
	go func() {
		defer m.waitGroup.Done()
		if err := m.metrics.Start(":9100"); err != nil {
			m.logger.Warn("启动 Prometheus metrics 服务失败", "error", err)
		}
	}()

	// 阻塞等待信号
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signalCh)

	select {
	case sig := <-signalCh:
		m.logger.Info("收到退出信号", "signal", sig.String())
	case <-m.stopCh:
		m.logger.Info("收到停止信号")
	}

	m.Shutdown()
	return nil
}

// watchHeartbeatState 监控心跳状态并更新指标。
func (m *manager) watchHeartbeatState() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.setOnline(m.heartbeatCoordinator.IsOnline())
			m.updateMetrics()
		}
	}
}

// handleConfigUpdate 处理心跳返回的配置更新。
func (m *manager) handleConfigUpdate(cfg *domainconfig.NodeConfig, version int) {
	if cfg == nil {
		m.logger.Warn("心跳返回配置更新，但未携带配置内容")
		return
	}

	prevVersion := m.configManager.GetVersion()
	err := m.configManager.HandleUpdate(cfg, version, func(nextConfig *domainconfig.NodeConfig) error {
		return m.rulesManager.ApplyConfig(nextConfig)
	})
	if err != nil {
		m.logger.Warn("应用新配置失败", "error", err)
		return
	}

	m.heartbeatCoordinator.SetCurrentConfigVersion(m.configManager.GetVersion())

	if m.configManager.GetVersion() != prevVersion {
		m.logger.Info("配置已更新", "from", prevVersion, "to", m.configManager.GetVersion())
	}
}

// applyConfigAndTrackState 在应用配置成功后同步配置管理器状态与心跳版本。
func (m *manager) applyConfigAndTrackState(cfg *domainconfig.NodeConfig, version int) error {
	return m.configManager.HandleUpdate(cfg, version, func(nextConfig *domainconfig.NodeConfig) error {
		if err := m.rulesManager.ApplyConfig(nextConfig); err != nil {
			return err
		}
		m.heartbeatCoordinator.SetCurrentConfigVersion(nextConfig.ConfigVersion)
		return nil
	})
}

// SnapshotHeartbeatMetrics 实现 RuntimeMetricsProvider 接口。
func (m *manager) SnapshotHeartbeatMetrics() (trafficIn int64, trafficOut int64, activeConnections int64) {
	stats := m.rulesManager.GetTrafficStats()
	for _, item := range stats {
		trafficIn += item.TrafficIn
		trafficOut += item.TrafficOut
	}

	ruleStatus := m.rulesManager.GetStatus()
	for _, item := range ruleStatus {
		if item.Status == nodepass.StatusRunning {
			activeConnections++
		}
	}
	return trafficIn, trafficOut, activeConnections
}

// Shutdown 停止所有服务并释放资源。
func (m *manager) Shutdown() {
	m.stopOnce.Do(func() {
		close(m.stopCh)

		m.heartbeatCoordinator.Stop()
		m.metrics.Stop()
		m.configManager.SaveFinalState()
		m.rulesManager.StopAll()

		m.waitGroup.Wait()

		if m.logger != nil {
			m.logger.Info("Manager 已停止")
			if err := m.logger.Close(); err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "[nodeclient] 关闭日志器失败: %v\n", err)
			}
		}
	})
}

// IsOnline 返回在线状态。
func (m *manager) IsOnline() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isOnline
}

// setOnline 设置在线状态。
func (m *manager) setOnline(online bool) {
	m.mu.Lock()
	prev := m.isOnline
	m.isOnline = online
	m.mu.Unlock()

	if prev == online {
		return
	}
	if online {
		m.logger.Info("运行模式切换: OFFLINE -> ONLINE")
	} else {
		m.logger.Info("运行模式切换: ONLINE -> OFFLINE")
	}
}

// updateMetrics 更新 Prometheus 指标。
func (m *manager) updateMetrics() {
	if m == nil || m.metrics == nil {
		return
	}

	// 更新配置版本
	m.metrics.SetConfigVersion(m.configManager.GetVersion())

	// 更新规则统计
	ruleStatus := m.rulesManager.GetStatus()
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
	m.metrics.SetRuleStats(len(ruleStatus), running, stopped, errored)

	// 更新流量统计
	trafficStats := m.rulesManager.GetTrafficStats()
	for _, stat := range trafficStats {
		ruleID := strconv.Itoa(stat.RuleID)
		m.metrics.SetTrafficStats(ruleID, stat.DeltaIn, stat.DeltaOut)
		// 注意：activeConnections 在 nodepass 中暂未实现，这里设为 0
		m.metrics.SetActiveConnections(ruleID, 0)
	}
}

// newLogger 创建日志器。
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
