package auditlog

import (
	"context"
	"time"
)

// Repository 审计日志仓储接口
type Repository interface {
	// Create 创建审计日志
	Create(ctx context.Context, log *AuditLog) error

	// BatchCreate 批量创建审计日志
	BatchCreate(ctx context.Context, logs []*AuditLog) error

	// FindByID 根据 ID 查找
	FindByID(ctx context.Context, id uint) (*AuditLog, error)

	// List 列表查询
	List(ctx context.Context, filter ListFilter) ([]*AuditLog, int64, error)

	// CountByAction 按操作统计
	CountByAction(ctx context.Context, action string, startTime, endTime time.Time) (int64, error)

	// CountByUser 按用户统计
	CountByUser(ctx context.Context, userID uint, startTime, endTime time.Time) (int64, error)

	// DeleteOldLogs 删除旧日志
	DeleteOldLogs(ctx context.Context, before time.Time) (int64, error)
}

// ListFilter 列表过滤器
type ListFilter struct {
	UserID       *uint
	Action       string
	ResourceType string
	StartTime    *time.Time
	EndTime      *time.Time
	Page         int
	PageSize     int
}
