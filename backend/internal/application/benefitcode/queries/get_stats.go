package queries

import (
	"context"

	"nodepass-pro/backend/internal/domain/benefitcode"
)

// GetStatsQuery 获取统计查询
type GetStatsQuery struct {
	// 可以扩展过滤条件
}

// GetStatsResult 统计结果
type GetStatsResult struct {
	TotalCodes   int64
	UnusedCodes  int64
	UsedCodes    int64
	RevokedCodes int64
}

// GetStatsHandler 获取统计处理器
type GetStatsHandler struct {
	repo benefitcode.Repository
}

// NewGetStatsHandler 创建获取统计处理器
func NewGetStatsHandler(repo benefitcode.Repository) *GetStatsHandler {
	return &GetStatsHandler{
		repo: repo,
	}
}

// Handle 处理获取统计查询
func (h *GetStatsHandler) Handle(ctx context.Context, query GetStatsQuery) (*GetStatsResult, error) {
	// 统计各状态的权益码数量
	unusedCount, err := h.repo.CountByStatus(ctx, benefitcode.BenefitCodeStatusUnused)
	if err != nil {
		return nil, err
	}

	usedCount, err := h.repo.CountByStatus(ctx, benefitcode.BenefitCodeStatusUsed)
	if err != nil {
		return nil, err
	}

	revokedCount, err := h.repo.CountByStatus(ctx, benefitcode.BenefitCodeStatusRevoked)
	if err != nil {
		return nil, err
	}

	return &GetStatsResult{
		TotalCodes:   unusedCount + usedCount + revokedCount,
		UnusedCodes:  unusedCount,
		UsedCodes:    usedCount,
		RevokedCodes: revokedCount,
	}, nil
}
