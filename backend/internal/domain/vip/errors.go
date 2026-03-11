package vip

import "errors"

var (
	// ErrLevelNotFound VIP 等级不存在
	ErrLevelNotFound = errors.New("vip level not found")

	// ErrLevelExists VIP 等级已存在
	ErrLevelExists = errors.New("vip level already exists")

	// ErrInvalidLevel 无效的 VIP 等级
	ErrInvalidLevel = errors.New("invalid vip level")

	// ErrInvalidTrafficQuota 无效的流量配额
	ErrInvalidTrafficQuota = errors.New("invalid traffic quota")

	// ErrUserNotFound 用户不存在
	ErrUserNotFound = errors.New("user not found")
)
