package commands

import (
	"context"
	"testing"
	"time"

	"nodepass-pro/backend/internal/domain/auth"
	"nodepass-pro/backend/internal/domain/vip"
)

// MockAuthRepository 模拟认证仓储
type MockAuthRepository struct {
	users         map[uint]*auth.User
	usersByEmail  map[string]*auth.User
	refreshTokens map[string]*auth.RefreshToken
}

func NewMockAuthRepository() *MockAuthRepository {
	return &MockAuthRepository{
		users:         make(map[uint]*auth.User),
		usersByEmail:  make(map[string]*auth.User),
		refreshTokens: make(map[string]*auth.RefreshToken),
	}
}

func (m *MockAuthRepository) FindUserByID(ctx context.Context, id uint) (*auth.User, error) {
	user, ok := m.users[id]
	if !ok {
		return nil, auth.ErrUserNotFound
	}
	return user, nil
}

func (m *MockAuthRepository) FindUserByEmail(ctx context.Context, email string) (*auth.User, error) {
	user, ok := m.usersByEmail[email]
	if !ok {
		return nil, auth.ErrUserNotFound
	}
	return user, nil
}

func (m *MockAuthRepository) FindUserByUsername(ctx context.Context, username string) (*auth.User, error) {
	for _, user := range m.users {
		if user.Username == username {
			return user, nil
		}
	}
	return nil, auth.ErrUserNotFound
}

func (m *MockAuthRepository) FindUserByAccount(ctx context.Context, account string) (*auth.User, error) {
	// 先尝试邮箱
	if user, ok := m.usersByEmail[account]; ok {
		return user, nil
	}
	// 再尝试用户名
	for _, user := range m.users {
		if user.Username == account {
			return user, nil
		}
	}
	return nil, auth.ErrUserNotFound
}

func (m *MockAuthRepository) CreateUser(ctx context.Context, user *auth.User) error {
	user.ID = uint(len(m.users) + 1)
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	m.users[user.ID] = user
	m.usersByEmail[user.Email] = user
	return nil
}

func (m *MockAuthRepository) UpdateUser(ctx context.Context, user *auth.User) error {
	if _, ok := m.users[user.ID]; !ok {
		return auth.ErrUserNotFound
	}
	user.UpdatedAt = time.Now()
	m.users[user.ID] = user
	return nil
}

func (m *MockAuthRepository) UpdateUserPassword(ctx context.Context, userID uint, passwordHash string) error {
	user, ok := m.users[userID]
	if !ok {
		return auth.ErrUserNotFound
	}
	user.PasswordHash = passwordHash
	user.UpdatedAt = time.Now()
	return nil
}

func (m *MockAuthRepository) UpdateUserEmail(ctx context.Context, userID uint, email string) error {
	user, ok := m.users[userID]
	if !ok {
		return auth.ErrUserNotFound
	}
	delete(m.usersByEmail, user.Email)
	user.Email = email
	user.UpdatedAt = time.Now()
	m.usersByEmail[email] = user
	return nil
}

func (m *MockAuthRepository) UpdateUserLastLogin(ctx context.Context, userID uint, loginTime time.Time) error {
	user, ok := m.users[userID]
	if !ok {
		return auth.ErrUserNotFound
	}
	user.LastLoginAt = &loginTime
	return nil
}

func (m *MockAuthRepository) CheckUserExists(ctx context.Context, username, email string) (bool, error) {
	for _, user := range m.users {
		if user.Username == username || user.Email == email {
			return true, nil
		}
	}
	return false, nil
}

func (m *MockAuthRepository) CheckEmailExists(ctx context.Context, email string, excludeUserID uint) (bool, error) {
	for _, user := range m.users {
		if user.Email == email && user.ID != excludeUserID {
			return true, nil
		}
	}
	return false, nil
}

