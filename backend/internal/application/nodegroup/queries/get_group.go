package queries

import (
	"context"

	"nodepass-pro/backend/internal/domain/nodegroup"
)

// GetGroupQuery 获取节点组查询
type GetGroupQuery struct {
	ID uint
}

// GetGroupHandler 获取节点组处理器
type GetGroupHandler struct {
	repo nodegroup.Repository
}

// NewGetGroupHandler 创建处理器实例
func NewGetGroupHandler(repo nodegroup.Repository) *GetGroupHandler {
	return &GetGroupHandler{
		repo: repo,
	}
}

// Handle 处理获取节点组查询
func (h *GetGroupHandler) Handle(ctx context.Context, query GetGroupQuery) (*nodegroup.NodeGroup, error) {
	return h.repo.FindByID(ctx, query.ID)
}
