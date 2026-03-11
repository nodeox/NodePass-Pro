package queries

import (
	"context"

	"nodepass-pro/backend/internal/domain/benefitcode"
)

// GetCodeQuery 获取权益码查询
type GetCodeQuery struct {
	ID uint
}

// GetCodeHandler 获取权益码处理器
type GetCodeHandler struct {
	repo benefitcode.Repository
}

// NewGetCodeHandler 创建获取权益码处理器
func NewGetCodeHandler(repo benefitcode.Repository) *GetCodeHandler {
	return &GetCodeHandler{
		repo: repo,
	}
}

// Handle 处理获取权益码查询
func (h *GetCodeHandler) Handle(ctx context.Context, query GetCodeQuery) (*benefitcode.BenefitCode, error) {
	return h.repo.FindByID(ctx, query.ID)
}
