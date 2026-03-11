package queries

import (
	"context"

	"nodepass-pro/backend/internal/domain/tunneltemplate"
)

// ListTemplatesQuery 列表查询
type ListTemplatesQuery struct {
	UserID   uint
	Protocol *string
	IsPublic *bool
	Page     int
	PageSize int
}

// ListTemplatesHandler 列表查询处理器
type ListTemplatesHandler struct {
	repo tunneltemplate.Repository
}

// NewListTemplatesHandler 创建处理器
func NewListTemplatesHandler(repo tunneltemplate.Repository) *ListTemplatesHandler {
	return &ListTemplatesHandler{repo: repo}
}

// Handle 处理查询
func (h *ListTemplatesHandler) Handle(ctx context.Context, query ListTemplatesQuery) ([]*tunneltemplate.TunnelTemplate, int64, error) {
	// 设置默认值
	page := query.Page
	if page <= 0 {
		page = 1
	}

	pageSize := query.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}

	// 构建过滤条件
	filter := tunneltemplate.ListFilter{
		UserID:   query.UserID,
		Protocol: query.Protocol,
		IsPublic: query.IsPublic,
		Page:     page,
		PageSize: pageSize,
	}

	return h.repo.List(ctx, filter)
}
