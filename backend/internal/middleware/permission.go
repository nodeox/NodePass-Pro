package middleware

import (
	"net/http"
	"strings"

	"nodepass-pro/backend/internal/database"
	"nodepass-pro/backend/internal/models"
	"nodepass-pro/backend/internal/utils"

	"github.com/gin-gonic/gin"
)

// RequireRole 角色权限校验中间件。
func RequireRole(roles ...string) gin.HandlerFunc {
	allowedRoles := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		allowedRoles[strings.ToLower(strings.TrimSpace(role))] = struct{}{}
	}

	return func(c *gin.Context) {
		roleValue, exists := c.Get("role")
		if !exists {
			utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
			c.Abort()
			return
		}

		role, ok := roleValue.(string)
		if !ok {
			utils.Error(c, http.StatusForbidden, "FORBIDDEN", "无效的角色信息")
			c.Abort()
			return
		}

		if _, ok = allowedRoles[strings.ToLower(role)]; !ok {
			utils.Error(c, http.StatusForbidden, "FORBIDDEN", "无权限访问")
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequirePermission 细粒度权限校验中间件。
func RequirePermission(permission string) gin.HandlerFunc {
	permission = strings.TrimSpace(permission)

	return func(c *gin.Context) {
		if permission == "" {
			c.Next()
			return
		}

		userID, ok := getContextUserID(c)
		if !ok {
			utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
			c.Abort()
			return
		}

		if database.DB == nil {
			utils.Error(c, http.StatusInternalServerError, "DB_NOT_INITIALIZED", "数据库未初始化")
			c.Abort()
			return
		}

		var count int64
		if err := database.DB.Model(&models.UserPermission{}).
			Where("user_id = ? AND permission = ?", userID, permission).
			Count(&count).Error; err != nil {
			utils.Error(c, http.StatusInternalServerError, "QUERY_PERMISSION_FAILED", "权限校验失败")
			c.Abort()
			return
		}

		if count == 0 {
			utils.Error(c, http.StatusForbidden, "FORBIDDEN", "缺少所需权限")
			c.Abort()
			return
		}

		c.Next()
	}
}

func getContextUserID(c *gin.Context) (uint, bool) {
	value, exists := c.Get("userID")
	if !exists {
		return 0, false
	}

	switch id := value.(type) {
	case uint:
		return id, true
	case int:
		if id > 0 {
			return uint(id), true
		}
	case int64:
		if id > 0 {
			return uint(id), true
		}
	case float64:
		if id > 0 {
			return uint(id), true
		}
	}

	return 0, false
}
