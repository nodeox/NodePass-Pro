package commands

import (
	"context"
	"time"

	"nodepass-pro/backend/internal/domain/nodegroup"
)

// UpdateGroupCommand 更新节点组命令
type UpdateGroupCommand struct {
	ID          uint
	Name        string
	Description string
	Config      nodegroup.NodeGroupConfig
}

// UpdateGroupHandler 更新节点组处理器
type UpdateGroupHandler struct {
	repo nodegroup.Repository
}

// NewUpdateGroupHandler 创建处理器实例
func NewUpdateGroupHandler(repo nodegroup.Repository) *UpdateGroupHandler {
	return &UpdateGroupHandler{
		repo: repo,
	}
}

// Handle 处理更新节点组命令
func (h *UpdateGroupHandler) Handle(ctx context.Context, cmd UpdateGroupCommand) (*nodegroup.NodeGroup, error) {
	// 查找节点组
	group, err := h.repo.FindByID(ctx, cmd.ID)
	if err != nil {
		return nil, err
	}

	// 验证端口范围
	if cmd.Config.PortRange.Start > 0 && cmd.Config.PortRange.End > 0 {
		if cmd.Config.PortRange.Start > cmd.Config.PortRange.End {
			return nil, nodegroup.ErrInvalidPortRange
		}
	}

	// 更新字段
	group.Name = cmd.Name
	group.Description = cmd.Description
	group.Config = cmd.Config
	group.UpdatedAt = time.Now()

	// 持久化
	if err := h.repo.Update(ctx, group); err != nil {
		return nil, err
	}

	return group, nil
}
