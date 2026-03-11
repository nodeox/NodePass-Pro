package announcement

import (
	"time"
)

// AnnouncementType 公告类型
type AnnouncementType string

const (
	AnnouncementTypeInfo    AnnouncementType = "info"
	AnnouncementTypeWarning AnnouncementType = "warning"
	AnnouncementTypeError   AnnouncementType = "error"
	AnnouncementTypeSuccess AnnouncementType = "success"
)

// Announcement 公告聚合根
type Announcement struct {
	ID        uint
	Title     string
	Content   string
	Type      AnnouncementType
	IsEnabled bool
	StartTime *time.Time
	EndTime   *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewAnnouncement 创建公告
func NewAnnouncement(title, content string, announcementType AnnouncementType) *Announcement {
	return &Announcement{
		Title:     title,
		Content:   content,
		Type:      announcementType,
		IsEnabled: true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// IsActive 检查公告是否在有效期内
func (a *Announcement) IsActive() bool {
	if !a.IsEnabled {
		return false
	}

	now := time.Now()

	// 检查开始时间
	if a.StartTime != nil && now.Before(*a.StartTime) {
		return false
	}

	// 检查结束时间
	if a.EndTime != nil && now.After(*a.EndTime) {
		return false
	}

	return true
}

// Enable 启用公告
func (a *Announcement) Enable() {
	a.IsEnabled = true
	a.UpdatedAt = time.Now()
}

// Disable 禁用公告
func (a *Announcement) Disable() {
	a.IsEnabled = false
	a.UpdatedAt = time.Now()
}

// UpdateInfo 更新基本信息
func (a *Announcement) UpdateInfo(title, content string, announcementType AnnouncementType) {
	a.Title = title
	a.Content = content
	a.Type = announcementType
	a.UpdatedAt = time.Now()
}

// SetTimeRange 设置时间范围
func (a *Announcement) SetTimeRange(startTime, endTime *time.Time) error {
	if startTime != nil && endTime != nil && endTime.Before(*startTime) {
		return ErrInvalidTimeRange
	}

	a.StartTime = startTime
	a.EndTime = endTime
	a.UpdatedAt = time.Now()
	return nil
}

// IsValidType 检查类型是否有效
func IsValidType(t AnnouncementType) bool {
	switch t {
	case AnnouncementTypeInfo, AnnouncementTypeWarning, AnnouncementTypeError, AnnouncementTypeSuccess:
		return true
	default:
		return false
	}
}
