package models

import "time"

// LicenseTransferLog 授权转移日志。
type LicenseTransferLog struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	LicenseID    uint      `gorm:"not null;index" json:"license_id"`
	OrderID      *uint     `gorm:"index" json:"order_id,omitempty"`
	FromCustomer string    `gorm:"size:255;not null" json:"from_customer"`
	ToCustomer   string    `gorm:"size:255;not null" json:"to_customer"`
	Reason       string    `gorm:"type:text" json:"reason"`
	OperatorID   uint      `gorm:"index" json:"operator_id"`
	CreatedAt    time.Time `json:"created_at"`
}

func (LicenseTransferLog) TableName() string {
	return "license_transfer_logs"
}
