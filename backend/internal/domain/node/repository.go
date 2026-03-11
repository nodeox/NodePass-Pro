package node

import (
	"context"
	"time"
)

// InstanceRepository 节点实例仓储接口
type InstanceRepository interface {
	// 基础 CRUD
	Create(ctx context.Context, instance *NodeInstance) error
	FindByID(ctx context.Context, id uint) (*NodeInstance, error)
	FindByNodeID(ctx context.Context, nodeID string) (*NodeInstance, error)
	Update(ctx context.Context, instance *NodeInstance) error
	Delete(ctx context.Context, id uint) error
	
	// 批量操作
	FindByGroupID(ctx context.Context, groupID uint) ([]*NodeInstance, error)
	FindByIDs(ctx context.Context, ids []uint) ([]*NodeInstance, error)
	List(ctx context.Context, filter InstanceListFilter) ([]*NodeInstance, int64, error)
	
	// 业务查询
	FindOnlineNodes(ctx context.Context) ([]*NodeInstance, error)
	FindOfflineNodes(ctx context.Context, timeout time.Duration) ([]*NodeInstance, error)
	CountByStatus(ctx context.Context, status string) (int64, error)
	
	// 心跳相关
	UpdateHeartbeat(ctx context.Context, nodeID string, data *HeartbeatData) error
	BatchUpdateHeartbeat(ctx context.Context, data []*HeartbeatData) error
	MarkOfflineByTimeout(ctx context.Context, timeout time.Duration) (int64, error)
}

// GroupRepository 节点组仓储接口
type GroupRepository interface {
	// 基础 CRUD
	Create(ctx context.Context, group *NodeGroup) error
	FindByID(ctx context.Context, id uint) (*NodeGroup, error)
	Update(ctx context.Context, group *NodeGroup) error
	Delete(ctx context.Context, id uint) error
	
	// 查询
	List(ctx context.Context, filter GroupListFilter) ([]*NodeGroup, int64, error)
	FindEnabled(ctx context.Context) ([]*NodeGroup, error)
	
	// 统计
	UpdateStats(ctx context.Context, groupID uint, nodeCount, onlineCount int) error
}

// InstanceListFilter 节点实例列表过滤器
type InstanceListFilter struct {
	Page        int
	PageSize    int
	GroupID     uint
	Status      string
	Keyword     string
	OnlineOnly  bool
}

// GroupListFilter 节点组列表过滤器
type GroupListFilter struct {
	Page        int
	PageSize    int
	EnabledOnly bool
	Keyword     string
}

// HeartbeatData 心跳数据
type HeartbeatData struct {
	NodeID        string
	CPUUsage      float64
	MemoryUsage   float64
	DiskUsage     float64
	TrafficIn     int64
	TrafficOut    int64
	ActiveRules   int
	ConfigVersion int
	ClientVersion string
	Timestamp     time.Time
}
