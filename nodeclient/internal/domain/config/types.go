package config

// HostPort 表示地址与端口。
type HostPort struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// RuleConfig 表示节点规则配置项。
type RuleConfig struct {
	RuleID   int       `json:"rule_id"`
	Mode     string    `json:"mode"`
	Listen   HostPort  `json:"listen"`
	ExitNode *HostPort `json:"exit_node,omitempty"`
	Target   HostPort  `json:"target"`
	Protocol string    `json:"protocol"`
}

// Settings 表示节点全局设置。
type Settings struct {
	HeartbeatInterval   int `json:"heartbeat_interval"`
	ConfigCheckInterval int `json:"config_check_interval"`
}

// NodeConfig 表示面板下发给节点的完整配置。
type NodeConfig struct {
	ConfigVersion int          `json:"config_version"`
	Rules         []RuleConfig `json:"rules"`
	Settings      Settings     `json:"settings"`
}
