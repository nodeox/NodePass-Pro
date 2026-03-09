package models

import "time"

// LicenseGroup 授权码分组
type LicenseGroup struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	Name        string `gorm:"size:128;not null;uniqueIndex" json:"name"`
	Description string `gorm:"type:text" json:"description"`
	Type        string `gorm:"size:32;not null;default:project;index" json:"type"` // project, customer, custom

	// 分组配置
	Color       string `gorm:"size:16" json:"color"`
	Icon        string `gorm:"size:64" json:"icon"`
	SortOrder   int    `gorm:"default:0" json:"sort_order"`

	// 统计
	LicenseCount int  `gorm:"default:0" json:"license_count"`
	IsEnabled    bool `gorm:"default:true;index" json:"is_enabled"`

	CreatedBy uint      `gorm:"index" json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (LicenseGroup) TableName() string {
	return "license_groups"
}

// LicenseGroupMember 授权码分组成员关系
type LicenseGroupMember struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	GroupID   uint      `gorm:"not null;index:idx_group_license" json:"group_id"`
	LicenseID uint      `gorm:"not null;index:idx_group_license" json:"license_id"`
	CreatedAt time.Time `json:"created_at"`
}

func (LicenseGroupMember) TableName() string {
	return "license_group_members"
}
