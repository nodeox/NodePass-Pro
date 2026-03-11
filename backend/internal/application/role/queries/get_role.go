package queries

import (
	"context"

	"nodepass-pro/backend/internal/domain/role"
)

// GetRoleQuery 获取角色查询
type GetRoleQuery struct {
	RoleID uint
}

// GetRoleHandler 获取角色处理器
type GetRoleHandler struct {
	repo role.Repository
}

// NewGetRoleHandler 创建处理器
func NewGetRoleHandler(repo role.Repository) *GetRoleHandler {
	return &GetRoleHandler{repo: repo}
}

// Handle 处理查询
func (h *GetRoleHandler) Handle(ctx context.Context, query GetRoleQuery) (*role.Role, error) {
	return h.repo.FindByID(ctx, query.RoleID)
}
