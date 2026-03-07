package models

import "time"

// BenefitCode 权益码模型（benefit_codes 表）。
type BenefitCode struct {
	ID           uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	Code         string `gorm:"type:varchar(50);not null;uniqueIndex:uk_benefit_codes_code" json:"code"`
	VipLevel     int    `gorm:"column:vip_level;not null" json:"vip_level"`
	DurationDays int    `gorm:"column:duration_days;not null" json:"duration_days"`

	Status    string `gorm:"type:varchar(20);not null;default:unused;index:idx_benefit_codes_status" json:"status"`
	IsEnabled bool   `gorm:"column:is_enabled;not null;default:true" json:"is_enabled"`

	UsedBy    *uint      `gorm:"column:used_by;index:idx_benefit_codes_used_by" json:"used_by"`
	UsedAt    *time.Time `gorm:"column:used_at" json:"used_at"`
	ExpiresAt *time.Time `gorm:"column:expires_at" json:"expires_at"`
	CreatedAt time.Time  `json:"created_at"`

	User *User `gorm:"foreignKey:UsedBy;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"user,omitempty"`
}

// TableName 指定表名。
func (BenefitCode) TableName() string {
	return "benefit_codes"
}
