package middleware

import (
	"github.com/gin-gonic/gin"
)

// SecurityHeaders 添加安全相关的 HTTP 响应头。
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 防止 MIME 类型嗅探
		c.Header("X-Content-Type-Options", "nosniff")

		// 防止页面被嵌入到 iframe 中（防止点击劫持）
		c.Header("X-Frame-Options", "DENY")

		// 启用浏览器的 XSS 过滤器
		c.Header("X-XSS-Protection", "1; mode=block")

		// 强制使用 HTTPS（仅在生产环境启用）
		if gin.Mode() == gin.ReleaseMode {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		// 内容安全策略（CSP）
		// 允许同源资源，允许内联样式（Ant Design 需要），允许 data: 图片
		c.Header("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self' data:; connect-src 'self'")

		// 控制浏览器发送的 Referer 信息
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// 控制浏览器功能和 API 的访问权限
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		c.Next()
	}
}
