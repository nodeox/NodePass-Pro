package utils

import (
	"net/http"
	"strings"
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

// SuccessResponse 统一成功响应（兼容两种调用）：
// 1) SuccessResponse(c, data, message)
// 2) SuccessResponse(c, statusCode, data, message)
func SuccessResponse(c *gin.Context, args ...interface{}) {
	statusCode := http.StatusOK
	var data interface{}
	message := "success"

	switch len(args) {
	case 2:
		data = args[0]
		if msg, ok := args[1].(string); ok && strings.TrimSpace(msg) != "" {
			message = msg
		}
	case 3:
		if code, ok := args[0].(int); ok {
			statusCode = code
		}
		data = args[1]
		if msg, ok := args[2].(string); ok && strings.TrimSpace(msg) != "" {
			message = msg
		}
	default:
		Error(c, http.StatusInternalServerError, "RESPONSE_FORMAT_ERROR", "SuccessResponse 参数错误")
		return
	}

	resp := APIResponse{
		Success:   true,
		Data:      data,
		Timestamp: time.Now().Format(time.RFC3339),
	}
	resp.Message = message
	c.JSON(statusCode, resp)
}

// ErrorResponse 兼容旧调用。
func ErrorResponse(c *gin.Context, statusCode int, errCode string, message string) {
	Error(c, statusCode, errCode, message)
}
