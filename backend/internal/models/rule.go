package models

import "time"

// Rule 规则模型（rules 表）。
type Rule struct {
	ID     uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID uint   `gorm:"not null;index:idx_rules_user_id;index:idx_rules_user_status,priority:1" json:"user_id"`
	Name   string `gorm:"type:varchar(100);not null" json:"name"`

	Mode     string `gorm:"type:varchar(20);not null;index:idx_rules_mode;check:chk_rules_mode_exit,((mode = 'single' AND exit_node_id IS NULL) OR (mode = 'tunnel' AND exit_node_id IS NOT NULL))" json:"mode"`
	Protocol string `gorm:"type:varchar(20);not null" json:"protocol"`

	EntryNodeID uint  `gorm:"column:entry_node_id;not null;index:idx_rules_entry_node_id" json:"entry_node_id"`
	ExitNodeID  *uint `gorm:"column:exit_node_id;index:idx_rules_exit_node_id" json:"exit_node_id"`

	TargetHost string `gorm:"column:target_host;type:varchar(255);not null" json:"target_host"`
	TargetPort int    `gorm:"column:target_port;not null" json:"target_port"`
	ListenHost string `gorm:"column:listen_host;type:varchar(255);not null;default:0.0.0.0" json:"listen_host"`
	ListenPort int    `gorm:"column:listen_port;not null" json:"listen_port"`

	Status     string `gorm:"type:varchar(20);not null;default:stopped;index:idx_rules_status;index:idx_rules_user_status,priority:2" json:"status"`
	SyncStatus string `gorm:"column:sync_status;type:varchar(20);not null;default:pending" json:"sync_status"`

	InstanceID     *string `gorm:"column:instance_id;type:varchar(100)" json:"instance_id"`
	InstanceStatus *string `gorm:"column:instance_status;type:text" json:"instance_status"`

	TrafficIn   int64 `gorm:"column:traffic_in;type:bigint;not null;default:0" json:"traffic_in"`
	TrafficOut  int64 `gorm:"column:traffic_out;type:bigint;not null;default:0" json:"traffic_out"`
	Connections int   `gorm:"not null;default:0" json:"connections"`

	ConfigJSON    *string `gorm:"column:config_json;type:text" json:"config_json"`
	ConfigVersion int     `gorm:"column:config_version;not null;default:0" json:"config_version"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	User      *User `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"user,omitempty"`
	EntryNode *Node `gorm:"foreignKey:EntryNodeID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;" json:"entry_node,omitempty"`
	ExitNode  *Node `gorm:"foreignKey:ExitNodeID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"exit_node,omitempty"`
}

// TableName 指定表名。
func (Rule) TableName() string {
	return "rules"
}
