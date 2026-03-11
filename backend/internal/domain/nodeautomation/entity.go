package nodeautomation

import (
	"time"
)

// AutomationPolicy 自动化策略聚合根
type AutomationPolicy struct {
	ID                    uint
	NodeGroupID           uint
	Enabled               bool
	AutoScalingEnabled    bool
	AutoFailoverEnabled   bool
	AutoRecoveryEnabled   bool
	MinNodes              int
	MaxNodes              int
	ScaleUpThreshold      float64
	ScaleDownThreshold    float64
	ScaleCooldown         int
	FailoverTimeout       int
	RecoveryCheckInterval int
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// NewAutomationPolicy 创建自动化策略
func NewAutomationPolicy(nodeGroupID uint) *AutomationPolicy {
	return &AutomationPolicy{
		NodeGroupID:           nodeGroupID,
		Enabled:               true,
		AutoScalingEnabled:    false,
		AutoFailoverEnabled:   true,
		AutoRecoveryEnabled:   true,
		MinNodes:              1,
		MaxNodes:              10,
		ScaleUpThreshold:      80.0,
		ScaleDownThreshold:    30.0,
		ScaleCooldown:         300,
		FailoverTimeout:       180,
		RecoveryCheckInterval: 60,
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}
}

// ActionType 操作类型
type ActionType string

const (
	ActionTypeScaleUp   ActionType = "scale_up"
	ActionTypeScaleDown ActionType = "scale_down"
	ActionTypeFailover  ActionType = "failover"
	ActionTypeRecovery  ActionType = "recovery"
	ActionTypeIsolate   ActionType = "isolate"
)

// ActionStatus 操作状态
type ActionStatus string

const (
	ActionStatusPending ActionStatus = "pending"
	ActionStatusSuccess ActionStatus = "success"
	ActionStatusFailed  ActionStatus = "failed"
)

// AutomationAction 自动化操作记录
type AutomationAction struct {
	ID             uint
	NodeGroupID    uint
	NodeInstanceID *uint
	ActionType     ActionType
	Reason         string
	Status         ActionStatus
	ErrorMessage   *string
	Metadata       string
	ExecutedAt     time.Time
	CompletedAt    *time.Time
}

// NewAutomationAction 创建自动化操作记录
func NewAutomationAction(nodeGroupID uint, actionType ActionType, reason string) *AutomationAction {
	return &AutomationAction{
		NodeGroupID: nodeGroupID,
		ActionType:  actionType,
		Reason:      reason,
		Status:      ActionStatusPending,
		ExecutedAt:  time.Now(),
	}
}

// MarkSuccess 标记成功
func (a *AutomationAction) MarkSuccess() {
	a.Status = ActionStatusSuccess
	now := time.Now()
	a.CompletedAt = &now
}

// MarkFailed 标记失败
func (a *AutomationAction) MarkFailed(errorMessage string) {
	a.Status = ActionStatusFailed
	a.ErrorMessage = &errorMessage
	now := time.Now()
	a.CompletedAt = &now
}

// NodeIsolation 节点隔离记录
type NodeIsolation struct {
	ID             uint
	NodeInstanceID uint
	Reason         string
	IsolatedBy     string
	IsolatedAt     time.Time
	RecoveredAt    *time.Time
	IsActive       bool
}

// NewNodeIsolation 创建节点隔离记录
func NewNodeIsolation(nodeInstanceID uint, reason, isolatedBy string) *NodeIsolation {
	return &NodeIsolation{
		NodeInstanceID: nodeInstanceID,
		Reason:         reason,
		IsolatedBy:     isolatedBy,
		IsolatedAt:     time.Now(),
		IsActive:       true,
	}
}

// Recover 恢复节点
func (n *NodeIsolation) Recover() {
	n.IsActive = false
	now := time.Now()
	n.RecoveredAt = &now
}
