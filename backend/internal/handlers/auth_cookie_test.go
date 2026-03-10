package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"nodepass-pro/backend/internal/config"

	"github.com/gin-gonic/gin"
)

func TestResolveRefreshCookiePolicy(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name                 string
		requestURL           string
		requestHost          string
		origin               string
		referer              string
		forwardedProto       string
		forwardedHost        string
		trustForwardedHeader bool
		wantSameSite         http.SameSite
		wantSecure           bool
	}{
		{
			name:         "同域 HTTPS 使用 Strict",
			requestURL:   "https://api.example.com/api/v1/auth/refresh",
			requestHost:  "api.example.com",
			origin:       "https://api.example.com",
			wantSameSite: http.SameSiteStrictMode,
			wantSecure:   true,
		},
		{
			name:         "跨域 HTTPS 使用 None",
			requestURL:   "https://api.example.com/api/v1/auth/refresh",
			requestHost:  "api.example.com",
			origin:       "https://panel.example.com",
			wantSameSite: http.SameSiteNoneMode,
			wantSecure:   true,
		},
		{
			name:           "跨域 HTTP 使用 Lax",
			requestURL:     "http://api.example.com/api/v1/auth/refresh",
			requestHost:    "api.example.com",
			origin:         "http://panel.example.com",
			forwardedProto: "http",
			wantSameSite:   http.SameSiteLaxMode,
			wantSecure:     false,
		},
		{
			name:                 "信任反代头时使用 X-Forwarded-Host 判断同域",
			requestURL:           "http://backend:8080/api/v1/auth/refresh",
			requestHost:          "backend:8080",
			origin:               "https://api.example.com",
			forwardedProto:       "https",
			forwardedHost:        "api.example.com",
			trustForwardedHeader: true,
			wantSameSite:         http.SameSiteStrictMode,
			wantSecure:           true,
		},
		{
			name:         "无 Origin 时默认 Strict",
			requestURL:   "https://api.example.com/api/v1/auth/refresh",
			requestHost:  "api.example.com",
			wantSameSite: http.SameSiteStrictMode,
			wantSecure:   true,
		},
	}

	originalConfig := config.GlobalConfig
	t.Cleanup(func() {
		config.GlobalConfig = originalConfig
	})

	gin.SetMode(gin.TestMode)
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tc.requestURL, nil)
			if tc.requestHost != "" {
				req.Host = tc.requestHost
			}
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			if tc.referer != "" {
				req.Header.Set("Referer", tc.referer)
			}
			if tc.forwardedProto != "" {
				req.Header.Set("X-Forwarded-Proto", tc.forwardedProto)
			}
			if tc.forwardedHost != "" {
				req.Header.Set("X-Forwarded-Host", tc.forwardedHost)
			}

			config.GlobalConfig = &config.Config{
				Server: config.ServerConfig{
					TrustForwardedHeaders: tc.trustForwardedHeader,
				},
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			policy := resolveRefreshCookiePolicy(c)
			if policy.sameSite != tc.wantSameSite {
				t.Fatalf("sameSite 不符合预期，got=%v want=%v", policy.sameSite, tc.wantSameSite)
			}
			if policy.secure != tc.wantSecure {
				t.Fatalf("secure 不符合预期，got=%v want=%v", policy.secure, tc.wantSecure)
			}
		})
	}
}
