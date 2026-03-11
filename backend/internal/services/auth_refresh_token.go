package services

import (
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"strings"
	"time"

	"nodepass-pro/backend/internal/config"
	"nodepass-pro/backend/internal/models"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// SessionInfo 登录会话信息（脱敏）。
type SessionInfo struct {
	ID         uint       `json:"id"`
	IPAddress  string     `json:"ip_address"`
	UserAgent  string     `json:"user_agent"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at"`
	ExpiresAt  time.Time  `json:"expires_at"`
	IsRevoked  bool       `json:"is_revoked"`
	IsExpired  bool       `json:"is_expired"`
	IsActive   bool       `json:"is_active"`
	IsCurrent  bool       `json:"is_current"`
}

// generateAccessToken 生成访问令牌（短期，30分钟）
func generateAccessToken(userID uint, role string) (string, int, error) {
	cfg := config.GlobalConfig
	if cfg == nil || strings.TrimSpace(cfg.JWT.Secret) == "" {
		return "", 0, fmt.Errorf("%w: JWT 配置无效", ErrInvalidParams)
	}

	// Access token 固定 30 分钟过期
	expiresIn := 30 * 60 // 30 分钟（秒）
	expiresAt := time.Now().Add(30 * time.Minute)

	claims := jwt.MapClaims{
		"user_id": userID,
		"role":    role,
		"type":    "access", // 标记为 access token
		"exp":     expiresAt.Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(cfg.JWT.Secret))
	if err != nil {
		return "", 0, fmt.Errorf("签名 access token 失败: %w", err)
	}

	return signedToken, expiresIn, nil
}

// LoginWithRefreshToken 登录并返回 access token 和 refresh token
func (s *AuthService) LoginWithRefreshToken(req *LoginRequest, ipAddress, userAgent string) (*LoginResult, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: 请求体不能为空", ErrInvalidParams)
	}

	account := strings.TrimSpace(req.Account)
	password := strings.TrimSpace(req.Password)
	if account == "" || password == "" {
		return nil, fmt.Errorf("%w: account 和 password 不能为空", ErrInvalidParams)
	}

	// 查询用户
	var user models.User
	if err := s.db.Model(&models.User{}).
		Where("email = ? OR username = ?", strings.ToLower(account), account).
		First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: 用户名/邮箱或密码错误", ErrUnauthorized)
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("%w: 用户名/邮箱或密码错误", ErrUnauthorized)
	}

	// 检查用户状态
	if strings.EqualFold(strings.TrimSpace(user.Status), "banned") {
		return nil, fmt.Errorf("%w: 账户已被封禁", ErrForbidden)
	}

	// 生成 access token
	accessToken, expiresIn, err := generateAccessToken(user.ID, user.Role)
	if err != nil {
		return nil, err
	}

	// 生成 refresh token
	refreshToken, err := s.refreshTokenService.CreateRefreshToken(user.ID, ipAddress, userAgent)
	if err != nil {
		return nil, fmt.Errorf("创建 refresh token 失败: %w", err)
	}

	// 更新最后登录时间
	now := time.Now()
	if err := s.db.Model(&models.User{}).Where("id = ?", user.ID).Update("last_login_at", &now).Error; err != nil {
		_ = err // 非关键错误
	}
	user.LastLoginAt = &now

	return &LoginResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
		TokenType:    "Bearer",
		User:         &user,
	}, nil
}

// RefreshAccessToken 使用 refresh token 刷新 access token
func (s *AuthService) RefreshAccessToken(refreshToken string, ipAddress, userAgent string) (*LoginResult, error) {
	// 验证 refresh token
	rt, err := s.refreshTokenService.VerifyRefreshToken(refreshToken)
	if err != nil {
		return nil, err
	}

	// 查询用户
	var user models.User
	if err := s.db.First(&user, rt.UserID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: 用户不存在", ErrNotFound)
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}

	// 检查用户状态
	if strings.EqualFold(strings.TrimSpace(user.Status), "banned") {
		return nil, fmt.Errorf("%w: 账户已被封禁", ErrForbidden)
	}

	// 生成新的 access token
	accessToken, expiresIn, err := generateAccessToken(user.ID, user.Role)
	if err != nil {
		return nil, err
	}

	// 可选：生成新的 refresh token（refresh token rotation）
	// 这是一种更安全的做法，每次刷新都生成新的 refresh token
	newRefreshToken, err := s.refreshTokenService.CreateRefreshToken(user.ID, ipAddress, userAgent)
	if err != nil {
		return nil, fmt.Errorf("创建新 refresh token 失败: %w", err)
	}

	// 撤销旧的 refresh token
	if err := s.refreshTokenService.RevokeRefreshToken(refreshToken); err != nil {
		// 记录错误但不影响流程
		fmt.Printf("撤销旧 refresh token 失败: %v\n", err)
	}

	return &LoginResult{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    expiresIn,
		TokenType:    "Bearer",
		User:         &user,
	}, nil
}

