package models

import "time"

// LicenseReminder 到期提醒任务。
type LicenseReminder struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	LicenseID uint       `gorm:"not null;index" json:"license_id"`
	Stage     string     `gorm:"size:32;not null;index" json:"stage"`
	RemindAt  time.Time  `gorm:"not null;index" json:"remind_at"`
	Channel   string     `gorm:"size:32;not null;default:webhook" json:"channel"`
	Status    string     `gorm:"size:32;not null;default:pending;index" json:"status"`
	LastError string     `gorm:"type:text" json:"last_error"`
	SentAt    *time.Time `gorm:"index" json:"sent_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

func (LicenseReminder) TableName() string {
	return "license_reminders"
}
