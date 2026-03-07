package agent

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"sync"
	"syscall"
	"time"

	"nodepass-pro/nodeclient/internal/api"
	"nodepass-pro/nodeclient/internal/config"
	"nodepass-pro/nodeclient/internal/heartbeat"
	"nodepass-pro/nodeclient/internal/nodepass"
	"nodepass-pro/nodeclient/internal/traffic"
)

const clientVersion = "0.1.0"

// Agent 定义节点客户端核心控制器。
type Agent struct {
	config      *config.Config
	apiClient   *api.Client
	configCache *config.ConfigCache
	nodePass    *nodepass.Integration
	heartbeat   *heartbeat.Service
	traffic     *traffic.Collector
	isOnline    bool
	nodeID      int
	mu          sync.RWMutex

	logger    *log.Logger
	logFile   *os.File
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

	logger, logFile := newLogger(cfg.LogPath)

	a := &Agent{
		config:        cfg,
		apiClient:     api.NewClient(cfg.HubURL, cfg.NodeToken),
		configCache:   config.NewConfigCache(cfg.CachePath),
		nodePass:      nodepass.NewIntegration(logger),
		isOnline:      false,
		nodeID:        0,
		logger:        logger,
		logFile:       logFile,
		stopCh:        make(chan struct{}),
		configVersion: 0,
	}

	a.configVersion = a.configCache.GetVersion()
	a.heartbeat = heartbeat.NewService(
		a.apiClient,
		a,
		time.Duration(cfg.HeartbeatInterval)*time.Second,
		logger,
	)
	a.traffic = traffic.NewCollector(
		a.apiClient,
		a.nodePass,
		time.Duration(cfg.TrafficReportInterval)*time.Second,
		logger,
	)

	return a
}

// Start 启动 Agent 并阻塞等待退出信号。
func (a *Agent) Start() error {
	if a == nil || a.config == nil {
		return fmt.Errorf("agent 配置不能为空")
	}

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown-node"
	}

	var activeConfig *config.NodeConfig

	registerResp, registerErr := a.apiClient.Register(hostname, clientVersion)
	if registerErr == nil {
		if registerResp == nil || registerResp.Config == nil {
			return fmt.Errorf("面板注册成功但未返回配置")
		}

		a.setOnline(true)
		a.setNodeID(registerResp.NodeID)
		a.setConfigVersion(registerResp.Config.ConfigVersion)

		activeConfig = registerResp.Config
		if err := a.configCache.Save(registerResp.Config); err != nil {
			a.logger.Printf("[WARN] 写入配置缓存失败: %v", err)
		}
		a.logger.Printf("[INFO] 在线模式: 从面板获取配置成功, node_id=%d, version=%d", registerResp.NodeID, registerResp.Config.ConfigVersion)
	} else {
		a.logger.Printf("[WARN] 无法连接面板: %v", registerErr)

		if a.configCache.Exists() {
			cached, cacheErr := a.configCache.Load()
			if cacheErr != nil {
				return fmt.Errorf("读取本地缓存配置失败: %w", cacheErr)
			}
			activeConfig = cached
			a.setOnline(false)
			a.setConfigVersion(cached.ConfigVersion)
			a.logger.Printf("[INFO] 离线模式: 使用本地缓存配置, version=%d", cached.ConfigVersion)
		} else {
			return fmt.Errorf("无缓存配置，无法启动")
		}
	}

	if err := a.applyConfig(activeConfig); err != nil {
		return fmt.Errorf("应用启动配置失败: %w", err)
	}

	a.waitGroup.Add(3)
	go func() {
		defer a.waitGroup.Done()
		a.heartbeat.Start()
	}()
	go func() {
		defer a.waitGroup.Done()
		a.traffic.Start()
	}()
	go func() {
		defer a.waitGroup.Done()
		a.watchConfigUpdates()
	}()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signalCh)

	select {
	case sig := <-signalCh:
		a.logger.Printf("[INFO] 收到退出信号: %s", sig.String())
	case <-a.stopCh:
		a.logger.Printf("[INFO] 收到停止信号")
	}

	a.Shutdown()
	return nil
}

