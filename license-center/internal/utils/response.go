package utils

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Success 返回成功响应。
func Success(c *gin.Context, data interface{}, message string) {
	if message == "" {
		message = "ok"
	}
	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"data":      data,
		"message":   message,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// Error 返回错误响应。
func Error(c *gin.Context, code int, errCode, message string) {
	if errCode == "" {
		errCode = "ERROR"
	}
	if message == "" {
		message = "请求失败"
	}
	c.JSON(code, gin.H{
		"success": false,
		"error": gin.H{
			"code":    errCode,
			"message": message,
		},
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}
