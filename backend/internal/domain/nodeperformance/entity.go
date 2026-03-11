package nodeperformance

import (
	"time"
)

// PerformanceMetric 性能指标聚合根
type PerformanceMetric struct {
	ID             uint
	NodeInstanceID uint
	CPUUsage       float64
	MemoryUsage    float64
	DiskUsage      float64
	NetworkIn      int64
	NetworkOut     int64
	Connections    int
	Latency        *int
	PacketLoss     *float64
	CollectedAt    time.Time
}

// NewPerformanceMetric 创建性能指标
func NewPerformanceMetric(nodeInstanceID uint) *PerformanceMetric {
	return &PerformanceMetric{
		NodeInstanceID: nodeInstanceID,
		CollectedAt:    time.Now(),
	}
}

// PerformanceAlert 性能告警配置聚合根
type PerformanceAlert struct {
	ID                  uint
	NodeInstanceID      uint
	Enabled             bool
	CPUThreshold        float64
	MemoryThreshold     float64
	DiskThreshold       float64
	LatencyThreshold    *int
	PacketLossThreshold *float64
	AlertCooldown       int
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// NewPerformanceAlert 创建性能告警配置
func NewPerformanceAlert(nodeInstanceID uint) *PerformanceAlert {
	return &PerformanceAlert{
		NodeInstanceID:  nodeInstanceID,
		Enabled:         true,
		CPUThreshold:    80.0,
		MemoryThreshold: 80.0,
		DiskThreshold:   90.0,
		AlertCooldown:   300,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
}

// CheckThresholds 检查是否超过阈值
func (a *PerformanceAlert) CheckThresholds(metric *PerformanceMetric) []string {
	var alerts []string

	if !a.Enabled {
		return alerts
	}

	if metric.CPUUsage > a.CPUThreshold {
		alerts = append(alerts, "cpu")
	}
	if metric.MemoryUsage > a.MemoryThreshold {
		alerts = append(alerts, "memory")
	}
	if metric.DiskUsage > a.DiskThreshold {
		alerts = append(alerts, "disk")
	}
	if a.LatencyThreshold != nil && metric.Latency != nil && *metric.Latency > *a.LatencyThreshold {
		alerts = append(alerts, "latency")
	}
	if a.PacketLossThreshold != nil && metric.PacketLoss != nil && *metric.PacketLoss > *a.PacketLossThreshold {
		alerts = append(alerts, "packet_loss")
	}

	return alerts
}

// PerformanceStats 性能统计
type PerformanceStats struct {
	SampleCount      int
	AvgCPUUsage      float64
	MaxCPUUsage      float64
	AvgMemoryUsage   float64
	MaxMemoryUsage   float64
	AvgDiskUsage     float64
	MaxDiskUsage     float64
	TotalNetworkIn   int64
	TotalNetworkOut  int64
	AvgConnections   int
	MaxConnections   int
}

// CalculateStats 计算统计数据
func CalculateStats(metrics []*PerformanceMetric) *PerformanceStats {
	if len(metrics) == 0 {
		return &PerformanceStats{}
	}

	stats := &PerformanceStats{
		SampleCount: len(metrics),
	}

	var totalCPU, totalMemory, totalDisk float64
	var totalConnections int

	for _, m := range metrics {
		totalCPU += m.CPUUsage
		totalMemory += m.MemoryUsage
		totalDisk += m.DiskUsage
		totalConnections += m.Connections
		stats.TotalNetworkIn += m.NetworkIn
		stats.TotalNetworkOut += m.NetworkOut

		if m.CPUUsage > stats.MaxCPUUsage {
			stats.MaxCPUUsage = m.CPUUsage
		}
		if m.MemoryUsage > stats.MaxMemoryUsage {
			stats.MaxMemoryUsage = m.MemoryUsage
		}
		if m.DiskUsage > stats.MaxDiskUsage {
			stats.MaxDiskUsage = m.DiskUsage
		}
		if m.Connections > stats.MaxConnections {
			stats.MaxConnections = m.Connections
		}
	}

	count := float64(len(metrics))
	stats.AvgCPUUsage = totalCPU / count
	stats.AvgMemoryUsage = totalMemory / count
	stats.AvgDiskUsage = totalDisk / count
	stats.AvgConnections = totalConnections / len(metrics)

	return stats
}
