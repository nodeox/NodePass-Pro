package queries

import (
	"context"
	"time"

	"nodepass-pro/backend/internal/domain/nodeperformance"
)

// GetMetricsStatsQuery 获取性能统计查询
type GetMetricsStatsQuery struct {
	NodeInstanceID uint
	Duration       time.Duration
}

// GetMetricsStatsHandler 获取性能统计处理器
type GetMetricsStatsHandler struct {
	repo nodeperformance.Repository
}

// NewGetMetricsStatsHandler 创建处理器
func NewGetMetricsStatsHandler(repo nodeperformance.Repository) *GetMetricsStatsHandler {
	return &GetMetricsStatsHandler{repo: repo}
}

// Handle 处理查询
func (h *GetMetricsStatsHandler) Handle(ctx context.Context, query GetMetricsStatsQuery) (*nodeperformance.PerformanceStats, error) {
	startTime := time.Now().Add(-query.Duration)
	metrics, err := h.repo.GetMetrics(ctx, query.NodeInstanceID, startTime, time.Time{}, 0)
	if err != nil {
		return nil, err
	}

	return nodeperformance.CalculateStats(metrics), nil
}
