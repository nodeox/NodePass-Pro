package models

import "time"

// TrafficRecord 流量记录模型（traffic_records 表）。
type TrafficRecord struct {
	ID     uint  `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID uint  `gorm:"not null;index:idx_traffic_records_user_hour,priority:1" json:"user_id"`
	RuleID *uint `gorm:"index:idx_traffic_records_rule_id" json:"rule_id"`
	NodeID *uint `gorm:"index:idx_traffic_records_node_id" json:"node_id"`

	TrafficIn  int64 `gorm:"column:traffic_in;type:bigint;not null;default:0" json:"traffic_in"`
	TrafficOut int64 `gorm:"column:traffic_out;type:bigint;not null;default:0" json:"traffic_out"`

	VipMultiplier   float64 `gorm:"column:vip_multiplier;type:decimal(5,2);not null;default:1.0" json:"vip_multiplier"`
	NodeMultiplier  float64 `gorm:"column:node_multiplier;type:decimal(5,2);not null;default:1.0" json:"node_multiplier"`
	FinalMultiplier float64 `gorm:"column:final_multiplier;type:decimal(5,2);not null;default:1.0" json:"final_multiplier"`

	CalculatedTraffic int64     `gorm:"column:calculated_traffic;type:bigint;not null;default:0" json:"calculated_traffic"`
	Hour              time.Time `gorm:"type:timestamp;not null;index:idx_traffic_records_user_hour,priority:2" json:"hour"`
	CreatedAt         time.Time `json:"created_at"`

	User *User `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"user,omitempty"`
	Rule *Rule `gorm:"foreignKey:RuleID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"rule,omitempty"`
	Node *Node `gorm:"foreignKey:NodeID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"node,omitempty"`
}

// TableName 指定表名。
func (TrafficRecord) TableName() string {
	return "traffic_records"
}
