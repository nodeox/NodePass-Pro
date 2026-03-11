package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// AuthCache 认证缓存
type AuthCache struct {
	client *redis.Client
	prefix string
}

// NewAuthCache 创建认证缓存
func NewAuthCache(client *redis.Client) *AuthCache {
	return &AuthCache{
		client: client,
		prefix: "auth:",
	}
}

// ========== RefreshToken 缓存 ==========

// SetRefreshToken 缓存 RefreshToken
func (c *AuthCache) SetRefreshToken(ctx context.Context, tokenHash string, userID uint, expiresAt time.Time) error {
	key := fmt.Sprintf("%srefresh_token:%s", c.prefix, tokenHash)
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		return fmt.Errorf("token already expired")
	}

	data := map[string]interface{}{
		"user_id":    userID,
		"expires_at": expiresAt.Unix(),
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, key, jsonData, ttl).Err()
}

// GetRefreshToken 获取 RefreshToken 信息
func (c *AuthCache) GetRefreshToken(ctx context.Context, tokenHash string) (userID uint, expiresAt time.Time, err error) {
	key := fmt.Sprintf("%srefresh_token:%s", c.prefix, tokenHash)
	data, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return 0, time.Time{}, nil // 缓存未命中
	}
	if err != nil {
		return 0, time.Time{}, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return 0, time.Time{}, err
	}

	userID = uint(result["user_id"].(float64))
	expiresAt = time.Unix(int64(result["expires_at"].(float64)), 0)

	return userID, expiresAt, nil
}

// RevokeRefreshToken 撤销 RefreshToken
func (c *AuthCache) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	key := fmt.Sprintf("%srefresh_token:%s", c.prefix, tokenHash)
	return c.client.Del(ctx, key).Err()
}

// RevokeUserRefreshTokens 撤销用户的所有 RefreshToken
func (c *AuthCache) RevokeUserRefreshTokens(ctx context.Context, userID uint) error {
	// 删除用户会话列表
	sessionKey := fmt.Sprintf("%suser_sessions:%d", c.prefix, userID)
	tokenHashes, err := c.client.SMembers(ctx, sessionKey).Result()
	if err != nil && err != redis.Nil {
		return err
	}

	// 删除所有 token
	for _, tokenHash := range tokenHashes {
		if err := c.RevokeRefreshToken(ctx, tokenHash); err != nil {
			// 继续删除其他 token
			continue
		}
	}

	// 删除会话列表
	return c.client.Del(ctx, sessionKey).Err()
}

// AddUserSession 添加用户会话
func (c *AuthCache) AddUserSession(ctx context.Context, userID uint, tokenHash string) error {
	key := fmt.Sprintf("%suser_sessions:%d", c.prefix, userID)
	return c.client.SAdd(ctx, key, tokenHash).Err()
}

// RemoveUserSession 移除用户会话
func (c *AuthCache) RemoveUserSession(ctx context.Context, userID uint, tokenHash string) error {
	key := fmt.Sprintf("%suser_sessions:%d", c.prefix, userID)
	return c.client.SRem(ctx, key, tokenHash).Err()
}

// GetUserSessionCount 获取用户会话数量
func (c *AuthCache) GetUserSessionCount(ctx context.Context, userID uint) (int64, error) {
	key := fmt.Sprintf("%suser_sessions:%d", c.prefix, userID)
	return c.client.SCard(ctx, key).Result()
}

// ========== 用户会话缓存 ==========

// SetUserSession 设置用户会话信息
func (c *AuthCache) SetUserSession(ctx context.Context, sessionID string, userID uint, ttl time.Duration) error {
	key := fmt.Sprintf("%ssession:%s", c.prefix, sessionID)
	return c.client.Set(ctx, key, userID, ttl).Err()
}

// GetUserSession 获取用户会话信息
func (c *AuthCache) GetUserSession(ctx context.Context, sessionID string) (uint, error) {
	key := fmt.Sprintf("%ssession:%s", c.prefix, sessionID)
	userID, err := c.client.Get(ctx, key).Uint64()
	if err == redis.Nil {
		return 0, nil // 会话不存在
	}
	if err != nil {
		return 0, err
	}
	return uint(userID), nil
}

// DeleteUserSession 删除用户会话
func (c *AuthCache) DeleteUserSession(ctx context.Context, sessionID string) error {
	key := fmt.Sprintf("%ssession:%s", c.prefix, sessionID)
	return c.client.Del(ctx, key).Err()
}

