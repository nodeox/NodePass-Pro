package middleware

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// 敏感路径列表（不记录详细信息）
var sensitivePaths = []string{
	"/auth/login",
	"/auth/register",
	"/auth/password",
	"/telegram/login",
}

// isSensitivePath 检查路径是否敏感
func isSensitivePath(path string) bool {
	for _, sensitive := range sensitivePaths {
		if strings.Contains(path, sensitive) {
			return true
		}
	}
	return false
}

// RequestLogger 记录请求日志。
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		startAt := time.Now()
		requestSize := c.Request.ContentLength
		c.Next()

		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		if requestSize < 0 {
			requestSize = 0
		}

		// 对敏感路径进行脱敏处理
		if isSensitivePath(path) {
			zap.L().Info("敏感请求",
				zap.String("method", c.Request.Method),
				zap.String("path", "[REDACTED]"),
				zap.Int("status", c.Writer.Status()),
				zap.Duration("latency", time.Since(startAt)),
				zap.String("ip", c.ClientIP()))
			return
		}

		zap.L().Info("HTTP 请求",
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", time.Since(startAt)),
			zap.String("ip", c.ClientIP()),
			zap.Int64("request_size", requestSize),
		)
	}
}
