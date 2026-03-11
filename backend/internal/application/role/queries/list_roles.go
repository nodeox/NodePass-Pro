package queries

import (
	"context"

	"nodepass-pro/backend/internal/domain/role"
)

// ListRolesQuery 列表查询
type ListRolesQuery struct {
	Keyword         string
	IncludeDisabled bool
	Page            int
	PageSize        int
}

// ListRolesHandler 列表查询处理器
type ListRolesHandler struct {
	repo role.Repository
}

// NewListRolesHandler 创建处理器
func NewListRolesHandler(repo role.Repository) *ListRolesHandler {
	return &ListRolesHandler{repo: repo}
}

// Handle 处理查询
func (h *ListRolesHandler) Handle(ctx context.Context, query ListRolesQuery) ([]*role.Role, int64, error) {
	// 确保系统角色存在
	if err := h.repo.EnsureSystemRoles(ctx); err != nil {
		return nil, 0, err
	}

	// 设置默认值
	page := query.Page
	if page <= 0 {
		page = 1
	}
	pageSize := query.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}

	// 查询
	return h.repo.List(ctx, role.ListFilter{
		Keyword:         query.Keyword,
		IncludeDisabled: query.IncludeDisabled,
		Page:            page,
		PageSize:        pageSize,
	})
}
