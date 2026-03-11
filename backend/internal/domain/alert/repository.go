package alert

import (
	"context"
	"time"
)

// Repository 告警仓储接口
type Repository interface {
	// Create 创建告警
	Create(ctx context.Context, alert *Alert) error

	// FindByID 根据 ID 查找
	FindByID(ctx context.Context, id uint) (*Alert, error)

	// FindByFingerprint 根据指纹查找
	FindByFingerprint(ctx context.Context, fingerprint string) (*Alert, error)

	// Update 更新告警
	Update(ctx context.Context, alert *Alert) error

	// List 列表查询
	List(ctx context.Context, filter ListFilter) ([]*Alert, int64, error)

	// CountByStatus 按状态统计
	CountByStatus(ctx context.Context, status AlertStatus) (int64, error)

	// CountByLevel 按级别统计
	CountByLevel(ctx context.Context, level AlertLevel) (int64, error)

	// FindFiringAlerts 查找正在触发的告警
	FindFiringAlerts(ctx context.Context) ([]*Alert, error)
}

// ListFilter 列表过滤器
type ListFilter struct {
	Status       AlertStatus
	Level        AlertLevel
	Type         string
	ResourceType string
	ResourceID   *uint
	StartTime    *time.Time
	EndTime      *time.Time
	Page         int
	PageSize     int
}
