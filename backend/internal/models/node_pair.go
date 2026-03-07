package models

import "time"

// NodePair 节点配对模型（node_pairs 表）。
type NodePair struct {
	ID     uint `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID uint `gorm:"not null;index:idx_node_pairs_user_id;uniqueIndex:uk_node_pairs_user_entry_exit" json:"user_id"`

	EntryNodeID uint `gorm:"column:entry_node_id;not null;index:idx_node_pairs_entry_node_id;uniqueIndex:uk_node_pairs_user_entry_exit" json:"entry_node_id"`
	ExitNodeID  uint `gorm:"column:exit_node_id;not null;index:idx_node_pairs_exit_node_id;uniqueIndex:uk_node_pairs_user_entry_exit" json:"exit_node_id"`

	Name        *string   `gorm:"type:varchar(100)" json:"name"`
	IsEnabled   bool      `gorm:"column:is_enabled;not null;default:true" json:"is_enabled"`
	Description *string   `gorm:"type:text" json:"description"`
	CreatedAt   time.Time `json:"created_at"`

	User      *User `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"user,omitempty"`
	EntryNode *Node `gorm:"foreignKey:EntryNodeID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;" json:"entry_node,omitempty"`
	ExitNode  *Node `gorm:"foreignKey:ExitNodeID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;" json:"exit_node,omitempty"`
}

// TableName 指定表名。
func (NodePair) TableName() string {
	return "node_pairs"
}
