package queries

import (
	"context"
	"strings"

	"nodepass-pro/backend/internal/domain/benefitcode"
)

// ValidateCodeQuery 验证权益码查询
type ValidateCodeQuery struct {
	Code string
}

// ValidateCodeResult 验证权益码结果
type ValidateCodeResult struct {
	Valid        bool
	Code         *benefitcode.BenefitCode
	ErrorMessage string
}

// ValidateCodeHandler 验证权益码处理器
type ValidateCodeHandler struct {
	repo benefitcode.Repository
}

// NewValidateCodeHandler 创建验证权益码处理器
func NewValidateCodeHandler(repo benefitcode.Repository) *ValidateCodeHandler {
	return &ValidateCodeHandler{
		repo: repo,
	}
}

// Handle 处理验证权益码查询
func (h *ValidateCodeHandler) Handle(ctx context.Context, query ValidateCodeQuery) (*ValidateCodeResult, error) {
	// 标准化权益码
	code := strings.ToUpper(strings.TrimSpace(query.Code))
	if code == "" {
		return &ValidateCodeResult{
			Valid:        false,
			ErrorMessage: "权益码不能为空",
		}, nil
	}

	// 查找权益码
	benefitCode, err := h.repo.FindByCode(ctx, code)
	if err != nil {
		if err == benefitcode.ErrBenefitCodeNotFound {
			return &ValidateCodeResult{
				Valid:        false,
				ErrorMessage: "权益码不存在",
			}, nil
		}
		return nil, err
	}

	// 验证权益码状态
	if !benefitCode.IsEnabled {
		return &ValidateCodeResult{
			Valid:        false,
			Code:         benefitCode,
			ErrorMessage: "权益码已禁用",
		}, nil
	}
	if benefitCode.IsUsed() {
		return &ValidateCodeResult{
			Valid:        false,
			Code:         benefitCode,
			ErrorMessage: "权益码已使用",
		}, nil
	}
	if benefitCode.IsRevoked() {
		return &ValidateCodeResult{
			Valid:        false,
			Code:         benefitCode,
			ErrorMessage: "权益码已撤销",
		}, nil
	}
	if benefitCode.IsExpired() {
		return &ValidateCodeResult{
			Valid:        false,
			Code:         benefitCode,
			ErrorMessage: "权益码已过期",
		}, nil
	}

	return &ValidateCodeResult{
		Valid: true,
		Code:  benefitCode,
	}, nil
}
