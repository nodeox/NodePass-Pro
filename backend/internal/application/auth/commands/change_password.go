package commands

import (
	"context"
	"fmt"

	"nodepass-pro/backend/internal/domain/auth"
	"nodepass-pro/backend/internal/infrastructure/cache"

	"golang.org/x/crypto/bcrypt"
)

// ChangePasswordCommand 修改密码命令
type ChangePasswordCommand struct {
	UserID      uint
	OldPassword string
	NewPassword string
}

// ChangePasswordHandler 修改密码命令处理器
type ChangePasswordHandler struct {
	authRepo  auth.Repository
	userCache *cache.UserCache
}

// NewChangePasswordHandler 创建修改密码命令处理器
func NewChangePasswordHandler(authRepo auth.Repository, userCache *cache.UserCache) *ChangePasswordHandler {
	return &ChangePasswordHandler{
		authRepo:  authRepo,
		userCache: userCache,
	}
}

// Handle 处理修改密码命令
func (h *ChangePasswordHandler) Handle(ctx context.Context, cmd ChangePasswordCommand) error {
	// 1. 查找用户
	user, err := h.authRepo.FindUserByID(ctx, cmd.UserID)
	if err != nil {
		return fmt.Errorf("用户不存在")
	}

	// 2. 验证旧密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(cmd.OldPassword)); err != nil {
		return fmt.Errorf("原密码错误")
	}

	// 3. 验证新密码强度
	newPassword, err := auth.NewPassword(cmd.NewPassword)
	if err != nil {
		return fmt.Errorf("新密码验证失败: %w", err)
	}

	// 4. 加密新密码
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(newPassword.String()), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("密码加密失败: %w", err)
	}

	// 5. 更新密码
	if err := h.authRepo.UpdateUserPassword(ctx, cmd.UserID, string(passwordHash)); err != nil {
		return fmt.Errorf("更新密码失败: %w", err)
	}

	// 6. 撤销所有 refresh token（强制重新登录）
	if err := h.authRepo.RevokeUserRefreshTokens(ctx, cmd.UserID); err != nil {
		// 记录日志但不影响修改密码流程
	}

	// 7. 清除用户缓存
	if h.userCache != nil {
		// TODO: 清除用户缓存
	}

	return nil
}
