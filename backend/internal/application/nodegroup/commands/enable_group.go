package commands

import (
	"context"
	"time"

	"nodepass-pro/backend/internal/domain/nodegroup"
)

// EnableGroupCommand 启用/禁用节点组命令
type EnableGroupCommand struct {
	ID      uint
	Enabled bool
}

// EnableGroupHandler 启用/禁用节点组处理器
type EnableGroupHandler struct {
	repo nodegroup.Repository
}

// NewEnableGroupHandler 创建处理器实例
func NewEnableGroupHandler(repo nodegroup.Repository) *EnableGroupHandler {
	return &EnableGroupHandler{
		repo: repo,
	}
}

// Handle 处理启用/禁用节点组命令
func (h *EnableGroupHandler) Handle(ctx context.Context, cmd EnableGroupCommand) error {
	// 查找节点组
	group, err := h.repo.FindByID(ctx, cmd.ID)
	if err != nil {
		return err
	}

	// 更新状态
	if cmd.Enabled {
		group.Enable()
	} else {
		group.Disable()
	}
	group.UpdatedAt = time.Now()

	// 持久化
	return h.repo.Update(ctx, group)
}
