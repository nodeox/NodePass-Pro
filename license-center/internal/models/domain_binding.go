package models

import "time"

// LicenseDomainBinding 域名绑定历史
type LicenseDomainBinding struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	LicenseID  uint      `gorm:"not null;index" json:"license_id"`
	OldDomain  string    `gorm:"size:255" json:"old_domain"`
	NewDomain  string    `gorm:"size:255;not null" json:"new_domain"`
	Reason     string    `gorm:"type:text" json:"reason"`
	OperatorID uint      `gorm:"index" json:"operator_id"`
	IPAddress  string    `gorm:"size:64" json:"ip_address"`
	CreatedAt  time.Time `json:"created_at"`
}

func (LicenseDomainBinding) TableName() string {
	return "license_domain_bindings"
}

// DomainIPBinding 域名 IP 绑定记录
type DomainIPBinding struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Domain    string    `gorm:"size:255;not null;uniqueIndex" json:"domain"`
	IPAddress string    `gorm:"size:64;not null" json:"ip_address"`
	FirstSeen time.Time `json:"first_seen"`
	LastSeen  time.Time `json:"last_seen"`
	HitCount  int64     `gorm:"default:0" json:"hit_count"`
}

func (DomainIPBinding) TableName() string {
	return "domain_ip_bindings"
}
