package tunnel

import (
	"context"
)

// Repository 隧道仓储接口
type Repository interface {
	// 基础 CRUD
	Create(ctx context.Context, tunnel *Tunnel) error
	FindByID(ctx context.Context, id uint) (*Tunnel, error)
	Update(ctx context.Context, tunnel *Tunnel) error
	Delete(ctx context.Context, id uint) error
	
	// 批量操作
	FindByUserID(ctx context.Context, userID uint) ([]*Tunnel, error)
	FindByIDs(ctx context.Context, ids []uint) ([]*Tunnel, error)
	List(ctx context.Context, filter ListFilter) ([]*Tunnel, int64, error)
	
	// 业务查询
	FindRunningTunnels(ctx context.Context) ([]*Tunnel, error)
	FindByPort(ctx context.Context, port int) (*Tunnel, error)
	CountByUserID(ctx context.Context, userID uint) (int64, error)
	
	// 流量统计
	UpdateTraffic(ctx context.Context, tunnelID uint, inBytes, outBytes int64) error
	BatchUpdateTraffic(ctx context.Context, data map[uint]TrafficData) error
}

// ListFilter 列表过滤器
type ListFilter struct {
	Page       int
	PageSize   int
	UserID     uint
	Status     string
	Protocol   string
	Keyword    string
	EnabledOnly bool
}

// TrafficData 流量数据
type TrafficData struct {
	InBytes  int64
	OutBytes int64
}
