package commands

import (
	"context"
	"fmt"

	"nodepass-pro/backend/internal/domain/benefitcode"
)

// DeleteCodesCommand 删除权益码命令
type DeleteCodesCommand struct {
	AdminID uint
	IDs     []uint
}

// DeleteCodesResult 删除权益码结果
type DeleteCodesResult struct {
	Deleted int64
}

// DeleteCodesHandler 删除权益码处理器
type DeleteCodesHandler struct {
	repo benefitcode.Repository
}

// NewDeleteCodesHandler 创建删除权益码处理器
func NewDeleteCodesHandler(repo benefitcode.Repository) *DeleteCodesHandler {
	return &DeleteCodesHandler{
		repo: repo,
	}
}

// Handle 处理删除权益码命令
func (h *DeleteCodesHandler) Handle(ctx context.Context, cmd DeleteCodesCommand) (*DeleteCodesResult, error) {
	if len(cmd.IDs) == 0 {
		return nil, fmt.Errorf("IDs 不能为空")
	}

	// 批量删除
	deleted, err := h.repo.BatchDelete(ctx, cmd.IDs)
	if err != nil {
		return nil, fmt.Errorf("删除权益码失败: %w", err)
	}

	return &DeleteCodesResult{
		Deleted: deleted,
	}, nil
}
