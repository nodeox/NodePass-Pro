package services

import (
	"fmt"
	"testing"
	"time"

	"nodepass-pro/backend/internal/config"
	"nodepass-pro/backend/internal/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	dsn := fmt.Sprintf("file:%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("初始化测试数据库失败: %v", err)
	}
	config.GlobalConfig = &config.Config{
		JWT: config.JWTConfig{
			Secret:     "test-secret-for-unit-tests",
			ExpireTime: 24,
		},
	}

	if err = db.AutoMigrate(
		&models.User{},
		&models.VIPLevel{},
		&models.RefreshToken{},
	); err != nil {
		t.Fatalf("自动迁移失败: %v", err)
	}

	// 创建默认 VIP 等级
	freeLevel := &models.VIPLevel{
		Level:                   0,
		Name:                    "免费版",
		TrafficQuota:            1073741824, // 1GB
		MaxRules:                5,
		MaxBandwidth:            100,
		MaxSelfHostedEntryNodes: 0,
		MaxSelfHostedExitNodes:  0,
	}
	if err = db.Create(freeLevel).Error; err != nil {
		t.Fatalf("创建默认 VIP 等级失败: %v", err)
	}

	return db
}

func TestAuthService_Register(t *testing.T) {
	db := setupTestDB(t)
	service := NewAuthService(db)

	tests := []struct {
		name        string
		req         *RegisterRequest
		expectError bool
		errorMsg    string
	}{
		{
			name: "成功注册",
			req: &RegisterRequest{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "Password123!",
			},
			expectError: false,
		},
		{
			name:        "空请求体",
			req:         nil,
			expectError: true,
			errorMsg:    "请求体不能为空",
		},
		{
			name: "用户名为空",
			req: &RegisterRequest{
				Username: "",
				Email:    "test@example.com",
				Password: "Password123!",
			},
			expectError: true,
			errorMsg:    "不能为空",
		},
		{
			name: "邮箱为空",
			req: &RegisterRequest{
				Username: "testuser",
				Email:    "",
				Password: "Password123!",
			},
			expectError: true,
			errorMsg:    "不能为空",
		},
		{
			name: "密码为空",
			req: &RegisterRequest{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "",
			},
			expectError: true,
			errorMsg:    "不能为空",
		},
		{
			name: "用户名过短",
			req: &RegisterRequest{
				Username: "ab",
				Email:    "test@example.com",
				Password: "Password123!",
			},
			expectError: true,
			errorMsg:    "用户名",
		},
		{
			name: "用户名过长",
			req: &RegisterRequest{
				Username: "verylongusernamethatexceedsthelimit1234567890",
				Email:    "test@example.com",
				Password: "Password123!",
			},
			expectError: true,
			errorMsg:    "用户名",
		},
		{
			name: "邮箱格式错误",
			req: &RegisterRequest{
				Username: "testuser",
				Email:    "invalid-email",
				Password: "Password123!",
			},
			expectError: true,
			errorMsg:    "邮箱",
		},
		{
			name: "密码过短",
			req: &RegisterRequest{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "Pass1!",
			},
			expectError: true,
			errorMsg:    "密码",
		},
		{
			name: "密码缺少数字",
			req: &RegisterRequest{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "Password!",
			},
			expectError: true,
			errorMsg:    "密码",
		},
		{
			name: "重复的用户名",
			req: &RegisterRequest{
				Username: "testuser", // 与第一个测试用例重复
				Email:    "another@example.com",
				Password: "Password123!",
			},
			expectError: true,
			errorMsg:    "已存在",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := service.Register(tt.req)

			if tt.expectError {
				if err == nil {
					t.Error("期望返回错误，但没有错误")
				}
				if tt.errorMsg != "" && err != nil {
					if !contains(err.Error(), tt.errorMsg) {
						t.Errorf("错误信息 = %q, 期望包含 %q", err.Error(), tt.errorMsg)
					}
				}
			} else {
				if err != nil {
					t.Errorf("不期望错误，但得到: %v", err)
				}
				if user == nil {
					t.Error("用户对象为 nil")
				} else {
					if user.Username != tt.req.Username {
						t.Errorf("用户名 = %q, 期望 %q", user.Username, tt.req.Username)
					}
					if user.Email != tt.req.Email {
						t.Errorf("邮箱 = %q, 期望 %q", user.Email, tt.req.Email)
					}
					if user.Role != "user" {
						t.Errorf("角色 = %q, 期望 %q", user.Role, "user")
					}
					if user.Status != "normal" {
						t.Errorf("状态 = %q, 期望 %q", user.Status, "normal")
					}
					if user.VipLevel != 0 {
						t.Errorf("VIP 等级 = %d, 期望 0", user.VipLevel)
					}
					if user.PasswordHash == "" {
						t.Error("密码哈希为空")
					}
					// 验证密码哈希
					if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(tt.req.Password)); err != nil {
						t.Error("密码哈希验证失败")
					}
				}
			}
		})
	}
}

