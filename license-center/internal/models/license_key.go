package models

import "time"

// LicenseKey 授权码。
type LicenseKey struct {
	ID             uint        `gorm:"primaryKey" json:"id"`
	Key            string      `gorm:"size:64;not null;uniqueIndex" json:"key"`
	PlanID         uint        `gorm:"not null;index" json:"plan_id"`
	Plan           LicensePlan `json:"plan"`
	Customer       string      `gorm:"size:255;not null;index" json:"customer"`
	Status         string      `gorm:"size:32;not null;default:active;index" json:"status"`
	ExpiresAt      *time.Time  `gorm:"index" json:"expires_at"`
	MaxMachines    *int        `json:"max_machines"`
	Note           string      `gorm:"type:text" json:"note"`
	MetadataJSON   string      `gorm:"column:metadata_json;type:text" json:"metadata_json"`
	BoundDomain    string      `gorm:"size:255;index" json:"bound_domain"`
	DomainLocked   bool        `gorm:"default:false" json:"domain_locked"`
	DomainBoundAt  *time.Time  `json:"domain_bound_at"`
	AllowedDomains string      `gorm:"type:text" json:"allowed_domains"` // JSON array
	MaxDomains     int         `gorm:"default:1" json:"max_domains"`
	CreatedBy      uint        `gorm:"index" json:"created_by"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
}

func (LicenseKey) TableName() string {
	return "license_keys"
}
