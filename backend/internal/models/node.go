package models

import "time"

// Node 节点模型（nodes 表）。
// 注意：节点不区分类型，无 type 字段。
type Node struct {
	ID     uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID uint   `gorm:"not null;index:idx_nodes_user_id" json:"user_id"`
	Name   string `gorm:"type:varchar(100);not null" json:"name"`
	Status string `gorm:"type:varchar(20);not null;default:offline;index:idx_nodes_status" json:"status"`

	Host   string  `gorm:"type:varchar(255);not null" json:"host"`
	Port   int     `gorm:"not null" json:"port"`
	Region *string `gorm:"type:varchar(50)" json:"region"`

	IsSelfHosted bool `gorm:"column:is_self_hosted;not null;default:false" json:"is_self_hosted"`
	IsPublic     bool `gorm:"column:is_public;not null;default:false" json:"is_public"`

	TrafficMultiplier float64 `gorm:"type:decimal(5,2);not null;default:1.0" json:"traffic_multiplier"`

	CpuUsage    *float64 `gorm:"column:cpu_usage;type:decimal(5,2)" json:"cpu_usage"`
	MemoryUsage *float64 `gorm:"column:memory_usage;type:decimal(5,2)" json:"memory_usage"`
	DiskUsage   *float64 `gorm:"column:disk_usage;type:decimal(5,2)" json:"disk_usage"`

	BandwidthIn  int64 `gorm:"column:bandwidth_in;type:bigint;not null;default:0" json:"bandwidth_in"`
	BandwidthOut int64 `gorm:"column:bandwidth_out;type:bigint;not null;default:0" json:"bandwidth_out"`
	Connections  int   `gorm:"not null;default:0" json:"connections"`

	TokenHash string `gorm:"column:token_hash;type:varchar(255);not null;uniqueIndex:uk_nodes_token_hash" json:"-"`

	ConfigVersion int     `gorm:"column:config_version;not null;default:0" json:"config_version"`
	Description   *string `gorm:"type:text" json:"description"`

	LastHeartbeatAt *time.Time `gorm:"column:last_heartbeat_at" json:"last_heartbeat_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`

	User *User `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"user,omitempty"`
}

// TableName 指定表名。
func (Node) TableName() string {
	return "nodes"
}
