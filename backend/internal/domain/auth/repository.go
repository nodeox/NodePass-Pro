package auth

import (
	"context"
	"time"
)

// Repository 认证仓储接口
type Repository interface {
	// 用户相关
	FindUserByID(ctx context.Context, id uint) (*User, error)
	FindUserByEmail(ctx context.Context, email string) (*User, error)
	FindUserByUsername(ctx context.Context, username string) (*User, error)
	FindUserByAccount(ctx context.Context, account string) (*User, error) // 通过用户名或邮箱查找
	CreateUser(ctx context.Context, user *User) error
	UpdateUser(ctx context.Context, user *User) error
	UpdateUserPassword(ctx context.Context, userID uint, passwordHash string) error
	UpdateUserEmail(ctx context.Context, userID uint, email string) error
	UpdateUserLastLogin(ctx context.Context, userID uint, loginTime time.Time) error
	CheckUserExists(ctx context.Context, username, email string) (bool, error)
	CheckEmailExists(ctx context.Context, email string, excludeUserID uint) (bool, error)

	// 刷新令牌相关
	CreateRefreshToken(ctx context.Context, token *RefreshToken) error
	FindRefreshTokenByHash(ctx context.Context, tokenHash string) (*RefreshToken, error)
	UpdateRefreshTokenLastUsed(ctx context.Context, tokenID uint, lastUsedAt time.Time) error
	RevokeRefreshToken(ctx context.Context, tokenID uint) error
	RevokeUserRefreshTokens(ctx context.Context, userID uint) error
	ListUserRefreshTokens(ctx context.Context, userID uint) ([]*RefreshToken, error)
	DeleteExpiredRefreshTokens(ctx context.Context) (int64, error)
}
