package models

import "time"

// VersionPolicy 版本策略（和授权校验一体化使用）。
type VersionPolicy struct {
	ID                  uint      `gorm:"primaryKey" json:"id"`
	Product             string    `gorm:"size:64;not null;index:idx_policy_product_channel" json:"product"`
	Channel             string    `gorm:"size:32;not null;default:stable;index:idx_policy_product_channel" json:"channel"`
	MinSupportedVersion string    `gorm:"size:64;not null" json:"min_supported_version"`
	RecommendedVersion  string    `gorm:"size:64" json:"recommended_version"`
	Message             string    `gorm:"type:text" json:"message"`
	IsActive            bool      `gorm:"not null;default:true;index" json:"is_active"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

func (VersionPolicy) TableName() string {
	return "version_policies"
}
