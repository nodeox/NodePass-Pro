package commands

import (
	"context"
	"fmt"
	"time"

	"nodepass-pro/backend/internal/domain/auth"
	"nodepass-pro/backend/internal/domain/vip"
	"nodepass-pro/backend/internal/infrastructure/cache"

	"golang.org/x/crypto/bcrypt"
)

// RegisterCommand 注册命令
type RegisterCommand struct {
	Username string
	Email    string
	Password string
}

// RegisterResult 注册结果
type RegisterResult struct {
	UserID    uint      `json:"user_id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// RegisterHandler 注册命令处理器
type RegisterHandler struct {
	authRepo  auth.Repository
	vipRepo   vip.Repository
	userCache *cache.UserCache
}

// NewRegisterHandler 创建注册命令处理器
func NewRegisterHandler(authRepo auth.Repository, vipRepo vip.Repository, userCache *cache.UserCache) *RegisterHandler {
	return &RegisterHandler{
		authRepo:  authRepo,
		vipRepo:   vipRepo,
		userCache: userCache,
	}
}

// Handle 处理注册命令
func (h *RegisterHandler) Handle(ctx context.Context, cmd RegisterCommand) (*RegisterResult, error) {
	// 1. 验证输入（使用值对象）
	username, err := auth.NewUsername(cmd.Username)
	if err != nil {
		return nil, fmt.Errorf("用户名验证失败: %w", err)
	}

	email, err := auth.NewEmail(cmd.Email)
	if err != nil {
		return nil, fmt.Errorf("邮箱验证失败: %w", err)
	}

	password, err := auth.NewPassword(cmd.Password)
	if err != nil {
		return nil, fmt.Errorf("密码验证失败: %w", err)
	}

	// 2. 检查用户是否已存在
	exists, err := h.authRepo.CheckUserExists(ctx, username.String(), email.String())
	if err != nil {
		return nil, fmt.Errorf("检查用户是否存在失败: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("用户名或邮箱已存在")
	}

	// 3. 加密密码
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password.String()), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("密码加密失败: %w", err)
	}

	// 4. 查询 VIP Level 0 配置（免费用户配额）
	freeLevel, err := h.vipRepo.FindByLevel(ctx, 0)
	if err != nil {
		// 如果查询失败，使用默认配额
		freeLevel = &vip.VIPLevel{
			Level:                   0,
			TrafficQuota:            10 * 1024 * 1024 * 1024, // 10GB
			MaxRules:                5,
			MaxBandwidth:            100,
			MaxSelfHostedEntryNodes: 0,
			MaxSelfHostedExitNodes:  0,
		}
	}

	// 5. 创建用户实体（使用数据库配置的配额）
	user := &auth.User{
		Username:                username.String(),
		Email:                   email.String(),
		PasswordHash:            string(passwordHash),
		Role:                    "user",
		Status:                  "normal",
		VipLevel:                0,
		VipExpiresAt:            nil,
		TrafficQuota:            freeLevel.TrafficQuota,
		TrafficUsed:             0,
		MaxRules:                freeLevel.MaxRules,
		MaxBandwidth:            freeLevel.MaxBandwidth,
		MaxSelfHostedEntryNodes: freeLevel.MaxSelfHostedEntryNodes,
		MaxSelfHostedExitNodes:  freeLevel.MaxSelfHostedExitNodes,
		CreatedAt:               time.Now(),
		UpdatedAt:               time.Now(),
	}

	// 6. 持久化用户
	if err := h.authRepo.CreateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("创建用户失败: %w", err)
	}

	// 7. 清除缓存（如果有）
	if h.userCache != nil {
		// 用户刚创建，无需清除缓存
	}

	return &RegisterResult{
		UserID:    user.ID,
		Username:  user.Username,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	}, nil
}
