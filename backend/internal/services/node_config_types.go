package services

// NodeConfig 下发给节点客户端的配置结构。
type NodeConfig struct {
	ConfigVersion int          `json:"config_version"`
	Rules         []RuleConfig `json:"rules"`
	Settings      Settings     `json:"settings"`
}

// HostPort 主机与端口。
type HostPort struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// RuleConfig 单条规则配置。
type RuleConfig struct {
	RuleID   int       `json:"rule_id"`
	Mode     string    `json:"mode"`
	Listen   HostPort  `json:"listen"`
	ExitNode *HostPort `json:"exit_node,omitempty"`
	Target   HostPort  `json:"target"`
	Protocol string    `json:"protocol"`
}

// Settings 节点运行配置。
type Settings struct {
	HeartbeatInterval   int `json:"heartbeat_interval"`
	ConfigCheckInterval int `json:"config_check_interval"`
}
