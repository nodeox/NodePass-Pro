package commands

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"nodepass-pro/backend/internal/config"
	"nodepass-pro/backend/internal/domain/auth"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

func TestRefreshTokenHandler_Handle(t *testing.T) {
	// 初始化配置
	config.GlobalConfig = &config.Config{
		JWT: config.JWTConfig{
			Secret: "test-secret-key-for-testing",
		},
	}

	authRepo := NewMockAuthRepository()

	// 创建测试用户
	passwordHash, _ := bcrypt.GenerateFromPassword([]byte("Test1234!"), bcrypt.DefaultCost)
	testUser := &auth.User{
		ID:           1,
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: string(passwordHash),
		Role:         "user",
		Status:       "normal",
		VipLevel:     0,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	authRepo.users[testUser.ID] = testUser
	authRepo.usersByEmail[testUser.Email] = testUser

	// 创建有效的 refresh token
	validToken := "valid-refresh-token-12345"
	hash := sha256.Sum256([]byte(validToken))
	tokenHash := hex.EncodeToString(hash[:])

	refreshToken := &auth.RefreshToken{
		ID:         1,
		UserID:     testUser.ID,
		TokenHash:  tokenHash,
		IPAddress:  "127.0.0.1",
		UserAgent:  "test-agent",
		ExpiresAt:  time.Now().Add(7 * 24 * time.Hour),
		LastUsedAt: nil,
		IsRevoked:  false,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	authRepo.refreshTokens[tokenHash] = refreshToken

	handler := NewRefreshTokenHandler(authRepo, nil)

	t.Run("成功刷新令牌", func(t *testing.T) {
		cmd := RefreshTokenCommand{
			RefreshToken: validToken,
			IPAddress:    "127.0.0.1",
			UserAgent:    "test-agent",
		}

		result, err := handler.Handle(context.Background(), cmd)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.AccessToken)
		assert.NotEmpty(t, result.RefreshToken)
		// 验证返回的是新的 refresh token（Token Rotation）
		assert.NotEqual(t, validToken, result.RefreshToken)
		assert.Equal(t, "Bearer", result.TokenType)
		assert.Equal(t, 1800, result.ExpiresIn)
		assert.NotNil(t, result.User)
		assert.Equal(t, testUser.ID, result.User.ID)

		// 验证旧 token 已被撤销
		oldToken, _ := authRepo.FindRefreshTokenByHash(context.Background(), tokenHash)
		assert.True(t, oldToken.IsRevoked)
	})

	t.Run("无效的 refresh token", func(t *testing.T) {
		cmd := RefreshTokenCommand{
			RefreshToken: "invalid-token",
			IPAddress:    "127.0.0.1",
			UserAgent:    "test-agent",
		}

		result, err := handler.Handle(context.Background(), cmd)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "无效")
	})

	t.Run("已过期的 refresh token", func(t *testing.T) {
		// 创建已过期的 token
		expiredToken := "expired-refresh-token-12345"
		hash := sha256.Sum256([]byte(expiredToken))
		expiredTokenHash := hex.EncodeToString(hash[:])

		expiredRefreshToken := &auth.RefreshToken{
			ID:         2,
			UserID:     testUser.ID,
			TokenHash:  expiredTokenHash,
			IPAddress:  "127.0.0.1",
			UserAgent:  "test-agent",
			ExpiresAt:  time.Now().Add(-1 * time.Hour), // 已过期
			LastUsedAt: nil,
			IsRevoked:  false,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		authRepo.refreshTokens[expiredTokenHash] = expiredRefreshToken

		cmd := RefreshTokenCommand{
			RefreshToken: expiredToken,
			IPAddress:    "127.0.0.1",
			UserAgent:    "test-agent",
		}

		result, err := handler.Handle(context.Background(), cmd)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "失效")
	})

	t.Run("已撤销的 refresh token", func(t *testing.T) {
		// 创建已撤销的 token
		revokedToken := "revoked-refresh-token-12345"
		hash := sha256.Sum256([]byte(revokedToken))
		revokedTokenHash := hex.EncodeToString(hash[:])

		revokedRefreshToken := &auth.RefreshToken{
			ID:         3,
			UserID:     testUser.ID,
			TokenHash:  revokedTokenHash,
			IPAddress:  "127.0.0.1",
			UserAgent:  "test-agent",
			ExpiresAt:  time.Now().Add(7 * 24 * time.Hour),
			LastUsedAt: nil,
			IsRevoked:  true, // 已撤销
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		authRepo.refreshTokens[revokedTokenHash] = revokedRefreshToken

		cmd := RefreshTokenCommand{
			RefreshToken: revokedToken,
			IPAddress:    "127.0.0.1",
			UserAgent:    "test-agent",
		}

		result, err := handler.Handle(context.Background(), cmd)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "失效")
	})
}

func TestRefreshTokenHandler_Handle_BannedUser(t *testing.T) {
	// 初始化配置
	config.GlobalConfig = &config.Config{
		JWT: config.JWTConfig{
			Secret: "test-secret-key-for-testing",
		},
	}

	authRepo := NewMockAuthRepository()

	// 创建被封禁的用户
	passwordHash, _ := bcrypt.GenerateFromPassword([]byte("Test1234!"), bcrypt.DefaultCost)
	bannedUser := &auth.User{
		ID:           1,
		Username:     "banneduser",
		Email:        "banned@example.com",
		PasswordHash: string(passwordHash),
		Role:         "user",
		Status:       "banned",
		VipLevel:     0,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	authRepo.users[bannedUser.ID] = bannedUser
	authRepo.usersByEmail[bannedUser.Email] = bannedUser

	// 创建有效的 refresh token
	validToken := "valid-refresh-token-12345"
	hash := sha256.Sum256([]byte(validToken))
	tokenHash := hex.EncodeToString(hash[:])

	refreshToken := &auth.RefreshToken{
		ID:         1,
		UserID:     bannedUser.ID,
		TokenHash:  tokenHash,
		IPAddress:  "127.0.0.1",
		UserAgent:  "test-agent",
		ExpiresAt:  time.Now().Add(7 * 24 * time.Hour),
		LastUsedAt: nil,
		IsRevoked:  false,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	authRepo.refreshTokens[tokenHash] = refreshToken

	handler := NewRefreshTokenHandler(authRepo, nil)

	cmd := RefreshTokenCommand{
		RefreshToken: validToken,
		IPAddress:    "127.0.0.1",
		UserAgent:    "test-agent",
	}

	result, err := handler.Handle(context.Background(), cmd)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "封禁")
}
