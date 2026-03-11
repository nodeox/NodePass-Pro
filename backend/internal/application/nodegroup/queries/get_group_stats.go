package queries

import (
	"context"

	"nodepass-pro/backend/internal/domain/nodegroup"
)

// GetGroupStatsQuery 获取节点组统计查询
type GetGroupStatsQuery struct {
	GroupID uint
}

// GetGroupStatsHandler 获取节点组统计处理器
type GetGroupStatsHandler struct {
	repo nodegroup.Repository
}

// NewGetGroupStatsHandler 创建处理器实例
func NewGetGroupStatsHandler(repo nodegroup.Repository) *GetGroupStatsHandler {
	return &GetGroupStatsHandler{
		repo: repo,
	}
}

// Handle 处理获取节点组统计查询
func (h *GetGroupStatsHandler) Handle(ctx context.Context, query GetGroupStatsQuery) (*nodegroup.NodeGroupStats, error) {
	// 确保节点组存在
	_, err := h.repo.FindByID(ctx, query.GroupID)
	if err != nil {
		return nil, err
	}

	// 获取统计信息
	return h.repo.GetStats(ctx, query.GroupID)
}
