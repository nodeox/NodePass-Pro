package commands

import (
	"context"

	"nodepass-pro/backend/internal/domain/auditlog"
)

// CreateAuditLogCommand 创建审计日志命令
type CreateAuditLogCommand struct {
	UserID       *uint
	Action       string
	ResourceType string
	ResourceID   *uint
	Details      string
	IPAddress    string
	UserAgent    string
}

// CreateAuditLogHandler 创建审计日志处理器
type CreateAuditLogHandler struct {
	repo auditlog.Repository
}

// NewCreateAuditLogHandler 创建处理器
func NewCreateAuditLogHandler(repo auditlog.Repository) *CreateAuditLogHandler {
	return &CreateAuditLogHandler{repo: repo}
}

// Handle 处理命令
func (h *CreateAuditLogHandler) Handle(ctx context.Context, cmd CreateAuditLogCommand) error {
	log := auditlog.NewAuditLog(
		cmd.UserID,
		cmd.Action,
		cmd.ResourceType,
		cmd.ResourceID,
		cmd.Details,
		cmd.IPAddress,
		cmd.UserAgent,
	)

	return h.repo.Create(ctx, log)
}
