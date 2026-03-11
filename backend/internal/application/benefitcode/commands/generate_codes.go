package commands

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"nodepass-pro/backend/internal/domain/benefitcode"
)

const benefitCodeCharset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

// GenerateCodesCommand 生成权益码命令
type GenerateCodesCommand struct {
	AdminID      uint
	VIPLevel     int
	DurationDays int
	Count        int
	ExpiresAt    *time.Time
}

// GenerateCodesResult 生成权益码结果
type GenerateCodesResult struct {
	Codes []*benefitcode.BenefitCode
	Total int
}

// GenerateCodesHandler 生成权益码处理器
type GenerateCodesHandler struct {
	repo benefitcode.Repository
}

// NewGenerateCodesHandler 创建生成权益码处理器
func NewGenerateCodesHandler(repo benefitcode.Repository) *GenerateCodesHandler {
	return &GenerateCodesHandler{
		repo: repo,
	}
}

// Handle 处理生成权益码命令
func (h *GenerateCodesHandler) Handle(ctx context.Context, cmd GenerateCodesCommand) (*GenerateCodesResult, error) {
	// 验证参数
	if cmd.Count <= 0 {
		return nil, benefitcode.ErrInvalidCount
	}
	if cmd.Count > 1000 {
		return nil, fmt.Errorf("%w: 单次最多生成 1000 个权益码", benefitcode.ErrInvalidCount)
	}
	if cmd.DurationDays <= 0 {
		return nil, benefitcode.ErrInvalidDuration
	}
	if cmd.VIPLevel < 0 {
		return nil, benefitcode.ErrInvalidVIPLevel
	}

	now := time.Now()
	if cmd.ExpiresAt != nil && cmd.ExpiresAt.Before(now) {
		return nil, fmt.Errorf("%w: 过期时间不能早于当前时间", benefitcode.ErrInvalidDuration)
	}

	// 生成权益码
	codes := make([]*benefitcode.BenefitCode, 0, cmd.Count)
	seen := make(map[string]struct{}, cmd.Count)

	for len(codes) < cmd.Count {
		code, err := generateBenefitCode()
		if err != nil {
			return nil, fmt.Errorf("生成权益码失败: %w", err)
		}

		// 检查重复
		if _, exists := seen[code]; exists {
			continue
		}
		seen[code] = struct{}{}

		// 创建实体
		entity := &benefitcode.BenefitCode{
			Code:         code,
			VIPLevel:     cmd.VIPLevel,
			DurationDays: cmd.DurationDays,
			Status:       benefitcode.BenefitCodeStatusUnused,
			IsEnabled:    true,
			ExpiresAt:    cmd.ExpiresAt,
			CreatedAt:    now,
			UpdatedAt:    now,
		}

		codes = append(codes, entity)
	}

	// 批量保存
	if err := h.repo.BatchCreate(ctx, codes); err != nil {
		return nil, fmt.Errorf("保存权益码失败: %w", err)
	}

	return &GenerateCodesResult{
		Codes: codes,
		Total: len(codes),
	}, nil
}

// generateBenefitCode 生成权益码字符串
func generateBenefitCode() (string, error) {
	raw := make([]byte, 12)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}

	chars := make([]byte, 12)
	for i, b := range raw {
		chars[i] = benefitCodeCharset[int(b)%len(benefitCodeCharset)]
	}

	return fmt.Sprintf("NP-%s-%s-%s",
		string(chars[0:4]),
		string(chars[4:8]),
		string(chars[8:12]),
	), nil
}
