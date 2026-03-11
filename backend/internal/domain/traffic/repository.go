package traffic

import (
	"context"
	"time"
)

// RecordRepository 流量记录仓储接口
type RecordRepository interface {
	// 基础 CRUD
	Create(ctx context.Context, record *TrafficRecord) error
	BatchCreate(ctx context.Context, records []*TrafficRecord) error
	FindByID(ctx context.Context, id uint) (*TrafficRecord, error)
	
	// 查询
	FindByUserID(ctx context.Context, userID uint, start, end time.Time) ([]*TrafficRecord, error)
	FindByTunnelID(ctx context.Context, tunnelID uint, start, end time.Time) ([]*TrafficRecord, error)
	List(ctx context.Context, filter RecordListFilter) ([]*TrafficRecord, int64, error)
	
	// 统计
	SumByUserID(ctx context.Context, userID uint, start, end time.Time) (int64, int64, error)
	SumByTunnelID(ctx context.Context, tunnelID uint, start, end time.Time) (int64, int64, error)
	
	// 清理
	DeleteOldRecords(ctx context.Context, before time.Time) (int64, error)
}

// QuotaRepository 流量配额仓储接口
type QuotaRepository interface {
	// 基础操作
	GetByUserID(ctx context.Context, userID uint) (*TrafficQuota, error)
	Update(ctx context.Context, quota *TrafficQuota) error
	
	// 批量操作
	BatchUpdate(ctx context.Context, quotas []*TrafficQuota) error
	FindAll(ctx context.Context) ([]*TrafficQuota, error)
	
	// 重置
	ResetAll(ctx context.Context) (int64, error)
	ResetByUserID(ctx context.Context, userID uint) error
}

// RecordListFilter 流量记录列表过滤器
type RecordListFilter struct {
	Page      int
	PageSize  int
	UserID    uint
	TunnelID  uint
	StartTime time.Time
	EndTime   time.Time
}
