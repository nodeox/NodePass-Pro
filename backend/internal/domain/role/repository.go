package role

import "context"

// Repository 角色仓储接口
type Repository interface {
	// Create 创建角色
	Create(ctx context.Context, role *Role) error

	// FindByID 根据 ID 查找
	FindByID(ctx context.Context, id uint) (*Role, error)

	// FindByCode 根据 Code 查找
	FindByCode(ctx context.Context, code string) (*Role, error)

	// Update 更新角色
	Update(ctx context.Context, role *Role) error

	// Delete 删除角色
	Delete(ctx context.Context, id uint) error

	// List 列表查询
	List(ctx context.Context, filter ListFilter) ([]*Role, int64, error)

	// CountUsersByRole 统计使用该角色的用户数量
	CountUsersByRole(ctx context.Context, roleCode string) (int64, error)

	// EnsureSystemRoles 确保系统角色存在
	EnsureSystemRoles(ctx context.Context) error

	// GetAvailablePermissions 获取所有可用权限
	GetAvailablePermissions(ctx context.Context) ([]Permission, error)
}

// ListFilter 列表过滤器
type ListFilter struct {
	Keyword         string
	IncludeDisabled bool
	Page            int
	PageSize        int
}
