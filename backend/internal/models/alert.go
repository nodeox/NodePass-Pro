package models

import (
	"time"

	"gorm.io/gorm"
)

// AlertLevel 告警级别
type AlertLevel string

const (
	AlertLevelInfo     AlertLevel = "info"
	AlertLevelWarning  AlertLevel = "warning"
	AlertLevelError    AlertLevel = "error"
	AlertLevelCritical AlertLevel = "critical"
)

// AlertStatus 告警状态
type AlertStatus string

const (
	AlertStatusPending   AlertStatus = "pending"   // 待处理
	AlertStatusFiring    AlertStatus = "firing"    // 触发中
	AlertStatusResolved  AlertStatus = "resolved"  // 已解决
	AlertStatusSilenced  AlertStatus = "silenced"  // 已静默
	AlertStatusAcknowledged AlertStatus = "acknowledged" // 已确认
)

// AlertType 告警类型
type AlertType string

const (
	AlertTypeNodeOffline      AlertType = "node_offline"       // 节点离线
	AlertTypeNodeHighCPU      AlertType = "node_high_cpu"      // CPU 使用率过高
	AlertTypeNodeHighMemory   AlertType = "node_high_memory"   // 内存使用率过高
	AlertTypeNodeHighDisk     AlertType = "node_high_disk"     // 磁盘使用率过高
	AlertTypeTrafficQuota     AlertType = "traffic_quota"      // 流量配额告警
	AlertTypeHighLatency      AlertType = "high_latency"       // 高延迟
	AlertTypeHighPacketLoss   AlertType = "high_packet_loss"   // 高丢包率
	AlertTypeSystemError      AlertType = "system_error"       // 系统错误
)

// Alert 告警记录
type Alert struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// 告警基本信息
	Type        AlertType   `gorm:"type:varchar(50);not null;index" json:"type"`
	Level       AlertLevel  `gorm:"type:varchar(20);not null;index" json:"level"`
	Status      AlertStatus `gorm:"type:varchar(20);not null;index;default:'pending'" json:"status"`
	Title       string      `gorm:"type:varchar(255);not null" json:"title"`
	Message     string      `gorm:"type:text" json:"message"`
	Fingerprint string      `gorm:"type:varchar(64);uniqueIndex" json:"fingerprint"` // 用于去重

	// 关联资源
	ResourceType string `gorm:"type:varchar(50);index" json:"resource_type"` // node_instance, node_group, user, system
	ResourceID   uint   `gorm:"index" json:"resource_id"`
	ResourceName string `gorm:"type:varchar(255)" json:"resource_name"`

	// 告警详情
	Labels      string `gorm:"type:json" json:"labels"`       // JSON 格式的标签
	Annotations string `gorm:"type:json" json:"annotations"`  // JSON 格式的注解
	Value       string `gorm:"type:varchar(255)" json:"value"` // 触发值
	Threshold   string `gorm:"type:varchar(255)" json:"threshold"` // 阈值

	// 时间信息
	FirstFiredAt  time.Time  `json:"first_fired_at"`  // 首次触发时间
	LastFiredAt   time.Time  `json:"last_fired_at"`   // 最后触发时间
	ResolvedAt    *time.Time `json:"resolved_at"`     // 解决时间
	AcknowledgedAt *time.Time `json:"acknowledged_at"` // 确认时间
	SilencedUntil *time.Time `json:"silenced_until"`  // 静默到期时间

	// 通知信息
	NotificationSent bool      `gorm:"default:false" json:"notification_sent"` // 是否已发送通知
	NotificationCount int      `gorm:"default:0" json:"notification_count"`    // 通知次数
	LastNotifiedAt   *time.Time `json:"last_notified_at"`                      // 最后通知时间

	// 处理信息
	AcknowledgedBy uint   `json:"acknowledged_by"` // 确认人 ID
	ResolvedBy     uint   `json:"resolved_by"`     // 解决人 ID
	Notes          string `gorm:"type:text" json:"notes"` // 处理备注
}

// TableName 指定表名
func (Alert) TableName() string {
	return "alerts"
}

// IsFiring 是否正在触发
func (a *Alert) IsFiring() bool {
	return a.Status == AlertStatusFiring
}

// IsResolved 是否已解决
func (a *Alert) IsResolved() bool {
	return a.Status == AlertStatusResolved
}

// IsSilenced 是否已静默
func (a *Alert) IsSilenced() bool {
	if a.Status == AlertStatusSilenced && a.SilencedUntil != nil {
		return time.Now().Before(*a.SilencedUntil)
	}
	return false
}

// AlertRule 告警规则
type AlertRule struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// 规则基本信息
	Name        string     `gorm:"type:varchar(255);not null;uniqueIndex" json:"name"`
	Description string     `gorm:"type:text" json:"description"`
	Type        AlertType  `gorm:"type:varchar(50);not null;index" json:"type"`
	Level       AlertLevel `gorm:"type:varchar(20);not null" json:"level"`
	IsEnabled   bool       `gorm:"default:true;index" json:"is_enabled"`

	// 规则条件
	Condition   string `gorm:"type:text;not null" json:"condition"`   // 条件表达式
	Threshold   string `gorm:"type:varchar(255)" json:"threshold"`    // 阈值
	Duration    int    `gorm:"default:60" json:"duration"`            // 持续时间（秒）
	EvalInterval int   `gorm:"default:60" json:"eval_interval"`       // 评估间隔（秒）

	// 通知配置
	NotifyChannels string `gorm:"type:json" json:"notify_channels"` // 通知渠道 JSON 数组
	NotifyInterval int    `gorm:"default:300" json:"notify_interval"` // 通知间隔（秒）
	MaxNotifications int  `gorm:"default:10" json:"max_notifications"` // 最大通知次数

	// 静默配置
	SilenceHours string `gorm:"type:varchar(255)" json:"silence_hours"` // 静默时间段，如 "22:00-08:00"

	// 标签和注解
	Labels      string `gorm:"type:json" json:"labels"`
	Annotations string `gorm:"type:json" json:"annotations"`

	// 统计信息
	FiredCount    int       `gorm:"default:0" json:"fired_count"`     // 触发次数
	LastFiredAt   *time.Time `json:"last_fired_at"`                   // 最后触发时间
	LastEvaluatedAt *time.Time `json:"last_evaluated_at"`             // 最后评估时间
}

// TableName 指定表名
func (AlertRule) TableName() string {
	return "alert_rules"
}

// NotificationChannel 通知渠道配置
type NotificationChannel struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// 渠道基本信息
	Name        string `gorm:"type:varchar(255);not null;uniqueIndex" json:"name"`
	Type        string `gorm:"type:varchar(50);not null;index" json:"type"` // email, telegram, webhook, slack
	Description string `gorm:"type:text" json:"description"`
	IsEnabled   bool   `gorm:"default:true;index" json:"is_enabled"`

	// 渠道配置
	Config string `gorm:"type:json;not null" json:"config"` // JSON 格式的配置

	// 统计信息
	SentCount      int       `gorm:"default:0" json:"sent_count"`
	FailedCount    int       `gorm:"default:0" json:"failed_count"`
	LastSentAt     *time.Time `json:"last_sent_at"`
	LastFailedAt   *time.Time `json:"last_failed_at"`
	LastError      string    `gorm:"type:text" json:"last_error"`
}

// TableName 指定表名
func (NotificationChannel) TableName() string {
	return "notification_channels"
}
