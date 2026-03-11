package commands

import (
	"context"
	"fmt"

	"nodepass-pro/backend/internal/domain/tunneltemplate"
)

// IncrementUsageCommand 增加使用次数命令
type IncrementUsageCommand struct {
	TemplateID uint
}

// IncrementUsageHandler 增加使用次数处理器
type IncrementUsageHandler struct {
	repo tunneltemplate.Repository
}

// NewIncrementUsageHandler 创建处理器
func NewIncrementUsageHandler(repo tunneltemplate.Repository) *IncrementUsageHandler {
	return &IncrementUsageHandler{repo: repo}
}

// Handle 处理命令
func (h *IncrementUsageHandler) Handle(ctx context.Context, cmd IncrementUsageCommand) error {
	// 直接增加使用次数（不需要加载整个对象）
	if err := h.repo.IncrementUsageCount(ctx, cmd.TemplateID); err != nil {
		return fmt.Errorf("增加使用次数失败: %w", err)
	}

	return nil
}
