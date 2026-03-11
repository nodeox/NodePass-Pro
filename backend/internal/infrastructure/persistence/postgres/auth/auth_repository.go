package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"nodepass-pro/backend/internal/domain/auth"
	"nodepass-pro/backend/internal/models"

	"gorm.io/gorm"
)

// AuthRepository 认证仓储实现
type AuthRepository struct {
	db *gorm.DB
}

// NewAuthRepository 创建认证仓储
func NewAuthRepository(db *gorm.DB) auth.Repository {
	return &AuthRepository{db: db}
}

// FindUserByID 通过 ID 查找用户
func (r *AuthRepository) FindUserByID(ctx context.Context, id uint) (*auth.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("用户不存在")
		}
		return nil, err
	}
	return r.toDomainUser(&user), nil
}

// FindUserByEmail 通过邮箱查找用户
func (r *AuthRepository) FindUserByEmail(ctx context.Context, email string) (*auth.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).Where("email = ?", strings.ToLower(email)).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("用户不存在")
		}
		return nil, err
	}
	return r.toDomainUser(&user), nil
}

// FindUserByUsername 通过用户名查找用户
func (r *AuthRepository) FindUserByUsername(ctx context.Context, username string) (*auth.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("用户不存在")
		}
		return nil, err
	}
	return r.toDomainUser(&user), nil
}

// FindUserByAccount 通过用户名或邮箱查找用户
func (r *AuthRepository) FindUserByAccount(ctx context.Context, account string) (*auth.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).
		Where("email = ? OR username = ?", strings.ToLower(account), account).
		First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("用户不存在")
		}
		return nil, err
	}
	return r.toDomainUser(&user), nil
}

// CreateUser 创建用户
func (r *AuthRepository) CreateUser(ctx context.Context, user *auth.User) error {
	modelUser := r.toModelUser(user)
	if err := r.db.WithContext(ctx).Create(modelUser).Error; err != nil {
		return err
	}
	user.ID = modelUser.ID
	user.CreatedAt = modelUser.CreatedAt
	user.UpdatedAt = modelUser.UpdatedAt
	return nil
}

// UpdateUser 更新用户
func (r *AuthRepository) UpdateUser(ctx context.Context, user *auth.User) error {
	modelUser := r.toModelUser(user)
	return r.db.WithContext(ctx).Save(modelUser).Error
}

// UpdateUserPassword 更新用户密码
func (r *AuthRepository) UpdateUserPassword(ctx context.Context, userID uint, passwordHash string) error {
	return r.db.WithContext(ctx).
		Model(&models.User{}).
		Where("id = ?", userID).
		Update("password_hash", passwordHash).Error
}

// UpdateUserEmail 更新用户邮箱
func (r *AuthRepository) UpdateUserEmail(ctx context.Context, userID uint, email string) error {
	return r.db.WithContext(ctx).
		Model(&models.User{}).
		Where("id = ?", userID).
		Update("email", strings.ToLower(email)).Error
}

// UpdateUserLastLogin 更新用户最后登录时间
func (r *AuthRepository) UpdateUserLastLogin(ctx context.Context, userID uint, loginTime time.Time) error {
	return r.db.WithContext(ctx).
		Model(&models.User{}).
		Where("id = ?", userID).
		Update("last_login_at", loginTime).Error
}

// CheckUserExists 检查用户是否存在
func (r *AuthRepository) CheckUserExists(ctx context.Context, username, email string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.User{}).
		Where("username = ? OR email = ?", username, strings.ToLower(email)).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// CheckEmailExists 检查邮箱是否存在（排除指定用户）
func (r *AuthRepository) CheckEmailExists(ctx context.Context, email string, excludeUserID uint) (bool, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&models.User{}).Where("email = ?", strings.ToLower(email))
	if excludeUserID > 0 {
		query = query.Where("id <> ?", excludeUserID)
	}
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// CreateRefreshToken 创建刷新令牌
func (r *AuthRepository) CreateRefreshToken(ctx context.Context, token *auth.RefreshToken) error {
	modelToken := &models.RefreshToken{
		UserID:     token.UserID,
		TokenHash:  token.TokenHash,
		IPAddress:  token.IPAddress,
		UserAgent:  token.UserAgent,
		ExpiresAt:  token.ExpiresAt,
		LastUsedAt: token.LastUsedAt,
		IsRevoked:  token.IsRevoked,
		CreatedAt:  token.CreatedAt,
		UpdatedAt:  token.UpdatedAt,
	}
	if err := r.db.WithContext(ctx).Create(modelToken).Error; err != nil {
		return err
	}
	token.ID = modelToken.ID
	return nil
}

// FindRefreshTokenByHash 通过 hash 查找刷新令牌
func (r *AuthRepository) FindRefreshTokenByHash(ctx context.Context, tokenHash string) (*auth.RefreshToken, error) {
	var token models.RefreshToken
	if err := r.db.WithContext(ctx).Where("token_hash = ?", tokenHash).First(&token).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("刷新令牌不存在")
		}
		return nil, err
	}
	return r.toDomainRefreshToken(&token), nil
}

