package queries

import (
	"context"

	"nodepass-pro/backend/internal/domain/auditlog"
)

// ListAuditLogsQuery 列表查询
type ListAuditLogsQuery struct {
	UserID       *uint
	Action       string
	ResourceType string
	StartTime    *string
	EndTime      *string
	Page         int
	PageSize     int
}

// ListAuditLogsHandler 列表查询处理器
type ListAuditLogsHandler struct {
	repo auditlog.Repository
}

// NewListAuditLogsHandler 创建处理器
func NewListAuditLogsHandler(repo auditlog.Repository) *ListAuditLogsHandler {
	return &ListAuditLogsHandler{repo: repo}
}

// Handle 处理查询
func (h *ListAuditLogsHandler) Handle(ctx context.Context, query ListAuditLogsQuery) ([]*auditlog.AuditLog, int64, error) {
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

	return h.repo.List(ctx, auditlog.ListFilter{
		UserID:       query.UserID,
		Action:       query.Action,
		ResourceType: query.ResourceType,
		Page:         page,
		PageSize:     pageSize,
	})
}
