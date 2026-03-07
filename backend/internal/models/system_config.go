package models

import "time"

// SystemConfig 系统配置模型（system_config 表）。
type SystemConfig struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Key         string    `gorm:"type:varchar(100);not null;uniqueIndex:uk_system_config_key" json:"key"`
	Value       *string   `gorm:"type:text" json:"value"`
	Description *string   `gorm:"type:text" json:"description"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName 指定表名。
func (SystemConfig) TableName() string {
	return "system_config"
}
