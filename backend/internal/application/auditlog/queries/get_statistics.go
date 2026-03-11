package queries

import (
	"context"
	"time"

	"nodepass-pro/backend/internal/domain/auditlog"
)

// GetStatisticsQuery 获取统计查询
type GetStatisticsQuery struct {
	StartTime time.Time
	EndTime   time.Time
}

// GetStatisticsResult 统计结果
type GetStatisticsResult struct {
	TotalLogs    int64
	UserActions  int64
	SystemActions int64
	TopActions   map[string]int64
}

// GetStatisticsHandler 获取统计处理器
type GetStatisticsHandler struct {
	repo auditlog.Repository
}

// NewGetStatisticsHandler 创建处理器
func NewGetStatisticsHandler(repo auditlog.Repository) *GetStatisticsHandler {
	return &GetStatisticsHandler{repo: repo}
}

// Handle 处理查询
func (h *GetStatisticsHandler) Handle(ctx context.Context, query GetStatisticsQuery) (*GetStatisticsResult, error) {
	// 查询所有日志
	logs, total, err := h.repo.List(ctx, auditlog.ListFilter{
		StartTime: &query.StartTime,
		EndTime:   &query.EndTime,
		Page:      1,
		PageSize:  10000, // 获取所有记录用于统计
	})
	if err != nil {
		return nil, err
	}

	// 统计
	userActions := int64(0)
	systemActions := int64(0)
	actionCounts := make(map[string]int64)

	for _, log := range logs {
		if log.IsUserAction() {
			userActions++
		} else {
			systemActions++
		}
		actionCounts[log.Action]++
	}

	return &GetStatisticsResult{
		TotalLogs:     total,
		UserActions:   userActions,
		SystemActions: systemActions,
		TopActions:    actionCounts,
	}, nil
}
