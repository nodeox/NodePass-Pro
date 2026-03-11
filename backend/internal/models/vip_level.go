package models

import "time"

// VIPLevel VIP 等级模型（vip_levels 表）。
//
// Deprecated: 此模型已被重构为 DDD 架构。
// 新代码请使用 internal/domain/vip/entity.go 中的 VIPLevel 实体。
// 迁移指南：
//   - 领域层: internal/domain/vip/entity.go
//   - 应用层: internal/application/vip/commands 和 queries
//   - 基础设施: internal/infrastructure/persistence/postgres/vip/vip_repository.go
//   - 缓存层: internal/infrastructure/cache/vip_cache.go
// 此模型将在所有旧代码迁移完成后删除。
type VIPLevel struct {
	ID    uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	Level int    `gorm:"not null;uniqueIndex:uk_vip_levels_level" json:"level"`
	Name  string `gorm:"type:varchar(50);not null" json:"name"`

	Description *string `gorm:"type:text" json:"description"`

	TrafficQuota int64 `gorm:"type:bigint;not null;default:0" json:"traffic_quota"`
	MaxRules     int   `gorm:"not null;default:0" json:"max_rules"`
	MaxBandwidth int   `gorm:"not null;default:0" json:"max_bandwidth"`

	MaxSelfHostedEntryNodes int `gorm:"column:max_self_hosted_entry_nodes;not null;default:0" json:"max_self_hosted_entry_nodes"`
	MaxSelfHostedExitNodes  int `gorm:"column:max_self_hosted_exit_nodes;not null;default:0" json:"max_self_hosted_exit_nodes"`

	AccessibleNodeLevel int     `gorm:"column:accessible_node_level;not null;default:1" json:"accessible_node_level"`
	TrafficMultiplier   float64 `gorm:"column:traffic_multiplier;type:decimal(5,2);not null;default:1.0" json:"traffic_multiplier"`

	CustomFeatures *string  `gorm:"column:custom_features;type:text" json:"custom_features"`
	Price          *float64 `gorm:"type:decimal(10,2)" json:"price"`
	DurationDays   *int     `gorm:"column:duration_days" json:"duration_days"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名。
func (VIPLevel) TableName() string {
	return "vip_levels"
}
