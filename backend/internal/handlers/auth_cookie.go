package handlers

import (
	"net/http"
	"strings"

	"nodepass-pro/backend/internal/services"

	"github.com/gin-gonic/gin"
)

const (
	refreshTokenCookieName   = "nodepass_rt"
	refreshTokenCookieMaxAge = 7 * 24 * 60 * 60
	refreshTokenCookiePath   = "/api/v1"
)

func setRefreshTokenCookie(c *gin.Context, refreshToken string) {
	if c == nil {
		return
	}
	token := strings.TrimSpace(refreshToken)
	if token == "" {
		clearRefreshTokenCookie(c)
		return
	}

	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		refreshTokenCookieName,
		token,
		refreshTokenCookieMaxAge,
		refreshTokenCookiePath,
		"",
		isHTTPSRequest(c),
		true,
	)
}

func clearRefreshTokenCookie(c *gin.Context) {
	if c == nil {
		return
	}
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		refreshTokenCookieName,
		"",
		-1,
		refreshTokenCookiePath,
		"",
		isHTTPSRequest(c),
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
