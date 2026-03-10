package models

import "time"

// Role 角色模型。
type Role struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Code        string    `gorm:"column:code;type:varchar(50);not null;uniqueIndex:uk_roles_code" json:"code"`
	Name        string    `gorm:"column:name;type:varchar(100);not null" json:"name"`
	Description *string   `gorm:"column:description;type:text" json:"description"`
	IsSystem    bool      `gorm:"column:is_system;not null;default:false;index:idx_roles_is_system" json:"is_system"`
	IsEnabled   bool      `gorm:"column:is_enabled;not null;default:true;index:idx_roles_is_enabled" json:"is_enabled"`
	CreatedAt   time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at" json:"updated_at"`

	Permissions []RolePermission `gorm:"foreignKey:RoleID;references:ID" json:"permissions,omitempty"`
}

// TableName 指定表名。
func (Role) TableName() string {
	return "roles"
}

// RolePermission 角色权限映射模型。
type RolePermission struct {
	ID         uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	RoleID     uint      `gorm:"column:role_id;not null;index:idx_role_permissions_role_id;uniqueIndex:uk_role_permissions_role_permission" json:"role_id"`
	Permission string    `gorm:"column:permission;type:varchar(100);not null;index:idx_role_permissions_permission;uniqueIndex:uk_role_permissions_role_permission" json:"permission"`
	CreatedAt  time.Time `gorm:"column:created_at" json:"created_at"`

	Role *Role `gorm:"foreignKey:RoleID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"role,omitempty"`
}

// TableName 指定表名。
func (RolePermission) TableName() string {
	return "role_permissions"
}
