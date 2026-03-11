package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"nodepass-pro/backend/internal/domain/announcement"
)

// UpdateAnnouncementCommand 更新公告命令
type UpdateAnnouncementCommand struct {
	AdminUserID uint
	ID          uint
	Title       *string
	Content     *string
	Type        *string
	IsEnabled   *bool
	StartTime   *time.Time
	EndTime     *time.Time
}

// UpdateAnnouncementHandler 更新公告处理器
type UpdateAnnouncementHandler struct {
	repo announcement.Repository
}

// NewUpdateAnnouncementHandler 创建处理器
func NewUpdateAnnouncementHandler(repo announcement.Repository) *UpdateAnnouncementHandler {
	return &UpdateAnnouncementHandler{repo: repo}
}

// Handle 处理命令
func (h *UpdateAnnouncementHandler) Handle(ctx context.Context, cmd UpdateAnnouncementCommand) (*announcement.Announcement, error) {
	// 查找公告
	ann, err := h.repo.FindByID(ctx, cmd.ID)
	if err != nil {
		return nil, err
	}

	// 更新基本信息
	if cmd.Title != nil || cmd.Content != nil || cmd.Type != nil {
		title := ann.Title
		if cmd.Title != nil {
			title = strings.TrimSpace(*cmd.Title)
			if title == "" {
				return nil, announcement.ErrInvalidTitle
			}
		}

		content := ann.Content
		if cmd.Content != nil {
			content = strings.TrimSpace(*cmd.Content)
			if content == "" {
				return nil, announcement.ErrInvalidContent
			}
		}

		announcementType := ann.Type
		if cmd.Type != nil {
			announcementType = announcement.AnnouncementType(strings.ToLower(strings.TrimSpace(*cmd.Type)))
			if !announcement.IsValidType(announcementType) {
				return nil, announcement.ErrInvalidType
			}
		}

		ann.UpdateInfo(title, content, announcementType)
	}

	// 更新启用状态
	if cmd.IsEnabled != nil {
		if *cmd.IsEnabled {
			ann.Enable()
		} else {
			ann.Disable()
		}
	}

	// 更新时间范围
	if cmd.StartTime != nil || cmd.EndTime != nil {
		startTime := ann.StartTime
		if cmd.StartTime != nil {
			startTime = cmd.StartTime
		}
		endTime := ann.EndTime
		if cmd.EndTime != nil {
			endTime = cmd.EndTime
		}
		if err := ann.SetTimeRange(startTime, endTime); err != nil {
			return nil, err
		}
	}

	// 保存
	if err := h.repo.Update(ctx, ann); err != nil {
		return nil, fmt.Errorf("更新公告失败: %w", err)
	}

	return ann, nil
}
