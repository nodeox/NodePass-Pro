package models

import "time"

// LicenseActivation 激活绑定记录。
type LicenseActivation struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	LicenseID       uint       `gorm:"not null;index:idx_license_machine,priority:1" json:"license_id"`
	License         LicenseKey `json:"license"`
	MachineID       string     `gorm:"size:128;not null;index:idx_license_machine,priority:2" json:"machine_id"`
	MachineName     string     `gorm:"size:255" json:"machine_name"`
	IPAddress       string     `gorm:"size:64" json:"ip_address"`
	FirstVerifiedAt time.Time  `json:"first_verified_at"`
	LastVerifiedAt  time.Time  `gorm:"index" json:"last_verified_at"`
	VerifyCount     int        `gorm:"not null;default:0" json:"verify_count"`
	IsActive        bool       `gorm:"not null;default:true;index" json:"is_active"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

func (LicenseActivation) TableName() string {
	return "license_activations"
}
