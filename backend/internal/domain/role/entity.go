package role

import (
	"regexp"
	"strings"
	"time"
)

var roleCodePattern = regexp.MustCompile(`^[a-z][a-z0-9_\-]{1,49}$`)

// Role 角色聚合根
type Role struct {
	ID          uint
	Code        string
	Name        string
	Description string
	IsSystem    bool
	IsEnabled   bool
	Permissions []Permission
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Permission 权限值对象
type Permission struct {
	Code        string
	Description string
}

// NewRole 创建角色
func NewRole(code, name, description string, isSystem bool) (*Role, error) {
	code = normalizeRoleCode(code)
	if !IsValidRoleCode(code) {
		return nil, ErrRoleCodeInvalid
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrRoleNameInvalid
	}
	if len([]rune(name)) > 100 {
		return nil, ErrRoleNameInvalid
	}

	now := time.Now()
	return &Role{
		Code:        code,
		Name:        name,
		Description: strings.TrimSpace(description),
		IsSystem:    isSystem,
		IsEnabled:   true,
		Permissions: []Permission{},
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// IsValidRoleCode 验证角色编码
func IsValidRoleCode(code string) bool {
	return roleCodePattern.MatchString(code)
}

// IsAdmin 是否为管理员角色
func (r *Role) IsAdmin() bool {
	return r.Code == "admin"
}

// IsUser 是否为普通用户角色
func (r *Role) IsUser() bool {
	return r.Code == "user"
}

// CanModify 是否可以修改
func (r *Role) CanModify() bool {
	return !r.IsSystem
}

// CanDelete 是否可以删除
func (r *Role) CanDelete() bool {
	return !r.IsSystem
}

// CanDisable 是否可以禁用
func (r *Role) CanDisable() bool {
	return !r.IsSystem
}

// UpdateName 更新名称
func (r *Role) UpdateName(name string) error {
	if !r.CanModify() {
		return ErrSystemRoleCannotModify
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return ErrRoleNameInvalid
	}
	if len([]rune(name)) > 100 {
		return ErrRoleNameInvalid
	}

	r.Name = name
	r.UpdatedAt = time.Now()
	return nil
}

// UpdateDescription 更新描述
func (r *Role) UpdateDescription(description string) error {
	if !r.CanModify() {
		return ErrSystemRoleCannotModify
	}

	r.Description = strings.TrimSpace(description)
	r.UpdatedAt = time.Now()
	return nil
}

// Enable 启用角色
func (r *Role) Enable() {
	r.IsEnabled = true
	r.UpdatedAt = time.Now()
}

// Disable 禁用角色
func (r *Role) Disable() error {
	if !r.CanDisable() {
		return ErrSystemRoleCannotDisable
	}

	r.IsEnabled = false
	r.UpdatedAt = time.Now()
	return nil
}

// SetPermissions 设置权限
func (r *Role) SetPermissions(permissions []Permission) {
	r.Permissions = permissions
	r.UpdatedAt = time.Now()
}

// HasPermission 是否拥有权限
func (r *Role) HasPermission(permissionCode string) bool {
	// 管理员拥有所有权限
	if r.IsAdmin() {
		return true
	}

	for _, p := range r.Permissions {
		if p.Code == permissionCode {
			return true
		}
	}
	return false
}

// GetPermissionCodes 获取权限代码列表
func (r *Role) GetPermissionCodes() []string {
	codes := make([]string, len(r.Permissions))
	for i, p := range r.Permissions {
		codes[i] = p.Code
	}
	return codes
}

// normalizeRoleCode 标准化角色编码
func normalizeRoleCode(code string) string {
	return strings.ToLower(strings.TrimSpace(code))
}

// NewPermission 创建权限
func NewPermission(code, description string) Permission {
	return Permission{
		Code:        strings.TrimSpace(code),
		Description: strings.TrimSpace(description),
	}
}
