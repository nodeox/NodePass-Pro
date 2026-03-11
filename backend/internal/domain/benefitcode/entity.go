package benefitcode

import (
	"time"
)

// BenefitCodeStatus 权益码状态
type BenefitCodeStatus string

const (
	BenefitCodeStatusUnused  BenefitCodeStatus = "unused"  // 未使用
	BenefitCodeStatusUsed    BenefitCodeStatus = "used"    // 已使用
	BenefitCodeStatusRevoked BenefitCodeStatus = "revoked" // 已撤销
)

// BenefitCode 权益码聚合根
type BenefitCode struct {
	ID           uint
	Code         string
	VIPLevel     int
	DurationDays int
	Status       BenefitCodeStatus
	IsEnabled    bool
	UsedBy       *uint
	UsedAt       *time.Time
	ExpiresAt    *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// IsUnused 是否未使用
func (b *BenefitCode) IsUnused() bool {
	return b.Status == BenefitCodeStatusUnused
}

// IsUsed 是否已使用
func (b *BenefitCode) IsUsed() bool {
	return b.Status == BenefitCodeStatusUsed
}

// IsRevoked 是否已撤销
func (b *BenefitCode) IsRevoked() bool {
	return b.Status == BenefitCodeStatusRevoked
}

// IsExpired 是否已过期
func (b *BenefitCode) IsExpired() bool {
	if b.ExpiresAt == nil {
		return false
	}
	return b.ExpiresAt.Before(time.Now())
}

// IsValid 是否有效（可以被兑换）
func (b *BenefitCode) IsValid() bool {
	return b.IsEnabled && b.IsUnused() && !b.IsExpired()
}

// MarkAsUsed 标记为已使用
func (b *BenefitCode) MarkAsUsed(userID uint) error {
	if !b.IsValid() {
		return ErrBenefitCodeInvalid
	}

	now := time.Now()
	b.Status = BenefitCodeStatusUsed
	b.UsedBy = &userID
	b.UsedAt = &now
	b.UpdatedAt = now

	return nil
}

// Revoke 撤销权益码
func (b *BenefitCode) Revoke() error {
	if b.IsUsed() {
		return ErrBenefitCodeAlreadyUsed
	}
	if b.IsRevoked() {
		return ErrBenefitCodeAlreadyRevoked
	}

	b.Status = BenefitCodeStatusRevoked
	b.IsEnabled = false
	b.UpdatedAt = time.Now()

	return nil
}

// Enable 启用权益码
func (b *BenefitCode) Enable() {
	b.IsEnabled = true
	b.UpdatedAt = time.Now()
}

// Disable 禁用权益码
func (b *BenefitCode) Disable() {
	b.IsEnabled = false
	b.UpdatedAt = time.Now()
}

// CalculateVIPExpiration 计算 VIP 过期时间
func (b *BenefitCode) CalculateVIPExpiration(currentExpiration *time.Time) time.Time {
	baseTime := time.Now()

	// 如果当前 VIP 未过期，从当前过期时间开始累加
	if currentExpiration != nil && currentExpiration.After(baseTime) {
		baseTime = *currentExpiration
	}

	return baseTime.AddDate(0, 0, b.DurationDays)
}
