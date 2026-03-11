package commands

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"nodepass-pro/backend/internal/domain/auth"
	"nodepass-pro/backend/internal/infrastructure/cache"
)

// RefreshTokenCommand 刷新令牌命令
type RefreshTokenCommand struct {
	RefreshToken string
	IPAddress    string
	UserAgent    string
}

// RefreshTokenHandler 刷新令牌命令处理器
type RefreshTokenHandler struct {
	authRepo  auth.Repository
	userCache *cache.UserCache
}

// NewRefreshTokenHandler 创建刷新令牌命令处理器
func NewRefreshTokenHandler(authRepo auth.Repository, userCache *cache.UserCache) *RefreshTokenHandler {
	return &RefreshTokenHandler{
		authRepo:  authRepo,
		userCache: userCache,
	}
}

// Handle 处理刷新令牌命令
func (h *RefreshTokenHandler) Handle(ctx context.Context, cmd RefreshTokenCommand) (*LoginResult, error) {
	// 1. 计算 token hash
	hash := sha256.Sum256([]byte(cmd.RefreshToken))
	tokenHash := hex.EncodeToString(hash[:])

	// 2. 查找 refresh token
	oldRefreshToken, err := h.authRepo.FindRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		return nil, fmt.Errorf("刷新令牌无效")
	}

	// 3. 验证 refresh token
	if !oldRefreshToken.IsValid() {
		return nil, fmt.Errorf("刷新令牌已失效")
	}

	// 4. 查找用户
	user, err := h.authRepo.FindUserByID(ctx, oldRefreshToken.UserID)
	if err != nil {
		return nil, fmt.Errorf("用户不存在")
	}

	// 5. 检查用户状态
	if user.IsBanned() {
		return nil, fmt.Errorf("账户已被封禁")
	}

	// 6. 生成新的 access token
	loginHandler := &LoginHandler{
		authRepo:  h.authRepo,
		userCache: h.userCache,
	}
	accessToken, err := loginHandler.generateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("生成访问令牌失败: %w", err)
	}

	// 7. 生成新的 refresh token（Token Rotation）
	newRefreshToken, newTokenHash, err := loginHandler.generateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("生成刷新令牌失败: %w", err)
	}

	// 8. 保存新的 refresh token
	refreshTokenEntity := &auth.RefreshToken{
		UserID:     user.ID,
		TokenHash:  newTokenHash,
		IPAddress:  cmd.IPAddress,
		UserAgent:  cmd.UserAgent,
		ExpiresAt:  time.Now().Add(7 * 24 * time.Hour), // 7 天
		LastUsedAt: nil,
		IsRevoked:  false,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if err := h.authRepo.CreateRefreshToken(ctx, refreshTokenEntity); err != nil {
		return nil, fmt.Errorf("保存刷新令牌失败: %w", err)
	}

	// 9. 撤销旧的 refresh token
	if err := h.authRepo.RevokeRefreshToken(ctx, oldRefreshToken.ID); err != nil {
		// 记录日志但不影响刷新流程
		// 新 token 已经生成，即使撤销失败也不应该阻止用户使用
	}

	return &LoginResult{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken, // 返回新的 refresh token
		ExpiresIn:    1800,             // 30 分钟
		TokenType:    "Bearer",
		User: &UserInfo{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			Role:     user.Role,
			VipLevel: user.VipLevel,
		},
	}, nil
}
