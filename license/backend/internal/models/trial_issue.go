package models

import "time"

// TrialIssue 试用发放记录。
type TrialIssue struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	LicenseID uint       `gorm:"not null;index" json:"license_id"`
	Customer  string     `gorm:"size:255;not null;index" json:"customer"`
	TrialDays int        `gorm:"not null" json:"trial_days"`
	IssuedBy  uint       `gorm:"index" json:"issued_by"`
	ExpiresAt *time.Time `gorm:"index" json:"expires_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

func (TrialIssue) TableName() string {
	return "trial_issues"
}
