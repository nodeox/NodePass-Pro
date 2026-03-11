package queries

import (
	"context"
	"fmt"
	"time"

	"nodepass-pro/backend/internal/domain/vip"
)

// GetMyLevelQuery 获取当前用户 VIP 等级查询
type GetMyLevelQuery struct {
	UserID uint
}

// GetMyLevelResult 获取当前用户 VIP 等级结果
type GetMyLevelResult struct {
	UserID        uint            `json:"user_id"`
	VIPLevel      int             `json:"vip_level"`
	VIPExpiresAt  *time.Time      `json:"vip_expires_at"`
	IsActive      bool            `json:"is_active"`
	IsExpired     bool            `json:"is_expired"`
	DaysRemaining int             `json:"days_remaining"`
	LevelDetail   *VIPLevelInfo   `json:"level_detail"`
}

// GetMyLevelHandler 获取当前用户 VIP 等级查询处理器
type GetMyLevelHandler struct {
	vipRepo vip.Repository
}

// NewGetMyLevelHandler 创建获取当前用户 VIP 等级查询处理器
func NewGetMyLevelHandler(vipRepo vip.Repository) *GetMyLevelHandler {
	return &GetMyLevelHandler{
		vipRepo: vipRepo,
	}
}

// Handle 处理获取当前用户 VIP 等级查询
func (h *GetMyLevelHandler) Handle(ctx context.Context, query GetMyLevelQuery) (*GetMyLevelResult, error) {
	if query.UserID == 0 {
		return nil, fmt.Errorf("用户 ID 无效")
	}

	userVIP, err := h.vipRepo.GetUserVIP(ctx, query.UserID)
	if err != nil {
		return nil, err
	}

	result := &GetMyLevelResult{
		UserID:        userVIP.UserID,
		VIPLevel:      userVIP.VIPLevel,
		VIPExpiresAt:  userVIP.VIPExpiresAt,
		IsActive:      userVIP.IsActive(),
		IsExpired:     userVIP.IsExpired(),
		DaysRemaining: userVIP.DaysRemaining(),
	}

	if userVIP.LevelDetail != nil {
		result.LevelDetail = &VIPLevelInfo{
			ID:                      userVIP.LevelDetail.ID,
			Level:                   userVIP.LevelDetail.Level,
			Name:                    userVIP.LevelDetail.Name,
			Description:             userVIP.LevelDetail.Description,
			TrafficQuota:            userVIP.LevelDetail.TrafficQuota,
			MaxRules:                userVIP.LevelDetail.MaxRules,
			MaxBandwidth:            userVIP.LevelDetail.MaxBandwidth,
			MaxSelfHostedEntryNodes: userVIP.LevelDetail.MaxSelfHostedEntryNodes,
			MaxSelfHostedExitNodes:  userVIP.LevelDetail.MaxSelfHostedExitNodes,
			AccessibleNodeLevel:     userVIP.LevelDetail.AccessibleNodeLevel,
			TrafficMultiplier:       userVIP.LevelDetail.TrafficMultiplier,
			CustomFeatures:          userVIP.LevelDetail.CustomFeatures,
			Price:                   userVIP.LevelDetail.Price,
			DurationDays:            userVIP.LevelDetail.DurationDays,
		}
	}

	return result, nil
}
