package models

import "time"

// AdminUser 管理员用户。
type AdminUser struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	Username     string     `gorm:"size:64;not null;uniqueIndex" json:"username"`
	Email        string     `gorm:"size:128;not null;uniqueIndex" json:"email"`
	PasswordHash string     `gorm:"size:255;not null" json:"-"`
	Role         string     `gorm:"size:32;not null;default:admin" json:"role"`
	Status       string     `gorm:"size:32;not null;default:active;index" json:"status"`
	LastLoginAt  *time.Time `json:"last_login_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

func (AdminUser) TableName() string {
	return "admin_users"
}
