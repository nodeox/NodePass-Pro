package auth

import "errors"

var (
	// ErrUserNotFound 用户不存在
	ErrUserNotFound = errors.New("user not found")

	// ErrTokenNotFound 令牌不存在
	ErrTokenNotFound = errors.New("token not found")

	// ErrInvalidCredentials 凭证无效
	ErrInvalidCredentials = errors.New("invalid credentials")

	// ErrUserBanned 用户已被封禁
	ErrUserBanned = errors.New("user banned")

	// ErrTokenExpired 令牌已过期
	ErrTokenExpired = errors.New("token expired")

	// ErrTokenRevoked 令牌已撤销
	ErrTokenRevoked = errors.New("token revoked")
)
