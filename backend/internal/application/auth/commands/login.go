package commands

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"nodepass-pro/backend/internal/config"
	"nodepass-pro/backend/internal/domain/auth"
	"nodepass-pro/backend/internal/infrastructure/cache"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// LoginCommand 登录命令
type LoginCommand struct {
	Account   string // 用户名或邮箱
	Password  string
	IPAddress string
	UserAgent string
}

// LoginResult 登录结果
type LoginResult struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresIn    int       `json:"expires_in"`
	TokenType    string    `json:"token_type"`
	User         *UserInfo `json:"user"`
}

// UserInfo 用户信息
type UserInfo struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	VipLevel int    `json:"vip_level"`
}

// LoginHandler 登录命令处理器
type LoginHandler struct {
	authRepo  auth.Repository
	userCache *cache.UserCache
}

// NewLoginHandler 创建登录命令处理器
func NewLoginHandler(authRepo auth.Repository, userCache *cache.UserCache) *LoginHandler {
	return &LoginHandler{
		authRepo:  authRepo,
		userCache: userCache,
	}
}

// Handle 处理登录命令
func (h *LoginHandler) Handle(ctx context.Context, cmd LoginCommand) (*LoginResult, error) {
	// 1. 查找用户（通过用户名或邮箱）
	user, err := h.authRepo.FindUserByAccount(ctx, cmd.Account)
	if err != nil {
		return nil, fmt.Errorf("用户名/邮箱或密码错误")
	}

	// 2. 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(cmd.Password)); err != nil {
		return nil, fmt.Errorf("用户名/邮箱或密码错误")
	}

	// 3. 检查用户状态
	if user.IsBanned() {
		return nil, fmt.Errorf("账户已被封禁")
	}

	// 4. 生成 JWT Access Token
	accessToken, err := h.generateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("生成访问令牌失败: %w", err)
	}

	// 5. 生成 Refresh Token
	refreshToken, tokenHash, err := h.generateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("生成刷新令牌失败: %w", err)
	}

	// 6. 保存 Refresh Token
	refreshTokenEntity := &auth.RefreshToken{
		UserID:     user.ID,
		TokenHash:  tokenHash,
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

	// 7. 更新最后登录时间
	now := time.Now()
	if err := h.authRepo.UpdateUserLastLogin(ctx, user.ID, now); err != nil {
		// 记录日志但不影响登录流程
	}

	// 8. 缓存用户信息
	if h.userCache != nil {
		// TODO: 缓存用户信息
	}

	return &LoginResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    1800, // 30 分钟
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

// generateAccessToken 生成 JWT 访问令牌
func (h *LoginHandler) generateAccessToken(user *auth.User) (string, error) {
	cfg := config.GlobalConfig
	if cfg == nil || cfg.JWT.Secret == "" {
		return "", fmt.Errorf("JWT 配置未初始化")
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"role":    user.Role,
		"iat":     now.Unix(),
		"exp":     now.Add(30 * time.Minute).Unix(), // 30 分钟过期
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.JWT.Secret))
}

// generateRefreshToken 生成刷新令牌
func (h *LoginHandler) generateRefreshToken() (token string, tokenHash string, err error) {
	// 生成 32 字节随机令牌
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", "", err
	}

	token = base64.RawURLEncoding.EncodeToString(tokenBytes)

	// 计算 SHA256 哈希
	hash := sha256.Sum256([]byte(token))
	tokenHash = hex.EncodeToString(hash[:])

	return token, tokenHash, nil
}
