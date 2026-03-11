package commands

import (
	"context"
	"fmt"

	"nodepass-pro/backend/internal/domain/benefitcode"
)

// RevokeCodeCommand 撤销权益码命令
type RevokeCodeCommand struct {
	AdminID uint
	CodeID  uint
}

// RevokeCodeHandler 撤销权益码处理器
type RevokeCodeHandler struct {
	repo benefitcode.Repository
}

// NewRevokeCodeHandler 创建撤销权益码处理器
func NewRevokeCodeHandler(repo benefitcode.Repository) *RevokeCodeHandler {
	return &RevokeCodeHandler{
		repo: repo,
	}
}

// Handle 处理撤销权益码命令
func (h *RevokeCodeHandler) Handle(ctx context.Context, cmd RevokeCodeCommand) error {
	// 查找权益码
	code, err := h.repo.FindByID(ctx, cmd.CodeID)
	if err != nil {
		return err
	}

	// 撤销权益码
	if err := code.Revoke(); err != nil {
		return err
	}

	// 更新权益码
	if err := h.repo.Update(ctx, code); err != nil {
		return fmt.Errorf("更新权益码失败: %w", err)
	}

	return nil
}
