package nodepass

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"syscall"
	"time"

	"nodepass-pro/nodeclient/internal/config"
)

const (
	// StatusRunning 表示规则实例运行中。
	StatusRunning = "running"
	// StatusStopped 表示规则实例已停止。
	StatusStopped = "stopped"
	// StatusError 表示规则实例异常退出。
	StatusError = "error"

	maxAutoRestartAttempts = 10

	gracefulStopTimeout = 8 * time.Second
	forcedStopTimeout   = 2 * time.Second
)

// RuleStatus 表示单条规则实例状态。
type RuleStatus struct {
	RuleID int    `json:"rule_id"`
	Mode   string `json:"mode"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// TrafficStat 表示单条规则流量统计。
type TrafficStat struct {
	RuleID     int   `json:"rule_id"`
	TrafficIn  int64 `json:"traffic_in"`
	TrafficOut int64 `json:"traffic_out"`
	DeltaIn    int64 `json:"delta_in"`
	DeltaOut   int64 `json:"delta_out"`
}

// Instance 表示 NodePass 规则实例。
type Instance struct {
	RuleID     int
	Mode       string
	Process    *os.Process
	Status     string
	TrafficIn  int64
	TrafficOut int64

	mu          sync.RWMutex
	cmd         *exec.Cmd
	rule        config.RuleConfig
	doneCh      chan struct{}
	stopCh      chan struct{}
	stopOnce    sync.Once
	doneClosed  bool
	lastErr     error
	stopFlag    bool
	restartCnt  int
	reportedIn  int64
	reportedOut int64
}

// Integration 负责 NodePass 实例生命周期管理。
type Integration struct {
	instances map[int]*Instance
	mu        sync.RWMutex
	logger    *log.Logger
}

// NewIntegration 创建 NodePass 集成管理器。
func NewIntegration(logger *log.Logger) *Integration {
	if logger == nil {
		logger = log.New(os.Stdout, "[nodepass] ", log.LstdFlags)
	}
	return &Integration{
		instances: make(map[int]*Instance),
		logger:    logger,
	}
}

// StartRule 启动指定规则实例。
func (i *Integration) StartRule(rule config.RuleConfig) error {
	if rule.RuleID <= 0 {
		return fmt.Errorf("无效规则 ID: %d", rule.RuleID)
	}

	i.mu.RLock()
	if _, exists := i.instances[rule.RuleID]; exists {
		i.mu.RUnlock()
		return nil
	}
	i.mu.RUnlock()

	cmd, err := buildCommand(rule)
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动规则 %d 进程失败: %w", rule.RuleID, err)
	}

	instance := &Instance{
		RuleID:  rule.RuleID,
		Mode:    rule.Mode,
		Process: cmd.Process,
		Status:  StatusRunning,
		cmd:     cmd,
		rule:    rule,
		doneCh:  make(chan struct{}),
		stopCh:  make(chan struct{}),
	}

	i.mu.Lock()
	i.instances[rule.RuleID] = instance
	i.mu.Unlock()

	i.logger.Printf(
		"[INFO] 规则已启动: rule_id=%d pid=%d mode=%s",
		rule.RuleID,
		cmd.Process.Pid,
		rule.Mode,
	)

	go i.watchProcessExit(instance)
	return nil
}

// StopRule 停止指定规则实例。
func (i *Integration) StopRule(ruleID int) error {
	if ruleID <= 0 {
		return fmt.Errorf("无效规则 ID: %d", ruleID)
	}

	i.mu.RLock()
	instance, exists := i.instances[ruleID]
	i.mu.RUnlock()
	if !exists {
		return nil
	}

	instance.mu.Lock()
	alreadyStopping := instance.stopFlag
	if !instance.stopFlag {
		instance.stopFlag = true
		instance.stopOnce.Do(func() {
			close(instance.stopCh)
		})
	}
	process := instance.Process
	doneCh := instance.doneCh
	instance.mu.Unlock()

	if process != nil {
		if err := process.Signal(syscall.SIGTERM); err != nil && !errors.Is(err, os.ErrProcessDone) {
			i.logger.Printf("[WARN] 发送 SIGTERM 失败, rule_id=%d err=%v", ruleID, err)
		}
	} else if !alreadyStopping {
		i.logger.Printf("[INFO] 规则进程已不存在，等待清理: rule_id=%d", ruleID)
	}

	stopped := false
	select {
	case <-doneCh:
		stopped = true
	case <-time.After(gracefulStopTimeout):
		if process != nil {
			_ = process.Kill()
		}
		select {
		case <-doneCh:
			stopped = true
		case <-time.After(forcedStopTimeout):
		}
	}
	if !stopped {
		return fmt.Errorf("停止规则 %d 超时", ruleID)
	}

	i.removeInstance(instance)

	i.logger.Printf("[INFO] 规则已停止: rule_id=%d", ruleID)
	return nil
}

// RestartRule 重启指定规则实例。
func (i *Integration) RestartRule(ruleID int, rule config.RuleConfig) error {
	if err := i.StopRule(ruleID); err != nil {
		return err
	}
	return i.StartRule(rule)
}

// RestartRuleWithRollback 重启规则，失败时回滚到旧配置。
func (i *Integration) RestartRuleWithRollback(ruleID int, oldRule config.RuleConfig, newRule config.RuleConfig) error {
	if err := i.StopRule(ruleID); err != nil {
		return err
	}
	if err := i.StartRule(newRule); err != nil {
		rollbackErr := i.StartRule(oldRule)
		if rollbackErr != nil {
			return fmt.Errorf("重启规则 %d 失败: %v；回滚旧配置失败: %w", ruleID, err, rollbackErr)
		}
		return fmt.Errorf("重启规则 %d 失败: %w（已回滚旧配置）", ruleID, err)
	}
	return nil
}

// GetStatus 查询单条规则状态。
func (i *Integration) GetStatus(ruleID int) (string, error) {
	i.mu.RLock()
	instance, exists := i.instances[ruleID]
	i.mu.RUnlock()
	if !exists {
		return "", fmt.Errorf("规则实例不存在: %d", ruleID)
	}

	instance.mu.RLock()
	defer instance.mu.RUnlock()
	return instance.Status, nil
}

// GetAllStatus 返回所有规则实例状态。
func (i *Integration) GetAllStatus() []RuleStatus {
	i.mu.RLock()
	defer i.mu.RUnlock()

	result := make([]RuleStatus, 0, len(i.instances))
	for ruleID, instance := range i.instances {
		instance.mu.RLock()
		item := RuleStatus{
			RuleID: ruleID,
			Mode:   instance.Mode,
			Status: instance.Status,
		}
		if instance.lastErr != nil {
			item.Error = instance.lastErr.Error()
		}
		instance.mu.RUnlock()
		result = append(result, item)
	}
	return result
}

// GetTrafficStats 返回所有规则实例流量统计。
func (i *Integration) GetTrafficStats() []TrafficStat {
	i.mu.RLock()
	defer i.mu.RUnlock()

	stats := make([]TrafficStat, 0, len(i.instances))
	for ruleID, instance := range i.instances {
		instance.mu.Lock()
		i.refreshTrafficCounters(instance)

		deltaIn := instance.TrafficIn - instance.reportedIn
		deltaOut := instance.TrafficOut - instance.reportedOut
		if deltaIn < 0 {
			deltaIn = 0
		}
		if deltaOut < 0 {
			deltaOut = 0
		}

		stats = append(stats, TrafficStat{
			RuleID:     ruleID,
			TrafficIn:  instance.TrafficIn,
			TrafficOut: instance.TrafficOut,
			DeltaIn:    deltaIn,
			DeltaOut:   deltaOut,
		})
		instance.mu.Unlock()
	}
	return stats
}

// MarkTrafficReported 标记流量统计已成功上报。
func (i *Integration) MarkTrafficReported(reported []TrafficStat) {
	if len(reported) == 0 {
		return
	}

	i.mu.RLock()
	defer i.mu.RUnlock()

	for _, item := range reported {
		instance, exists := i.instances[item.RuleID]
		if !exists {
			continue
		}

		instance.mu.Lock()
		instance.reportedIn = item.TrafficIn
		instance.reportedOut = item.TrafficOut
		instance.mu.Unlock()
	}
}

// StopAll 停止所有实例。
func (i *Integration) StopAll() {
	i.mu.RLock()
	ids := make([]int, 0, len(i.instances))
	for ruleID := range i.instances {
		ids = append(ids, ruleID)
	}
	i.mu.RUnlock()

	for _, ruleID := range ids {
		if err := i.StopRule(ruleID); err != nil {
			i.logger.Printf("[WARN] 停止规则失败: rule_id=%d err=%v", ruleID, err)
		}
	}
}

// SnapshotRules 返回当前运行规则配置快照。
func (i *Integration) SnapshotRules() map[int]config.RuleConfig {
	i.mu.RLock()
	defer i.mu.RUnlock()

	result := make(map[int]config.RuleConfig, len(i.instances))
	for ruleID, instance := range i.instances {
		instance.mu.RLock()
		result[ruleID] = instance.rule
		instance.mu.RUnlock()
	}
	return result
}

func (i *Integration) watchProcessExit(instance *Instance) {
	for {
		instance.mu.RLock()
		cmd := instance.cmd
		instance.mu.RUnlock()

		if cmd != nil {
			waitErr := cmd.Wait()

			shouldStop := false
			attempt := 0
			instance.mu.Lock()
			instance.Process = nil
			instance.cmd = nil
			if instance.stopFlag {
				instance.Status = StatusStopped
				instance.lastErr = nil
				shouldStop = true
			} else {
				instance.lastErr = waitErr
				if waitErr != nil {
					instance.Status = StatusError
					i.logger.Printf("[WARN] 规则实例异常退出: rule_id=%d err=%v", instance.RuleID, waitErr)
				} else {
					instance.Status = StatusStopped
					i.logger.Printf("[WARN] 规则实例退出: rule_id=%d", instance.RuleID)
				}
				instance.restartCnt++
				attempt = instance.restartCnt
			}
			instance.mu.Unlock()

			if shouldStop {
				i.closeDoneOnce(instance)
				i.removeInstance(instance)
				return
			}
			if !i.restartInstance(instance, attempt) {
				return
			}
			continue
		}

		if i.shouldStopInstance(instance) {
			i.markInstanceStopped(instance)
			i.closeDoneOnce(instance)
			i.removeInstance(instance)
			return
		}

		instance.mu.Lock()
		instance.restartCnt++
		attempt := instance.restartCnt
		instance.mu.Unlock()

		if !i.restartInstance(instance, attempt) {
			return
		}
	}
}

func (i *Integration) restartInstance(instance *Instance, attempt int) bool {
	if attempt <= 0 {
		attempt = 1
	}

	delay := restartBackoff(attempt)
	i.logger.Printf("[INFO] 尝试自动恢复规则: rule_id=%d attempt=%d delay=%s", instance.RuleID, attempt, delay)

	instance.mu.RLock()
	stopCh := instance.stopCh
	rule := instance.rule
	instance.mu.RUnlock()

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-stopCh:
		i.markInstanceStopped(instance)
		i.closeDoneOnce(instance)
		i.removeInstance(instance)
		return false
	case <-timer.C:
	}

	if i.shouldStopInstance(instance) {
		i.markInstanceStopped(instance)
		i.closeDoneOnce(instance)
		i.removeInstance(instance)
		return false
	}

	cmd, err := buildCommand(rule)
	if err != nil {
		return i.handleRestartFailure(instance, attempt, fmt.Errorf("构建规则命令失败: %w", err))
	}

	if err := cmd.Start(); err != nil {
		return i.handleRestartFailure(instance, attempt, fmt.Errorf("启动规则进程失败: %w", err))
	}

	instance.mu.Lock()
	if instance.stopFlag {
		instance.mu.Unlock()
		_ = cmd.Process.Signal(syscall.SIGTERM)
		_ = cmd.Process.Kill()
		i.markInstanceStopped(instance)
		i.closeDoneOnce(instance)
		i.removeInstance(instance)
		return false
	}
	instance.Process = cmd.Process
	instance.cmd = cmd
	instance.Status = StatusRunning
	instance.lastErr = nil
	instance.mu.Unlock()

	i.logger.Printf("[INFO] 规则自动恢复成功: rule_id=%d pid=%d", instance.RuleID, cmd.Process.Pid)
	return true
}

func (i *Integration) handleRestartFailure(instance *Instance, attempt int, err error) bool {
	instance.mu.Lock()
	instance.Status = StatusError
	instance.lastErr = err
	reachedLimit := attempt >= maxAutoRestartAttempts
	instance.mu.Unlock()

	if reachedLimit {
		i.logger.Printf("[ERROR] 规则自动恢复达到上限，已暂停自动恢复: rule_id=%d attempts=%d err=%v", instance.RuleID, attempt, err)
		i.closeDoneOnce(instance)
		return false
	}

	i.logger.Printf("[WARN] 自动恢复规则失败: rule_id=%d attempt=%d err=%v", instance.RuleID, attempt, err)
	return true
}

func (i *Integration) shouldStopInstance(instance *Instance) bool {
	instance.mu.RLock()
	defer instance.mu.RUnlock()
	return instance.stopFlag
}

func (i *Integration) markInstanceStopped(instance *Instance) {
	instance.mu.Lock()
	instance.Process = nil
	instance.cmd = nil
	instance.Status = StatusStopped
	instance.lastErr = nil
	instance.mu.Unlock()
}

func (i *Integration) closeDoneOnce(instance *Instance) {
	instance.mu.Lock()
	defer instance.mu.Unlock()
	if instance.doneClosed {
		return
	}
	close(instance.doneCh)
	instance.doneClosed = true
}

func (i *Integration) removeInstance(instance *Instance) {
	if instance == nil {
		return
	}

	i.mu.Lock()
	defer i.mu.Unlock()
	current, exists := i.instances[instance.RuleID]
	if !exists || current != instance {
		return
	}
	delete(i.instances, instance.RuleID)
}

func restartBackoff(attempt int) time.Duration {
	if attempt <= 0 {
		return time.Second
	}
	shift := attempt - 1
	if shift > 5 {
		shift = 5
	}
	return time.Duration(1<<shift) * time.Second
}

func (i *Integration) refreshTrafficCounters(instance *Instance) {
	if instance == nil {
		return
	}
	if instance.Process == nil || instance.Status != StatusRunning {
		return
	}

	// 预留: 这里接入 NodePass 原生流量计数器读取逻辑。
	// 当前以实例内累计值为准，确保接口稳定并可平滑升级。
}

func buildCommand(rule config.RuleConfig) (*exec.Cmd, error) {
	if rule.Mode == "" {
		return nil, fmt.Errorf("规则模式不能为空")
	}

	binary := os.Getenv("NODEPASS_BIN")
	if binary == "" {
		binary = "nodepass"
	} else {
		normalized := strings.TrimSpace(binary)
		if normalized == "" {
			return nil, fmt.Errorf("NODEPASS_BIN 不能为空")
		}
		if !filepath.IsAbs(normalized) {
			return nil, fmt.Errorf("NODEPASS_BIN 必须为绝对路径: %s", normalized)
		}
		info, err := os.Stat(normalized)
		if err != nil {
			return nil, fmt.Errorf("NODEPASS_BIN 无法访问: %w", err)
		}
		if info.IsDir() {
			return nil, fmt.Errorf("NODEPASS_BIN 不能是目录: %s", normalized)
		}
		if info.Mode()&0o111 == 0 {
			return nil, fmt.Errorf("NODEPASS_BIN 不可执行: %s", normalized)
		}
		binary = normalized
	}

	listenAddr := fmt.Sprintf("%s:%d", rule.Listen.Host, rule.Listen.Port)
	targetAddr := fmt.Sprintf("%s:%d", rule.Target.Host, rule.Target.Port)
	protocol := rule.Protocol
	if protocol == "" {
		protocol = "tcp"
	}

	var args []string
	switch rule.Mode {
	case "single":
		args = []string{
			"server",
			"--listen", listenAddr,
			"--target", targetAddr,
			"--protocol", protocol,
		}
	case "tunnel":
		if rule.ExitNode == nil {
			return nil, fmt.Errorf("隧道模式缺少出口节点配置: rule_id=%d", rule.RuleID)
		}
		exitAddr := fmt.Sprintf("%s:%d", rule.ExitNode.Host, rule.ExitNode.Port)
		args = []string{
			"client",
			"--listen", listenAddr,
			"--exit", exitAddr,
			"--target", targetAddr,
			"--protocol", protocol,
		}
	default:
		return nil, fmt.Errorf("不支持的规则模式: %s", rule.Mode)
	}

	cmd := exec.Command(binary, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd, nil
}

// IsSameRuleConfig 判断两条规则配置是否一致。
func IsSameRuleConfig(left config.RuleConfig, right config.RuleConfig) bool {
	return reflect.DeepEqual(left, right)
}
