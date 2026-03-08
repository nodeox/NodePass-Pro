package models

import "time"

// LicenseTag 授权码标签
type LicenseTag struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:64;not null;uniqueIndex" json:"name"`
	Color     string    `gorm:"size:32" json:"color"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (LicenseTag) TableName() string {
	return "license_tags"
}

// LicenseKeyTag 授权码与标签关联
type LicenseKeyTag struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	LicenseKeyID uint      `gorm:"not null;index" json:"license_key_id"`
	TagID        uint      `gorm:"not null;index" json:"tag_id"`
	CreatedAt    time.Time `json:"created_at"`
}

func (LicenseKeyTag) TableName() string {
	return "license_key_tags"
}

// WebhookConfig Webhook 配置
type WebhookConfig struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:128;not null" json:"name"`
	URL       string    `gorm:"size:512;not null" json:"url"`
	Secret    string    `gorm:"size:128" json:"secret"`
	Events    string    `gorm:"type:text" json:"events"` // JSON array
	IsEnabled bool      `gorm:"not null;default:true" json:"is_enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (WebhookConfig) TableName() string {
	return "webhook_configs"
}

// WebhookLog Webhook 日志
type WebhookLog struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	WebhookID    uint      `gorm:"not null;index" json:"webhook_id"`
	Event        string    `gorm:"size:64;not null;index" json:"event"`
	Payload      string    `gorm:"type:text" json:"payload"`
	Response     string    `gorm:"type:text" json:"response"`
	StatusCode   int       `json:"status_code"`
	Success      bool      `gorm:"index" json:"success"`
	ErrorMessage string    `gorm:"type:text" json:"error_message"`
	CreatedAt    time.Time `json:"created_at"`
}

func (WebhookLog) TableName() string {
	return "webhook_logs"
}

// Alert 告警记录
type Alert struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Type        string    `gorm:"size:64;not null;index" json:"type"` // license_expiring, license_expired, quota_exceeded, etc.
	Level       string    `gorm:"size:32;not null;index" json:"level"` // info, warning, error, critical
	Title       string    `gorm:"size:255;not null" json:"title"`
	Message     string    `gorm:"type:text" json:"message"`
	LicenseID   *uint     `gorm:"index" json:"license_id"`
	IsRead      bool      `gorm:"not null;default:false;index" json:"is_read"`
	IsSent      bool      `gorm:"not null;default:false" json:"is_sent"`
	MetadataJSON string   `gorm:"column:metadata_json;type:text" json:"metadata_json"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Alert) TableName() string {
	return "alerts"
}

// LicenseTransferLog 授权码转移日志
type LicenseTransferLog struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	LicenseID    uint      `gorm:"not null;index" json:"license_id"`
	FromCustomer string    `gorm:"size:255;not null" json:"from_customer"`
	ToCustomer   string    `gorm:"size:255;not null" json:"to_customer"`
	Reason       string    `gorm:"type:text" json:"reason"`
	OperatorID   uint      `gorm:"index" json:"operator_id"`
	CreatedAt    time.Time `json:"created_at"`
}

func (LicenseTransferLog) TableName() string {
	return "license_transfer_logs"
}
