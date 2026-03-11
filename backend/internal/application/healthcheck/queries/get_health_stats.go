package queries

import (
	"context"
	"time"

	"nodepass-pro/backend/internal/domain/healthcheck"
)

// GetHealthStatsQuery 获取健康统计查询
type GetHealthStatsQuery struct {
	NodeInstanceID uint
	Duration       time.Duration
}

// GetHealthStatsHandler 获取健康统计处理器
type GetHealthStatsHandler struct {
	repo healthcheck.Repository
}

// NewGetHealthStatsHandler 创建处理器
func NewGetHealthStatsHandler(repo healthcheck.Repository) *GetHealthStatsHandler {
	return &GetHealthStatsHandler{repo: repo}
}

// Handle 处理查询
func (h *GetHealthStatsHandler) Handle(ctx context.Context, query GetHealthStatsQuery) (*healthcheck.HealthStats, error) {
	duration := query.Duration
	if duration <= 0 {
		duration = 24 * time.Hour
	}

	startTime := time.Now().Add(-duration)
	records, err := h.repo.FindHealthRecordsByTimeRange(ctx, query.NodeInstanceID, startTime)
	if err != nil {
		return nil, err
	}

	return healthcheck.NewHealthStats(records, duration.String()), nil
}
