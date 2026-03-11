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

	// 部署镜像上传允许更大的请求体（约 11GB）
	deployAssetUploadMaxBodySize = 11 * 1024 * 1024 * 1024
)

var requestBodyLimitOverrides = map[string]int64{
	"/api/v1/system/deploy-assets/upload": deployAssetUploadMaxBodySize,
}

// RequestBodyLimit 限制请求体大小的中间件。
func RequestBodyLimit(maxSize int64) gin.HandlerFunc {
	if maxSize <= 0 {
		maxSize = defaultMaxBodySize
	}

	return func(c *gin.Context) {
		effectiveMaxSize := maxSize
		if overrideSize, ok := requestBodyLimitOverrides[c.Request.URL.Path]; ok && overrideSize > effectiveMaxSize {
			effectiveMaxSize = overrideSize
		}

		// 检查 Content-Length 头
		if c.Request.ContentLength > effectiveMaxSize {
			utils.Error(c, http.StatusRequestEntityTooLarge, "REQUEST_TOO_LARGE",
				"请求体过大，最大允许 "+formatBytes(effectiveMaxSize))
			c.Abort()
			return
		}

		// 限制请求体读取大小
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, effectiveMaxSize)

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
