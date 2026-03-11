package commands

import (
	"context"
	"testing"
	"time"

	"nodepass-pro/backend/internal/config"
	"nodepass-pro/backend/internal/domain/auth"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

func TestLoginHandler_Handle(t *testing.T) {
	// 初始化配置
	config.GlobalConfig = &config.Config{
		JWT: config.JWTConfig{
			Secret: "test-secret-key-for-testing",
		},
	}

	authRepo := NewMockAuthRepository()

	// 先创建一个测试用户
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

	handler := NewLoginHandler(authRepo, nil)

	tests := []struct {
		name    string
		cmd     LoginCommand
		wantErr bool
	}{
		{
			name: "使用邮箱登录成功",
			cmd: LoginCommand{
				Account:   "test@example.com",
				Password:  "Test1234!",
				IPAddress: "127.0.0.1",
				UserAgent: "test-agent",
			},
			wantErr: false,
		},
		{
			name: "使用用户名登录成功",
			cmd: LoginCommand{
				Account:   "testuser",
				Password:  "Test1234!",
				IPAddress: "127.0.0.1",
				UserAgent: "test-agent",
			},
			wantErr: false,
		},
		{
			name: "密码错误",
			cmd: LoginCommand{
				Account:   "test@example.com",
				Password:  "WrongPassword",
				IPAddress: "127.0.0.1",
				UserAgent: "test-agent",
			},
			wantErr: true,
		},
		{
			name: "用户不存在",
			cmd: LoginCommand{
				Account:   "nonexistent@example.com",
				Password:  "Test1234!",
				IPAddress: "127.0.0.1",
				UserAgent: "test-agent",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := handler.Handle(context.Background(), tt.cmd)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.NotEmpty(t, result.AccessToken)
				assert.NotEmpty(t, result.RefreshToken)
				assert.Equal(t, "Bearer", result.TokenType)
				assert.Equal(t, 1800, result.ExpiresIn)
				assert.NotNil(t, result.User)
				assert.Equal(t, testUser.ID, result.User.ID)
				assert.Equal(t, testUser.Username, result.User.Username)
				assert.Equal(t, testUser.Email, result.User.Email)
			}
		})
	}
}

func TestLoginHandler_Handle_BannedUser(t *testing.T) {
	// 初始化配置
	config.GlobalConfig = &config.Config{
		JWT: config.JWTConfig{
			Secret: "test-secret-key-for-testing",
		},
	}

	authRepo := NewMockAuthRepository()

	// 创建一个被封禁的用户
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

	handler := NewLoginHandler(authRepo, nil)

	cmd := LoginCommand{
		Account:   "banned@example.com",
		Password:  "Test1234!",
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	}

	result, err := handler.Handle(context.Background(), cmd)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "封禁")
}
