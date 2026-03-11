package commands

import (
	"context"
	"fmt"

	"nodepass-pro/backend/internal/domain/role"
)

// CreateRoleCommand 创建角色命令
type CreateRoleCommand struct {
	AdminID     uint
	Code        string
	Name        string
	Description string
	Permissions []string
}

// CreateRoleHandler 创建角色处理器
type CreateRoleHandler struct {
	repo role.Repository
}

// NewCreateRoleHandler 创建处理器
func NewCreateRoleHandler(repo role.Repository) *CreateRoleHandler {
	return &CreateRoleHandler{repo: repo}
}

// Handle 处理命令
func (h *CreateRoleHandler) Handle(ctx context.Context, cmd CreateRoleCommand) (*role.Role, error) {
	// 检查角色是否已存在
	existing, err := h.repo.FindByCode(ctx, cmd.Code)
	if err == nil && existing != nil {
		return nil, role.ErrRoleAlreadyExists
	}

	// 创建角色实体
	r, err := role.NewRole(cmd.Code, cmd.Name, cmd.Description, false)
	if err != nil {
		return nil, err
	}

	// 设置权限
	permissions := make([]role.Permission, len(cmd.Permissions))
	for i, p := range cmd.Permissions {
		permissions[i] = role.NewPermission(p, "")
	}
	r.SetPermissions(permissions)

	// 保存
	if err := h.repo.Create(ctx, r); err != nil {
		return nil, fmt.Errorf("创建角色失败: %w", err)
	}

	return r, nil
}
