package systemconfig

import (
	"time"
)

// SystemConfig 系统配置聚合根
type SystemConfig struct {
	ID          uint
	Key         string
	Value       *string
	Description *string
	UpdatedAt   time.Time
}

// NewSystemConfig 创建系统配置
func NewSystemConfig(key string, value *string) *SystemConfig {
	return &SystemConfig{
		Key:       key,
		Value:     value,
		UpdatedAt: time.Now(),
	}
}

// UpdateValue 更新值
func (c *SystemConfig) UpdateValue(value *string) {
	c.Value = value
	c.UpdatedAt = time.Now()
}

// SetDescription 设置描述
func (c *SystemConfig) SetDescription(description *string) {
	c.Description = description
	c.UpdatedAt = time.Now()
}

// GetValueOrDefault 获取值或默认值
func (c *SystemConfig) GetValueOrDefault(defaultValue string) string {
	if c.Value == nil {
		return defaultValue
	}
	return *c.Value
}
