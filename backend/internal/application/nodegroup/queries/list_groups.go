package queries

import (
	"context"

	"nodepass-pro/backend/internal/domain/nodegroup"
)

// ListGroupsQuery 列表查询节点组
type ListGroupsQuery struct {
	UserID      uint
	Type        nodegroup.NodeGroupType
	EnabledOnly bool
	Keyword     string
	Page        int
	PageSize    int
}

// ListGroupsResult 列表查询结果
type ListGroupsResult struct {
	Groups []*nodegroup.NodeGroup
	Total  int64
}

// ListGroupsHandler 列表查询处理器
type ListGroupsHandler struct {
	repo nodegroup.Repository
}

// NewListGroupsHandler 创建处理器实例
func NewListGroupsHandler(repo nodegroup.Repository) *ListGroupsHandler {
	return &ListGroupsHandler{
		repo: repo,
	}
}

// Handle 处理列表查询
func (h *ListGroupsHandler) Handle(ctx context.Context, query ListGroupsQuery) (*ListGroupsResult, error) {
	filter := nodegroup.ListFilter{
		UserID:      query.UserID,
		Type:        query.Type,
		EnabledOnly: query.EnabledOnly,
		Keyword:     query.Keyword,
		Page:        query.Page,
		PageSize:    query.PageSize,
	}

	groups, total, err := h.repo.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	return &ListGroupsResult{
		Groups: groups,
		Total:  total,
	}, nil
}
