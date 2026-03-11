package queries

import (
	"context"

	"nodepass-pro/backend/internal/domain/systemconfig"
)

// GetAllConfigQuery 获取所有配置查询
type GetAllConfigQuery struct{}

// GetAllConfigHandler 获取所有配置处理器
type GetAllConfigHandler struct {
	repo systemconfig.Repository
}

// NewGetAllConfigHandler 创建处理器
func NewGetAllConfigHandler(repo systemconfig.Repository) *GetAllConfigHandler {
	return &GetAllConfigHandler{repo: repo}
}

// Handle 处理查询
func (h *GetAllConfigHandler) Handle(ctx context.Context, query GetAllConfigQuery) (map[string]string, error) {
	return h.repo.GetAllAsMap(ctx)
}
