package commands

import (
	"context"

	"nodepass-pro/backend/internal/domain/nodegroup"
)

// DeleteGroupCommand 删除节点组命令
type DeleteGroupCommand struct {
	ID uint
}

// DeleteGroupHandler 删除节点组处理器
type DeleteGroupHandler struct {
	repo nodegroup.Repository
}

// NewDeleteGroupHandler 创建处理器实例
func NewDeleteGroupHandler(repo nodegroup.Repository) *DeleteGroupHandler {
	return &DeleteGroupHandler{
		repo: repo,
	}
}

// Handle 处理删除节点组命令
func (h *DeleteGroupHandler) Handle(ctx context.Context, cmd DeleteGroupCommand) error {
	// 查找节点组（确保存在）
	_, err := h.repo.FindByID(ctx, cmd.ID)
	if err != nil {
		return err
	}

	// 删除节点组
	return h.repo.Delete(ctx, cmd.ID)
}
