package healthcheck

import (
	"context"
	"time"
)

// Repository 健康检查仓储接口
type Repository interface {
	// CreateHealthCheck 创建健康检查配置
	CreateHealthCheck(ctx context.Context, check *HealthCheck) error

	// FindHealthCheckByID 根据 ID 查找健康检查配置
	FindHealthCheckByID(ctx context.Context, id uint) (*HealthCheck, error)

	// FindHealthCheckByNodeInstance 根据节点实例 ID 查找健康检查配置
	FindHealthCheckByNodeInstance(ctx context.Context, nodeInstanceID uint) (*HealthCheck, error)

	// UpdateHealthCheck 更新健康检查配置
	UpdateHealthCheck(ctx context.Context, check *HealthCheck) error

	// DeleteHealthCheck 删除健康检查配置
	DeleteHealthCheck(ctx context.Context, id uint) error

	// ListEnabledHealthChecks 列出所有启用的健康检查配置
	ListEnabledHealthChecks(ctx context.Context) ([]*HealthCheck, error)

	// CreateHealthRecord 创建健康检查记录
	CreateHealthRecord(ctx context.Context, record *HealthRecord) error

	// FindHealthRecordsByNodeInstance 根据节点实例 ID 查找健康检查记录
	FindHealthRecordsByNodeInstance(ctx context.Context, nodeInstanceID uint, limit int) ([]*HealthRecord, error)

	// FindHealthRecordsByTimeRange 根据时间范围查找健康检查记录
	FindHealthRecordsByTimeRange(ctx context.Context, nodeInstanceID uint, startTime time.Time) ([]*HealthRecord, error)

	// DeleteOldHealthRecords 删除旧的健康检查记录
	DeleteOldHealthRecords(ctx context.Context, cutoffTime time.Time) (int64, error)

	// CreateOrUpdateQualityScore 创建或更新质量评分
	CreateOrUpdateQualityScore(ctx context.Context, score *QualityScore) error

	// FindQualityScoreByNodeInstance 根据节点实例 ID 查找质量评分
	FindQualityScoreByNodeInstance(ctx context.Context, nodeInstanceID uint) (*QualityScore, error)

	// ListQualityScoresByUser 列出用户的所有质量评分
	ListQualityScoresByUser(ctx context.Context, userID uint) ([]*QualityScore, error)
}
