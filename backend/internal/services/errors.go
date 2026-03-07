package services

import "errors"

var (
	// ErrUnauthorized 未认证。
	ErrUnauthorized = errors.New("unauthorized")
	// ErrInvalidParams 请求参数不合法。
	ErrInvalidParams = errors.New("invalid params")
	// ErrNotFound 资源不存在。
	ErrNotFound = errors.New("not found")
	// ErrForbidden 无权限访问资源。
	ErrForbidden = errors.New("forbidden")
	// ErrConflict 资源冲突。
	ErrConflict = errors.New("conflict")
	// ErrQuotaExceeded 配额不足或超限。
	ErrQuotaExceeded = errors.New("quota exceeded")
)
