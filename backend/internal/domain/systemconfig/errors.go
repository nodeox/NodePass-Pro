package systemconfig

import "errors"

var (
	// ErrConfigNotFound 配置不存在
	ErrConfigNotFound = errors.New("配置不存在")

	// ErrInvalidKey 无效的键
	ErrInvalidKey = errors.New("无效的键")

	// ErrUnauthorized 未授权
	ErrUnauthorized = errors.New("未授权操作")
)
