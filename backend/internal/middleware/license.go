package middleware

import (
	"net/http"
	"strings"

	"nodepass-panel/backend/internal/license"
	"nodepass-panel/backend/internal/utils"

	"github.com/gin-gonic/gin"
)

var licenseBypassPaths = []string{
	"/health",
	"/api/v1/ping",
	"/api/v1/license/status",
}

// LicenseGuard 在授权失效时拦截业务请求。
func LicenseGuard(manager *license.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if manager == nil || !manager.Enabled() {
			c.Next()
			return
		}

		path := c.Request.URL.Path
		for _, bypass := range licenseBypassPaths {
			if strings.EqualFold(path, bypass) {
				c.Next()
				return
			}
		}

		if manager.IsAllowed() {
			c.Next()
			return
		}

		status := manager.Status()
		message := strings.TrimSpace(status.Message)
		if message == "" {
			message = "授权已失效，服务不可用"
		}
		utils.Error(c, http.StatusForbidden, "LICENSE_INVALID", message)
		c.Abort()
	}
}