// ExtendUserSession 延长用户会话
func (c *AuthCache) ExtendUserSession(ctx context.Context, sessionID string, ttl time.Duration) error {
	key := fmt.Sprintf("%ssession:%s", c.prefix, sessionID)
	return c.client.Expire(ctx, key, ttl).Err()
}

// ========== 登录失败计数器 ==========

// IncrementLoginFailure 增加登录失败次数
func (c *AuthCache) IncrementLoginFailure(ctx context.Context, identifier string) (int64, error) {
	key := fmt.Sprintf("%slogin_failure:%s", c.prefix, identifier)
	count, err := c.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}

	// 设置过期时间（15分钟）
	if count == 1 {
		c.client.Expire(ctx, key, 15*time.Minute)
	}

	return count, nil
}

// GetLoginFailureCount 获取登录失败次数
func (c *AuthCache) GetLoginFailureCount(ctx context.Context, identifier string) (int64, error) {
	key := fmt.Sprintf("%slogin_failure:%s", c.prefix, identifier)
	count, err := c.client.Get(ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return count, err
}

// ResetLoginFailure 重置登录失败次数
func (c *AuthCache) ResetLoginFailure(ctx context.Context, identifier string) error {
	key := fmt.Sprintf("%slogin_failure:%s", c.prefix, identifier)
	return c.client.Del(ctx, key).Err()
}

// IsAccountLocked 检查账户是否被锁定
func (c *AuthCache) IsAccountLocked(ctx context.Context, identifier string, maxAttempts int64) (bool, error) {
	count, err := c.GetLoginFailureCount(ctx, identifier)
	if err != nil {
		return false, err
	}
	return count >= maxAttempts, nil
}

// LockAccount 锁定账户
func (c *AuthCache) LockAccount(ctx context.Context, identifier string, duration time.Duration) error {
	key := fmt.Sprintf("%saccount_locked:%s", c.prefix, identifier)
	return c.client.Set(ctx, key, "1", duration).Err()
}

// UnlockAccount 解锁账户
func (c *AuthCache) UnlockAccount(ctx context.Context, identifier string) error {
	key := fmt.Sprintf("%saccount_locked:%s", c.prefix, identifier)
	return c.client.Del(ctx, key).Err()
}

// IsAccountLockedExplicitly 检查账户是否被显式锁定
func (c *AuthCache) IsAccountLockedExplicitly(ctx context.Context, identifier string) (bool, error) {
	key := fmt.Sprintf("%saccount_locked:%s", c.prefix, identifier)
	exists, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

// ========== 验证码缓存 ==========

// SetVerificationCode 设置验证码
func (c *AuthCache) SetVerificationCode(ctx context.Context, identifier string, code string, ttl time.Duration) error {
	key := fmt.Sprintf("%sverification_code:%s", c.prefix, identifier)
	return c.client.Set(ctx, key, code, ttl).Err()
}

// GetVerificationCode 获取验证码
func (c *AuthCache) GetVerificationCode(ctx context.Context, identifier string) (string, error) {
	key := fmt.Sprintf("%sverification_code:%s", c.prefix, identifier)
	code, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	return code, err
}

// DeleteVerificationCode 删除验证码
func (c *AuthCache) DeleteVerificationCode(ctx context.Context, identifier string) error {
	key := fmt.Sprintf("%sverification_code:%s", c.prefix, identifier)
	return c.client.Del(ctx, key).Err()
}

// ========== 密码重置令牌 ==========

// SetPasswordResetToken 设置密码重置令牌
func (c *AuthCache) SetPasswordResetToken(ctx context.Context, token string, userID uint, ttl time.Duration) error {
	key := fmt.Sprintf("%spassword_reset:%s", c.prefix, token)
	return c.client.Set(ctx, key, userID, ttl).Err()
}

// GetPasswordResetToken 获取密码重置令牌
func (c *AuthCache) GetPasswordResetToken(ctx context.Context, token string) (uint, error) {
	key := fmt.Sprintf("%spassword_reset:%s", c.prefix, token)
	userID, err := c.client.Get(ctx, key).Uint64()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return uint(userID), nil
}

// DeletePasswordResetToken 删除密码重置令牌
func (c *AuthCache) DeletePasswordResetToken(ctx context.Context, token string) error {
	key := fmt.Sprintf("%spassword_reset:%s", c.prefix, token)
	return c.client.Del(ctx, key).Err()
}
