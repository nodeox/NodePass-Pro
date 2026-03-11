package commands

import (
	"context"
	"fmt"

	"nodepass-pro/backend/internal/domain/tunneltemplate"
)

// DeleteTemplateCommand 删除模板命令
type DeleteTemplateCommand struct {
	TemplateID uint
	UserID     uint
}

// DeleteTemplateHandler 删除模板处理器
type DeleteTemplateHandler struct {
	repo tunneltemplate.Repository
}

// NewDeleteTemplateHandler 创建处理器
func NewDeleteTemplateHandler(repo tunneltemplate.Repository) *DeleteTemplateHandler {
	return &DeleteTemplateHandler{repo: repo}
}

// Handle 处理命令
func (h *DeleteTemplateHandler) Handle(ctx context.Context, cmd DeleteTemplateCommand) error {
	// 查找模板
	template, err := h.repo.FindByID(ctx, cmd.TemplateID)
	if err != nil {
		return err
	}

	// 检查权限
	if !template.IsOwnedBy(cmd.UserID) {
		return tunneltemplate.ErrUnauthorized
	}

	// 删除
	if err := h.repo.Delete(ctx, cmd.TemplateID); err != nil {
		return fmt.Errorf("删除模板失败: %w", err)
	}

	return nil
}
