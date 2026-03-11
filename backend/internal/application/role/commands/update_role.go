package commands

import (
	"context"
	"fmt"

	"nodepass-pro/backend/internal/domain/role"
)

// UpdateRoleCommand 更新角色命令
type UpdateRoleCommand struct {
	AdminID     uint
	RoleID      uint
	Name        *string
	Description *string
	IsEnabled   *bool
}

// UpdateRoleHandler 更新角色处理器
type UpdateRoleHandler struct {
	repo role.Repository
}

// NewUpdateRoleHandler 创建处理器
func NewUpdateRoleHandler(repo role.Repository) *UpdateRoleHandler {
	return &UpdateRoleHandler{repo: repo}
}

// Handle 处理命令
func (h *UpdateRoleHandler) Handle(ctx context.Context, cmd UpdateRoleCommand) (*role.Role, error) {
	// 查找角色
	r, err := h.repo.FindByID(ctx, cmd.RoleID)
	if err != nil {
		return nil, err
	}

	// 更新名称
	if cmd.Name != nil {
		if err := r.UpdateName(*cmd.Name); err != nil {
			return nil, err
		}
	}

	// 更新描述
	if cmd.Description != nil {
		if err := r.UpdateDescription(*cmd.Description); err != nil {
			return nil, err
		}
	}

	// 更新启用状态
	if cmd.IsEnabled != nil {
		if *cmd.IsEnabled {
			r.Enable()
		} else {
			if err := r.Disable(); err != nil {
				return nil, err
			}
		}
	}

	// 保存
	if err := h.repo.Update(ctx, r); err != nil {
		return nil, fmt.Errorf("更新角色失败: %w", err)
	}

	return r, nil
}
