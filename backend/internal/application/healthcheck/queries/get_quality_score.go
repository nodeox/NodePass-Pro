package queries

import (
	"context"

	"nodepass-pro/backend/internal/domain/healthcheck"
)

// GetQualityScoreQuery 获取质量评分查询
type GetQualityScoreQuery struct {
	NodeInstanceID uint
}

// GetQualityScoreHandler 获取质量评分处理器
type GetQualityScoreHandler struct {
	repo healthcheck.Repository
}

// NewGetQualityScoreHandler 创建处理器
func NewGetQualityScoreHandler(repo healthcheck.Repository) *GetQualityScoreHandler {
	return &GetQualityScoreHandler{repo: repo}
}

// Handle 处理查询
func (h *GetQualityScoreHandler) Handle(ctx context.Context, query GetQualityScoreQuery) (*healthcheck.QualityScore, error) {
	return h.repo.FindQualityScoreByNodeInstance(ctx, query.NodeInstanceID)
}
