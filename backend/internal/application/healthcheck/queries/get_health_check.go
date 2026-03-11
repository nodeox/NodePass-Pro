package queries

import (
	"context"

	"nodepass-pro/backend/internal/domain/healthcheck"
)

// GetHealthCheckQuery 获取健康检查配置查询
type GetHealthCheckQuery struct {
	NodeInstanceID uint
}

// GetHealthCheckHandler 获取健康检查配置处理器
type GetHealthCheckHandler struct {
	repo healthcheck.Repository
}

// NewGetHealthCheckHandler 创建处理器
func NewGetHealthCheckHandler(repo healthcheck.Repository) *GetHealthCheckHandler {
	return &GetHealthCheckHandler{repo: repo}
}

// Handle 处理查询
func (h *GetHealthCheckHandler) Handle(ctx context.Context, query GetHealthCheckQuery) (*healthcheck.HealthCheck, error) {
	return h.repo.FindHealthCheckByNodeInstance(ctx, query.NodeInstanceID)
}
