package models

import "time"

// NodeConfig 节点配置模型（node_configs 表）。
type NodeConfig struct {
	ID            uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	NodeID        uint      `gorm:"not null;index:idx_node_configs_node_id_version,priority:1" json:"node_id"`
	ConfigVersion int       `gorm:"column:config_version;not null;index:idx_node_configs_node_id_version,priority:2" json:"config_version"`
	ConfigData    string    `gorm:"column:config_data;type:text;not null" json:"config_data"`
	CreatedAt     time.Time `json:"created_at"`

	Node *Node `gorm:"foreignKey:NodeID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"node,omitempty"`
}

// TableName 指定表名。
func (NodeConfig) TableName() string {
	return "node_configs"
}