func TestAuthService_Login(t *testing.T) {
	db := setupTestDB(t)
	service := NewAuthService(db)

	// 先注册一个用户
	password := "Password123!"
	registerReq := &RegisterRequest{
		Username: "logintest",
		Email:    "login@example.com",
		Password: password,
	}
	_, err := service.Register(registerReq)
	if err != nil {
		t.Fatalf("注册测试用户失败: %v", err)
	}

	tests := []struct {
		name        string
		req         *LoginRequest
		expectError bool
		errorMsg    string
	}{
		{
			name: "使用用户名登录成功",
			req: &LoginRequest{
				Account:  "logintest",
				Password: password,
			},
			expectError: false,
		},
		{
			name: "使用邮箱登录成功",
			req: &LoginRequest{
				Account:  "login@example.com",
				Password: password,
			},
			expectError: false,
		},
		{
			name:        "空请求体",
			req:         nil,
			expectError: true,
			errorMsg:    "请求体不能为空",
		},
		{
			name: "账号为空",
			req: &LoginRequest{
				Account:  "",
				Password: password,
			},
			expectError: true,
			errorMsg:    "不能为空",
		},
		{
			name: "密码为空",
			req: &LoginRequest{
				Account:  "logintest",
				Password: "",
			},
			expectError: true,
			errorMsg:    "不能为空",
		},
		{
			name: "用户不存在",
			req: &LoginRequest{
				Account:  "nonexistent",
				Password: password,
			},
			expectError: true,
			errorMsg:    "用户名/邮箱或密码错误",
		},
		{
			name: "密码错误",
			req: &LoginRequest{
				Account:  "logintest",
				Password: "WrongPassword123!",
			},
			expectError: true,
			errorMsg:    "密码错误",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.Login(tt.req)

			if tt.expectError {
				if err == nil {
					t.Error("期望返回错误，但没有错误")
				}
				if tt.errorMsg != "" && err != nil {
					if !contains(err.Error(), tt.errorMsg) {
						t.Errorf("错误信息 = %q, 期望包含 %q", err.Error(), tt.errorMsg)
					}
				}
			} else {
				if err != nil {
					t.Errorf("不期望错误，但得到: %v", err)
				}
				if result == nil {
					t.Error("登录结果为 nil")
				} else {
					if result.AccessToken == "" {
						t.Error("访问令牌为空")
					}
					if result.TokenType != "Bearer" {
						t.Errorf("令牌类型 = %q, 期望 %q", result.TokenType, "Bearer")
					}
					if result.ExpiresIn <= 0 {
						t.Errorf("过期时间 = %d, 应该大于 0", result.ExpiresIn)
					}
					if result.User == nil {
						t.Error("用户对象为 nil")
					} else {
						if result.User.Username != "logintest" {
							t.Errorf("用户名 = %q, 期望 %q", result.User.Username, "logintest")
						}
						if result.User.LastLoginAt == nil {
							t.Error("最后登录时间未更新")
						}
					}
				}
			}
		})
	}
}

