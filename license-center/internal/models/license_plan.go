package models

import "time"

// LicensePlan 授权套餐。
type LicensePlan struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	Name         string `gorm:"size:128;not null" json:"name"`
	Code         string `gorm:"size:64;not null;uniqueIndex" json:"code"`
	Description  string `gorm:"type:text" json:"description"`
	IsEnabled    bool   `gorm:"not null;default:true;index" json:"is_enabled"`
	MaxMachines  int    `gorm:"not null;default:1" json:"max_machines"`
	DurationDays int    `gorm:"not null;default:365" json:"duration_days"`

	MinPanelVersion      string `gorm:"size:32" json:"min_panel_version"`
	MaxPanelVersion      string `gorm:"size:32" json:"max_panel_version"`
	MinBackendVersion    string `gorm:"size:32" json:"min_backend_version"`
	MaxBackendVersion    string `gorm:"size:32" json:"max_backend_version"`
	MinFrontendVersion   string `gorm:"size:32" json:"min_frontend_version"`
	MaxFrontendVersion   string `gorm:"size:32" json:"max_frontend_version"`
	MinNodeclientVersion string `gorm:"size:32" json:"min_nodeclient_version"`
	MaxNodeclientVersion string `gorm:"size:32" json:"max_nodeclient_version"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (LicensePlan) TableName() string {
	return "license_plans"
}
