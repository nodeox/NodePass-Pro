package models

import (
	"time"

	"gorm.io/gorm"
)

// RefreshToken 刷新令牌模型
type RefreshToken struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联用户
	UserID uint  `gorm:"not null;index" json:"user_id"`
	User   *User `gorm:"foreignKey:UserID" json:"user,omitempty"`

	// Token 信息
	TokenHash string    `gorm:"type:varchar(64);uniqueIndex;not null" json:"-"` // SHA256 哈希
	ExpiresAt time.Time `gorm:"not null;index" json:"expires_at"`
	IsRevoked bool      `gorm:"default:false;index" json:"is_revoked"` // 是否已撤销

	// 元数据
	IPAddress string `gorm:"type:varchar(45)" json:"ip_address"` // 创建时的 IP
	UserAgent string `gorm:"type:varchar(512)" json:"user_agent"` // 创建时的 User-Agent
	LastUsedAt *time.Time `json:"last_used_at"` // 最后使用时间
}

// TableName 指定表名
func (RefreshToken) TableName() string {
	return "refresh_tokens"
}

// IsExpired 检查是否已过期
func (rt *RefreshToken) IsExpired() bool {
	return time.Now().After(rt.ExpiresAt)
}

// IsValid 检查是否有效（未过期且未撤销）
func (rt *RefreshToken) IsValid() bool {
	return !rt.IsExpired() && !rt.IsRevoked
}
