package middleware

import (
	"net/http"
	"strings"

	"nodepass-license-unified/internal/services"
	"nodepass-license-unified/internal/utils"

	"github.com/gin-gonic/gin"
)

// Auth 管理员 JWT 中间件。
func Auth(authService *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "缺少 Authorization")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authorization 格式错误")
			c.Abort()
			return
		}

		claims, err := authService.ParseToken(parts[1])
		if err != nil {
			utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Token 无效或已过期")
			c.Abort()
			return
		}

		c.Set("admin_id", claims.AdminID)
		c.Set("admin_username", claims.Username)
		c.Next()
	}
}
