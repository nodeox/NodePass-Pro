package handlers

import (
	"net/http"
	"net/url"
	"strings"

	"nodepass-pro/backend/internal/config"
	"nodepass-pro/backend/internal/services"

	"github.com/gin-gonic/gin"
)

const (
	refreshTokenCookieName   = "nodepass_rt"
	refreshTokenCookieMaxAge = 7 * 24 * 60 * 60
	refreshTokenCookiePath   = "/api/v1"
)

type refreshCookiePolicy struct {
	sameSite http.SameSite
	secure   bool
}

func setRefreshTokenCookie(c *gin.Context, refreshToken string) {
	if c == nil {
		return
	}
	token := strings.TrimSpace(refreshToken)
	if token == "" {
		clearRefreshTokenCookie(c)
		return
	}

	policy := resolveRefreshCookiePolicy(c)
	c.SetSameSite(policy.sameSite)
	c.SetCookie(
		refreshTokenCookieName,
		token,
		refreshTokenCookieMaxAge,
		refreshTokenCookiePath,
		"",
		policy.secure,
		true,
	)
}

func clearRefreshTokenCookie(c *gin.Context) {
	if c == nil {
		return
	}
	policy := resolveRefreshCookiePolicy(c)
	c.SetSameSite(policy.sameSite)
	c.SetCookie(
		refreshTokenCookieName,
		"",
		-1,
		refreshTokenCookiePath,
		"",
		policy.secure,
		true,
	)
}

func resolveRefreshToken(c *gin.Context, bodyToken string) string {
	if token := strings.TrimSpace(bodyToken); token != "" {
		return token
	}
	if c == nil {
		return ""
	}
	if token, err := c.Cookie(refreshTokenCookieName); err == nil {
		return strings.TrimSpace(token)
	}
	return ""
}

func isHTTPSRequest(c *gin.Context) bool {
	if c == nil || c.Request == nil {
		return false
	}
	if c.Request.TLS != nil {
		return true
	}
	proto := strings.TrimSpace(c.GetHeader("X-Forwarded-Proto"))
	if idx := strings.Index(proto, ","); idx >= 0 {
		proto = strings.TrimSpace(proto[:idx])
	}
	return strings.EqualFold(proto, "https")
}

func resolveRefreshCookiePolicy(c *gin.Context) refreshCookiePolicy {
	https := isHTTPSRequest(c)
	policy := refreshCookiePolicy{
		sameSite: http.SameSiteStrictMode,
		secure:   https,
	}

	if !isCrossOriginRequest(c) {
		return policy
	}

	// 跨域场景下必须放宽 SameSite，且 HTTPS 才能使用 None。
	if https {
		policy.sameSite = http.SameSiteNoneMode
		policy.secure = true
		return policy
	}

	policy.sameSite = http.SameSiteLaxMode
	policy.secure = false
	return policy
}

func isCrossOriginRequest(c *gin.Context) bool {
	if c == nil || c.Request == nil {
		return false
	}

	requestHost := resolveRequestHost(c)
	originHost := resolveOriginHost(c)
	if requestHost == "" || originHost == "" {
		return false
	}

	return !strings.EqualFold(requestHost, originHost)
}

func resolveRequestHost(c *gin.Context) string {
	if c == nil || c.Request == nil {
		return ""
	}

	trustForwarded := false
	if cfg := config.GlobalConfig; cfg != nil {
		trustForwarded = cfg.Server.TrustForwardedHeaders
	}

	if trustForwarded {
		if host := normalizeHost(c.GetHeader("X-Forwarded-Host")); host != "" {
			return host
		}
	}

	return normalizeHost(c.Request.Host)
}

func resolveOriginHost(c *gin.Context) string {
	if c == nil {
		return ""
	}
	if originHost := normalizeHost(c.GetHeader("Origin")); originHost != "" {
		return originHost
	}
	return normalizeHost(c.GetHeader("Referer"))
}

func normalizeHost(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	if idx := strings.Index(value, ","); idx >= 0 {
		value = strings.TrimSpace(value[:idx])
	}

	parseTarget := value
	if !strings.Contains(parseTarget, "://") {
		parseTarget = "//" + parseTarget
	}

	parsed, err := url.Parse(parseTarget)
	if err != nil {
		return ""
	}

	host := strings.TrimSpace(parsed.Hostname())
	if host == "" {
		return ""
	}
	return strings.ToLower(host)
}

func buildAuthLoginPayload(result *services.LoginResult) gin.H {
	if result == nil {
		return gin.H{}
	}
	return gin.H{
		"token":        result.AccessToken,
		"access_token": result.AccessToken,
		"expires_in":   result.ExpiresIn,
		"token_type":   result.TokenType,
		"user":         result.User,
	}
}
