package models

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// NodeGroupType 节点组类型。
type NodeGroupType string

const (
	NodeGroupTypeEntry NodeGroupType = "entry"
	NodeGroupTypeExit  NodeGroupType = "exit"
)

// LoadBalanceStrategy 负载均衡策略。
type LoadBalanceStrategy string

const (
	LoadBalanceRoundRobin       LoadBalanceStrategy = "round_robin"       // 轮询
	LoadBalanceLeastConnections LoadBalanceStrategy = "least_connections" // 最少连接数
	LoadBalanceRandom           LoadBalanceStrategy = "random"            // 随机
	LoadBalanceFailover         LoadBalanceStrategy = "failover"          // 主备
	LoadBalanceHash             LoadBalanceStrategy = "hash"              // 哈希
	LoadBalanceLatency          LoadBalanceStrategy = "latency"           // 最小延迟
)

// NodeInstanceStatus 节点实例状态。
type NodeInstanceStatus string

const (
	NodeInstanceStatusOnline   NodeInstanceStatus = "online"
	NodeInstanceStatusOffline  NodeInstanceStatus = "offline"
	NodeInstanceStatusMaintain NodeInstanceStatus = "maintain"
)

// TunnelStatus 隧道状态。
type TunnelStatus string

const (
	TunnelStatusRunning TunnelStatus = "running"
	TunnelStatusStopped TunnelStatus = "stopped"
	TunnelStatusPaused  TunnelStatus = "paused"
)

// PortRange 端口范围配置。
type PortRange struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

// EntryGroupConfig 入口组扩展配置。
type EntryGroupConfig struct {
	RequireExitGroup  bool    `json:"require_exit_group"`
	TrafficMultiplier float64 `json:"traffic_multiplier"`
	DNSLoadBalance    bool    `json:"dns_load_balance"`
}

// ExitGroupConfig 出口组扩展配置。
type ExitGroupConfig struct {
	LoadBalanceStrategy string `json:"load_balance_strategy"`
	HealthCheckInterval int    `json:"health_check_interval"`
	HealthCheckTimeout  int    `json:"health_check_timeout"`
}

// NodeGroupConfig 节点组配置（JSON）。
type NodeGroupConfig struct {
	AllowedProtocols []string          `json:"allowed_protocols"`
	PortRange        PortRange         `json:"port_range"`
	EntryConfig      *EntryGroupConfig `json:"entry_config,omitempty"`
	ExitConfig       *ExitGroupConfig  `json:"exit_config,omitempty"`
}

