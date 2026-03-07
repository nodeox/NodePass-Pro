package middleware

import (
	"net/http"

	"nodepass-pro/backend/internal/utils"

	"github.com/gin-gonic/gin"
)

// ErrorHandler 统一处理 Gin 错误栈。
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			utils.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "服务器内部错误")
			c.Abort()
		}
	}
}