func (a *Agent) applyConfig(nextConfig *config.NodeConfig) error {
	if nextConfig == nil {
		return fmt.Errorf("配置不能为空")
	}

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

func (a *Agent) watchConfigUpdates() {
	interval := time.Duration(a.config.ConfigCheckInterval) * time.Second
	if interval <= 0 {
		a.logger.Printf("[WARN] 配置检查间隔无效，跳过配置监听")
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	a.logger.Printf("[INFO] 配置变更监听已启动，间隔=%s", interval)
	for {
		select {
		case <-a.stopCh:
			a.logger.Printf("[INFO] 配置变更监听已停止")
			return
		case <-ticker.C:
			if !a.getOnline() {
				continue
			}

			nodeID := a.getNodeID()
			if nodeID <= 0 {
				continue
			}

			latestConfig, err := a.apiClient.PullConfig(nodeID)
			if err != nil {
				a.logger.Printf("[WARN] 拉取配置失败: %v", err)
				continue
			}
			if latestConfig == nil {
				continue
			}

			currentVersion := a.getConfigVersion()
			if latestConfig.ConfigVersion <= currentVersion {
				continue
			}

			a.ApplyNewConfig(latestConfig)

			a.logger.Printf("[INFO] 配置已更新: %d -> %d", currentVersion, latestConfig.ConfigVersion)
		}
	}
}

// Shutdown 停止所有服务并释放资源。
func (a *Agent) Shutdown() {
	a.stopOnce.Do(func() {
		close(a.stopCh)

		a.heartbeat.Stop()
		a.traffic.Stop()

		a.nodePass.StopAll()

		a.waitGroup.Wait()

		if a.logFile != nil {
			if err := a.logFile.Close(); err != nil {
				a.logger.Printf("[WARN] 关闭日志文件失败: %v", err)
			}
			a.logFile = nil
		}

		a.logger.Printf("[INFO] Agent 已停止")
	})
}

// GetRulesStatus 返回当前规则实例状态。
func (a *Agent) GetRulesStatus() []api.RuleRuntimeStatus {
	allStatus := a.nodePass.GetAllStatus()
	statuses := make([]api.RuleRuntimeStatus, 0, len(allStatus))

	for _, item := range allStatus {
		statuses = append(statuses, api.RuleRuntimeStatus{
			RuleID: item.RuleID,
			Status: item.Status,
			Error:  item.Error,
		})
	}

	sort.Slice(statuses, func(i int, j int) bool {
		return statuses[i].RuleID < statuses[j].RuleID
	})
	return statuses
}

// GetConfigVersion 返回当前配置版本。
func (a *Agent) GetConfigVersion() int {
	return a.getConfigVersion()
}

// SetOnline 设置在线状态。
func (a *Agent) SetOnline(online bool) {
	a.setOnline(online)
}

// ApplyNewConfig 应用新配置并更新缓存。
func (a *Agent) ApplyNewConfig(cfg *api.NodeConfig) {
	if cfg == nil {
		return
	}

	if err := a.configCache.Save(cfg); err != nil {
		a.logger.Printf("[WARN] 保存配置缓存失败: %v", err)
	}
	if err := a.applyConfig(cfg); err != nil {
		a.logger.Printf("[WARN] 应用新配置失败: %v", err)
	}
}

func (a *Agent) setOnline(online bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.isOnline = online
}

func (a *Agent) getOnline() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.isOnline
}

func (a *Agent) setNodeID(nodeID int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.nodeID = nodeID
}

func (a *Agent) getNodeID() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.nodeID
}

func (a *Agent) setConfigVersion(version int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.configVersion = version
}

func (a *Agent) getConfigVersion() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.configVersion
}

func newLogger(logPath string) (*log.Logger, *os.File) {
	baseWriter := io.Writer(os.Stdout)
	var logFile *os.File

	if logPath != "" {
		if err := os.MkdirAll(logPath, 0o755); err == nil {
			filePath := filepath.Join(logPath, "nodeclient.log")
			file, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
			if err == nil {
				logFile = file
				baseWriter = io.MultiWriter(os.Stdout, file)
			}
		}
	}

	logger := log.New(baseWriter, "[nodeclient] ", log.LstdFlags)
	return logger, logFile
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
