package commands

import (
	"context"
	"errors"
	
	"nodepass-pro/backend/internal/domain/user"
	"nodepass-pro/backend/internal/infrastructure/cache"
	
	"golang.org/x/crypto/bcrypt"
)

// CreateUserCommand 创建用户命令
type CreateUserCommand struct {
	Username string
	Email    string
	Password string
	Role     string
}

// CreateUserResult 创建用户结果
type CreateUserResult struct {
	User *user.User
}

// CreateUserHandler 创建用户处理器
type CreateUserHandler struct {
	userRepo  user.Repository
	userCache *cache.UserCache
}

// NewCreateUserHandler 创建处理器
func NewCreateUserHandler(repo user.Repository, cache *cache.UserCache) *CreateUserHandler {
	return &CreateUserHandler{
		userRepo:  repo,
		userCache: cache,
	}
}

// Handle 处理命令
func (h *CreateUserHandler) Handle(ctx context.Context, cmd CreateUserCommand) (*CreateUserResult, error) {
	// 1. 验证邮箱是否已存在
	if _, err := h.userRepo.FindByEmail(ctx, cmd.Email); err == nil {
		return nil, user.ErrEmailExists
	} else if !errors.Is(err, user.ErrUserNotFound) {
		return nil, err
	}
	
	// 2. 验证用户名是否已存在
	if _, err := h.userRepo.FindByUsername(ctx, cmd.Username); err == nil {
		return nil, user.ErrUsernameExists
	} else if !errors.Is(err, user.ErrUserNotFound) {
		return nil, err
	}
	
	// 3. 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(cmd.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	
	// 4. 创建用户实体
	newUser := &user.User{
		Username:     cmd.Username,
		Email:        cmd.Email,
		PasswordHash: string(hashedPassword),
		Role:         cmd.Role,
		Status:       "normal",
		VipLevel:     0,
		TrafficQuota: 10 * 1024 * 1024 * 1024, // 10GB
		TrafficUsed:  0,
		MaxRules:     5,
		MaxBandwidth: 100,
	}
	
	// 5. 保存到数据库
	if err := h.userRepo.Create(ctx, newUser); err != nil {
		return nil, err
	}
	
	// 6. 写入缓存
	userData := map[string]interface{}{
		"id":       newUser.ID,
		"username": newUser.Username,
		"email":    newUser.Email,
		"role":     newUser.Role,
		"status":   newUser.Status,
	}
	h.userCache.Set(ctx, newUser.ID, userData)
	h.userCache.SetEmailIndex(ctx, newUser.Email, newUser.ID)
	
	return &CreateUserResult{User: newUser}, nil
}
