package models

import "time"

// VersionSyncConfig GitHub 版本镜像同步配置。
type VersionSyncConfig struct {
	ID                uint       `gorm:"primaryKey" json:"id"`
	Product           string     `gorm:"size:64;not null;uniqueIndex:idx_version_sync_product" json:"product"`
	Enabled           bool       `gorm:"not null;default:false" json:"enabled"`
	AutoSync          bool       `gorm:"not null;default:false" json:"auto_sync"`
	IntervalMinutes   int        `gorm:"not null;default:60" json:"interval_minutes"`
	GitHubOwner       string     `gorm:"size:255" json:"github_owner"`
	GitHubRepo        string     `gorm:"size:255" json:"github_repo"`
	GitHubToken       string     `gorm:"size:512" json:"-"`
	Channel           string     `gorm:"size:32;not null;default:stable" json:"channel"`
	IncludePrerelease bool       `gorm:"not null;default:false" json:"include_prerelease"`
	APIBaseURL        string     `gorm:"size:255;not null;default:https://api.github.com" json:"api_base_url"`
	LastSyncAt        *time.Time `json:"last_sync_at"`
	LastSyncStatus    string     `gorm:"size:32" json:"last_sync_status"`
	LastSyncMessage   string     `gorm:"size:255" json:"last_sync_message"`
	LastSyncedCount   int        `gorm:"not null;default:0" json:"last_synced_count"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

func (VersionSyncConfig) TableName() string {
	return "version_sync_configs"
}
