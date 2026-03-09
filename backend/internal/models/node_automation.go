package models

import "time"

// NodeAutomationPolicy 节点自动化策略
type NodeAutomationPolicy struct {
	ID                    uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	NodeGroupID           uint      `gorm:"column:node_group_id;not null;uniqueIndex:uk_node_automation_policies_group;index:idx_node_automation_policies_group" json:"node_group_id"`
	Enabled               bool      `gorm:"column:enabled;not null;default:true" json:"enabled"`
	AutoScalingEnabled    bool      `gorm:"column:auto_scaling_enabled;not null;default:false" json:"auto_scaling_enabled"`
	AutoFailoverEnabled   bool      `gorm:"column:auto_failover_enabled;not null;default:true" json:"auto_failover_enabled"`
	AutoRecoveryEnabled   bool      `gorm:"column:auto_recovery_enabled;not null;default:true" json:"auto_recovery_enabled"`
	MinNodes              int       `gorm:"column:min_nodes;not null;default:1" json:"min_nodes"`                         // 最小节点数
	MaxNodes              int       `gorm:"column:max_nodes;not null;default:10" json:"max_nodes"`                        // 最大节点数
	ScaleUpThreshold      float64   `gorm:"column:scale_up_threshold;type:decimal(5,2);not null;default:80" json:"scale_up_threshold"`     // 扩容阈值（CPU/内存使用率）
	ScaleDownThreshold    float64   `gorm:"column:scale_down_threshold;type:decimal(5,2);not null;default:30" json:"scale_down_threshold"` // 缩容阈值
	ScaleCooldown         int       `gorm:"column:scale_cooldown;not null;default:300" json:"scale_cooldown"`             // 扩缩容冷却时间（秒）
	FailoverTimeout       int       `gorm:"column:failover_timeout;not null;default:180" json:"failover_timeout"`         // 故障转移超时（秒）
	RecoveryCheckInterval int       `gorm:"column:recovery_check_interval;not null;default:60" json:"recovery_check_interval"` // 恢复检查间隔（秒）
	CreatedAt             time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt             time.Time `gorm:"column:updated_at" json:"updated_at"`

	NodeGroup *NodeGroup `gorm:"foreignKey:NodeGroupID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"node_group,omitempty"`
}

// TableName 指定表名
func (NodeAutomationPolicy) TableName() string {
	return "node_automation_policies"
}

// NodeAutomationAction 节点自动化操作记录
type NodeAutomationAction struct {
	ID             uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	NodeGroupID    uint      `gorm:"column:node_group_id;not null;index:idx_node_automation_actions_group" json:"node_group_id"`
	NodeInstanceID *uint     `gorm:"column:node_instance_id;index:idx_node_automation_actions_instance" json:"node_instance_id"`
	ActionType     string    `gorm:"column:action_type;type:varchar(50);not null;index:idx_node_automation_actions_type" json:"action_type"` // scale_up, scale_down, failover, recovery, isolate
	Reason         string    `gorm:"column:reason;type:text;not null" json:"reason"`
	Status         string    `gorm:"column:status;type:varchar(20);not null;default:pending;index:idx_node_automation_actions_status" json:"status"` // pending, success, failed
	ErrorMessage   *string   `gorm:"column:error_message;type:text" json:"error_message"`
	Metadata       string    `gorm:"column:metadata;type:text" json:"metadata"` // JSON 格式的额外信息
	ExecutedAt     time.Time `gorm:"column:executed_at;not null;index:idx_node_automation_actions_executed_at" json:"executed_at"`
	CompletedAt    *time.Time `gorm:"column:completed_at" json:"completed_at"`

	NodeGroup    *NodeGroup    `gorm:"foreignKey:NodeGroupID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"node_group,omitempty"`
	NodeInstance *NodeInstance `gorm:"foreignKey:NodeInstanceID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"node_instance,omitempty"`
}

// TableName 指定表名
func (NodeAutomationAction) TableName() string {
	return "node_automation_actions"
}

// NodeIsolation 节点隔离记录
type NodeIsolation struct {
	ID             uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	NodeInstanceID uint       `gorm:"column:node_instance_id;not null;index:idx_node_isolations_instance" json:"node_instance_id"`
	Reason         string     `gorm:"column:reason;type:text;not null" json:"reason"`
	IsolatedBy     string     `gorm:"column:isolated_by;type:varchar(50);not null" json:"isolated_by"` // auto, manual
	IsolatedAt     time.Time  `gorm:"column:isolated_at;not null;index:idx_node_isolations_isolated_at" json:"isolated_at"`
	RecoveredAt    *time.Time `gorm:"column:recovered_at" json:"recovered_at"`
	IsActive       bool       `gorm:"column:is_active;not null;default:true;index:idx_node_isolations_is_active" json:"is_active"`

	NodeInstance *NodeInstance `gorm:"foreignKey:NodeInstanceID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"node_instance,omitempty"`
}

// TableName 指定表名
func (NodeIsolation) TableName() string {
	return "node_isolations"
}

// NodeOptimizationSuggestion 节点优化建议
type NodeOptimizationSuggestion struct {
	ID             uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	NodeGroupID    *uint     `gorm:"column:node_group_id;index:idx_node_optimization_suggestions_group" json:"node_group_id"`
	NodeInstanceID *uint     `gorm:"column:node_instance_id;index:idx_node_optimization_suggestions_instance" json:"node_instance_id"`
	Category       string    `gorm:"column:category;type:varchar(50);not null;index:idx_node_optimization_suggestions_category" json:"category"` // performance, cost, reliability, security
	Priority       string    `gorm:"column:priority;type:varchar(20);not null;default:medium" json:"priority"` // low, medium, high, critical
	Title          string    `gorm:"column:title;type:varchar(200);not null" json:"title"`
	Description    string    `gorm:"column:description;type:text;not null" json:"description"`
	Impact         string    `gorm:"column:impact;type:text" json:"impact"`           // 预期影响
	Action         string    `gorm:"column:action;type:text" json:"action"`           // 建议操作
	Status         string    `gorm:"column:status;type:varchar(20);not null;default:pending" json:"status"` // pending, applied, dismissed
	AppliedAt      *time.Time `gorm:"column:applied_at" json:"applied_at"`
	DismissedAt    *time.Time `gorm:"column:dismissed_at" json:"dismissed_at"`
	CreatedAt      time.Time `gorm:"column:created_at;index:idx_node_optimization_suggestions_created_at" json:"created_at"`

	NodeGroup    *NodeGroup    `gorm:"foreignKey:NodeGroupID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"node_group,omitempty"`
	NodeInstance *NodeInstance `gorm:"foreignKey:NodeInstanceID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"node_instance,omitempty"`
}

// TableName 指定表名
func (NodeOptimizationSuggestion) TableName() string {
	return "node_optimization_suggestions"
}
