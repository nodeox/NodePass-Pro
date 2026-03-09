package models

import "time"

// AutoRenewalRule 自动续期规则
type AutoRenewalRule struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	Name        string `gorm:"size:128;not null" json:"name"`
	Description string `gorm:"type:text" json:"description"`

	// 触发条件
	TriggerType    string `gorm:"size:32;not null;default:before_expire" json:"trigger_type"` // before_expire, on_expire
	TriggerDays    int    `gorm:"default:7" json:"trigger_days"` // 提前多少天触发

	// 续期配置
	RenewalDays    int  `gorm:"not null;default:365" json:"renewal_days"`
	AutoApprove    bool `gorm:"default:false" json:"auto_approve"`
	NotifyCustomer bool `gorm:"default:true" json:"notify_customer"`

	// 应用范围
	PlanID    *uint  `gorm:"index" json:"plan_id"`    // null 表示所有套餐
	GroupID   *uint  `gorm:"index" json:"group_id"`   // null 表示所有分组
	Customer  string `gorm:"size:255;index" json:"customer"` // 空表示所有客户

	IsEnabled bool `gorm:"default:true;index" json:"is_enabled"`

	CreatedBy uint      `gorm:"index" json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (AutoRenewalRule) TableName() string {
	return "auto_renewal_rules"
}

// ExpiryNotification 过期通知记录
type ExpiryNotification struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	LicenseID  uint   `gorm:"not null;index" json:"license_id"`
	NotifyType string `gorm:"size:32;not null" json:"notify_type"` // email, webhook, internal
	NotifyDays int    `gorm:"not null" json:"notify_days"` // 提前多少天通知
	Status     string `gorm:"size:32;not null;default:pending" json:"status"` // pending, sent, failed
	SentAt     *time.Time `json:"sent_at"`
	ErrorMsg   string `gorm:"type:text" json:"error_msg"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (ExpiryNotification) TableName() string {
	return "expiry_notifications"
}

// SavedSearch 保存的搜索条件
type SavedSearch struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	Name        string `gorm:"size:128;not null" json:"name"`
	Description string `gorm:"type:text" json:"description"`

	// 搜索条件 (JSON)
	FilterJSON string `gorm:"column:filter_json;type:text;not null" json:"filter_json"`

	// 统计
	UsageCount int  `gorm:"default:0" json:"usage_count"`
	IsPublic   bool `gorm:"default:false" json:"is_public"`

	CreatedBy uint      `gorm:"index" json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (SavedSearch) TableName() string {
	return "saved_searches"
}
