package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"nodepass-pro/backend/internal/domain/announcement"
)

// CreateAnnouncementCommand 创建公告命令
type CreateAnnouncementCommand struct {
	AdminUserID uint
	Title       string
	Content     string
	Type        string
	IsEnabled   *bool
	StartTime   *time.Time
	EndTime     *time.Time
}

// CreateAnnouncementHandler 创建公告处理器
type CreateAnnouncementHandler struct {
	repo announcement.Repository
}

// NewCreateAnnouncementHandler 创建处理器
func NewCreateAnnouncementHandler(repo announcement.Repository) *CreateAnnouncementHandler {
	return &CreateAnnouncementHandler{repo: repo}
}

// Handle 处理命令
func (h *CreateAnnouncementHandler) Handle(ctx context.Context, cmd CreateAnnouncementCommand) (*announcement.Announcement, error) {
	// 验证标题
	title := strings.TrimSpace(cmd.Title)
	if title == "" {
		return nil, announcement.ErrInvalidTitle
	}

	// 验证内容
	content := strings.TrimSpace(cmd.Content)
	if content == "" {
		return nil, announcement.ErrInvalidContent
	}

	// 验证类型
	announcementType := announcement.AnnouncementType(strings.ToLower(strings.TrimSpace(cmd.Type)))
	if announcementType == "" {
		announcementType = announcement.AnnouncementTypeInfo
	}
	if !announcement.IsValidType(announcementType) {
		return nil, announcement.ErrInvalidType
	}

	// 验证时间范围
	if cmd.StartTime != nil && cmd.EndTime != nil && cmd.EndTime.Before(*cmd.StartTime) {
		return nil, announcement.ErrInvalidTimeRange
	}

	// 创建公告
	ann := announcement.NewAnnouncement(title, content, announcementType)
	ann.StartTime = cmd.StartTime
	ann.EndTime = cmd.EndTime

	// 设置启用状态
	if cmd.IsEnabled != nil && !*cmd.IsEnabled {
		ann.Disable()
	}

	// 保存
	if err := h.repo.Create(ctx, ann); err != nil {
		return nil, fmt.Errorf("创建公告失败: %w", err)
	}

	return ann, nil
}
