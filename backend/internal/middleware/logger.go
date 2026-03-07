package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

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
