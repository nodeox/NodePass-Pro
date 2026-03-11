package queries

import (
	"context"

	"nodepass-pro/backend/internal/domain/alert"
)

// ListAlertsQuery 列表查询
type ListAlertsQuery struct {
	Status       alert.AlertStatus
	Level        alert.AlertLevel
	Type         string
	ResourceType string
	ResourceID   *uint
	Page         int
	PageSize     int
}

// ListAlertsHandler 列表查询处理器
type ListAlertsHandler struct {
	repo alert.Repository
}

// NewListAlertsHandler 创建处理器
func NewListAlertsHandler(repo alert.Repository) *ListAlertsHandler {
	return &ListAlertsHandler{repo: repo}
}

// Handle 处理查询
func (h *ListAlertsHandler) Handle(ctx context.Context, query ListAlertsQuery) ([]*alert.Alert, int64, error) {
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

	return h.repo.List(ctx, alert.ListFilter{
		Status:       query.Status,
		Level:        query.Level,
		Type:         query.Type,
		ResourceType: query.ResourceType,
		ResourceID:   query.ResourceID,
		Page:         page,
		PageSize:     pageSize,
	})
}
