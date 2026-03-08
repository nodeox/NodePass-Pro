package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// ErrorCode 错误码类型
type ErrorCode string

// 定义标准错误码
const (
	// 通用错误码 (1000-1999)
	ErrCodeInternal       ErrorCode = "INTERNAL_ERROR"        // 内部错误
	ErrCodeInvalidRequest ErrorCode = "INVALID_REQUEST"       // 请求参数错误
	ErrCodeInvalidParams  ErrorCode = "INVALID_PARAMS"        // 参数验证失败
	ErrCodeNotFound       ErrorCode = "NOT_FOUND"             // 资源不存在
	ErrCodeConflict       ErrorCode = "CONFLICT"              // 资源冲突
	ErrCodeForbidden      ErrorCode = "FORBIDDEN"             // 禁止访问
	ErrCodeUnauthorized   ErrorCode = "UNAUTHORIZED"          // 未授权
	ErrCodeTooManyReqs    ErrorCode = "TOO_MANY_REQUESTS"     // 请求过于频繁
	ErrCodeServiceUnavail ErrorCode = "SERVICE_UNAVAILABLE"   // 服务不可用

	// 认证相关错误码 (2000-2999)
	ErrCodeAuthFailed        ErrorCode = "AUTH_FAILED"           // 认证失败
	ErrCodeTokenInvalid      ErrorCode = "TOKEN_INVALID"         // Token 无效
	ErrCodeTokenExpired      ErrorCode = "TOKEN_EXPIRED"         // Token 过期
	ErrCodePasswordIncorrect ErrorCode = "PASSWORD_INCORRECT"    // 密码错误
	ErrCodeUserNotFound      ErrorCode = "USER_NOT_FOUND"        // 用户不存在
	ErrCodeUserExists        ErrorCode = "USER_EXISTS"           // 用户已存在
	ErrCodeUserDisabled      ErrorCode = "USER_DISABLED"         // 用户已禁用
	ErrCodePermissionDenied  ErrorCode = "PERMISSION_DENIED"     // 权限不足

	// 业务相关错误码 (3000-3999)
	ErrCodeTunnelNotFound    ErrorCode = "TUNNEL_NOT_FOUND"      // 隧道不存在
	ErrCodeTunnelExists      ErrorCode = "TUNNEL_EXISTS"         // 隧道已存在
	ErrCodeTunnelStartFailed ErrorCode = "TUNNEL_START_FAILED"   // 隧道启动失败
	ErrCodeNodeNotFound      ErrorCode = "NODE_NOT_FOUND"        // 节点不存在
	ErrCodeNodeOffline       ErrorCode = "NODE_OFFLINE"          // 节点离线
	ErrCodeGroupNotFound     ErrorCode = "GROUP_NOT_FOUND"       // 节点组不存在
	ErrCodeGroupExists       ErrorCode = "GROUP_EXISTS"          // 节点组已存在
	ErrCodeVIPNotFound       ErrorCode = "VIP_NOT_FOUND"         // VIP 等级不存在
	ErrCodeVIPExpired        ErrorCode = "VIP_EXPIRED"           // VIP 已过期
	ErrCodeCodeInvalid       ErrorCode = "CODE_INVALID"          // 权益码无效
	ErrCodeCodeUsed          ErrorCode = "CODE_USED"             // 权益码已使用
	ErrCodeCodeExpired       ErrorCode = "CODE_EXPIRED"          // 权益码已过期

	// 配额相关错误码 (4000-4999)
	ErrCodeQuotaExceeded    ErrorCode = "QUOTA_EXCEEDED"        // 配额已超限
	ErrCodeTrafficExceeded  ErrorCode = "TRAFFIC_EXCEEDED"      // 流量已超限
	ErrCodeRuleLimitReached ErrorCode = "RULE_LIMIT_REACHED"    // 规则数量达到上限
	ErrCodeNodeLimitReached ErrorCode = "NODE_LIMIT_REACHED"    // 节点数量达到上限

	// 授权相关错误码 (5000-5999)
	ErrCodeLicenseInvalid  ErrorCode = "LICENSE_INVALID"        // 授权无效
	ErrCodeLicenseExpired  ErrorCode = "LICENSE_EXPIRED"        // 授权已过期
	ErrCodeLicenseNotFound ErrorCode = "LICENSE_NOT_FOUND"      // 授权不存在
)

// AppError 应用错误类型
type AppError struct {
	Code       ErrorCode              // 错误码
	Message    string                 // 错误消息
	HTTPStatus int                    // HTTP 状态码
	Err        error                  // 原始错误
	Details    map[string]interface{} // 额外详情
}

// Error 实现 error 接口
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap 实现 errors.Unwrap 接口
func (e *AppError) Unwrap() error {
	return e.Err
}

// WithError 添加原始错误
func (e *AppError) WithError(err error) *AppError {
	e.Err = err
	return e
}

// WithDetail 添加详情
func (e *AppError) WithDetail(key string, value interface{}) *AppError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// WithMessage 覆盖错误消息
func (e *AppError) WithMessage(message string) *AppError {
	e.Message = message
	return e
}

// New 创建新的应用错误
func New(code ErrorCode, message string, httpStatus int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
	}
}

