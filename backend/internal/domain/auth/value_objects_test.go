package auth

import (
	"testing"
)

func TestNewEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{
			name:    "有效邮箱",
			email:   "user@example.com",
			wantErr: false,
		},
		{
			name:    "有效邮箱（大写）",
			email:   "User@Example.COM",
			wantErr: false,
		},
		{
			name:    "空邮箱",
			email:   "",
			wantErr: true,
		},
		{
			name:    "空格邮箱",
			email:   "   ",
			wantErr: true,
		},
		{
			name:    "无效格式（缺少@）",
			email:   "userexample.com",
			wantErr: true,
		},
		{
			name:    "无效格式（缺少域名）",
			email:   "user@",
			wantErr: true,
		},
		{
			name:    "无效格式（缺少用户名）",
			email:   "@example.com",
			wantErr: true,
		},
		{
			name:    "邮箱过长（超过254字符）",
			email:   "a" + string(make([]byte, 250)) + "@example.com",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			email, err := NewEmail(tt.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewEmail() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if email == nil {
					t.Error("NewEmail() returned nil for valid email")
					return
				}
				// 验证邮箱已转换为小写
				if email.String() != "user@example.com" && tt.email == "User@Example.COM" {
					t.Errorf("NewEmail() did not convert to lowercase, got %v", email.String())
				}
			}
		})
	}
}

func TestNewUsername(t *testing.T) {
	tests := []struct {
		name     string
		username string
		wantErr  bool
	}{
		{
			name:     "有效用户名",
			username: "testuser",
			wantErr:  false,
		},
		{
			name:     "有效用户名（带数字）",
			username: "user123",
			wantErr:  false,
		},
		{
			name:     "有效用户名（带下划线）",
			username: "test_user",
			wantErr:  false,
		},
		{
			name:     "有效用户名（带连字符）",
			username: "test-user",
			wantErr:  false,
		},
		{
			name:     "空用户名",
			username: "",
			wantErr:  true,
		},
		{
			name:     "用户名过短",
			username: "ab",
			wantErr:  true,
		},
		{
			name:     "用户名过长",
			username: "verylongusernamethatexceedsthirtytwocharacters",
			wantErr:  true,
		},
		{
			name:     "包含非法字符（空格）",
			username: "test user",
			wantErr:  true,
		},
		{
			name:     "包含非法字符（特殊符号）",
			username: "test@user",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			username, err := NewUsername(tt.username)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewUsername() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && username == nil {
				t.Error("NewUsername() returned nil for valid username")
			}
		})
	}
}

func TestNewPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "有效密码",
			password: "Test1234!",
			wantErr:  false,
		},
		{
			name:     "有效密码（复杂）",
			password: "MyP@ssw0rd123",
			wantErr:  false,
		},
		{
			name:     "密码过短",
			password: "Test1!",
			wantErr:  true,
		},
		{
			name:     "密码过长",
			password: "Test1234!" + string(make([]byte, 120)),
			wantErr:  true,
		},
		{
			name:     "缺少小写字母",
			password: "TEST1234!",
			wantErr:  true,
		},
		{
			name:     "缺少大写字母",
			password: "test1234!",
			wantErr:  true,
		},
		{
			name:     "缺少数字",
			password: "TestTest!",
			wantErr:  true,
		},
		{
			name:     "缺少特殊字符",
			password: "Test1234",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			password, err := NewPassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewPassword() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && password == nil {
				t.Error("NewPassword() returned nil for valid password")
			}
		})
	}
}
