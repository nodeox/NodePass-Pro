package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"

	"nodepass-pro/backend/internal/models"

	"gorm.io/gorm"
)

// RefreshTokenService 刷新令牌服务
type RefreshTokenService struct {
	db *gorm.DB
}

// NewRefreshTokenService 创建刷新令牌服务
func NewRefreshTokenService(db *gorm.DB) *RefreshTokenService {
	return &RefreshTokenService{db: db}
}

// CreateRefreshToken 创建刷新令牌
func (s *RefreshTokenService) CreateRefreshToken(userID uint, ipAddress, userAgent string) (string, error) {
	// 生成随机 token（48 字节 = 64 字符 base64）
	tokenBytes := make([]byte, 48)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("生成 refresh token 失败: %w", err)
	}
	token := base64.URLEncoding.EncodeToString(tokenBytes)

	// 计算 token 哈希
	hash := sha256.Sum256([]byte(token))
	tokenHash := fmt.Sprintf("%x", hash)

	// 创建数据库记录
	refreshToken := &models.RefreshToken{
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour), // 7 天有效期
		IPAddress: ipAddress,
		UserAgent: userAgent,
		IsRevoked: false,
	}

	if err := s.db.Create(refreshToken).Error; err != nil {
		return "", fmt.Errorf("保存 refresh token 失败: %w", err)
	}

	return token, nil
}

// VerifyRefreshToken 验证刷新令牌
func (s *RefreshTokenService) VerifyRefreshToken(token string) (*models.RefreshToken, error) {
	// 计算 token 哈希
	hash := sha256.Sum256([]byte(token))
	tokenHash := fmt.Sprintf("%x", hash)

	// 查询数据库
	var refreshToken models.RefreshToken
	if err := s.db.Where("token_hash = ?", tokenHash).First(&refreshToken).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("%w: refresh token 不存在", ErrUnauthorized)
		}
		return nil, fmt.Errorf("查询 refresh token 失败: %w", err)
	}

	// 检查是否有效
	if !refreshToken.IsValid() {
		if refreshToken.IsRevoked {
			return nil, fmt.Errorf("%w: refresh token 已被撤销", ErrUnauthorized)
		}
		return nil, fmt.Errorf("%w: refresh token 已过期", ErrUnauthorized)
	}

	// 更新最后使用时间
	now := time.Now()
	refreshToken.LastUsedAt = &now
	if err := s.db.Model(&refreshToken).Update("last_used_at", now).Error; err != nil {
		// 记录错误但不影响验证结果
		fmt.Printf("更新 refresh token 使用时间失败: %v\n", err)
	}

	return &refreshToken, nil
}

// RevokeRefreshToken 撤销刷新令牌
func (s *RefreshTokenService) RevokeRefreshToken(token string) error {
	// 计算 token 哈希
	hash := sha256.Sum256([]byte(token))
	tokenHash := fmt.Sprintf("%x", hash)

	// 更新数据库
	result := s.db.Model(&models.RefreshToken{}).
		Where("token_hash = ?", tokenHash).
		Update("is_revoked", true)

	if result.Error != nil {
		return fmt.Errorf("撤销 refresh token 失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("%w: refresh token 不存在", ErrNotFound)
	}

	return nil
}

// RevokeUserRefreshTokens 撤销用户的所有刷新令牌
func (s *RefreshTokenService) RevokeUserRefreshTokens(userID uint) error {
	result := s.db.Model(&models.RefreshToken{}).
		Where("user_id = ? AND is_revoked = ?", userID, false).
		Update("is_revoked", true)

	if result.Error != nil {
		return fmt.Errorf("撤销用户 refresh tokens 失败: %w", result.Error)
	}

	return nil
}

// CleanupExpiredTokens 清理过期的刷新令牌
func (s *RefreshTokenService) CleanupExpiredTokens() error {
	// 删除过期超过 30 天的 token
	cutoff := time.Now().Add(-30 * 24 * time.Hour)
	result := s.db.Where("expires_at < ?", cutoff).Delete(&models.RefreshToken{})

	if result.Error != nil {
		return fmt.Errorf("清理过期 refresh tokens 失败: %w", result.Error)
	}

	return nil
}

// GetUserRefreshTokens 获取用户的刷新令牌列表
func (s *RefreshTokenService) GetUserRefreshTokens(userID uint) ([]*models.RefreshToken, error) {
	var tokens []*models.RefreshToken
	if err := s.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&tokens).Error; err != nil {
		return nil, fmt.Errorf("查询用户 refresh tokens 失败: %w", err)
	}

	return tokens, nil
}
