package announcement

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewAnnouncement(t *testing.T) {
	ann := NewAnnouncement("Test Title", "Test Content", AnnouncementTypeInfo)

	assert.Equal(t, "Test Title", ann.Title)
	assert.Equal(t, "Test Content", ann.Content)
	assert.Equal(t, AnnouncementTypeInfo, ann.Type)
	assert.True(t, ann.IsEnabled)
}

func TestAnnouncement_IsActive(t *testing.T) {
	ann := NewAnnouncement("Test", "Content", AnnouncementTypeInfo)

	// 默认启用且无时间限制
	assert.True(t, ann.IsActive())

	// 禁用后不活跃
	ann.Disable()
	assert.False(t, ann.IsActive())

	// 重新启用
	ann.Enable()
	assert.True(t, ann.IsActive())

	// 设置未来开始时间
	future := time.Now().Add(1 * time.Hour)
	ann.StartTime = &future
	assert.False(t, ann.IsActive())

	// 设置过去开始时间
	past := time.Now().Add(-1 * time.Hour)
	ann.StartTime = &past
	assert.True(t, ann.IsActive())

	// 设置过去结束时间
	ann.EndTime = &past
	assert.False(t, ann.IsActive())

	// 设置未来结束时间
	ann.EndTime = &future
	assert.True(t, ann.IsActive())
}

func TestAnnouncement_SetTimeRange(t *testing.T) {
	ann := NewAnnouncement("Test", "Content", AnnouncementTypeInfo)

	start := time.Now()
	end := start.Add(1 * time.Hour)

	err := ann.SetTimeRange(&start, &end)
	assert.NoError(t, err)
	assert.Equal(t, &start, ann.StartTime)
	assert.Equal(t, &end, ann.EndTime)
}

func TestAnnouncement_SetTimeRange_Invalid(t *testing.T) {
	ann := NewAnnouncement("Test", "Content", AnnouncementTypeInfo)

	start := time.Now()
	end := start.Add(-1 * time.Hour) // 结束时间早于开始时间

	err := ann.SetTimeRange(&start, &end)
	assert.ErrorIs(t, err, ErrInvalidTimeRange)
}

func TestIsValidType(t *testing.T) {
	assert.True(t, IsValidType(AnnouncementTypeInfo))
	assert.True(t, IsValidType(AnnouncementTypeWarning))
	assert.True(t, IsValidType(AnnouncementTypeError))
	assert.True(t, IsValidType(AnnouncementTypeSuccess))
	assert.False(t, IsValidType("invalid"))
}
