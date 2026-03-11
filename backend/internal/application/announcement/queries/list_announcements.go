package queries

import (
	"context"

	"nodepass-pro/backend/internal/domain/announcement"
)

// ListAnnouncementsQuery 列表查询
type ListAnnouncementsQuery struct {
	OnlyEnabled bool
}

// ListAnnouncementsHandler 列表查询处理器
type ListAnnouncementsHandler struct {
	repo announcement.Repository
}

// NewListAnnouncementsHandler 创建处理器
func NewListAnnouncementsHandler(repo announcement.Repository) *ListAnnouncementsHandler {
	return &ListAnnouncementsHandler{repo: repo}
}

// Handle 处理查询
func (h *ListAnnouncementsHandler) Handle(ctx context.Context, query ListAnnouncementsQuery) ([]*announcement.Announcement, error) {
	if query.OnlyEnabled {
		return h.repo.ListEnabled(ctx)
	}
	return h.repo.ListAll(ctx)
}
