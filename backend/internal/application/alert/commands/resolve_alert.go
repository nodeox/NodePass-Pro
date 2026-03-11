package commands

import (
	"context"
	"fmt"

	"nodepass-pro/backend/internal/domain/alert"
)

// ResolveAlertCommand 解决告警命令
type ResolveAlertCommand struct {
	AlertID    uint
	ResolvedBy uint
	Notes      string
}

// ResolveAlertHandler 解决告警处理器
type ResolveAlertHandler struct {
	repo alert.Repository
}

// NewResolveAlertHandler 创建处理器
func NewResolveAlertHandler(repo alert.Repository) *ResolveAlertHandler {
	return &ResolveAlertHandler{repo: repo}
}

// Handle 处理命令
func (h *ResolveAlertHandler) Handle(ctx context.Context, cmd ResolveAlertCommand) error {
	// 查找告警
	a, err := h.repo.FindByID(ctx, cmd.AlertID)
	if err != nil {
		return err
	}

	// 解决告警
	if err := a.Resolve(cmd.ResolvedBy, cmd.Notes); err != nil {
		return err
	}

	// 更新
	if err := h.repo.Update(ctx, a); err != nil {
		return fmt.Errorf("更新告警失败: %w", err)
	}

	return nil
}
