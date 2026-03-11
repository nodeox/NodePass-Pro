package vip

import (
	"time"
)

// VIPLevel VIP 等级领域实体
type VIPLevel struct {
	ID    uint
	Level int
	Name  string

	Description *string

	TrafficQuota int64
	MaxRules     int
	MaxBandwidth int

	MaxSelfHostedEntryNodes int
	MaxSelfHostedExitNodes  int

	AccessibleNodeLevel int
	TrafficMultiplier   float64

	CustomFeatures *string
	Price          *float64
	DurationDays   *int

	CreatedAt time.Time
	UpdatedAt time.Time
}

// IsFree 是否是免费等级
func (v *VIPLevel) IsFree() bool {
	return v.Level == 0
}

// HasUnlimitedBandwidth 是否有无限带宽
func (v *VIPLevel) HasUnlimitedBandwidth() bool {
	return v.MaxBandwidth == -1
}

// HasUnlimitedRules 是否有无限规则数
func (v *VIPLevel) HasUnlimitedRules() bool {
	return v.MaxRules == -1
}

// CanAccessNodeLevel 是否可以访问指定等级的节点
func (v *VIPLevel) CanAccessNodeLevel(nodeLevel int) bool {
	return v.AccessibleNodeLevel >= nodeLevel
}

// CalculateTraffic 计算实际流量（应用流量倍率）
func (v *VIPLevel) CalculateTraffic(rawTraffic int64) int64 {
	if v.TrafficMultiplier <= 0 {
		return rawTraffic
	}
	return int64(float64(rawTraffic) * v.TrafficMultiplier)
}

// UserVIP 用户 VIP 信息聚合根
type UserVIP struct {
	UserID       uint
	VIPLevel     int
	VIPExpiresAt *time.Time
	LevelDetail  *VIPLevel
}

// IsActive VIP 是否激活
func (u *UserVIP) IsActive() bool {
	if u.VIPLevel == 0 {
		return false // 免费用户不算 VIP
	}
	if u.VIPExpiresAt == nil {
		return true // 永久 VIP
	}
	return time.Now().Before(*u.VIPExpiresAt)
}

// IsExpired VIP 是否过期
func (u *UserVIP) IsExpired() bool {
	if u.VIPExpiresAt == nil {
		return false // 永久 VIP 不过期
	}
	return time.Now().After(*u.VIPExpiresAt)
}

// DaysRemaining VIP 剩余天数
func (u *UserVIP) DaysRemaining() int {
	if u.VIPExpiresAt == nil {
		return -1 // 永久 VIP
	}
	if u.IsExpired() {
		return 0
	}
	duration := time.Until(*u.VIPExpiresAt)
	return int(duration.Hours() / 24)
}
