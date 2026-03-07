package models

import "time"

// User 用户模型（users 表）。
type User struct {
	ID       uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	Username string `gorm:"type:varchar(50);not null;uniqueIndex:uk_users_username" json:"username"`
	Email    string `gorm:"type:varchar(100);not null;uniqueIndex:uk_users_email" json:"email"`

	PasswordHash string `gorm:"column:password_hash;type:varchar(255);not null" json:"-"`
	Role         string `gorm:"type:varchar(20);not null;default:user" json:"role"`
	Status       string `gorm:"type:varchar(20);not null;default:normal;index:idx_users_status" json:"status"`

	VipLevel     int        `gorm:"column:vip_level;not null;default:0" json:"vip_level"`
	VipExpiresAt *time.Time `gorm:"column:vip_expires_at" json:"vip_expires_at"`

	TrafficQuota int64 `gorm:"type:bigint;not null;default:0" json:"traffic_quota"`
	TrafficUsed  int64 `gorm:"type:bigint;not null;default:0" json:"traffic_used"`

	MaxRules                int `gorm:"not null;default:5" json:"max_rules"`
	MaxBandwidth            int `gorm:"not null;default:100" json:"max_bandwidth"`
	MaxSelfHostedEntryNodes int `gorm:"column:max_self_hosted_entry_nodes;not null;default:0" json:"max_self_hosted_entry_nodes"`
	MaxSelfHostedExitNodes  int `gorm:"column:max_self_hosted_exit_nodes;not null;default:0" json:"max_self_hosted_exit_nodes"`

	TelegramID       *string `gorm:"column:telegram_id;type:varchar(50);uniqueIndex:uk_users_telegram_id" json:"telegram_id"`
	TelegramUsername *string `gorm:"column:telegram_username;type:varchar(100)" json:"telegram_username"`

	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	LastLoginAt *time.Time `gorm:"column:last_login_at" json:"last_login_at"`
}

// TableName 指定表名。
func (User) TableName() string {
	return "users"
}
