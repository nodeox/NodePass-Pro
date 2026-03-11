package cache

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func setupAuthCacheTest(t *testing.T) (*AuthCache, context.Context) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15, // 使用测试数据库
	})

	ctx := context.Background()

	// 清空测试数据库
	err := client.FlushDB(ctx).Err()
	assert.NoError(t, err)

	cache := NewAuthCache(client)
	return cache, ctx
}

// ========== RefreshToken 测试 ==========

func TestAuthCache_SetAndGetRefreshToken(t *testing.T) {
	cache, ctx := setupAuthCacheTest(t)

	tokenHash := "test_token_hash_123"
	userID := uint(100)
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	// 设置 token
	err := cache.SetRefreshToken(ctx, tokenHash, userID, expiresAt)
	assert.NoError(t, err)

	// 获取 token
	gotUserID, gotExpiresAt, err := cache.GetRefreshToken(ctx, tokenHash)
	assert.NoError(t, err)
	assert.Equal(t, userID, gotUserID)
	assert.WithinDuration(t, expiresAt, gotExpiresAt, time.Second)
}

func TestAuthCache_GetRefreshToken_NotFound(t *testing.T) {
	cache, ctx := setupAuthCacheTest(t)

	userID, expiresAt, err := cache.GetRefreshToken(ctx, "non_existent_token")
	assert.NoError(t, err)
	assert.Equal(t, uint(0), userID)
	assert.True(t, expiresAt.IsZero())
}

func TestAuthCache_RevokeRefreshToken(t *testing.T) {
	cache, ctx := setupAuthCacheTest(t)

	tokenHash := "test_token_hash_456"
	userID := uint(200)
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	// 设置 token
	err := cache.SetRefreshToken(ctx, tokenHash, userID, expiresAt)
	assert.NoError(t, err)

	// 撤销 token
	err = cache.RevokeRefreshToken(ctx, tokenHash)
	assert.NoError(t, err)

	// 验证已删除
	gotUserID, _, err := cache.GetRefreshToken(ctx, tokenHash)
	assert.NoError(t, err)
	assert.Equal(t, uint(0), gotUserID)
}

func TestAuthCache_UserSessions(t *testing.T) {
	cache, ctx := setupAuthCacheTest(t)

	userID := uint(300)
	token1 := "token_hash_1"
	token2 := "token_hash_2"

	// 添加会话
	err := cache.AddUserSession(ctx, userID, token1)
	assert.NoError(t, err)
	err = cache.AddUserSession(ctx, userID, token2)
	assert.NoError(t, err)

	// 获取会话数量
	count, err := cache.GetUserSessionCount(ctx, userID)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), count)

	// 移除一个会话
	err = cache.RemoveUserSession(ctx, userID, token1)
	assert.NoError(t, err)

	count, err = cache.GetUserSessionCount(ctx, userID)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestAuthCache_RevokeUserRefreshTokens(t *testing.T) {
	cache, ctx := setupAuthCacheTest(t)

	userID := uint(400)
	token1 := "token_hash_a"
	token2 := "token_hash_b"
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	// 设置多个 token
	err := cache.SetRefreshToken(ctx, token1, userID, expiresAt)
	assert.NoError(t, err)
	err = cache.SetRefreshToken(ctx, token2, userID, expiresAt)
	assert.NoError(t, err)

	// 添加到会话列表
	err = cache.AddUserSession(ctx, userID, token1)
	assert.NoError(t, err)
	err = cache.AddUserSession(ctx, userID, token2)
	assert.NoError(t, err)

	// 撤销所有 token
	err = cache.RevokeUserRefreshTokens(ctx, userID)
	assert.NoError(t, err)

	// 验证会话数量为 0
	count, err := cache.GetUserSessionCount(ctx, userID)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

// ========== 用户会话测试 ==========

func TestAuthCache_UserSession(t *testing.T) {
	cache, ctx := setupAuthCacheTest(t)

	sessionID := "session_123"
	userID := uint(500)
	ttl := 30 * time.Minute

	// 设置会话
	err := cache.SetUserSession(ctx, sessionID, userID, ttl)
	assert.NoError(t, err)

	// 获取会话
	gotUserID, err := cache.GetUserSession(ctx, sessionID)
	assert.NoError(t, err)
	assert.Equal(t, userID, gotUserID)

	// 删除会话
	err = cache.DeleteUserSession(ctx, sessionID)
	assert.NoError(t, err)

	// 验证已删除
	gotUserID, err = cache.GetUserSession(ctx, sessionID)
	assert.NoError(t, err)
	assert.Equal(t, uint(0), gotUserID)
}

func TestAuthCache_ExtendUserSession(t *testing.T) {
	cache, ctx := setupAuthCacheTest(t)

	sessionID := "session_456"
	userID := uint(600)
	ttl := 1 * time.Second

	// 设置会话（短 TTL）
	err := cache.SetUserSession(ctx, sessionID, userID, ttl)
	assert.NoError(t, err)

	// 延长会话
	err = cache.ExtendUserSession(ctx, sessionID, 10*time.Minute)
	assert.NoError(t, err)

	// 等待原 TTL 过期
	time.Sleep(2 * time.Second)

	// 验证会话仍然存在
	gotUserID, err := cache.GetUserSession(ctx, sessionID)
	assert.NoError(t, err)
	assert.Equal(t, userID, gotUserID)
}

// ========== 登录失败计数器测试 ==========

func TestAuthCache_LoginFailure(t *testing.T) {
	cache, ctx := setupAuthCacheTest(t)

	identifier := "user@example.com"

	// 增加失败次数
	count, err := cache.IncrementLoginFailure(ctx, identifier)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count)

	count, err = cache.IncrementLoginFailure(ctx, identifier)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), count)

	// 获取失败次数
	count, err = cache.GetLoginFailureCount(ctx, identifier)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), count)

	// 重置失败次数
	err = cache.ResetLoginFailure(ctx, identifier)
	assert.NoError(t, err)

	count, err = cache.GetLoginFailureCount(ctx, identifier)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestAuthCache_IsAccountLocked(t *testing.T) {
	cache, ctx := setupAuthCacheTest(t)

	identifier := "user2@example.com"
	maxAttempts := int64(5)

	// 未锁定
	locked, err := cache.IsAccountLocked(ctx, identifier, maxAttempts)
	assert.NoError(t, err)
	assert.False(t, locked)

	// 增加到阈值
	for i := 0; i < 5; i++ {
		_, err := cache.IncrementLoginFailure(ctx, identifier)
		assert.NoError(t, err)
	}

	// 已锁定
	locked, err = cache.IsAccountLocked(ctx, identifier, maxAttempts)
	assert.NoError(t, err)
	assert.True(t, locked)
}

