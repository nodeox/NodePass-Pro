package auditlog

import (
	"time"
)

// AuditLog 审计日志聚合根
type AuditLog struct {
	ID           uint
	UserID       *uint
	Action       string
	ResourceType string
	ResourceID   *uint
	Details      string
	IPAddress    string
	UserAgent    string
	CreatedAt    time.Time
}

// ActionType 操作类型
type ActionType string

const (
	ActionCreate ActionType = "create"
	ActionUpdate ActionType = "update"
	ActionDelete ActionType = "delete"
	ActionLogin  ActionType = "login"
	ActionLogout ActionType = "logout"
	ActionAccess ActionType = "access"
)

// ResourceType 资源类型
type ResourceType string

const (
	ResourceUser         ResourceType = "user"
	ResourceRole         ResourceType = "role"
	ResourceNodeGroup    ResourceType = "node_group"
	ResourceNodeInstance ResourceType = "node_instance"
	ResourceTunnel       ResourceType = "tunnel"
	ResourceBenefitCode  ResourceType = "benefit_code"
	ResourceVIP          ResourceType = "vip"
)

// NewAuditLog 创建审计日志
func NewAuditLog(userID *uint, action, resourceType string, resourceID *uint, details, ipAddress, userAgent string) *AuditLog {
	return &AuditLog{
		UserID:       userID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Details:      details,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		CreatedAt:    time.Now(),
	}
}

// IsUserAction 是否为用户操作
func (a *AuditLog) IsUserAction() bool {
	return a.UserID != nil
}

// IsSystemAction 是否为系统操作
func (a *AuditLog) IsSystemAction() bool {
	return a.UserID == nil
}

// GetActionType 获取操作类型
func (a *AuditLog) GetActionType() ActionType {
	return ActionType(a.Action)
}

// GetResourceType 获取资源类型
func (a *AuditLog) GetResourceType() ResourceType {
	return ResourceType(a.ResourceType)
}
