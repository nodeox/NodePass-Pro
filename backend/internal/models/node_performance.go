package models

import "time"

// NodePerformanceMetric 节点性能指标
type NodePerformanceMetric struct {
	ID             uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	NodeInstanceID uint      `gorm:"column:node_instance_id;not null;index:idx_node_performance_metrics_instance" json:"node_instance_id"`
	CPUUsage       float64   `gorm:"column:cpu_usage;type:decimal(5,2);not null;default:0" json:"cpu_usage"`           // CPU 使用率 (0-100)
	MemoryUsage    float64   `gorm:"column:memory_usage;type:decimal(5,2);not null;default:0" json:"memory_usage"`     // 内存使用率 (0-100)
	DiskUsage      float64   `gorm:"column:disk_usage;type:decimal(5,2);not null;default:0" json:"disk_usage"`         // 磁盘使用率 (0-100)
	NetworkIn      int64     `gorm:"column:network_in;type:bigint;not null;default:0" json:"network_in"`               // 入站流量 (字节)
	NetworkOut     int64     `gorm:"column:network_out;type:bigint;not null;default:0" json:"network_out"`             // 出站流量 (字节)
	Connections    int       `gorm:"column:connections;not null;default:0" json:"connections"`                          // 连接数
	Latency        *int      `gorm:"column:latency" json:"latency"`                                                     // 延迟 (毫秒)
	PacketLoss     *float64  `gorm:"column:packet_loss;type:decimal(5,2)" json:"packet_loss"`                          // 丢包率 (0-100)
	CollectedAt    time.Time `gorm:"column:collected_at;not null;index:idx_node_performance_metrics_collected_at" json:"collected_at"`

	NodeInstance *NodeInstance `gorm:"foreignKey:NodeInstanceID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"node_instance,omitempty"`
}

// TableName 指定表名
func (NodePerformanceMetric) TableName() string {
	return "node_performance_metrics"
}

// NodePerformanceAlert 节点性能告警配置
type NodePerformanceAlert struct {
	ID              uint    `gorm:"primaryKey;autoIncrement" json:"id"`
	NodeInstanceID  uint    `gorm:"column:node_instance_id;not null;uniqueIndex:uk_node_performance_alerts_instance;index:idx_node_performance_alerts_instance" json:"node_instance_id"`
	Enabled         bool    `gorm:"column:enabled;not null;default:true" json:"enabled"`
	CPUThreshold    float64 `gorm:"column:cpu_threshold;type:decimal(5,2);not null;default:80" json:"cpu_threshold"`       // CPU 告警阈值
	MemoryThreshold float64 `gorm:"column:memory_threshold;type:decimal(5,2);not null;default:80" json:"memory_threshold"` // 内存告警阈值
	DiskThreshold   float64 `gorm:"column:disk_threshold;type:decimal(5,2);not null;default:90" json:"disk_threshold"`     // 磁盘告警阈值
	LatencyThreshold *int   `gorm:"column:latency_threshold" json:"latency_threshold"`                                      // 延迟告警阈值 (毫秒)
	PacketLossThreshold *float64 `gorm:"column:packet_loss_threshold;type:decimal(5,2)" json:"packet_loss_threshold"`      // 丢包率告警阈值
	AlertCooldown   int       `gorm:"column:alert_cooldown;not null;default:300" json:"alert_cooldown"`                    // 告警冷却时间 (秒)
	CreatedAt       time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at" json:"updated_at"`

	NodeInstance *NodeInstance `gorm:"foreignKey:NodeInstanceID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"node_instance,omitempty"`
}

// TableName 指定表名
func (NodePerformanceAlert) TableName() string {
	return "node_performance_alerts"
}

// NodePerformanceAlertRecord 节点性能告警记录
type NodePerformanceAlertRecord struct {
	ID             uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	NodeInstanceID uint      `gorm:"column:node_instance_id;not null;index:idx_node_performance_alert_records_instance" json:"node_instance_id"`
	AlertType      string    `gorm:"column:alert_type;type:varchar(50);not null" json:"alert_type"` // cpu, memory, disk, latency, packet_loss
	Threshold      float64   `gorm:"column:threshold;type:decimal(10,2);not null" json:"threshold"`
	ActualValue    float64   `gorm:"column:actual_value;type:decimal(10,2);not null" json:"actual_value"`
	Message        string    `gorm:"column:message;type:text;not null" json:"message"`
	Resolved       bool      `gorm:"column:resolved;not null;default:false;index:idx_node_performance_alert_records_resolved" json:"resolved"`
	ResolvedAt     *time.Time `gorm:"column:resolved_at" json:"resolved_at"`
	CreatedAt      time.Time `gorm:"column:created_at;index:idx_node_performance_alert_records_created_at" json:"created_at"`

	NodeInstance *NodeInstance `gorm:"foreignKey:NodeInstanceID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"node_instance,omitempty"`
}

// TableName 指定表名
func (NodePerformanceAlertRecord) TableName() string {
	return "node_performance_alert_records"
}

// NodePerformanceSummary 节点性能汇总 (按小时/天聚合)
type NodePerformanceSummary struct {
	ID              uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	NodeInstanceID  uint      `gorm:"column:node_instance_id;not null;index:idx_node_performance_summaries_instance" json:"node_instance_id"`
	Period          string    `gorm:"column:period;type:varchar(20);not null;index:idx_node_performance_summaries_period" json:"period"` // hourly, daily
	PeriodStart     time.Time `gorm:"column:period_start;not null;index:idx_node_performance_summaries_period_start" json:"period_start"`
	AvgCPUUsage     float64   `gorm:"column:avg_cpu_usage;type:decimal(5,2);not null;default:0" json:"avg_cpu_usage"`
	MaxCPUUsage     float64   `gorm:"column:max_cpu_usage;type:decimal(5,2);not null;default:0" json:"max_cpu_usage"`
	AvgMemoryUsage  float64   `gorm:"column:avg_memory_usage;type:decimal(5,2);not null;default:0" json:"avg_memory_usage"`
	MaxMemoryUsage  float64   `gorm:"column:max_memory_usage;type:decimal(5,2);not null;default:0" json:"max_memory_usage"`
	AvgDiskUsage    float64   `gorm:"column:avg_disk_usage;type:decimal(5,2);not null;default:0" json:"avg_disk_usage"`
	MaxDiskUsage    float64   `gorm:"column:max_disk_usage;type:decimal(5,2);not null;default:0" json:"max_disk_usage"`
	TotalNetworkIn  int64     `gorm:"column:total_network_in;type:bigint;not null;default:0" json:"total_network_in"`
	TotalNetworkOut int64     `gorm:"column:total_network_out;type:bigint;not null;default:0" json:"total_network_out"`
	AvgConnections  int       `gorm:"column:avg_connections;not null;default:0" json:"avg_connections"`
	MaxConnections  int       `gorm:"column:max_connections;not null;default:0" json:"max_connections"`
	SampleCount     int       `gorm:"column:sample_count;not null;default:0" json:"sample_count"`
	CreatedAt       time.Time `gorm:"column:created_at" json:"created_at"`

	NodeInstance *NodeInstance `gorm:"foreignKey:NodeInstanceID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"node_instance,omitempty"`
}

// TableName 指定表名
func (NodePerformanceSummary) TableName() string {
	return "node_performance_summaries"
}
