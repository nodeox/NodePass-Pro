package queries

import (
	"context"

	"nodepass-pro/backend/internal/domain/role"
)

// GetAvailablePermissionsQuery 获取可用权限查询
type GetAvailablePermissionsQuery struct {
	AdminID uint
}

// GetAvailablePermissionsHandler 获取可用权限处理器
type GetAvailablePermissionsHandler struct {
	repo role.Repository
}

// NewGetAvailablePermissionsHandler 创建处理器
func NewGetAvailablePermissionsHandler(repo role.Repository) *GetAvailablePermissionsHandler {
	return &GetAvailablePermissionsHandler{repo: repo}
}

// Handle 处理查询
func (h *GetAvailablePermissionsHandler) Handle(ctx context.Context, query GetAvailablePermissionsQuery) ([]role.Permission, error) {
	return h.repo.GetAvailablePermissions(ctx)
}
