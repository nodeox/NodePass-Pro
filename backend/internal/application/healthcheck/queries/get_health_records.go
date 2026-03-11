package queries

import (
	"context"

	"nodepass-pro/backend/internal/domain/healthcheck"
)

// GetHealthRecordsQuery 获取健康检查记录查询
type GetHealthRecordsQuery struct {
	NodeInstanceID uint
	Limit          int
}

// GetHealthRecordsHandler 获取健康检查记录处理器
type GetHealthRecordsHandler struct {
	repo healthcheck.Repository
}

// NewGetHealthRecordsHandler 创建处理器
func NewGetHealthRecordsHandler(repo healthcheck.Repository) *GetHealthRecordsHandler {
	return &GetHealthRecordsHandler{repo: repo}
}

// Handle 处理查询
func (h *GetHealthRecordsHandler) Handle(ctx context.Context, query GetHealthRecordsQuery) ([]*healthcheck.HealthRecord, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	return h.repo.FindHealthRecordsByNodeInstance(ctx, query.NodeInstanceID, limit)
}
