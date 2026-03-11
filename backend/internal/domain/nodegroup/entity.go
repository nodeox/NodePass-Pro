package nodegroup

import (
	"time"
)

// NodeGroupType 节点组类型
type NodeGroupType string

const (
	NodeGroupTypeEntry NodeGroupType = "entry" // 入口组
	NodeGroupTypeExit  NodeGroupType = "exit"  // 出口组
)

// LoadBalanceStrategy 负载均衡策略
type LoadBalanceStrategy string

const (
	LoadBalanceRoundRobin       LoadBalanceStrategy = "round_robin"       // 轮询
	LoadBalanceLeastConnections LoadBalanceStrategy = "least_connections" // 最少连接数
	LoadBalanceRandom           LoadBalanceStrategy = "random"            // 随机
	LoadBalanceFailover         LoadBalanceStrategy = "failover"          // 主备
	LoadBalanceHash             LoadBalanceStrategy = "hash"              // 哈希
	LoadBalanceLatency          LoadBalanceStrategy = "latency"           // 最小延迟
)

// PortRange 端口范围
type PortRange struct {
	Start int
	End   int
}

// EntryGroupConfig 入口组配置
type EntryGroupConfig struct {
	RequireExitGroup  bool
	TrafficMultiplier float64
	DNSLoadBalance    bool
}

// ExitGroupConfig 出口组配置
type ExitGroupConfig struct {
	LoadBalanceStrategy LoadBalanceStrategy
	HealthCheckInterval int // 秒
	HealthCheckTimeout  int // 秒
}

// NodeGroupConfig 节点组配置
type NodeGroupConfig struct {
	AllowedProtocols []string
	PortRange        PortRange
	EntryConfig      *EntryGroupConfig
	ExitConfig       *ExitGroupConfig
}

// NodeGroup 节点组聚合根
type NodeGroup struct {
	ID          uint
	UserID      uint
	Name        string
	Type        NodeGroupType
	Description string
	IsEnabled   bool
	Config      NodeGroupConfig
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// IsEntry 是否是入口组
func (g *NodeGroup) IsEntry() bool {
	return g.Type == NodeGroupTypeEntry
}

// IsExit 是否是出口组
func (g *NodeGroup) IsExit() bool {
	return g.Type == NodeGroupTypeExit
}

// Enable 启用节点组
func (g *NodeGroup) Enable() {
	g.IsEnabled = true
}

// Disable 禁用节点组
func (g *NodeGroup) Disable() {
	g.IsEnabled = false
}

// IsProtocolAllowed 检查协议是否允许
func (g *NodeGroup) IsProtocolAllowed(protocol string) bool {
	if len(g.Config.AllowedProtocols) == 0 {
		return true // 未配置则允许所有
	}
	for _, p := range g.Config.AllowedProtocols {
		if p == protocol {
			return true
		}
	}
	return false
}

// IsPortInRange 检查端口是否在允许范围内
func (g *NodeGroup) IsPortInRange(port int) bool {
	if g.Config.PortRange.Start == 0 && g.Config.PortRange.End == 0 {
		return true // 未配置则允许所有
	}
	return port >= g.Config.PortRange.Start && port <= g.Config.PortRange.End
}

// GetLoadBalanceStrategy 获取负载均衡策略
func (g *NodeGroup) GetLoadBalanceStrategy() LoadBalanceStrategy {
	if g.IsExit() && g.Config.ExitConfig != nil {
		return g.Config.ExitConfig.LoadBalanceStrategy
	}
	return LoadBalanceRoundRobin // 默认轮询
}

// GetTrafficMultiplier 获取流量倍率
func (g *NodeGroup) GetTrafficMultiplier() float64 {
	if g.IsEntry() && g.Config.EntryConfig != nil {
		return g.Config.EntryConfig.TrafficMultiplier
	}
	return 1.0 // 默认 1.0
}

// NodeGroupStats 节点组统计
type NodeGroupStats struct {
	NodeGroupID      uint
	TotalNodes       int
	OnlineNodes      int
	TotalTrafficIn   int64
	TotalTrafficOut  int64
	TotalConnections int
	UpdatedAt        time.Time
}

// GetOnlineRate 获取在线率
func (s *NodeGroupStats) GetOnlineRate() float64 {
	if s.TotalNodes == 0 {
		return 0
	}
	return float64(s.OnlineNodes) / float64(s.TotalNodes)
}

// GetTotalTraffic 获取总流量
func (s *NodeGroupStats) GetTotalTraffic() int64 {
	return s.TotalTrafficIn + s.TotalTrafficOut
}
