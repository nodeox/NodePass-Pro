package middleware

import (
	"net/http"
	"strconv"

	"nodepass-pro/backend/internal/utils"

	"github.com/gin-gonic/gin"
)

const (
	// 默认请求体大小限制：10MB
	defaultMaxBodySize = 10 * 1024 * 1024
)

// RequestBodyLimit 限制请求体大小的中间件。
func RequestBodyLimit(maxSize int64) gin.HandlerFunc {
	if maxSize <= 0 {
		maxSize = defaultMaxBodySize
	}

	return func(c *gin.Context) {
		// 检查 Content-Length 头
		if c.Request.ContentLength > maxSize {
			utils.Error(c, http.StatusRequestEntityTooLarge, "REQUEST_TOO_LARGE",
				"请求体过大，最大允许 "+formatBytes(maxSize))
			c.Abort()
			return
		}

		// 限制请求体读取大小
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSize)

		c.Next()
	}
}

// formatBytes 格式化字节数为人类可读的格式。
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return strconv.FormatInt(bytes, 10) + " B"
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	units := []string{"KB", "MB", "GB", "TB"}
	return strconv.FormatInt(bytes/div, 10) + " " + units[exp]
}