func (m *MockAuthRepository) CreateRefreshToken(ctx context.Context, token *auth.RefreshToken) error {
	token.ID = uint(len(m.refreshTokens) + 1)
	token.CreatedAt = time.Now()
	token.UpdatedAt = time.Now()
	m.refreshTokens[token.TokenHash] = token
	return nil
}

func (m *MockAuthRepository) FindRefreshTokenByHash(ctx context.Context, tokenHash string) (*auth.RefreshToken, error) {
	token, ok := m.refreshTokens[tokenHash]
	if !ok {
		return nil, auth.ErrTokenNotFound
	}
	return token, nil
}

func (m *MockAuthRepository) UpdateRefreshTokenLastUsed(ctx context.Context, tokenID uint, lastUsedAt time.Time) error {
	for _, token := range m.refreshTokens {
		if token.ID == tokenID {
			token.LastUsedAt = &lastUsedAt
			return nil
		}
	}
	return auth.ErrTokenNotFound
}

func (m *MockAuthRepository) RevokeRefreshToken(ctx context.Context, tokenID uint) error {
	for _, token := range m.refreshTokens {
		if token.ID == tokenID {
			token.IsRevoked = true
			return nil
		}
	}
	return auth.ErrTokenNotFound
}

func (m *MockAuthRepository) RevokeUserRefreshTokens(ctx context.Context, userID uint) error {
	for _, token := range m.refreshTokens {
		if token.UserID == userID {
			token.IsRevoked = true
		}
	}
	return nil
}

func (m *MockAuthRepository) ListUserRefreshTokens(ctx context.Context, userID uint) ([]*auth.RefreshToken, error) {
	var tokens []*auth.RefreshToken
	for _, token := range m.refreshTokens {
		if token.UserID == userID {
			tokens = append(tokens, token)
		}
	}
	return tokens, nil
}

func (m *MockAuthRepository) DeleteExpiredRefreshTokens(ctx context.Context) (int64, error) {
	count := int64(0)
	now := time.Now()
	for hash, token := range m.refreshTokens {
		if token.ExpiresAt.Before(now) {
			delete(m.refreshTokens, hash)
			count++
		}
	}
	return count, nil
}

// MockVIPRepository 模拟 VIP 仓储
type MockVIPRepository struct {
	levels map[int]*vip.VIPLevel
}

func NewMockVIPRepository() *MockVIPRepository {
	repo := &MockVIPRepository{
		levels: make(map[int]*vip.VIPLevel),
	}
	// 添加默认的 Level 0 配置
	repo.levels[0] = &vip.VIPLevel{
		ID:                      1,
		Level:                   0,
		Name:                    "免费用户",
		TrafficQuota:            10 * 1024 * 1024 * 1024, // 10GB
		MaxRules:                5,
		MaxBandwidth:            100,
		MaxSelfHostedEntryNodes: 0,
		MaxSelfHostedExitNodes:  0,
	}
	return repo
}

func (m *MockVIPRepository) FindLevelByID(ctx context.Context, id uint) (*vip.VIPLevel, error) {
	for _, level := range m.levels {
		if level.ID == id {
			return level, nil
		}
	}
	return nil, vip.ErrLevelNotFound
}

func (m *MockVIPRepository) FindLevelByLevel(ctx context.Context, level int) (*vip.VIPLevel, error) {
	l, ok := m.levels[level]
	if !ok {
		return nil, vip.ErrLevelNotFound
	}
	return l, nil
}

func (m *MockVIPRepository) FindByLevel(ctx context.Context, level int) (*vip.VIPLevel, error) {
	return m.FindLevelByLevel(ctx, level)
}

func (m *MockVIPRepository) ListLevels(ctx context.Context) ([]*vip.VIPLevel, error) {
	result := make([]*vip.VIPLevel, 0, len(m.levels))
	for _, level := range m.levels {
		result = append(result, level)
	}
	return result, nil
}

func (m *MockVIPRepository) CreateLevel(ctx context.Context, level *vip.VIPLevel) error {
	level.ID = uint(len(m.levels) + 1)
	m.levels[level.Level] = level
	return nil
}

