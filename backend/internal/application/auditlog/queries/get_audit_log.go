package queries

import (
	"context"

	"nodepass-pro/backend/internal/domain/auditlog"
)

// GetAuditLogQuery 获取审计日志查询
type GetAuditLogQuery struct {
	ID uint
}

// GetAuditLogHandler 获取审计日志处理器
type GetAuditLogHandler struct {
	repo auditlog.Repository
}

// NewGetAuditLogHandler 创建处理器
func NewGetAuditLogHandler(repo auditlog.Repository) *GetAuditLogHandler {
	return &GetAuditLogHandler{repo: repo}
}

// Handle 处理查询
func (h *GetAuditLogHandler) Handle(ctx context.Context, query GetAuditLogQuery) (*auditlog.AuditLog, error) {
	return h.repo.FindByID(ctx, query.ID)
}
