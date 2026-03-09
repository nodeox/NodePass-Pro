package models

import "time"

// License 授权实例。
type License struct {
	ID           uint                `gorm:"primaryKey" json:"id"`
	Key          string              `gorm:"size:64;not null;uniqueIndex" json:"key"`
	PlanID       uint                `gorm:"not null;index" json:"plan_id"`
	Plan         LicensePlan         `json:"plan"`
	Customer     string              `gorm:"size:255;not null;index" json:"customer"`
	Status       string              `gorm:"size:32;not null;default:active;index" json:"status"`
	ExpiresAt    *time.Time          `gorm:"index" json:"expires_at"`
	MaxMachines  *int                `json:"max_machines"`
	MetadataJSON string              `gorm:"type:text" json:"metadata_json"`
	Note         string              `gorm:"type:text" json:"note"`
	CreatedBy    uint                `gorm:"index" json:"created_by"`
	Activations  []LicenseActivation `json:"activations,omitempty"`
	CreatedAt    time.Time           `json:"created_at"`
	UpdatedAt    time.Time           `json:"updated_at"`
}

func (License) TableName() string {
	return "licenses"
}
