package tunneltemplate

import "context"

// Repository 隧道模板仓储接口
type Repository interface {
	// Create 创建模板
	Create(ctx context.Context, template *TunnelTemplate) error

	// FindByID 根据 ID 查找模板
	FindByID(ctx context.Context, id uint) (*TunnelTemplate, error)

	// Update 更新模板
	Update(ctx context.Context, template *TunnelTemplate) error

	// Delete 删除模板
	Delete(ctx context.Context, id uint) error

	// List 列表查询
	List(ctx context.Context, filter ListFilter) ([]*TunnelTemplate, int64, error)

	// FindByUserAndName 根据用户和名称查找模板
	FindByUserAndName(ctx context.Context, userID uint, name string) (*TunnelTemplate, error)

	// IncrementUsageCount 增加使用次数
	IncrementUsageCount(ctx context.Context, id uint) error
}
