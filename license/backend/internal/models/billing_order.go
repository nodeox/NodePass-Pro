package models

import "time"

// BillingOrder 商业化订单。
type BillingOrder struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	OrderNo        string     `gorm:"size:64;not null;uniqueIndex" json:"order_no"`
	LicenseID      uint       `gorm:"not null;index" json:"license_id"`
	Action         string     `gorm:"size:32;not null;index" json:"action"`
	FromPlanID     *uint      `gorm:"index" json:"from_plan_id,omitempty"`
	ToPlanID       *uint      `gorm:"index" json:"to_plan_id,omitempty"`
	TargetCustomer string     `gorm:"size:255" json:"target_customer,omitempty"`
	PeriodDays     int        `json:"period_days"`
	AmountCents    int64      `gorm:"not null;default:0" json:"amount_cents"`
	Currency       string     `gorm:"size:16;not null;default:CNY" json:"currency"`
	Status         string     `gorm:"size:32;not null;default:pending;index" json:"status"`
	PaymentChannel string     `gorm:"size:32" json:"payment_channel"`
	PaymentTxnID   *string    `gorm:"size:128;uniqueIndex" json:"payment_txn_id,omitempty"`
	PayURL         string     `gorm:"type:text" json:"pay_url"`
	ExpireAt       *time.Time `gorm:"index" json:"expire_at,omitempty"`
	PaidAt         *time.Time `gorm:"index" json:"paid_at,omitempty"`
	CreatedBy      uint       `gorm:"index" json:"created_by"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func (BillingOrder) TableName() string {
	return "billing_orders"
}
