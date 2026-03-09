package models

import "time"

// LicensePlan 授权套餐。
type LicensePlan struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Code         string    `gorm:"size:64;not null;uniqueIndex" json:"code"`
	Name         string    `gorm:"size:255;not null" json:"name"`
	Description  string    `gorm:"type:text" json:"description"`
	MaxMachines  int       `gorm:"not null;default:1" json:"max_machines"`
	DurationDays int       `gorm:"not null;default:365" json:"duration_days"`
	Status       string    `gorm:"size:32;not null;default:active;index" json:"status"`
	LicenseCount int64     `gorm:"-" json:"license_count,omitempty"`
	ActiveCount  int64     `gorm:"-" json:"active_license_count,omitempty"`
	BindingCount int64     `gorm:"-" json:"activation_count,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (LicensePlan) TableName() string {
	return "license_plans"
}