// Wrap 包装现有错误
func Wrap(err error, code ErrorCode, message string, httpStatus int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
		Err:        err,
	}
}

// 预定义的错误实例
var (
	// 通用错误
	ErrInternal       = New(ErrCodeInternal, "内部错误", http.StatusInternalServerError)
	ErrInvalidRequest = New(ErrCodeInvalidRequest, "请求参数错误", http.StatusBadRequest)
	ErrInvalidParams  = New(ErrCodeInvalidParams, "参数验证失败", http.StatusBadRequest)
	ErrNotFound       = New(ErrCodeNotFound, "资源不存在", http.StatusNotFound)
	ErrConflict       = New(ErrCodeConflict, "资源冲突", http.StatusConflict)
	ErrForbidden      = New(ErrCodeForbidden, "禁止访问", http.StatusForbidden)
	ErrUnauthorized   = New(ErrCodeUnauthorized, "未授权", http.StatusUnauthorized)
	ErrTooManyReqs    = New(ErrCodeTooManyReqs, "请求过于频繁", http.StatusTooManyRequests)
	ErrServiceUnavail = New(ErrCodeServiceUnavail, "服务不可用", http.StatusServiceUnavailable)

	// 认证错误
	ErrAuthFailed        = New(ErrCodeAuthFailed, "认证失败", http.StatusUnauthorized)
	ErrTokenInvalid      = New(ErrCodeTokenInvalid, "Token 无效", http.StatusUnauthorized)
	ErrTokenExpired      = New(ErrCodeTokenExpired, "Token 已过期", http.StatusUnauthorized)
	ErrPasswordIncorrect = New(ErrCodePasswordIncorrect, "密码错误", http.StatusUnauthorized)
	ErrUserNotFound      = New(ErrCodeUserNotFound, "用户不存在", http.StatusNotFound)
	ErrUserExists        = New(ErrCodeUserExists, "用户已存在", http.StatusConflict)
	ErrUserDisabled      = New(ErrCodeUserDisabled, "用户已禁用", http.StatusForbidden)
	ErrPermissionDenied  = New(ErrCodePermissionDenied, "权限不足", http.StatusForbidden)

	// 业务错误
	ErrTunnelNotFound    = New(ErrCodeTunnelNotFound, "隧道不存在", http.StatusNotFound)
	ErrTunnelExists      = New(ErrCodeTunnelExists, "隧道已存在", http.StatusConflict)
	ErrTunnelStartFailed = New(ErrCodeTunnelStartFailed, "隧道启动失败", http.StatusInternalServerError)
	ErrNodeNotFound      = New(ErrCodeNodeNotFound, "节点不存在", http.StatusNotFound)
	ErrNodeOffline       = New(ErrCodeNodeOffline, "节点离线", http.StatusServiceUnavailable)
	ErrGroupNotFound     = New(ErrCodeGroupNotFound, "节点组不存在", http.StatusNotFound)
	ErrGroupExists       = New(ErrCodeGroupExists, "节点组已存在", http.StatusConflict)
	ErrVIPNotFound       = New(ErrCodeVIPNotFound, "VIP 等级不存在", http.StatusNotFound)
	ErrVIPExpired        = New(ErrCodeVIPExpired, "VIP 已过期", http.StatusForbidden)
	ErrCodeInvalid       = New(ErrCodeCodeInvalid, "权益码无效", http.StatusBadRequest)
	ErrCodeUsed          = New(ErrCodeCodeUsed, "权益码已使用", http.StatusConflict)
	ErrCodeExpired       = New(ErrCodeCodeExpired, "权益码已过期", http.StatusBadRequest)

	// 配额错误
	ErrQuotaExceeded    = New(ErrCodeQuotaExceeded, "配额已超限", http.StatusForbidden)
	ErrTrafficExceeded  = New(ErrCodeTrafficExceeded, "流量已超限", http.StatusForbidden)
	ErrRuleLimitReached = New(ErrCodeRuleLimitReached, "规则数量达到上限", http.StatusForbidden)
	ErrNodeLimitReached = New(ErrCodeNodeLimitReached, "节点数量达到上限", http.StatusForbidden)

	// 授权错误
	ErrLicenseInvalid  = New(ErrCodeLicenseInvalid, "授权无效", http.StatusForbidden)
	ErrLicenseExpired  = New(ErrCodeLicenseExpired, "授权已过期", http.StatusForbidden)
	ErrLicenseNotFound = New(ErrCodeLicenseNotFound, "授权不存在", http.StatusNotFound)
)

// Is 检查错误是否为指定的 AppError
func Is(err error, target *AppError) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == target.Code
	}
	return false
}

// GetHTTPStatus 获取错误对应的 HTTP 状态码
func GetHTTPStatus(err error) int {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.HTTPStatus
	}
	return http.StatusInternalServerError
}

// GetErrorCode 获取错误码
func GetErrorCode(err error) ErrorCode {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code
	}
	return ErrCodeInternal
}

// ToAppError 将普通错误转换为 AppError
func ToAppError(err error) *AppError {
	if err == nil {
		return nil
	}

	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}

	return Wrap(err, ErrCodeInternal, "内部错误", http.StatusInternalServerError)
}