// NodeGroup 节点组。
type NodeGroup struct {
	ID          uint          `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID      uint          `gorm:"column:user_id;not null;index:idx_node_groups_user_id" json:"user_id"`
	Name        string        `gorm:"column:name;type:varchar(100);not null" json:"name"`
	Type        NodeGroupType `gorm:"column:type;type:varchar(20);not null;index:idx_node_groups_type" json:"type"`
	Description *string       `gorm:"column:description;type:text" json:"description"`
	IsEnabled   bool          `gorm:"column:is_enabled;not null;default:true;index:idx_node_groups_is_enabled" json:"is_enabled"`
	Config      string        `gorm:"column:config;type:text;not null;default:'{}'" json:"config"`
	CreatedAt   time.Time     `gorm:"column:created_at" json:"created_at"`
	UpdatedAt   time.Time     `gorm:"column:updated_at" json:"updated_at"`

	User          *User           `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"user,omitempty"`
	NodeInstances []NodeInstance  `gorm:"foreignKey:NodeGroupID;references:ID" json:"node_instances,omitempty"`
	Stats         *NodeGroupStats `gorm:"foreignKey:NodeGroupID;references:ID" json:"stats,omitempty"`
}

// TableName 指定表名。
func (NodeGroup) TableName() string {
	return "node_groups"
}

// GetConfig 读取并反序列化节点组配置。
func (g *NodeGroup) GetConfig() (*NodeGroupConfig, error) {
	if g == nil {
		return nil, fmt.Errorf("node group 不能为空")
	}

	raw := strings.TrimSpace(g.Config)
	if raw == "" {
		return &NodeGroupConfig{}, nil
	}

	cfg := &NodeGroupConfig{}
	if err := json.Unmarshal([]byte(raw), cfg); err != nil {
		return nil, fmt.Errorf("解析节点组配置失败: %w", err)
	}
	return cfg, nil
}

// SetConfig 序列化并写入节点组配置。
func (g *NodeGroup) SetConfig(cfg *NodeGroupConfig) error {
	if g == nil {
		return fmt.Errorf("node group 不能为空")
	}
	if cfg == nil {
		g.Config = "{}"
		return nil
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("序列化节点组配置失败: %w", err)
	}
	g.Config = string(data)
	return nil
}

// MarshalJSON 自定义 JSON 序列化，将 Config 字符串解析为对象。
func (g NodeGroup) MarshalJSON() ([]byte, error) {
	type Alias NodeGroup

	// 解析 Config 字符串为对象
	var configObj interface{}
	if g.Config != "" {
		if err := json.Unmarshal([]byte(g.Config), &configObj); err != nil {
			configObj = g.Config // 解析失败时保持原字符串
		}
	} else {
		configObj = map[string]interface{}{}
	}

	return json.Marshal(&struct {
		*Alias
		Config interface{} `json:"config"`
	}{
		Alias:  (*Alias)(&g),
		Config: configObj,
	})
}

// NodeInstance 节点实例。
type NodeInstance struct {
	ID              uint               `gorm:"primaryKey;autoIncrement" json:"id"`
	NodeGroupID     uint               `gorm:"column:node_group_id;not null;index:idx_node_instances_node_group_id" json:"node_group_id"`
	NodeID          string             `gorm:"column:node_id;type:varchar(100);not null;uniqueIndex:uk_node_instances_node_id" json:"node_id"`
	AuthTokenHash   string             `gorm:"column:auth_token_hash;type:varchar(255);index:idx_node_instances_auth_token_hash" json:"-"`
	Name            string             `gorm:"column:name;type:varchar(100);not null" json:"name"`
	Host            *string            `gorm:"column:host;type:varchar(255)" json:"host"`
	Port            *int               `gorm:"column:port" json:"port"`
	Status          NodeInstanceStatus `gorm:"column:status;type:varchar(20);not null;default:offline;index:idx_node_instances_status" json:"status"`
	IsEnabled       bool               `gorm:"column:is_enabled;not null;default:true;index:idx_node_instances_is_enabled" json:"is_enabled"`
	SystemInfo      string             `gorm:"column:system_info;type:text" json:"system_info"`
	TrafficStats    string             `gorm:"column:traffic_stats;type:text" json:"traffic_stats"`
	ConfigVersion   int                `gorm:"column:config_version;not null;default:0" json:"config_version"`
	LastHeartbeatAt *time.Time         `gorm:"column:last_heartbeat_at;index:idx_node_instances_last_heartbeat_at" json:"last_heartbeat_at"`
	CreatedAt       time.Time          `gorm:"column:created_at" json:"created_at"`
	UpdatedAt       time.Time          `gorm:"column:updated_at" json:"updated_at"`

	NodeGroup *NodeGroup `gorm:"foreignKey:NodeGroupID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"node_group,omitempty"`
}

// TableName 指定表名。
func (NodeInstance) TableName() string {
	return "node_instances"
}

// MarshalJSON 自定义 JSON 序列化，将 SystemInfo 和 TrafficStats 字符串解析为对象。
func (n NodeInstance) MarshalJSON() ([]byte, error) {
	type Alias NodeInstance

	// 解析 SystemInfo 字符串为对象
	var systemInfoObj interface{}
	if n.SystemInfo != "" {
		if err := json.Unmarshal([]byte(n.SystemInfo), &systemInfoObj); err != nil {
			systemInfoObj = n.SystemInfo // 解析失败时保持原字符串
		}
	} else {
		systemInfoObj = map[string]interface{}{}
	}

	// 解析 TrafficStats 字符串为对象
	var trafficStatsObj interface{}
	if n.TrafficStats != "" {
		if err := json.Unmarshal([]byte(n.TrafficStats), &trafficStatsObj); err != nil {
			trafficStatsObj = n.TrafficStats // 解析失败时保持原字符串
		}
	} else {
		trafficStatsObj = map[string]interface{}{}
	}

	return json.Marshal(&struct {
		*Alias
		SystemInfo   interface{} `json:"system_info"`
		TrafficStats interface{} `json:"traffic_stats"`
	}{
		Alias:        (*Alias)(&n),
		SystemInfo:   systemInfoObj,
		TrafficStats: trafficStatsObj,
	})
}

