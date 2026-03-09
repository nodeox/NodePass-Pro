package models

import "time"

// VerifyLog 统一校验日志。
type VerifyLog struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	LicenseID     *uint     `gorm:"index" json:"license_id"`
	LicenseKey    string    `gorm:"size:64;index" json:"license_key"`
	MachineID     string    `gorm:"size:128;index" json:"machine_id"`
	Product       string    `gorm:"size:64;index" json:"product"`
	ClientVersion string    `gorm:"size:64;index" json:"client_version"`
	Verified      bool      `gorm:"index" json:"verified"`
	Status        string    `gorm:"size:64;index" json:"status"`
	Reason        string    `gorm:"type:text" json:"reason"`
	ClientIP      string    `gorm:"size:64" json:"client_ip"`
	UserAgent     string    `gorm:"type:text" json:"user_agent"`
	CreatedAt     time.Time `gorm:"index" json:"created_at"`
}

func (VerifyLog) TableName() string {
	return "verify_logs"
}
