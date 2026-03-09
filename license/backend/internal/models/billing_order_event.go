package models

import "time"

// BillingOrderEvent 订单事件日志。
type BillingOrderEvent struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	OrderID     uint      `gorm:"not null;index" json:"order_id"`
	EventType   string    `gorm:"size:64;not null;index" json:"event_type"`
	PayloadJSON string    `gorm:"type:text" json:"payload_json"`
	CreatedAt   time.Time `json:"created_at"`
}

func (BillingOrderEvent) TableName() string {
	return "billing_order_events"
}
