package vip

import (
	"context"
	"time"
)

// Repository VIP 仓储接口
type Repository interface {
	// VIP 等级相关
	FindLevelByID(ctx context.Context, id uint) (*VIPLevel, error)
	FindLevelByLevel(ctx context.Context, level int) (*VIPLevel, error)
	FindByLevel(ctx context.Context, level int) (*VIPLevel, error) // 别名，与 FindLevelByLevel 相同
	ListLevels(ctx context.Context) ([]*VIPLevel, error)
	CreateLevel(ctx context.Context, level *VIPLevel) error
	UpdateLevel(ctx context.Context, level *VIPLevel) error
	DeleteLevel(ctx context.Context, id uint) error
	CheckLevelExists(ctx context.Context, level int) (bool, error)

	// 用户 VIP 相关
	GetUserVIP(ctx context.Context, userID uint) (*UserVIP, error)
	UpgradeUserVIP(ctx context.Context, userID uint, level int, expiresAt *time.Time) error
	CheckExpiredUsers(ctx context.Context) ([]uint, error)
	DegradeExpiredUsers(ctx context.Context, userIDs []uint) (int64, error)
}
