package queries

import (
	"context"

	"nodepass-pro/backend/internal/domain/healthcheck"
)

// ListQualityScoresQuery 列出质量评分查询
type ListQualityScoresQuery struct {
	UserID uint
}

// ListQualityScoresHandler 列出质量评分处理器
type ListQualityScoresHandler struct {
	repo healthcheck.Repository
}

// NewListQualityScoresHandler 创建处理器
func NewListQualityScoresHandler(repo healthcheck.Repository) *ListQualityScoresHandler {
	return &ListQualityScoresHandler{repo: repo}
}

// Handle 处理查询
func (h *ListQualityScoresHandler) Handle(ctx context.Context, query ListQualityScoresQuery) ([]*healthcheck.QualityScore, error) {
	return h.repo.ListQualityScoresByUser(ctx, query.UserID)
}
