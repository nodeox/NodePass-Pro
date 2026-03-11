package auth

import (
	"time"
)

// User 用户领域实体
type User struct {
	ID       uint
	Username string
	Email    string

	PasswordHash string
	Role         string
	Status       string

	VipLevel     int
	VipExpiresAt *time.Time

	TrafficQuota int64
	TrafficUsed  int64

	MaxRules                int
	MaxBandwidth            int
	MaxSelfHostedEntryNodes int
	MaxSelfHostedExitNodes  int

	TelegramID       *string
	TelegramUsername *string

	CreatedAt   time.Time
	UpdatedAt   time.Time
	LastLoginAt *time.Time
}

// IsActive 用户是否激活
func (u *User) IsActive() bool {
	return u.Status == "normal"
}

// IsBanned 用户是否被封禁
func (u *User) IsBanned() bool {
	return u.Status == "banned"
}

// IsVIP 用户是否是 VIP
func (u *User) IsVIP() bool {
	return u.VipLevel > 0
}

// IsVIPExpired VIP 是否过期
func (u *User) IsVIPExpired() bool {
	if u.VipExpiresAt == nil {
		return false
	}
	return time.Now().After(*u.VipExpiresAt)
}

// RefreshToken 刷新令牌领域实体
type RefreshToken struct {
	ID        uint
	UserID    uint
	TokenHash string

	IPAddress string
	UserAgent string

	ExpiresAt  time.Time
	LastUsedAt *time.Time
	IsRevoked  bool

	CreatedAt time.Time
	UpdatedAt time.Time
}

// IsValid 刷新令牌是否有效
func (rt *RefreshToken) IsValid() bool {
	if rt.IsRevoked {
		return false
	}
	if time.Now().After(rt.ExpiresAt) {
		return false
	}
	return true
}

// IsExpired 刷新令牌是否过期
func (rt *RefreshToken) IsExpired() bool {
	return time.Now().After(rt.ExpiresAt)
}
