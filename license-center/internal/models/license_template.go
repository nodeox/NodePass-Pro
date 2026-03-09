package models

import "time"

// LicenseTemplate 授权码模板
type LicenseTemplate struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	Name        string     `gorm:"size:128;not null;uniqueIndex" json:"name"`
	Description string     `gorm:"type:text" json:"description"`
	PlanID      uint       `gorm:"not null;index" json:"plan_id"`
	Plan        LicensePlan `json:"plan"`

	// 模板配置
	DurationDays *int    `json:"duration_days"` // null 表示使用套餐默认值
	MaxMachines  *int    `json:"max_machines"`  // null 表示使用套餐默认值
	MaxDomains   *int    `json:"max_domains"`
	Prefix       string  `gorm:"size:16" json:"prefix"`
	Note         string  `gorm:"type:text" json:"note"`

	// 统计
	UsageCount  int  `gorm:"default:0" json:"usage_count"`
	IsEnabled   bool `gorm:"default:true;index" json:"is_enabled"`

	CreatedBy uint      `gorm:"index" json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (LicenseTemplate) TableName() string {
	return "license_templates"
}