func TestAuthCache_LockAndUnlockAccount(t *testing.T) {
	cache, ctx := setupAuthCacheTest(t)

	identifier := "user3@example.com"

	// 锁定账户
	err := cache.LockAccount(ctx, identifier, 10*time.Minute)
	assert.NoError(t, err)

	// 检查是否锁定
	locked, err := cache.IsAccountLockedExplicitly(ctx, identifier)
	assert.NoError(t, err)
	assert.True(t, locked)

	// 解锁账户
	err = cache.UnlockAccount(ctx, identifier)
	assert.NoError(t, err)

	// 检查是否解锁
	locked, err = cache.IsAccountLockedExplicitly(ctx, identifier)
	assert.NoError(t, err)
	assert.False(t, locked)
}

// ========== 验证码测试 ==========

func TestAuthCache_VerificationCode(t *testing.T) {
	cache, ctx := setupAuthCacheTest(t)

	identifier := "user@example.com"
	code := "123456"
	ttl := 5 * time.Minute

	// 设置验证码
	err := cache.SetVerificationCode(ctx, identifier, code, ttl)
	assert.NoError(t, err)

	// 获取验证码
	gotCode, err := cache.GetVerificationCode(ctx, identifier)
	assert.NoError(t, err)
	assert.Equal(t, code, gotCode)

	// 删除验证码
	err = cache.DeleteVerificationCode(ctx, identifier)
	assert.NoError(t, err)

	// 验证已删除
	gotCode, err = cache.GetVerificationCode(ctx, identifier)
	assert.NoError(t, err)
	assert.Equal(t, "", gotCode)
}

// ========== 密码重置令牌测试 ==========

func TestAuthCache_PasswordResetToken(t *testing.T) {
	cache, ctx := setupAuthCacheTest(t)

	token := "reset_token_xyz"
	userID := uint(700)
	ttl := 1 * time.Hour

	// 设置重置令牌
	err := cache.SetPasswordResetToken(ctx, token, userID, ttl)
	assert.NoError(t, err)

	// 获取重置令牌
	gotUserID, err := cache.GetPasswordResetToken(ctx, token)
	assert.NoError(t, err)
	assert.Equal(t, userID, gotUserID)

	// 删除重置令牌
	err = cache.DeletePasswordResetToken(ctx, token)
	assert.NoError(t, err)

	// 验证已删除
	gotUserID, err = cache.GetPasswordResetToken(ctx, token)
	assert.NoError(t, err)
	assert.Equal(t, uint(0), gotUserID)
}

func TestAuthCache_PasswordResetToken_NotFound(t *testing.T) {
	cache, ctx := setupAuthCacheTest(t)

	userID, err := cache.GetPasswordResetToken(ctx, "non_existent_token")
	assert.NoError(t, err)
	assert.Equal(t, uint(0), userID)
}
