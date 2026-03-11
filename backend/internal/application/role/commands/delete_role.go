package commands

import (
	"context"
	"fmt"

	"nodepass-pro/backend/internal/domain/role"
)

// DeleteRoleCommand 删除角色命令
type DeleteRoleCommand struct {
	AdminID uint
	RoleID  uint
}

// DeleteRoleHandler 删除角色处理器
type DeleteRoleHandler struct {
	repo role.Repository
}

// NewDeleteRoleHandler 创建处理器
func NewDeleteRoleHandler(repo role.Repository) *DeleteRoleHandler {
	return &DeleteRoleHandler{repo: repo}
}

// Handle 处理命令
func (h *DeleteRoleHandler) Handle(ctx context.Context, cmd DeleteRoleCommand) error {
	// 查找角色
	r, err := h.repo.FindByID(ctx, cmd.RoleID)
	if err != nil {
		return err
	}

	// 检查是否可以删除
	if !r.CanDelete() {
		return role.ErrSystemRoleCannotDelete
	}

	// 检查是否有用户使用该角色
	count, err := h.repo.CountUsersByRole(ctx, r.Code)
	if err != nil {
		return fmt.Errorf("检查角色使用情况失败: %w", err)
	}
	if count > 0 {
		return role.ErrRoleInUse
	}

	// 删除
	if err := h.repo.Delete(ctx, cmd.RoleID); err != nil {
		return fmt.Errorf("删除角色失败: %w", err)
	}

	return nil
}
