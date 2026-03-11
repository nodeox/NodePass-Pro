package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"nodepass-pro/backend/internal/domain/benefitcode"
)

// RedeemCodeCommand 兑换权益码命令
type RedeemCodeCommand struct {
	UserID            uint
	Code              string
	CurrentVIPLevel   int
	CurrentVIPExpires *time.Time
}

// RedeemCodeResult 兑换权益码结果
type RedeemCodeResult struct {
	Code         string
	AppliedLevel int
	VIPExpiresAt time.Time
	DurationDays int
}

// RedeemCodeHandler 兑换权益码处理器
type RedeemCodeHandler struct {
	repo benefitcode.Repository
}

// NewRedeemCodeHandler 创建兑换权益码处理器
func NewRedeemCodeHandler(repo benefitcode.Repository) *RedeemCodeHandler {
	return &RedeemCodeHandler{
		repo: repo,
	}
}

// Handle 处理兑换权益码命令
func (h *RedeemCodeHandler) Handle(ctx context.Context, cmd RedeemCodeCommand) (*RedeemCodeResult, error) {
	// 标准化权益码
	code := strings.ToUpper(strings.TrimSpace(cmd.Code))
	if code == "" {
		return nil, benefitcode.ErrBenefitCodeInvalid
	}

	// 查找权益码
	benefitCode, err := h.repo.FindByCode(ctx, code)
	if err != nil {
		return nil, err
	}

	// 验证权益码状态
	if !benefitCode.IsEnabled {
		return nil, benefitcode.ErrBenefitCodeDisabled
	}
	if benefitCode.IsUsed() {
		return nil, benefitcode.ErrBenefitCodeAlreadyUsed
	}
	if benefitCode.IsRevoked() {
		return nil, benefitcode.ErrBenefitCodeAlreadyRevoked
	}
	if benefitCode.IsExpired() {
		return nil, benefitcode.ErrBenefitCodeExpired
	}

	// 计算应用的 VIP 等级（取最高等级）
	appliedLevel := benefitCode.VIPLevel
	if cmd.CurrentVIPLevel > appliedLevel {
		appliedLevel = cmd.CurrentVIPLevel
	}

	// 计算 VIP 过期时间
	vipExpiresAt := benefitCode.CalculateVIPExpiration(cmd.CurrentVIPExpires)

	// 标记为已使用
	if err := benefitCode.MarkAsUsed(cmd.UserID); err != nil {
		return nil, err
	}

	// 更新权益码
	if err := h.repo.Update(ctx, benefitCode); err != nil {
		return nil, fmt.Errorf("更新权益码失败: %w", err)
	}

	return &RedeemCodeResult{
		Code:         benefitCode.Code,
		AppliedLevel: appliedLevel,
		VIPExpiresAt: vipExpiresAt,
		DurationDays: benefitCode.DurationDays,
	}, nil
}