func (m *MockVIPRepository) UpdateLevel(ctx context.Context, level *vip.VIPLevel) error {
	if _, ok := m.levels[level.Level]; !ok {
		return vip.ErrLevelNotFound
	}
	m.levels[level.Level] = level
	return nil
}

func (m *MockVIPRepository) DeleteLevel(ctx context.Context, id uint) error {
	for level, l := range m.levels {
		if l.ID == id {
			delete(m.levels, level)
			return nil
		}
	}
	return vip.ErrLevelNotFound
}

func (m *MockVIPRepository) CheckLevelExists(ctx context.Context, level int) (bool, error) {
	_, ok := m.levels[level]
	return ok, nil
}

func (m *MockVIPRepository) GetUserVIP(ctx context.Context, userID uint) (*vip.UserVIP, error) {
	return nil, vip.ErrLevelNotFound
}

func (m *MockVIPRepository) UpgradeUserVIP(ctx context.Context, userID uint, level int, expiresAt *time.Time) error {
	return nil
}

func (m *MockVIPRepository) CheckExpiredUsers(ctx context.Context) ([]uint, error) {
	return nil, nil
}

func (m *MockVIPRepository) DegradeExpiredUsers(ctx context.Context, userIDs []uint) (int64, error) {
	return 0, nil
}

func TestRegisterHandler_Handle(t *testing.T) {
	authRepo := NewMockAuthRepository()
	vipRepo := NewMockVIPRepository()
	handler := NewRegisterHandler(authRepo, vipRepo, nil)

	tests := []struct {
		name    string
		cmd     RegisterCommand
		wantErr bool
	}{
		{
			name: "成功注册",
			cmd: RegisterCommand{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "Test1234!",
			},
			wantErr: false,
		},
		{
			name: "用户名过短",
			cmd: RegisterCommand{
				Username: "ab",
				Email:    "test@example.com",
				Password: "Test1234!",
			},
			wantErr: true,
		},
		{
			name: "邮箱格式错误",
			cmd: RegisterCommand{
				Username: "testuser",
				Email:    "invalid-email",
				Password: "Test1234!",
			},
			wantErr: true,
		},
		{
			name: "密码强度不足",
			cmd: RegisterCommand{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "weak",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := handler.Handle(context.Background(), tt.cmd)
			if (err != nil) != tt.wantErr {
				t.Errorf("RegisterHandler.Handle() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if result == nil {
					t.Error("RegisterHandler.Handle() returned nil result")
					return
				}
				if result.UserID == 0 {
					t.Error("RegisterHandler.Handle() returned zero UserID")
				}
				if result.Username != tt.cmd.Username {
					t.Errorf("RegisterHandler.Handle() username = %v, want %v", result.Username, tt.cmd.Username)
				}
			}
		})
	}
}

func TestRegisterHandler_Handle_DuplicateUser(t *testing.T) {
	authRepo := NewMockAuthRepository()
	vipRepo := NewMockVIPRepository()
	handler := NewRegisterHandler(authRepo, vipRepo, nil)

	// 先注册一个用户
	cmd := RegisterCommand{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "Test1234!",
	}
	_, err := handler.Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("First registration failed: %v", err)
	}

	// 尝试注册相同用户名
	cmd2 := RegisterCommand{
		Username: "testuser",
		Email:    "another@example.com",
		Password: "Test1234!",
	}
	_, err = handler.Handle(context.Background(), cmd2)
	if err == nil {
		t.Error("Expected error for duplicate username, got nil")
	}

	// 尝试注册相同邮箱
	cmd3 := RegisterCommand{
		Username: "anotheruser",
		Email:    "test@example.com",
		Password: "Test1234!",
	}
	_, err = handler.Handle(context.Background(), cmd3)
	if err == nil {
		t.Error("Expected error for duplicate email, got nil")
	}
}
