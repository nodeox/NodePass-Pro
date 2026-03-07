package utils

import (
	"time"

	"github.com/gin-gonic/gin"
)

// ErrorInfo 错误信息。
type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// APIResponse 统一响应体。
type APIResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     *ErrorInfo  `json:"error,omitempty"`
	Message   string      `json:"message,omitempty"`
	Timestamp string      `json:"timestamp"`
}

// Success 返回成功响应。
func Success(c *gin.Context, data interface{}) {
	c.JSON(200, APIResponse{
		Success:   true,
		Data:      data,
		Message:   "success",
		Timestamp: time.Now().Format(time.RFC3339),
	})
}

// Error 返回错误响应。
func Error(c *gin.Context, code int, errCode string, message string) {
	c.JSON(code, APIResponse{
		Success: false,
		Error: &ErrorInfo{
			Code:    errCode,
			Message: message,
		},
		Timestamp: time.Now().Format(time.RFC3339),
	})
}

// SuccessResponse 兼容旧调用。
func SuccessResponse(c *gin.Context, data interface{}, message string) {
	resp := APIResponse{
		Success:   true,
		Data:      data,
		Timestamp: time.Now().Format(time.RFC3339),
	}
	if message == "" {
		resp.Message = "success"
	} else {
		resp.Message = message
	}
	c.JSON(200, resp)
}

// ErrorResponse 兼容旧调用。
func ErrorResponse(c *gin.Context, statusCode int, errCode string, message string) {
	Error(c, statusCode, errCode, message)
}
