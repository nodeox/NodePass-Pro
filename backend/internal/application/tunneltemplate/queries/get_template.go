package queries

import (
	"context"

	"nodepass-pro/backend/internal/domain/tunneltemplate"
)

// GetTemplateQuery 获取模板查询
type GetTemplateQuery struct {
	TemplateID uint
	UserID     uint
}

// GetTemplateHandler 获取模板处理器
type GetTemplateHandler struct {
	repo tunneltemplate.Repository
}

// NewGetTemplateHandler 创建处理器
func NewGetTemplateHandler(repo tunneltemplate.Repository) *GetTemplateHandler {
	return &GetTemplateHandler{repo: repo}
}

// Handle 处理查询
func (h *GetTemplateHandler) Handle(ctx context.Context, query GetTemplateQuery) (*tunneltemplate.TunnelTemplate, error) {
	// 查找模板
	template, err := h.repo.FindByID(ctx, query.TemplateID)
	if err != nil {
		return nil, err
	}

	// 检查访问权限
	if !template.CanBeAccessedBy(query.UserID) {
		return nil, tunneltemplate.ErrUnauthorized
	}

	return template, nil
}
