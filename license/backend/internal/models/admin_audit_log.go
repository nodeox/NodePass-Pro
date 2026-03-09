package models

import "time"

// AdminAuditLog 管理员审计日志。
type AdminAuditLog struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	AdminID     uint      `gorm:"not null;index" json:"admin_id"`
	Action      string    `gorm:"size:64;not null;index" json:"action"`
	TargetType  string    `gorm:"size:64;not null;index" json:"target_type"`
	PayloadJSON string    `gorm:"type:text" json:"payload_json"`
	CreatedAt   time.Time `json:"created_at"`
}

func (AdminAuditLog) TableName() string {
	return "admin_audit_logs"
}