// NodeGroupRelation 节点组关联（入口组 <-> 出口组）。
type NodeGroupRelation struct {
	ID           uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	EntryGroupID uint      `gorm:"column:entry_group_id;not null;index:idx_node_group_relations_entry_group_id;uniqueIndex:uk_node_group_relations_entry_exit" json:"entry_group_id"`
	ExitGroupID  uint      `gorm:"column:exit_group_id;not null;index:idx_node_group_relations_exit_group_id;uniqueIndex:uk_node_group_relations_entry_exit" json:"exit_group_id"`
	IsEnabled    bool      `gorm:"column:is_enabled;not null;default:true;index:idx_node_group_relations_is_enabled" json:"is_enabled"`
	CreatedAt    time.Time `gorm:"column:created_at" json:"created_at"`

	EntryGroup *NodeGroup `gorm:"foreignKey:EntryGroupID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"entry_group,omitempty"`
	ExitGroup  *NodeGroup `gorm:"foreignKey:ExitGroupID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"exit_group,omitempty"`
}

// TableName 指定表名。
func (NodeGroupRelation) TableName() string {
	return "node_group_relations"
}

// NodeGroupStats 节点组统计。
type NodeGroupStats struct {
	ID               uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	NodeGroupID      uint      `gorm:"column:node_group_id;not null;uniqueIndex:uk_node_group_stats_group_id;index:idx_node_group_stats_node_group_id" json:"node_group_id"`
	TotalNodes       int       `gorm:"column:total_nodes;not null;default:0" json:"total_nodes"`
	OnlineNodes      int       `gorm:"column:online_nodes;not null;default:0" json:"online_nodes"`
	TotalTrafficIn   int64     `gorm:"column:total_traffic_in;type:bigint;not null;default:0" json:"total_traffic_in"`
	TotalTrafficOut  int64     `gorm:"column:total_traffic_out;type:bigint;not null;default:0" json:"total_traffic_out"`
	TotalConnections int       `gorm:"column:total_connections;not null;default:0" json:"total_connections"`
	UpdatedAt        time.Time `gorm:"column:updated_at;index:idx_node_group_stats_updated_at" json:"updated_at"`
}

// TableName 指定表名。
func (NodeGroupStats) TableName() string {
	return "node_group_stats"
}

// Tunnel 隧道。
type Tunnel struct {
	ID           uint         `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID       uint         `gorm:"column:user_id;not null;index:idx_tunnels_user_id" json:"user_id"`
	Name         string       `gorm:"column:name;type:varchar(100);not null" json:"name"`
	Description  *string      `gorm:"column:description;type:text" json:"description"`
	EntryGroupID uint         `gorm:"column:entry_group_id;not null;index:idx_tunnels_entry_group_id" json:"entry_group_id"`
	ExitGroupID  *uint        `gorm:"column:exit_group_id;index:idx_tunnels_exit_group_id" json:"exit_group_id"`
	Protocol     string       `gorm:"column:protocol;type:varchar(20);not null;index:idx_tunnels_protocol" json:"protocol"`
	ListenHost   string       `gorm:"column:listen_host;type:varchar(255);default:'0.0.0.0'" json:"listen_host"`
	ListenPort   int          `gorm:"column:listen_port" json:"listen_port"`
	RemoteHost   string       `gorm:"column:remote_host;type:varchar(255);not null" json:"remote_host"`
	RemotePort   int          `gorm:"column:remote_port;not null" json:"remote_port"`
	Status       TunnelStatus `gorm:"column:status;type:varchar(20);not null;default:stopped;index:idx_tunnels_status" json:"status"`
	TrafficIn    int64        `gorm:"column:traffic_in;type:bigint;not null;default:0" json:"traffic_in"`
	TrafficOut   int64        `gorm:"column:traffic_out;type:bigint;not null;default:0" json:"traffic_out"`
	ConfigJSON   string       `gorm:"column:config_json;type:text" json:"config_json"`
	CreatedAt    time.Time    `gorm:"column:created_at" json:"created_at"`
	UpdatedAt    time.Time    `gorm:"column:updated_at" json:"updated_at"`

	User       *User      `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"user,omitempty"`
	EntryGroup *NodeGroup `gorm:"foreignKey:EntryGroupID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"entry_group,omitempty"`
	ExitGroup  *NodeGroup `gorm:"foreignKey:ExitGroupID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"exit_group,omitempty"`
}