// UpdateRefreshTokenLastUsed 更新刷新令牌最后使用时间
func (r *AuthRepository) UpdateRefreshTokenLastUsed(ctx context.Context, tokenID uint, lastUsedAt time.Time) error {
	return r.db.WithContext(ctx).
		Model(&models.RefreshToken{}).
		Where("id = ?", tokenID).
		Update("last_used_at", lastUsedAt).Error
}

// RevokeRefreshToken 撤销刷新令牌
func (r *AuthRepository) RevokeRefreshToken(ctx context.Context, tokenID uint) error {
	return r.db.WithContext(ctx).
		Model(&models.RefreshToken{}).
		Where("id = ?", tokenID).
		Update("is_revoked", true).Error
}

// RevokeUserRefreshTokens 撤销用户所有刷新令牌
func (r *AuthRepository) RevokeUserRefreshTokens(ctx context.Context, userID uint) error {
	return r.db.WithContext(ctx).
		Model(&models.RefreshToken{}).
		Where("user_id = ?", userID).
		Update("is_revoked", true).Error
}

// ListUserRefreshTokens 列出用户所有刷新令牌
func (r *AuthRepository) ListUserRefreshTokens(ctx context.Context, userID uint) ([]*auth.RefreshToken, error) {
	var tokens []models.RefreshToken
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&tokens).Error; err != nil {
		return nil, err
	}

	result := make([]*auth.RefreshToken, len(tokens))
	for i, token := range tokens {
		result[i] = r.toDomainRefreshToken(&token)
	}
	return result, nil
}

// DeleteExpiredRefreshTokens 删除过期的刷新令牌
func (r *AuthRepository) DeleteExpiredRefreshTokens(ctx context.Context) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("expires_at < ?", time.Now()).
		Delete(&models.RefreshToken{})
	return result.RowsAffected, result.Error
}

// toDomainUser 转换为领域用户
func (r *AuthRepository) toDomainUser(user *models.User) *auth.User {
	return &auth.User{
		ID:                      user.ID,
		Username:                user.Username,
		Email:                   user.Email,
		PasswordHash:            user.PasswordHash,
		Role:                    user.Role,
		Status:                  user.Status,
		VipLevel:                user.VipLevel,
		VipExpiresAt:            user.VipExpiresAt,
		TrafficQuota:            user.TrafficQuota,
		TrafficUsed:             user.TrafficUsed,
		MaxRules:                user.MaxRules,
		MaxBandwidth:            user.MaxBandwidth,
		MaxSelfHostedEntryNodes: user.MaxSelfHostedEntryNodes,
		MaxSelfHostedExitNodes:  user.MaxSelfHostedExitNodes,
		TelegramID:              user.TelegramID,
		TelegramUsername:        user.TelegramUsername,
		CreatedAt:               user.CreatedAt,
		UpdatedAt:               user.UpdatedAt,
		LastLoginAt:             user.LastLoginAt,
	}
}

// toModelUser 转换为模型用户
func (r *AuthRepository) toModelUser(user *auth.User) *models.User {
	return &models.User{
		ID:                      user.ID,
		Username:                user.Username,
		Email:                   user.Email,
		PasswordHash:            user.PasswordHash,
		Role:                    user.Role,
		Status:                  user.Status,
		VipLevel:                user.VipLevel,
		VipExpiresAt:            user.VipExpiresAt,
		TrafficQuota:            user.TrafficQuota,
		TrafficUsed:             user.TrafficUsed,
		MaxRules:                user.MaxRules,
		MaxBandwidth:            user.MaxBandwidth,
		MaxSelfHostedEntryNodes: user.MaxSelfHostedEntryNodes,
		MaxSelfHostedExitNodes:  user.MaxSelfHostedExitNodes,
		TelegramID:              user.TelegramID,
		TelegramUsername:        user.TelegramUsername,
		CreatedAt:               user.CreatedAt,
		UpdatedAt:               user.UpdatedAt,
		LastLoginAt:             user.LastLoginAt,
	}
}

// toDomainRefreshToken 转换为领域刷新令牌
func (r *AuthRepository) toDomainRefreshToken(token *models.RefreshToken) *auth.RefreshToken {
	return &auth.RefreshToken{
		ID:         token.ID,
		UserID:     token.UserID,
		TokenHash:  token.TokenHash,
		IPAddress:  token.IPAddress,
		UserAgent:  token.UserAgent,
		ExpiresAt:  token.ExpiresAt,
		LastUsedAt: token.LastUsedAt,
		IsRevoked:  token.IsRevoked,
		CreatedAt:  token.CreatedAt,
		UpdatedAt:  token.UpdatedAt,
	}
}
