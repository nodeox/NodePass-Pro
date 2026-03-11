package nodegroup

import (
	"context"
)

// Repository 节点组仓储接口
type Repository interface {
	// Create 创建节点组
	Create(ctx context.Context, group *NodeGroup) error

	// FindByID 根据 ID 查找节点组
	FindByID(ctx context.Context, id uint) (*NodeGroup, error)

	// Update 更新节点组
	Update(ctx context.Context, group *NodeGroup) error

	// Delete 删除节点组
	Delete(ctx context.Context, id uint) error

	// FindByUserID 根据用户 ID 查找节点组
	FindByUserID(ctx context.Context, userID uint) ([]*NodeGroup, error)

	// FindByType 根据类型查找节点组
	FindByType(ctx context.Context, groupType NodeGroupType) ([]*NodeGroup, error)

	// FindByUserIDAndType 根据用户 ID 和类型查找节点组
	FindByUserIDAndType(ctx context.Context, userID uint, groupType NodeGroupType) ([]*NodeGroup, error)

	// List 列表查询
	List(ctx context.Context, filter ListFilter) ([]*NodeGroup, int64, error)

	// CountByUserID 统计用户的节点组数量
	CountByUserID(ctx context.Context, userID uint) (int64, error)

	// UpdateStats 更新节点组统计
	UpdateStats(ctx context.Context, stats *NodeGroupStats) error

	// GetStats 获取节点组统计
	GetStats(ctx context.Context, groupID uint) (*NodeGroupStats, error)
}

// ListFilter 列表查询过滤条件
type ListFilter struct {
	UserID      uint
	Type        NodeGroupType
	EnabledOnly bool
	Keyword     string
	Page        int
	PageSize    int
}
