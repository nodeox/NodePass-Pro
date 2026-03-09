package models

import "time"

// HealthCheckType 健康检查类型
type HealthCheckType string

const (
	HealthCheckTypeTCP  HealthCheckType = "tcp"
	HealthCheckTypeHTTP HealthCheckType = "http"
	HealthCheckTypeICMP HealthCheckType = "icmp"
)

// HealthCheckStatus 健康检查状态
type HealthCheckStatus string

const (
	HealthCheckStatusHealthy   HealthCheckStatus = "healthy"
	HealthCheckStatusUnhealthy HealthCheckStatus = "unhealthy"
	HealthCheckStatusUnknown   HealthCheckStatus = "unknown"
)

// NodeHealthCheck 节点健康检查配置
type NodeHealthCheck struct {
	ID              uint              `gorm:"primaryKey;autoIncrement" json:"id"`
	NodeInstanceID  uint              `gorm:"column:node_instance_id;not null;uniqueIndex:uk_node_health_checks_instance;index:idx_node_health_checks_instance" json:"node_instance_id"`
	Type            HealthCheckType   `gorm:"column:type;type:varchar(20);not null;default:tcp" json:"type"`
	Enabled         bool              `gorm:"column:enabled;not null;default:true" json:"enabled"`
	Interval        int               `gorm:"column:interval;not null;default:30" json:"interval"`         // 检查间隔（秒）
	Timeout         int               `gorm:"column:timeout;not null;default:5" json:"timeout"`            // 超时时间（秒）
	Retries         int               `gorm:"column:retries;not null;default:3" json:"retries"`            // 失败重试次数
	SuccessThreshold int              `gorm:"column:success_threshold;not null;default:2" json:"success_threshold"` // 成功阈值
	FailureThreshold int              `gorm:"column:failure_threshold;not null;default:3" json:"failure_threshold"` // 失败阈值
	HTTPPath        *string           `gorm:"column:http_path;type:varchar(255)" json:"http_path"`         // HTTP 检查路径
	HTTPMethod      *string           `gorm:"column:http_method;type:varchar(10)" json:"http_method"`      // HTTP 方法
	ExpectedStatus  *int              `gorm:"column:expected_status" json:"expected_status"`               // 期望的 HTTP 状态码
	CreatedAt       time.Time         `gorm:"column:created_at" json:"created_at"`
	UpdatedAt       time.Time         `gorm:"column:updated_at" json:"updated_at"`

	NodeInstance *NodeInstance `gorm:"foreignKey:NodeInstanceID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"node_instance,omitempty"`
}

// TableName 指定表名
func (NodeHealthCheck) TableName() string {
	return "node_health_checks"
}

// NodeHealthRecord 节点健康检查记录
type NodeHealthRecord struct {
	ID             uint              `gorm:"primaryKey;autoIncrement" json:"id"`
	NodeInstanceID uint              `gorm:"column:node_instance_id;not null;index:idx_node_health_records_instance" json:"node_instance_id"`
	CheckType      HealthCheckType   `gorm:"column:check_type;type:varchar(20);not null" json:"check_type"`
	Status         HealthCheckStatus `gorm:"column:status;type:varchar(20);not null;index:idx_node_health_records_status" json:"status"`
	Latency        *int              `gorm:"column:latency" json:"latency"`                 // 延迟（毫秒）
	ErrorMessage   *string           `gorm:"column:error_message;type:text" json:"error_message"`
	CheckedAt      time.Time         `gorm:"column:checked_at;not null;index:idx_node_health_records_checked_at" json:"checked_at"`

	NodeInstance *NodeInstance `gorm:"foreignKey:NodeInstanceID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"node_instance,omitempty"`
}

// TableName 指定表名
func (NodeHealthRecord) TableName() string {
	return "node_health_records"
}

// NodeQualityScore 节点质量评分
type NodeQualityScore struct {
	ID              uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	NodeInstanceID  uint      `gorm:"column:node_instance_id;not null;uniqueIndex:uk_node_quality_scores_instance;index:idx_node_quality_scores_instance" json:"node_instance_id"`
	LatencyScore    float64   `gorm:"column:latency_score;type:decimal(5,2);not null;default:0" json:"latency_score"`       // 延迟评分 (0-100)
	StabilityScore  float64   `gorm:"column:stability_score;type:decimal(5,2);not null;default:0" json:"stability_score"`   // 稳定性评分 (0-100)
	LoadScore       float64   `gorm:"column:load_score;type:decimal(5,2);not null;default:0" json:"load_score"`             // 负载评分 (0-100)
	OverallScore    float64   `gorm:"column:overall_score;type:decimal(5,2);not null;default:0" json:"overall_score"`       // 综合评分 (0-100)
	AvgLatency      *int      `gorm:"column:avg_latency" json:"avg_latency"`                                                 // 平均延迟（毫秒）
	Uptime          float64   `gorm:"column:uptime;type:decimal(5,2);not null;default:0" json:"uptime"`                     // 可用性 (0-100)
	SuccessRate     float64   `gorm:"column:success_rate;type:decimal(5,2);not null;default:0" json:"success_rate"`         // 成功率 (0-100)
	LastCheckedAt   *time.Time `gorm:"column:last_checked_at" json:"last_checked_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at" json:"updated_at"`

	NodeInstance *NodeInstance `gorm:"foreignKey:NodeInstanceID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"node_instance,omitempty"`
}

// TableName 指定表名
func (NodeQualityScore) TableName() string {
	return "node_quality_scores"
}

// CalculateOverallScore 计算综合评分
func (s *NodeQualityScore) CalculateOverallScore() {
	// 权重：延迟 30%，稳定性 40%，负载 30%
	s.OverallScore = s.LatencyScore*0.3 + s.StabilityScore*0.4 + s.LoadScore*0.3
}
