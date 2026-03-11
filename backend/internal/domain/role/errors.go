package role

import "errors"

var (
	// ErrRoleNotFound 角色不存在
	ErrRoleNotFound = errors.New("角色不存在")

	// ErrRoleAlreadyExists 角色已存在
	ErrRoleAlreadyExists = errors.New("角色已存在")

	// ErrRoleCodeInvalid 角色编码无效
	ErrRoleCodeInvalid = errors.New("角色编码无效")

	// ErrRoleNameInvalid 角色名称无效
	ErrRoleNameInvalid = errors.New("角色名称无效")

	// ErrSystemRoleCannotModify 系统角色不可修改
	ErrSystemRoleCannotModify = errors.New("系统角色不可修改")

	// ErrSystemRoleCannotDelete 系统角色不可删除
	ErrSystemRoleCannotDelete = errors.New("系统角色不可删除")

	// ErrSystemRoleCannotDisable 系统角色不可禁用
	ErrSystemRoleCannotDisable = errors.New("系统角色不可禁用")

	// ErrRoleInUse 角色正在使用中
	ErrRoleInUse = errors.New("角色正在使用中")

	// ErrPermissionInvalid 权限无效
	ErrPermissionInvalid = errors.New("权限无效")

	// ErrPermissionDenied 权限被拒绝
	ErrPermissionDenied = errors.New("权限被拒绝")
)
