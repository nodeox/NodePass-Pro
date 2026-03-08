package errors

import (
	"errors"
	"net/http"
	"testing"
)

func TestAppError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *AppError
		expected string
	}{
		{
			name:     "错误不带原始错误",
			err:      New(ErrCodeUserNotFound, "用户不存在", http.StatusNotFound),
			expected: "[USER_NOT_FOUND] 用户不存在",
		},
		{
			name:     "错误带原始错误",
			err:      Wrap(errors.New("database error"), ErrCodeInternal, "内部错误", http.StatusInternalServerError),
			expected: "[INTERNAL_ERROR] 内部错误: database error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("Error() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAppError_WithError(t *testing.T) {
	originalErr := errors.New("original error")
	appErr := ErrInternal.WithError(originalErr)

	if appErr.Err != originalErr {
		t.Errorf("WithError() did not set error correctly")
	}

	if !errors.Is(appErr, originalErr) {
		t.Errorf("errors.Is() should return true for wrapped error")
	}
}

func TestAppError_WithDetail(t *testing.T) {
	appErr := ErrInvalidParams.WithDetail("field", "username").WithDetail("reason", "too short")

	if appErr.Details["field"] != "username" {
		t.Errorf("WithDetail() did not set field correctly")
	}

	if appErr.Details["reason"] != "too short" {
		t.Errorf("WithDetail() did not set reason correctly")
	}
}

func TestAppError_WithMessage(t *testing.T) {
	customMessage := "自定义错误消息"
	appErr := ErrUserNotFound.WithMessage(customMessage)

	if appErr.Message != customMessage {
		t.Errorf("WithMessage() = %v, want %v", appErr.Message, customMessage)
	}
}

func TestIs(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		target   *AppError
		expected bool
	}{
		{
			name:     "相同的错误码",
			err:      ErrUserNotFound,
			target:   ErrUserNotFound,
			expected: true,
		},
		{
			name:     "不同的错误码",
			err:      ErrUserNotFound,
			target:   ErrUserExists,
			expected: false,
		},
		{
			name:     "包装的错误",
			err:      ErrUserNotFound.WithError(errors.New("db error")),
			target:   ErrUserNotFound,
			expected: true,
		},
		{
			name:     "普通错误",
			err:      errors.New("normal error"),
			target:   ErrInternal,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Is(tt.err, tt.target); got != tt.expected {
				t.Errorf("Is() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetHTTPStatus(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{
			name:     "AppError",
			err:      ErrUserNotFound,
			expected: http.StatusNotFound,
		},
		{
			name:     "包装的 AppError",
			err:      ErrUnauthorized.WithError(errors.New("token invalid")),
			expected: http.StatusUnauthorized,
		},
		{
			name:     "普通错误",
			err:      errors.New("normal error"),
			expected: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetHTTPStatus(tt.err); got != tt.expected {
				t.Errorf("GetHTTPStatus() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetErrorCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected ErrorCode
	}{
		{
			name:     "AppError",
			err:      ErrUserNotFound,
			expected: ErrCodeUserNotFound,
		},
		{
			name:     "包装的 AppError",
			err:      ErrTokenExpired.WithError(errors.New("jwt expired")),
			expected: ErrCodeTokenExpired,
		},
		{
			name:     "普通错误",
			err:      errors.New("normal error"),
			expected: ErrCodeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetErrorCode(tt.err); got != tt.expected {
				t.Errorf("GetErrorCode() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestToAppError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected *AppError
	}{
		{
			name:     "nil 错误",
			err:      nil,
			expected: nil,
		},
		{
			name:     "已经是 AppError",
			err:      ErrUserNotFound,
			expected: ErrUserNotFound,
		},
		{
			name:     "普通错误",
			err:      errors.New("normal error"),
			expected: nil, // 会被转换为 ErrInternal
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToAppError(tt.err)

			if tt.expected == nil && tt.err == nil {
				if got != nil {
					t.Errorf("ToAppError() = %v, want nil", got)
				}
				return
			}

			if tt.expected == nil && tt.err != nil {
				// 普通错误应该被转换为 ErrInternal
				if got.Code != ErrCodeInternal {
					t.Errorf("ToAppError() code = %v, want %v", got.Code, ErrCodeInternal)
				}
				return
			}

			if got.Code != tt.expected.Code {
				t.Errorf("ToAppError() code = %v, want %v", got.Code, tt.expected.Code)
			}
		})
	}
}

func TestPredefinedErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        *AppError
		expectCode ErrorCode
		expectHTTP int
	}{
		{"ErrInternal", ErrInternal, ErrCodeInternal, http.StatusInternalServerError},
		{"ErrInvalidRequest", ErrInvalidRequest, ErrCodeInvalidRequest, http.StatusBadRequest},
		{"ErrNotFound", ErrNotFound, ErrCodeNotFound, http.StatusNotFound},
		{"ErrUnauthorized", ErrUnauthorized, ErrCodeUnauthorized, http.StatusUnauthorized},
		{"ErrUserNotFound", ErrUserNotFound, ErrCodeUserNotFound, http.StatusNotFound},
		{"ErrUserExists", ErrUserExists, ErrCodeUserExists, http.StatusConflict},
		{"ErrTunnelNotFound", ErrTunnelNotFound, ErrCodeTunnelNotFound, http.StatusNotFound},
		{"ErrQuotaExceeded", ErrQuotaExceeded, ErrCodeQuotaExceeded, http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Code != tt.expectCode {
				t.Errorf("%s Code = %v, want %v", tt.name, tt.err.Code, tt.expectCode)
			}
			if tt.err.HTTPStatus != tt.expectHTTP {
				t.Errorf("%s HTTPStatus = %v, want %v", tt.name, tt.err.HTTPStatus, tt.expectHTTP)
			}
		})
	}
}
