package models

import "time"

// VerifyLog 授权验证日志。
type VerifyLog struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	LicenseID         *uint     `gorm:"index" json:"license_id"`
	LicenseKey        string    `gorm:"size:64;not null;index" json:"license_key"`
	MachineID         string    `gorm:"size:128;not null;index" json:"machine_id"`
	Action            string    `gorm:"size:32;not null;index" json:"action"`
	Result            string    `gorm:"size:32;not null;index" json:"result"`
	Reason            string    `gorm:"type:text" json:"reason"`
	PanelVersion      string    `gorm:"size:32" json:"panel_version"`
	BackendVersion    string    `gorm:"size:32" json:"backend_version"`
	FrontendVersion   string    `gorm:"size:32" json:"frontend_version"`
	NodeclientVersion string    `gorm:"size:32" json:"nodeclient_version"`
	Branch            string    `gorm:"size:64" json:"branch"`
	Commit            string    `gorm:"size:64" json:"commit"`
	IPAddress         string    `gorm:"size:64" json:"ip_address"`
	UserAgent         string    `gorm:"size:255" json:"user_agent"`
	CreatedAt         time.Time `gorm:"index" json:"created_at"`
}

func (VerifyLog) TableName() string {
	return "verify_logs"
}
