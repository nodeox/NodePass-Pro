package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	// RequestIDHeader 请求 ID 的 HTTP 头名称
	RequestIDHeader = "X-Request-ID"
	// RequestIDKey 请求 ID 在 gin.Context 中的键名
	RequestIDKey = "request_id"
)

// RequestID 请求 ID 中间件，为每个请求生成唯一标识符
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 尝试从请求头获取请求 ID
		requestID := c.GetHeader(RequestIDHeader)

		// 如果请求头中没有，则生成新的 UUID
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// 将请求 ID 存储到 context 中
		c.Set(RequestIDKey, requestID)

		// 将请求 ID 添加到响应头中
		c.Header(RequestIDHeader, requestID)

		c.Next()
	}
}

// GetRequestID 从 gin.Context 中获取请求 ID
func GetRequestID(c *gin.Context) string {
	if requestID, exists := c.Get(RequestIDKey); exists {
		if id, ok := requestID.(string); ok {
			return id
		}
	}
	return ""
}
