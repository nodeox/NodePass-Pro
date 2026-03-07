package models

import "time"

// AuditLog 审计日志模型（audit_logs 表）。
type AuditLog struct {
	ID     uint  `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID *uint `gorm:"index:idx_audit_logs_user_id" json:"user_id"`

	Action       string  `gorm:"type:varchar(100);not null;index:idx_audit_logs_action" json:"action"`
	ResourceType *string `gorm:"column:resource_type;type:varchar(50);index:idx_audit_logs_resource_type" json:"resource_type"`
	ResourceID   *uint   `gorm:"column:resource_id" json:"resource_id"`
	Details      *string `gorm:"type:text" json:"details"`
	IPAddress    *string `gorm:"column:ip_address;type:varchar(50)" json:"ip_address"`
	UserAgent    *string `gorm:"column:user_agent;type:text" json:"user_agent"`

	CreatedAt time.Time `gorm:"index:idx_audit_logs_created_at" json:"created_at"`

	User *User `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"user,omitempty"`
}

// TableName 指定表名。
func (AuditLog) TableName() string {
	return "audit_logs"
}
