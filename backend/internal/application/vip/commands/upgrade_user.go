package commands

import (
	"context"
	"fmt"
	"time"

	"nodepass-pro/backend/internal/domain/vip"
)

// UpgradeUserCommand 升级用户 VIP 命令
type UpgradeUserCommand struct {
	UserID       uint
	Level        int
	DurationDays int // 升级天数，0 表示永久
}

// UpgradeUserResult 升级用户 VIP 结果
type UpgradeUserResult struct {
	UserID       uint       `json:"user_id"`
	VIPLevel     int        `json:"vip_level"`
	VIPExpiresAt *time.Time `json:"vip_expires_at"`
}

// UpgradeUserHandler 升级用户 VIP 命令处理器
type UpgradeUserHandler struct {
	vipRepo vip.Repository
}

// NewUpgradeUserHandler 创建升级用户 VIP 命令处理器
func NewUpgradeUserHandler(vipRepo vip.Repository) *UpgradeUserHandler {
	return &UpgradeUserHandler{
		vipRepo: vipRepo,
	}
}

// Handle 处理升级用户 VIP 命令
func (h *UpgradeUserHandler) Handle(ctx context.Context, cmd UpgradeUserCommand) (*UpgradeUserResult, error) {
	// 1. 验证输入
	if cmd.UserID == 0 {
		return nil, fmt.Errorf("用户 ID 无效")
	}
	if cmd.Level < 0 {
		return nil, vip.ErrInvalidLevel
	}
	if cmd.DurationDays < 0 {
		return nil, fmt.Errorf("升级天数不能为负数")
	}

	// 2. 检查 VIP 等级是否存在
	_, err := h.vipRepo.FindLevelByLevel(ctx, cmd.Level)
	if err != nil {
		return nil, fmt.Errorf("VIP 等级不存在")
	}

	// 3. 计算过期时间
	var expiresAt *time.Time
	if cmd.DurationDays > 0 {
		expiry := time.Now().AddDate(0, 0, cmd.DurationDays)
		expiresAt = &expiry
	}
	// DurationDays == 0 表示永久 VIP，expiresAt 为 nil

	// 4. 升级用户 VIP
	if err := h.vipRepo.UpgradeUserVIP(ctx, cmd.UserID, cmd.Level, expiresAt); err != nil {
		return nil, fmt.Errorf("升级用户 VIP 失败: %w", err)
	}

	return &UpgradeUserResult{
		UserID:       cmd.UserID,
		VIPLevel:     cmd.Level,
		VIPExpiresAt: expiresAt,
	}, nil
}
