package models

import (
	"time"

	"gorm.io/gorm"
)

// ProductRelease 产品发布记录。
type ProductRelease struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	Product      string         `gorm:"size:64;not null;index:idx_product_channel" json:"product"`
	Version      string         `gorm:"size:64;not null;index" json:"version"`
	Channel      string         `gorm:"size:32;not null;default:stable;index:idx_product_channel" json:"channel"`
	IsMandatory  bool           `gorm:"not null;default:false" json:"is_mandatory"`
	ReleaseNotes string         `gorm:"type:text" json:"release_notes"`
	FileName     string         `gorm:"size:255" json:"file_name"`
	FilePath     string         `gorm:"size:1024" json:"-"`
	FileSize     int64          `gorm:"not null;default:0" json:"file_size"`
	FileSHA256   string         `gorm:"size:64" json:"file_sha256"`
	PublishedAt  *time.Time     `gorm:"index" json:"published_at"`
	IsActive     bool           `gorm:"not null;default:true;index" json:"is_active"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

func (ProductRelease) TableName() string {
	return "product_releases"
}