func TestAuthService_ChangePassword(t *testing.T) {
	db := setupTestDB(t)
	service := NewAuthService(db)

	// 注册测试用户
	oldPassword := "OldPassword123!"
	registerReq := &RegisterRequest{
		Username: "pwdtest",
		Email:    "pwd@example.com",
		Password: oldPassword,
	}
	user, err := service.Register(registerReq)
	if err != nil {
		t.Fatalf("注册测试用户失败: %v", err)
	}

	tests := []struct {
		name        string
		userID      uint
		oldPassword string
		newPassword string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "成功修改密码",
			userID:      user.ID,
			oldPassword: oldPassword,
			newPassword: "NewPassword123!",
			expectError: false,
		},
		{
			name:        "旧密码错误",
			userID:      user.ID,
			oldPassword: "WrongPassword123!",
			newPassword: "NewPassword123!",
			expectError: true,
			errorMsg:    "原密码错误",
		},
		{
			name:        "新密码格式错误",
			userID:      user.ID,
			oldPassword: "NewPassword123!", // 使用上次修改后的密码
			newPassword: "weak",
			expectError: true,
			errorMsg:    "密码",
		},
		{
			name:        "用户不存在",
			userID:      99999,
			oldPassword: oldPassword,
			newPassword: "NewPassword123!",
			expectError: true,
			errorMsg:    "用户不存在",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ChangePassword(tt.userID, tt.oldPassword, tt.newPassword)

			if tt.expectError {
				if err == nil {
					t.Error("期望返回错误，但没有错误")
				}
				if tt.errorMsg != "" && err != nil {
					if !contains(err.Error(), tt.errorMsg) {
						t.Errorf("错误信息 = %q, 期望包含 %q", err.Error(), tt.errorMsg)
					}
				}
			} else {
				if err != nil {
					t.Errorf("不期望错误，但得到: %v", err)
				}

				// 验证新密码
				var updatedUser models.User
				if err := db.First(&updatedUser, tt.userID).Error; err != nil {
					t.Fatalf("查询用户失败: %v", err)
				}
				if err := bcrypt.CompareHashAndPassword([]byte(updatedUser.PasswordHash), []byte(tt.newPassword)); err != nil {
					t.Error("新密码验证失败")
				}
			}
		})
	}
}

func TestAuthService_GetUserByID(t *testing.T) {
	db := setupTestDB(t)
	service := NewAuthService(db)

	// 创建测试用户
	registerReq := &RegisterRequest{
		Username: "gettest",
		Email:    "get@example.com",
		Password: "Password123!",
	}
	user, err := service.Register(registerReq)
	if err != nil {
		t.Fatalf("注册测试用户失败: %v", err)
	}

	tests := []struct {
		name        string
		userID      uint
		expectError bool
		errorMsg    string
	}{
		{
			name:        "成功获取用户",
			userID:      user.ID,
			expectError: false,
		},
		{
			name:        "用户不存在",
			userID:      99999,
			expectError: true,
			errorMsg:    "用户不存在",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.GetMe(tt.userID)

			if tt.expectError {
				if err == nil {
					t.Error("期望返回错误，但没有错误")
				}
				if tt.errorMsg != "" && err != nil {
					if !contains(err.Error(), tt.errorMsg) {
						t.Errorf("错误信息 = %q, 期望包含 %q", err.Error(), tt.errorMsg)
					}
				}
			} else {
				if err != nil {
					t.Errorf("不期望错误，但得到: %v", err)
				}
				if result == nil {
					t.Error("用户对象为 nil")
				} else {
					if result.ID != tt.userID {
						t.Errorf("用户 ID = %d, 期望 %d", result.ID, tt.userID)
					}
					if result.Username != "gettest" {
						t.Errorf("用户名 = %q, 期望 %q", result.Username, "gettest")
					}
				}
			}
		})
	}
}

func TestAuthService_UpdateLastLogin(t *testing.T) {
	db := setupTestDB(t)
	service := NewAuthService(db)

	// 创建测试用户
	registerReq := &RegisterRequest{
		Username: "loginupdate",
		Email:    "loginupdate@example.com",
		Password: "Password123!",
	}
	user, err := service.Register(registerReq)
	if err != nil {
		t.Fatalf("注册测试用户失败: %v", err)
	}

	// 初始状态 LastLoginAt 应该为 nil
	if user.LastLoginAt != nil {
		t.Error("初始 LastLoginAt 应该为 nil")
	}

	// 通过登录流程更新最后登录时间
	_, err = service.Login(&LoginRequest{
		Account:  registerReq.Username,
		Password: registerReq.Password,
	})
	if err != nil {
		t.Fatalf("登录失败: %v", err)
	}

	// 验证更新
	var updatedUser models.User
	if err := db.First(&updatedUser, user.ID).Error; err != nil {
		t.Fatalf("查询用户失败: %v", err)
	}

	if updatedUser.LastLoginAt == nil {
		t.Error("LastLoginAt 未更新")
	} else {
		// 验证时间在合理范围内（1 分钟内）
		if time.Since(*updatedUser.LastLoginAt) > time.Minute {
			t.Error("LastLoginAt 时间不正确")
		}
	}
}

// 辅助函数：检查字符串是否包含子串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
