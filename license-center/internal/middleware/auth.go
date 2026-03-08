package middleware

import (
	"net/http"
	"strings"

	"nodepass-license-center/internal/services"
	"nodepass-license-center/internal/utils"

	"github.com/gin-gonic/gin"
)

// AdminAuth 管理员 JWT 认证中间件。
func AdminAuth(authService *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := strings.TrimSpace(c.GetHeader("Authorization"))
		if header == "" {
			utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "缺少 Authorization")
			c.Abort()
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authorization 格式无效")
			c.Abort()
			return
		}

		claims, err := authService.ParseToken(strings.TrimSpace(parts[1]))
		if err != nil {
			utils.Error(c, http.StatusUnauthorized, "TOKEN_INVALID", err.Error())
			c.Abort()
			return
		}

		c.Set("adminID", claims.AdminID)
		c.Set("admin_id", claims.AdminID)
		c.Set("role", claims.Role)
		c.Set("username", claims.Username)
		c.Next()
	}
}
