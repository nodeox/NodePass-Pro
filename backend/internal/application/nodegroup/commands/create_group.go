package commands

import (
	"context"
	"time"

	"nodepass-pro/backend/internal/domain/nodegroup"
)

// CreateGroupCommand 创建节点组命令
type CreateGroupCommand struct {
	UserID      uint
	Name        string
	Type        nodegroup.NodeGroupType
	Description string
	Config      nodegroup.NodeGroupConfig
}

// CreateGroupHandler 创建节点组处理器
type CreateGroupHandler struct {
	repo nodegroup.Repository
}

// NewCreateGroupHandler 创建处理器实例
func NewCreateGroupHandler(repo nodegroup.Repository) *CreateGroupHandler {
	return &CreateGroupHandler{
		repo: repo,
	}
}

// Handle 处理创建节点组命令
func (h *CreateGroupHandler) Handle(ctx context.Context, cmd CreateGroupCommand) (*nodegroup.NodeGroup, error) {
	// 验证节点组类型
	if cmd.Type != nodegroup.NodeGroupTypeEntry && cmd.Type != nodegroup.NodeGroupTypeExit {
		return nil, nodegroup.ErrInvalidNodeGroupType
	}

	// 验证端口范围
	if cmd.Config.PortRange.Start > 0 && cmd.Config.PortRange.End > 0 {
		if cmd.Config.PortRange.Start > cmd.Config.PortRange.End {
			return nil, nodegroup.ErrInvalidPortRange
		}
	}

	// 创建节点组实体
	group := &nodegroup.NodeGroup{
		UserID:      cmd.UserID,
		Name:        cmd.Name,
		Type:        cmd.Type,
		Description: cmd.Description,
		IsEnabled:   true, // 默认启用
		Config:      cmd.Config,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// 持久化
	if err := h.repo.Create(ctx, group); err != nil {
		return nil, err
	}

	return group, nil
}
