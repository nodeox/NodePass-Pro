package commands

import (
	"context"
	"fmt"

	"nodepass-pro/backend/internal/domain/announcement"
)

// DeleteAnnouncementCommand 删除公告命令
type DeleteAnnouncementCommand struct {
	AdminUserID uint
	ID          uint
}

// DeleteAnnouncementHandler 删除公告处理器
type DeleteAnnouncementHandler struct {
	repo announcement.Repository
}

// NewDeleteAnnouncementHandler 创建处理器
func NewDeleteAnnouncementHandler(repo announcement.Repository) *DeleteAnnouncementHandler {
	return &DeleteAnnouncementHandler{repo: repo}
}

// Handle 处理命令
func (h *DeleteAnnouncementHandler) Handle(ctx context.Context, cmd DeleteAnnouncementCommand) error {
	// 验证公告是否存在
	_, err := h.repo.FindByID(ctx, cmd.ID)
	if err != nil {
		return err
	}

	// 删除
	if err := h.repo.Delete(ctx, cmd.ID); err != nil {
		return fmt.Errorf("删除公告失败: %w", err)
	}

	return nil
}
