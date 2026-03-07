package models

import "time"

// UserPermission 用户权限模型（user_permissions 表）。
type UserPermission struct {
	ID         uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID     uint      `gorm:"not null;index:idx_user_permissions_user_id" json:"user_id"`
	Permission string    `gorm:"type:varchar(100);not null;index:idx_user_permissions_permission" json:"permission"`
	CreatedAt  time.Time `json:"created_at"`

	User *User `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"user,omitempty"`
}

// TableName 指定表名。
func (UserPermission) TableName() string {
	return "user_permissions"
}
