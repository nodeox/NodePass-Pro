package auth

import (
	"testing"
	"time"
)

func TestUser_IsActive(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   bool
	}{
		{
			name:   "正常用户",
			status: "normal",
			want:   true,
		},
		{
			name:   "封禁用户",
			status: "banned",
			want:   false,
		},
		{
			name:   "其他状态",
			status: "pending",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &User{Status: tt.status}
			if got := u.IsActive(); got != tt.want {
				t.Errorf("User.IsActive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUser_IsBanned(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   bool
	}{
		{
			name:   "封禁用户",
			status: "banned",
			want:   true,
		},
		{
			name:   "正常用户",
			status: "normal",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &User{Status: tt.status}
			if got := u.IsBanned(); got != tt.want {
				t.Errorf("User.IsBanned() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUser_IsVIP(t *testing.T) {
	tests := []struct {
		name     string
		vipLevel int
		want     bool
	}{
		{
			name:     "免费用户",
			vipLevel: 0,
			want:     false,
		},
		{
			name:     "VIP 用户",
			vipLevel: 1,
			want:     true,
		},
		{
			name:     "高级 VIP 用户",
			vipLevel: 3,
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &User{VipLevel: tt.vipLevel}
			if got := u.IsVIP(); got != tt.want {
				t.Errorf("User.IsVIP() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUser_IsVIPExpired(t *testing.T) {
	now := time.Now()
	past := now.Add(-24 * time.Hour)
	future := now.Add(24 * time.Hour)

	tests := []struct {
		name         string
		vipExpiresAt *time.Time
		want         bool
	}{
		{
			name:         "VIP 未过期",
			vipExpiresAt: &future,
			want:         false,
		},
		{
			name:         "VIP 已过期",
			vipExpiresAt: &past,
			want:         true,
		},
		{
			name:         "永久 VIP",
			vipExpiresAt: nil,
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &User{VipExpiresAt: tt.vipExpiresAt}
			if got := u.IsVIPExpired(); got != tt.want {
				t.Errorf("User.IsVIPExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRefreshToken_IsValid(t *testing.T) {
	now := time.Now()
	past := now.Add(-24 * time.Hour)
	future := now.Add(24 * time.Hour)

	tests := []struct {
		name      string
		token     *RefreshToken
		want      bool
	}{
		{
			name: "有效令牌",
			token: &RefreshToken{
				IsRevoked: false,
				ExpiresAt: future,
			},
			want: true,
		},
		{
			name: "已撤销令牌",
			token: &RefreshToken{
				IsRevoked: true,
				ExpiresAt: future,
			},
			want: false,
		},
		{
			name: "已过期令牌",
			token: &RefreshToken{
				IsRevoked: false,
				ExpiresAt: past,
			},
			want: false,
		},
		{
			name: "已撤销且已过期令牌",
			token: &RefreshToken{
				IsRevoked: true,
				ExpiresAt: past,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.token.IsValid(); got != tt.want {
				t.Errorf("RefreshToken.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRefreshToken_IsExpired(t *testing.T) {
	now := time.Now()
	past := now.Add(-24 * time.Hour)
	future := now.Add(24 * time.Hour)

	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{
			name:      "未过期",
			expiresAt: future,
			want:      false,
		},
		{
			name:      "已过期",
			expiresAt: past,
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rt := &RefreshToken{ExpiresAt: tt.expiresAt}
			if got := rt.IsExpired(); got != tt.want {
				t.Errorf("RefreshToken.IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}
