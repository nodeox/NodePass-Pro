package queries

import (
	"context"

	"nodepass-pro/backend/internal/domain/benefitcode"
)

// ListCodesQuery 列表查询
type ListCodesQuery struct {
	Status   benefitcode.BenefitCodeStatus
	VIPLevel *int
	UsedBy   *uint
	Page     int
	PageSize int
}

// ListCodesResult 列表查询结果
type ListCodesResult struct {
	List     []*benefitcode.BenefitCode
	Total    int64
	Page     int
	PageSize int
}

// ListCodesHandler 列表查询处理器
type ListCodesHandler struct {
	repo benefitcode.Repository
}

// NewListCodesHandler 创建列表查询处理器
func NewListCodesHandler(repo benefitcode.Repository) *ListCodesHandler {
	return &ListCodesHandler{
		repo: repo,
	}
}

// Handle 处理列表查询
func (h *ListCodesHandler) Handle(ctx context.Context, query ListCodesQuery) (*ListCodesResult, error) {
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

	// 查询
	list, total, err := h.repo.List(ctx, benefitcode.ListFilter{
		Status:   query.Status,
		VIPLevel: query.VIPLevel,
		UsedBy:   query.UsedBy,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		return nil, err
	}

	return &ListCodesResult{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}