// TunnelConfig 隧道配置（存储在 config_json 字段）。
type TunnelConfig struct {
	LoadBalanceStrategy LoadBalanceStrategy `json:"load_balance_strategy"`
	IPType              string              `json:"ip_type"` // ipv4, ipv6, auto
	EnableProxyProtocol bool                `json:"enable_proxy_protocol"`
	ForwardTargets      []ForwardTarget     `json:"forward_targets"`
	HealthCheckInterval int                 `json:"health_check_interval"` // 秒
	HealthCheckTimeout  int                 `json:"health_check_timeout"`  // 秒
	ProtocolConfig      *ProtocolConfig     `json:"protocol_config,omitempty"`
}

// ProtocolConfig 协议特定配置。
type ProtocolConfig struct {
	// TCP 配置
	TCPKeepalive      *bool `json:"tcp_keepalive,omitempty"`
	KeepaliveInterval *int  `json:"keepalive_interval,omitempty"` // 秒
	ConnectTimeout    *int  `json:"connect_timeout,omitempty"`    // 秒
	ReadTimeout       *int  `json:"read_timeout,omitempty"`       // 秒

	// UDP 配置
	BufferSize     *int `json:"buffer_size,omitempty"`     // 字节
	SessionTimeout *int `json:"session_timeout,omitempty"` // 秒

	// WebSocket 配置
	WSPath         *string `json:"ws_path,omitempty"`
	PingInterval   *int    `json:"ping_interval,omitempty"`    // 秒
	MaxMessageSize *int    `json:"max_message_size,omitempty"` // KB
	Compression    *bool   `json:"compression,omitempty"`

	// TLS 配置
	TLSVersion *string `json:"tls_version,omitempty"` // tls1.2, tls1.3
	VerifyCert *bool   `json:"verify_cert,omitempty"`
	SNI        *string `json:"sni,omitempty"`

	// QUIC 配置
	MaxStreams    *int  `json:"max_streams,omitempty"`
	InitialWindow *int  `json:"initial_window,omitempty"` // KB
	IdleTimeout   *int  `json:"idle_timeout,omitempty"`   // 秒
	Enable0RTT    *bool `json:"enable_0rtt,omitempty"`
}

// ForwardTarget 转发目标。
type ForwardTarget struct {
	Host   string `json:"host"`
	Port   int    `json:"port"`
	Weight int    `json:"weight"` // 权重，用于随机负载均衡
}

// GetConfig 读取并反序列化隧道配置。
func (t *Tunnel) GetConfig() (*TunnelConfig, error) {
	if t == nil {
		return nil, fmt.Errorf("tunnel 不能为空")
	}

	raw := strings.TrimSpace(t.ConfigJSON)
	if raw == "" || raw == "{}" {
		return &TunnelConfig{
			LoadBalanceStrategy: LoadBalanceRoundRobin,
			IPType:              "auto",
			ForwardTargets:      []ForwardTarget{},
		}, nil
	}

	cfg := &TunnelConfig{}
	if err := json.Unmarshal([]byte(raw), cfg); err != nil {
		return nil, fmt.Errorf("解析隧道配置失败: %w", err)
	}
	return cfg, nil
}

// SetConfig 序列化并写入隧道配置。
func (t *Tunnel) SetConfig(cfg *TunnelConfig) error {
	if t == nil {
		return fmt.Errorf("tunnel 不能为空")
	}
	if cfg == nil {
		t.ConfigJSON = "{}"
		return nil
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("序列化隧道配置失败: %w", err)
	}
	t.ConfigJSON = string(data)
	return nil
}

// TableName 指定表名。
func (Tunnel) TableName() string {
	return "tunnels"
}
