package commands

import (
	"context"
	"fmt"

	"nodepass-pro/backend/internal/domain/role"
)

// AssignPermissionsCommand 分配权限命令
type AssignPermissionsCommand struct {
	AdminID     uint
	RoleID      uint
	Permissions []string
}

// AssignPermissionsHandler 分配权限处理器
type AssignPermissionsHandler struct {
	repo role.Repository
}

// NewAssignPermissionsHandler 创建处理器
func NewAssignPermissionsHandler(repo role.Repository) *AssignPermissionsHandler {
	return &AssignPermissionsHandler{repo: repo}
}

// Handle 处理命令
func (h *AssignPermissionsHandler) Handle(ctx context.Context, cmd AssignPermissionsCommand) (*role.Role, error) {
	// 查找角色
	r, err := h.repo.FindByID(ctx, cmd.RoleID)
	if err != nil {
		return nil, err
	}

	// 转换为权限对象
	permissions := make([]role.Permission, len(cmd.Permissions))
	for i, p := range cmd.Permissions {
		permissions[i] = role.NewPermission(p, "")
	}

	// 设置权限
	r.SetPermissions(permissions)

	// 保存
	if err := h.repo.Update(ctx, r); err != nil {
		return nil, fmt.Errorf("更新角色权限失败: %w", err)
	}

	return r, nil
}
