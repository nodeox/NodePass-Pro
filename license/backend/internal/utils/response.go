package utils

import (
	"time"

	"github.com/gin-gonic/gin"
)

type successResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data"`
	Message   string      `json:"message"`
	Timestamp int64       `json:"timestamp"`
}

type errorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type errorResponse struct {
	Success   bool         `json:"success"`
	Error     errorPayload `json:"error"`
	Timestamp int64        `json:"timestamp"`
}

// Success 统一成功响应。
func Success(c *gin.Context, data interface{}, message string) {
	c.JSON(200, successResponse{
		Success:   true,
		Data:      data,
		Message:   message,
		Timestamp: time.Now().Unix(),
	})
}

// Error 统一错误响应。
func Error(c *gin.Context, code int, errCode, message string) {
	c.JSON(code, errorResponse{
		Success: false,
		Error: errorPayload{
			Code:    errCode,
			Message: message,
		},
		Timestamp: time.Now().Unix(),
	})
}
