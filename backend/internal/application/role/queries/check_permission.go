package queries

import (
	"context"

	"nodepass-pro/backend/internal/domain/role"
)

// CheckPermissionQuery 检查权限查询
type CheckPermissionQuery struct {
	RoleCode       string
	PermissionCode string
}

// CheckPermissionResult 检查权限结果
type CheckPermissionResult struct {
	HasPermission bool
	Role          *role.Role
}

// CheckPermissionHandler 检查权限处理器
type CheckPermissionHandler struct {
	repo role.Repository
}

// NewCheckPermissionHandler 创建处理器
func NewCheckPermissionHandler(repo role.Repository) *CheckPermissionHandler {
	return &CheckPermissionHandler{repo: repo}
}

// Handle 处理查询
func (h *CheckPermissionHandler) Handle(ctx context.Context, query CheckPermissionQuery) (*CheckPermissionResult, error) {
	// 查找角色
	r, err := h.repo.FindByCode(ctx, query.RoleCode)
	if err != nil {
		return &CheckPermissionResult{
			HasPermission: false,
		}, nil
	}

	// 检查权限
	hasPermission := r.HasPermission(query.PermissionCode)

	return &CheckPermissionResult{
		HasPermission: hasPermission,
		Role:          r,
	}, nil
}
