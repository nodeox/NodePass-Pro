package heartbeat

import "nodepass-pro/nodeclient/internal/domain/config"

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
	ClientVersion        string         `json:"client_version,omitempty"`
	NodeRole             string         `json:"node_role,omitempty"`
	CurrentConfigVersion int            `json:"current_config_version"`
	ConnectionAddress    string         `json:"connection_address,omitempty"`
	SystemInfo           SystemInfoData `json:"system_info"`
	TrafficStats         TrafficData    `json:"traffic_stats"`
}

// HeartbeatResponseData 表示心跳返回数据。
type HeartbeatResponseData struct {
	ConfigUpdated    bool `json:"config_updated"`
	NewConfigVersion int  `json:"new_config_version"`
	Config           *config.NodeConfig `json:"config,omitempty"`
}
