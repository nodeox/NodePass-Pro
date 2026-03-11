package queries

import (
	"context"
	"fmt"

	"nodepass-pro/backend/internal/domain/auth"
	"nodepass-pro/backend/internal/infrastructure/cache"
)

// GetUserQuery 获取用户查询
type GetUserQuery struct {
	UserID uint
}

// GetUserResult 获取用户结果
type GetUserResult struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	Status   string `json:"status"`

	VipLevel     int    `json:"vip_level"`
	VipExpiresAt *int64 `json:"vip_expires_at"` // Unix 时间戳

	TrafficQuota int64 `json:"traffic_quota"`
	TrafficUsed  int64 `json:"traffic_used"`

	MaxRules                int `json:"max_rules"`
	MaxBandwidth            int `json:"max_bandwidth"`
	MaxSelfHostedEntryNodes int `json:"max_self_hosted_entry_nodes"`
	MaxSelfHostedExitNodes  int `json:"max_self_hosted_exit_nodes"`

	TelegramID       *string `json:"telegram_id"`
	TelegramUsername *string `json:"telegram_username"`

	CreatedAt   int64  `json:"created_at"`   // Unix 时间戳
	UpdatedAt   int64  `json:"updated_at"`   // Unix 时间戳
	LastLoginAt *int64 `json:"last_login_at"` // Unix 时间戳
}

// GetUserHandler 获取用户查询处理器
type GetUserHandler struct {
	authRepo  auth.Repository
	userCache *cache.UserCache
}

// NewGetUserHandler 创建获取用户查询处理器
func NewGetUserHandler(authRepo auth.Repository, userCache *cache.UserCache) *GetUserHandler {
	return &GetUserHandler{
		authRepo:  authRepo,
		userCache: userCache,
	}
}

// Handle 处理获取用户查询
func (h *GetUserHandler) Handle(ctx context.Context, query GetUserQuery) (*GetUserResult, error) {
	// 1. 尝试从缓存获取
	// TODO: 实现缓存逻辑

	// 2. 从数据库获取
	user, err := h.authRepo.FindUserByID(ctx, query.UserID)
	if err != nil {
		return nil, fmt.Errorf("用户不存在")
	}

	// 3. 转换为结果
	result := &GetUserResult{
		ID:                      user.ID,
		Username:                user.Username,
		Email:                   user.Email,
		Role:                    user.Role,
		Status:                  user.Status,
		VipLevel:                user.VipLevel,
		TrafficQuota:            user.TrafficQuota,
		TrafficUsed:             user.TrafficUsed,
		MaxRules:                user.MaxRules,
		MaxBandwidth:            user.MaxBandwidth,
		MaxSelfHostedEntryNodes: user.MaxSelfHostedEntryNodes,
		MaxSelfHostedExitNodes:  user.MaxSelfHostedExitNodes,
		TelegramID:              user.TelegramID,
		TelegramUsername:        user.TelegramUsername,
		CreatedAt:               user.CreatedAt.Unix(),
		UpdatedAt:               user.UpdatedAt.Unix(),
	}

	if user.VipExpiresAt != nil {
		timestamp := user.VipExpiresAt.Unix()
		result.VipExpiresAt = &timestamp
	}

	if user.LastLoginAt != nil {
		timestamp := user.LastLoginAt.Unix()
		result.LastLoginAt = &timestamp
	}

	// 4. 缓存结果
	// TODO: 缓存用户信息

	return result, nil
}
