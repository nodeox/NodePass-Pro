package models

import "time"

// ComponentType 组件类型
type ComponentType string

const (
	ComponentTypeBackend       ComponentType = "backend"
	ComponentTypeFrontend      ComponentType = "frontend"
	ComponentTypeNodeClient    ComponentType = "node_client"
	ComponentTypeLicenseCenter ComponentType = "license_center"
)

// ComponentVersion 组件版本信息
type ComponentVersion struct {
	ID          uint          `gorm:"primaryKey" json:"id"`
	Component   ComponentType `gorm:"type:varchar(50);not null;uniqueIndex" json:"component"`
	Version     string        `gorm:"type:varchar(50);not null" json:"version"`
	BuildTime   *time.Time    `json:"build_time,omitempty"`
	GitCommit   string        `gorm:"type:varchar(100)" json:"git_commit,omitempty"`
	GitBranch   string        `gorm:"type:varchar(100)" json:"git_branch,omitempty"`
	Description string        `gorm:"type:text" json:"description,omitempty"`
	IsActive    bool          `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// TableName 指定表名
func (ComponentVersion) TableName() string {
	return "component_versions"
}

// VersionCompatibility 版本兼容性配置
type VersionCompatibility struct {
	ID                      uint      `gorm:"primaryKey" json:"id"`
	BackendVersion          string    `gorm:"type:varchar(50);not null;uniqueIndex" json:"backend_version"`
	MinFrontendVersion      string    `gorm:"type:varchar(50);not null" json:"min_frontend_version"`
	MinNodeClientVersion    string    `gorm:"type:varchar(50);not null" json:"min_node_client_version"`
	MinLicenseCenterVersion string    `gorm:"type:varchar(50);not null" json:"min_license_center_version"`
	Description             string    `gorm:"type:text" json:"description,omitempty"`
	IsActive                bool      `gorm:"default:true" json:"is_active"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
}

// TableName 指定表名
func (VersionCompatibility) TableName() string {
	return "version_compatibility"
}
