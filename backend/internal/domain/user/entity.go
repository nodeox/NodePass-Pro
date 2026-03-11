package user

import (
	"errors"
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

// 领域错误
var (
	ErrUserNotFound      = errors.New("用户不存在")
	ErrEmailExists       = errors.New("邮箱已存在")
	ErrUsernameExists    = errors.New("用户名已存在")
	ErrInvalidPassword   = errors.New("密码不正确")
	ErrUserDisabled      = errors.New("用户已被禁用")
	ErrTrafficExceeded   = errors.New("流量已超限")
	ErrVIPExpired        = errors.New("VIP 已过期")
)

// IsActive 检查用户是否激活
func (u *User) IsActive() bool {
	return u.Status == "normal"
}

// IsAdmin 检查是否为管理员
func (u *User) IsAdmin() bool {
	return u.Role == "admin"
}

// IsVIP 检查是否为 VIP
func (u *User) IsVIP() bool {
	if u.VipLevel <= 0 {
		return false
	}
	if u.VipExpiresAt == nil {
		return true
	}
	return u.VipExpiresAt.After(time.Now())
}

// CanCreateTunnel 检查是否可以创建隧道
func (u *User) CanCreateTunnel(currentCount int) bool {
	if u.MaxRules < 0 {
		return true // 无限制
	}
	return currentCount < u.MaxRules
}

// HasTrafficQuota 检查是否有流量配额
func (u *User) HasTrafficQuota(required int64) bool {
	if u.TrafficQuota < 0 {
		return true // 无限制
	}
	return u.TrafficUsed+required <= u.TrafficQuota
}

// ConsumeTraffic 消耗流量
func (u *User) ConsumeTraffic(amount int64) error {
	if !u.HasTrafficQuota(amount) {
		return ErrTrafficExceeded
	}
	u.TrafficUsed += amount
	return nil
}
