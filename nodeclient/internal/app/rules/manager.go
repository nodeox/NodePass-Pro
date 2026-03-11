package rules

import (
	"fmt"
	"sync"

	domainconfig "nodepass-pro/nodeclient/internal/domain/config"
	domainrules "nodepass-pro/nodeclient/internal/domain/rules"
	"nodepass-pro/nodeclient/internal/infra/logger"
	"nodepass-pro/nodeclient/internal/infra/nodepass"
)

// Manager 负责规则生命周期管理和编排。
type Manager interface {
	ApplyConfig(nextConfig *domainconfig.NodeConfig) error
	GetStatus() []domainrules.RuleStatus
	GetTrafficStats() []domainrules.TrafficStat
	SnapshotRules() map[int]domainconfig.RuleConfig
	StopAll()
}

type manager struct {
	nodePass *nodepass.Integration
	logger   *logger.Logger
	applyMu  sync.Mutex
}

// NewManager 创建规则管理器。
func NewManager(nodePass *nodepass.Integration, logger *logger.Logger) Manager {
	return &manager{
		nodePass: nodePass,
		logger:   logger,
	}
}

// ApplyConfig 应用配置：diff 规则并执行启动/停止/重启操作。
func (m *manager) ApplyConfig(nextConfig *domainconfig.NodeConfig) error {
	if nextConfig == nil {
		return fmt.Errorf("配置不能为空")
	}
	m.applyMu.Lock()
	defer m.applyMu.Unlock()

	currentRules := m.nodePass.SnapshotRules()
	nextRules := make(map[int]domainconfig.RuleConfig, len(nextConfig.Rules))
	for _, rule := range nextConfig.Rules {
		nextRules[rule.RuleID] = rule
	}

	// 停止不在新配置中的规则
	for ruleID := range currentRules {
		if _, exists := nextRules[ruleID]; exists {
			continue
		}
		if err := m.nodePass.StopRule(ruleID); err != nil {
			return fmt.Errorf("停止规则 %d 失败: %w", ruleID, err)
		}
	}

	// 启动新规则或重启变更的规则
	for ruleID, nextRule := range nextRules {
		currentRule, exists := currentRules[ruleID]
		if !exists {
			if err := m.nodePass.StartRule(nextRule); err != nil {
				return fmt.Errorf("启动规则 %d 失败: %w", ruleID, err)
			}
			continue
		}

		if nodepass.IsSameRuleConfig(currentRule, nextRule) {
			continue
		}
		if err := m.nodePass.RestartRule(ruleID, nextRule); err != nil {
			return fmt.Errorf("重启规则 %d 失败: %w", ruleID, err)
		}
	}

	return nil
}

// GetStatus 返回所有规则的状态。
func (m *manager) GetStatus() []domainrules.RuleStatus {
	return m.nodePass.GetAllStatus()
}

// GetTrafficStats 返回所有规则的流量统计。
func (m *manager) GetTrafficStats() []domainrules.TrafficStat {
	return m.nodePass.GetTrafficStats()
}

// SnapshotRules 返回当前运行的规则快照。
func (m *manager) SnapshotRules() map[int]domainconfig.RuleConfig {
	return m.nodePass.SnapshotRules()
}

// StopAll 停止所有规则。
func (m *manager) StopAll() {
	m.nodePass.StopAll()
}
