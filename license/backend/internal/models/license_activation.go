package models

import "time"

// LicenseActivation 授权绑定机器记录。
type LicenseActivation struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	LicenseID  uint      `gorm:"not null;uniqueIndex:idx_license_machine" json:"license_id"`
	MachineID  string    `gorm:"size:128;not null;uniqueIndex:idx_license_machine" json:"machine_id"`
	Hostname   string    `gorm:"size:255" json:"hostname"`
	IPAddress  string    `gorm:"size:64" json:"ip_address"`
	LastSeenAt time.Time `gorm:"not null;index" json:"last_seen_at"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (LicenseActivation) TableName() string {
	return "license_activations"
}
