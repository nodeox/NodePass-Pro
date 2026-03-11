package interfaces

import (
	"nodepass-pro/nodeclient/internal/domain/config"
	"nodepass-pro/nodeclient/internal/domain/rules"
)

// RuleRunner 定义规则编排接口。
type RuleRunner interface {
	Start(rule config.RuleConfig) error
	Stop(ruleID int) error
	Restart(ruleID int, rule config.RuleConfig) error
	RestartWithRollback(ruleID int, oldRule, newRule config.RuleConfig) error
	SnapshotRules() map[int]config.RuleConfig
	GetAllStatus() []rules.RuleStatus
	GetTrafficStats() []rules.TrafficStat
	MarkTrafficReported([]rules.TrafficStat)
	StopAll()
}
