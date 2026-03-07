package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"sync"
	"time"

	"nodepass-panel/backend/internal/config"
	"nodepass-panel/backend/internal/utils"

	"github.com/gin-gonic/gin"
)

const (
	csrfTokenLength = 32
	csrfTokenTTL    = 24 * time.Hour
	csrfHeaderName  = "X-CSRF-Token"
	csrfCookieName  = "csrf_token"
)

// csrfToken 存储 CSRF 令牌及其过期时间。
type csrfToken struct {
	Token     string
	ExpiresAt time.Time
}

// csrfStore CSRF 令牌存储。
type csrfStore struct {
	mu     sync.RWMutex
	tokens map[string]*csrfToken
}

var store = &csrfStore{
	tokens: make(map[string]*csrfToken),
}

// CSRFProtection CSRF 保护中间件。
// 对于 GET、HEAD、OPTIONS 请求，生成并返回 CSRF 令牌。
// 对于 POST、PUT、DELETE、PATCH 请求，验证 CSRF 令牌。
// 注意：此中间件主要用于浏览器环境，对于纯 API 调用（如移动端、节点客户端），
// 应该跳过 CSRF 验证，仅依赖 JWT 认证。
func CSRFProtection() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method

		// 跳过节点客户端的请求（通过路径判断）
		path := c.Request.URL.Path
		if isNodeAgentPath(path) {
			c.Next()
			return
		}

		// 对于安全方法（GET、HEAD、OPTIONS），生成并设置 CSRF 令牌
		if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
			token, err := generateCSRFToken()
			if err != nil {
				utils.Error(c, http.StatusInternalServerError, "CSRF_GENERATION_FAILED", "生成 CSRF 令牌失败")
				c.Abort()
				return
			}

			// 存储令牌
			store.mu.Lock()
			store.tokens[token] = &csrfToken{
				Token:     token,
				ExpiresAt: time.Now().Add(csrfTokenTTL),
			}
			store.mu.Unlock()

			// 判断是否为生产环境
			cfg := config.GlobalConfig
			isProduction := cfg != nil && cfg.Server.Mode == "release"

			// 设置 Cookie
			c.SetCookie(
				csrfCookieName,
				token,
				int(csrfTokenTTL.Seconds()),
				"/",
				"",
				isProduction, // 生产环境使用 HTTPS
				true,         // HttpOnly
			)

			// 设置 SameSite 属性（Gin 的 SetCookie 不支持，需要手动设置）
			sameSite := "Strict"
			if !isProduction {
				sameSite = "Lax" // 开发环境使用 Lax 更方便
			}
			c.Header("Set-Cookie", c.Writer.Header().Get("Set-Cookie")+"; SameSite="+sameSite)

			// 同时在响应头中返回令牌（方便前端获取）
			c.Header(csrfHeaderName, token)
			c.Next()
			return
		}

		// 对于不安全方法（POST、PUT、DELETE、PATCH），验证 CSRF 令牌
		if method == http.MethodPost || method == http.MethodPut ||
			method == http.MethodDelete || method == http.MethodPatch {

			// 从请求头获取令牌
			headerToken := c.GetHeader(csrfHeaderName)
			if headerToken == "" {
				utils.Error(c, http.StatusForbidden, "CSRF_TOKEN_MISSING", "缺少 CSRF 令牌")
				c.Abort()
				return
			}

			// 从 Cookie 获取令牌
			cookieToken, err := c.Cookie(csrfCookieName)
			if err != nil {
				utils.Error(c, http.StatusForbidden, "CSRF_TOKEN_MISSING", "缺少 CSRF Cookie")
				c.Abort()
				return
			}

			// 验证令牌是否匹配
			if headerToken != cookieToken {
				utils.Error(c, http.StatusForbidden, "CSRF_TOKEN_MISMATCH", "CSRF 令牌不匹配")
				c.Abort()
				return
			}

			// 验证令牌是否存在且未过期
			store.mu.RLock()
			storedToken, exists := store.tokens[headerToken]
			store.mu.RUnlock()

			if !exists {
				utils.Error(c, http.StatusForbidden, "CSRF_TOKEN_INVALID", "CSRF 令牌无效")
				c.Abort()
				return
			}

			if time.Now().After(storedToken.ExpiresAt) {
				// 清理过期令牌
				store.mu.Lock()
				delete(store.tokens, headerToken)
				store.mu.Unlock()

				utils.Error(c, http.StatusForbidden, "CSRF_TOKEN_EXPIRED", "CSRF 令牌已过期")
				c.Abort()
				return
			}

			c.Next()
			return
		}

		// 其他方法直接通过
		c.Next()
	}
}

// generateCSRFToken 生成随机 CSRF 令牌。
func generateCSRFToken() (string, error) {
	bytes := make([]byte, csrfTokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// CleanupExpiredTokens 清理过期的 CSRF 令牌（应该定期调用）。
func CleanupExpiredTokens() {
	store.mu.Lock()
	defer store.mu.Unlock()

	now := time.Now()
	for token, csrfToken := range store.tokens {
		if now.After(csrfToken.ExpiresAt) {
			delete(store.tokens, token)
		}
	}
}

// isNodeAgentPath 判断是否是节点客户端的请求路径。
// 节点客户端的请求不需要 CSRF 保护，仅依赖 JWT 或 Token 认证。
func isNodeAgentPath(path string) bool {
	nodeAgentPaths := []string{
		"/api/v1/nodes/register",
		"/api/v1/nodes/heartbeat",
		"/api/v1/nodes/",
		"/api/v1/traffic/report",
	}

	for _, prefix := range nodeAgentPaths {
		if len(path) >= len(prefix) && path[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}
