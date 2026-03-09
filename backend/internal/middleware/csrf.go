package middleware

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strings"
	"time"

	"nodepass-pro/backend/internal/cache"
	"nodepass-pro/backend/internal/config"
	"nodepass-pro/backend/internal/utils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const (
	csrfTokenLength = 32
	csrfTokenTTL    = 24 * time.Hour
	csrfHeaderName  = "X-CSRF-Token"
	csrfCookieName  = "csrf_token"
	csrfKeyPrefix   = "csrf:token:"
)

// csrfToken 存储 CSRF 令牌及其过期时间。
type csrfToken struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
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

		// 检查是否启用严格 CSRF 模式
		cfg := config.GlobalConfig
		strictMode := false
		if cfg != nil {
			strictMode = cfg.Server.StrictCSRF
		}

		// 非浏览器请求（无 Origin/Referer）的处理
		if !isBrowserRequest(c) {
			if strictMode {
				// 严格模式：拒绝无 Origin/Referer 的不安全请求
				if method == http.MethodPost || method == http.MethodPut ||
					method == http.MethodDelete || method == http.MethodPatch {
					zap.L().Warn("严格 CSRF 模式：拒绝无 Origin/Referer 的请求",
						zap.String("method", method),
						zap.String("path", path),
						zap.String("ip", c.ClientIP()))
					utils.Error(c, http.StatusForbidden, "CSRF_REQUIRED", "此请求需要 CSRF 保护")
					c.Abort()
					return
				}
			}
			// 非严格模式：跳过 CSRF，便于 CLI/脚本调用
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

			// 存储令牌到 Redis（如果可用）或内存
			if err := storeCSRFToken(c.Request.Context(), token); err != nil {
				zap.L().Warn("存储 CSRF 令牌失败", zap.Error(err))
			}

			// 判断是否为生产环境
			cfg := config.GlobalConfig
			isProduction := cfg != nil && cfg.Server.Mode == "release"

			// 设置 Cookie
			// 注意：CSRF Cookie 不能设置 HttpOnly，因为前端需要读取它来放入请求头
			c.SetCookie(
				csrfCookieName,
				token,
				int(csrfTokenTTL.Seconds()),
				"/",
				"",
				isProduction, // 生产环境使用 HTTPS
				false,        // 不设置 HttpOnly，允许 JavaScript 读取
			)

			// 设置 SameSite 属性
			// 注意：不能使用 c.Header() 因为会覆盖已有的 Set-Cookie
			// 需要手动修改最后一个 Set-Cookie 头
			sameSite := "Strict"
			if !isProduction {
				sameSite = "Lax" // 开发环境使用 Lax 更方便
			}

			// 获取所有 Set-Cookie 头
			cookies := c.Writer.Header().Values("Set-Cookie")
			if len(cookies) > 0 {
				// 修改最后一个 Set-Cookie（刚刚设置的 CSRF cookie）
				lastCookie := cookies[len(cookies)-1]
				if !strings.Contains(lastCookie, "SameSite=") {
					cookies[len(cookies)-1] = lastCookie + "; SameSite=" + sameSite
					// 清除并重新设置所有 cookies
					c.Writer.Header().Del("Set-Cookie")
					for _, cookie := range cookies {
						c.Writer.Header().Add("Set-Cookie", cookie)
					}
				}
			}

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
			valid, err := verifyCSRFToken(c.Request.Context(), headerToken)
			if err != nil {
				zap.L().Warn("验证 CSRF 令牌失败", zap.Error(err))
				utils.Error(c, http.StatusForbidden, "CSRF_TOKEN_INVALID", "CSRF 令牌验证失败")
				c.Abort()
				return
			}

			if !valid {
				utils.Error(c, http.StatusForbidden, "CSRF_TOKEN_INVALID", "CSRF 令牌无效或已过期")
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

func isBrowserRequest(c *gin.Context) bool {
	if c == nil {
		return false
	}
	origin := strings.TrimSpace(c.GetHeader("Origin"))
	referer := strings.TrimSpace(c.GetHeader("Referer"))
	return origin != "" || referer != ""
}

// generateCSRFToken 生成随机 CSRF 令牌。
func generateCSRFToken() (string, error) {
	bytes := make([]byte, csrfTokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// storeCSRFToken 存储 CSRF 令牌到 Redis（如果可用）。
func storeCSRFToken(ctx context.Context, token string) error {
	if !cache.Enabled() {
		// Redis 未启用时，令牌仅存在于 Cookie 中，依赖双重提交模式
		return nil
	}

	csrfData := &csrfToken{
		Token:     token,
		ExpiresAt: time.Now().Add(csrfTokenTTL),
	}

	key := csrfKeyPrefix + token
	return cache.SetJSON(ctx, key, csrfData, csrfTokenTTL)
}

// verifyCSRFToken 验证 CSRF 令牌是否有效。
func verifyCSRFToken(ctx context.Context, token string) (bool, error) {
	if !cache.Enabled() {
		// Redis 未启用时，仅验证 Cookie 和 Header 的令牌是否匹配（双重提交模式）
		// 实际的匹配验证已在调用方完成
		return true, nil
	}

	key := csrfKeyPrefix + token
	var csrfData csrfToken
	exists, err := cache.GetJSON(ctx, key, &csrfData)
	if err != nil {
		return false, err
	}

	if !exists {
		return false, nil
	}

	// 检查是否过期
	if time.Now().After(csrfData.ExpiresAt) {
		// 删除过期令牌
		_ = cache.Delete(ctx, key)
		return false, nil
	}

	return true, nil
}

// CleanupExpiredTokens 清理过期的 CSRF 令牌。
// 注意：使用 Redis 存储时，令牌会自动过期，此函数主要用于兼容性。
func CleanupExpiredTokens() {
	// 使用 Redis 时，令牌通过 TTL 自动过期，无需手动清理
	if cache.Enabled() {
		return
	}
	// Redis 未启用时，无需清理（令牌仅存在于 Cookie 中）
}

// isNodeAgentPath 判断是否是节点客户端的请求路径。
// 节点客户端的请求不需要 CSRF 保护，仅依赖 JWT 或 Token 认证。
func isNodeAgentPath(path string) bool {
	// 精确匹配的路径
	exactPaths := []string{
		"/api/v1/node-instances/heartbeat",
	}

	for _, exactPath := range exactPaths {
		if path == exactPath {
			return true
		}
	}
	return false
}
