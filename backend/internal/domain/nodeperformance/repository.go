package nodeperformance

import (
	"context"
	"time"
)

// Repository 性能监控仓储接口
type Repository interface {
	// RecordMetric 记录性能指标
	RecordMetric(ctx context.Context, metric *PerformanceMetric) error

	// GetLatestMetric 获取最新指标
	GetLatestMetric(ctx context.Context, nodeInstanceID uint) (*PerformanceMetric, error)

	// GetMetrics 获取指标历史
	GetMetrics(ctx context.Context, nodeInstanceID uint, startTime, endTime time.Time, limit int) ([]*PerformanceMetric, error)

	// FindAlertByNodeInstance 根据节点实例查找告警配置
	FindAlertByNodeInstance(ctx context.Context, nodeInstanceID uint) (*PerformanceAlert, error)

	// UpsertAlert 创建或更新告警配置
	UpsertAlert(ctx context.Context, alert *PerformanceAlert) error
}