// RevokeRefreshToken 撤销 refresh token（登出）
func (s *AuthService) RevokeRefreshToken(refreshToken string) error {
	return s.refreshTokenService.RevokeRefreshToken(refreshToken)
}

// RevokeAllUserTokens 撤销用户的所有 refresh tokens
func (s *AuthService) RevokeAllUserTokens(userID uint) error {
	return s.refreshTokenService.RevokeUserRefreshTokens(userID)
}

// ListUserSessions 列出当前用户所有登录会话。
func (s *AuthService) ListUserSessions(userID uint, currentRefreshToken string) ([]SessionInfo, error) {
	if userID == 0 {
		return nil, fmt.Errorf("%w: 用户 ID 无效", ErrInvalidParams)
	}

	tokens, err := s.refreshTokenService.GetUserRefreshTokens(userID)
	if err != nil {
		return nil, err
	}

	currentHash := hashRefreshToken(currentRefreshToken)
	result := make([]SessionInfo, 0, len(tokens))
	for _, item := range tokens {
		if item == nil {
			continue
		}
		isExpired := item.IsExpired()
		isRevoked := item.IsRevoked
		result = append(result, SessionInfo{
			ID:         item.ID,
			IPAddress:  item.IPAddress,
			UserAgent:  item.UserAgent,
			CreatedAt:  item.CreatedAt,
			LastUsedAt: item.LastUsedAt,
			ExpiresAt:  item.ExpiresAt,
			IsRevoked:  isRevoked,
			IsExpired:  isExpired,
			IsActive:   !isRevoked && !isExpired,
			IsCurrent:  currentHash != "" && subtle.ConstantTimeCompare([]byte(currentHash), []byte(item.TokenHash)) == 1,
		})
	}
	return result, nil
}

// RevokeUserSession 按会话 ID 撤销当前用户的会话。
func (s *AuthService) RevokeUserSession(userID uint, sessionID uint) error {
	if userID == 0 || sessionID == 0 {
		return fmt.Errorf("%w: 参数无效", ErrInvalidParams)
	}

	var token models.RefreshToken
	if err := s.db.Where("id = ? AND user_id = ?", sessionID, userID).First(&token).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("%w: 会话不存在", ErrNotFound)
		}
		return fmt.Errorf("查询会话失败: %w", err)
	}

	if token.IsRevoked {
		return nil
	}
	if err := s.db.Model(&models.RefreshToken{}).Where("id = ?", token.ID).Update("is_revoked", true).Error; err != nil {
		return fmt.Errorf("撤销会话失败: %w", err)
	}
	return nil
}

// RevokeCurrentSession 撤销当前用户指定的 refresh token 会话（用于当前会话下线）。
func (s *AuthService) RevokeCurrentSession(userID uint, refreshToken string) error {
	if userID == 0 {
		return fmt.Errorf("%w: 用户 ID 无效", ErrInvalidParams)
	}
	tokenHash := hashRefreshToken(refreshToken)
	if tokenHash == "" {
		return nil
	}

	var token models.RefreshToken
	if err := s.db.Where("user_id = ? AND token_hash = ?", userID, tokenHash).First(&token).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return fmt.Errorf("查询当前会话失败: %w", err)
	}
	if token.IsRevoked {
		return nil
	}
	if err := s.db.Model(&models.RefreshToken{}).Where("id = ?", token.ID).Update("is_revoked", true).Error; err != nil {
		return fmt.Errorf("撤销当前会话失败: %w", err)
	}
	return nil
}

func hashRefreshToken(refreshToken string) string {
	token := strings.TrimSpace(refreshToken)
	if token == "" {
		return ""
	}
	hash := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", hash)
}
