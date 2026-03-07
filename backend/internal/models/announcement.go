package models

import "time"

// Announcement 公告模型（announcements 表）。
type Announcement struct {
	ID      uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	Title   string `gorm:"type:varchar(200);not null" json:"title"`
	Content string `gorm:"type:text;not null" json:"content"`
	Type    string `gorm:"type:varchar(20);not null;check:chk_announcements_type,type IN ('info','warning','error','success')" json:"type"`

	IsEnabled bool       `gorm:"column:is_enabled;not null;default:true;index:idx_announcements_is_enabled" json:"is_enabled"`
	StartTime *time.Time `gorm:"column:start_time;index:idx_announcements_start_time" json:"start_time"`
	EndTime   *time.Time `gorm:"column:end_time;index:idx_announcements_end_time" json:"end_time"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名。
func (Announcement) TableName() string {
	return "announcements"
}
