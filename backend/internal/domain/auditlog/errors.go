package auditlog

import "errors"

var (
	// ErrAuditLogNotFound 审计日志不存在
	ErrAuditLogNotFound = errors.New("审计日志不存在")

	// ErrInvalidAction 无效的操作
	ErrInvalidAction = errors.New("无效的操作")

	// ErrInvalidResourceType 无效的资源类型
	ErrInvalidResourceType = errors.New("无效的资源类型")
)
