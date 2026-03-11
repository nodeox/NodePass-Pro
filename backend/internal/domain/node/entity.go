package node

import (
	"errors"
	"time"
)

// NodeInstance 节点实例实体
type NodeInstance struct {
	ID              uint
	GroupID         uint
	NodeID          string
	ServiceName     string
	Status          string
	ConnectionAddr  string
	ExitNetwork     string
	
	// 心跳相关
	LastHeartbeatAt *time.Time
	ConfigVersion   int
	ClientVersion   string
	
	// 系统指标
	CPUUsage        float64
	MemoryUsage     float64
	DiskUsage       float64
	
	// 流量统计
	TrafficIn       int64
	TrafficOut      int64
	ActiveRules     int
	
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// NodeGroup 节点组实体
type NodeGroup struct {
	ID          uint
	Name        string
	Description string
	IsEnabled   bool
	NodeCount   int
	OnlineCount int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// 领域错误
var (
	ErrNodeNotFound      = errors.New("节点不存在")
	ErrNodeGroupNotFound = errors.New("节点组不存在")
	ErrNodeOffline       = errors.New("节点离线")
	ErrNodeDisabled      = errors.New("节点已禁用")
	ErrHeartbeatTimeout  = errors.New("心跳超时")
)

// IsOnline 检查节点是否在线
func (n *NodeInstance) IsOnline() bool {
	if n.LastHeartbeatAt == nil {
		return false
	}
	// 3 分钟内有心跳认为在线
	return time.Since(*n.LastHeartbeatAt) < 3*time.Minute
}

// IsHealthy 检查节点是否健康
func (n *NodeInstance) IsHealthy() bool {
	if !n.IsOnline() {
		return false
	}
	// CPU 使用率 < 90%，内存使用率 < 90%
	return n.CPUUsage < 90.0 && n.MemoryUsage < 90.0
}

// UpdateHeartbeat 更新心跳信息
func (n *NodeInstance) UpdateHeartbeat(cpuUsage, memoryUsage, diskUsage float64, trafficIn, trafficOut int64, activeRules int) {
	now := time.Now()
	n.LastHeartbeatAt = &now
	n.CPUUsage = cpuUsage
	n.MemoryUsage = memoryUsage
	n.DiskUsage = diskUsage
	n.TrafficIn = trafficIn
	n.TrafficOut = trafficOut
	n.ActiveRules = activeRules
	n.Status = "online"
}

// MarkOffline 标记为离线
func (n *NodeInstance) MarkOffline() {
	n.Status = "offline"
}

// UpdateConfig 更新配置版本
func (n *NodeInstance) UpdateConfig(version int) {
	n.ConfigVersion = version
}
