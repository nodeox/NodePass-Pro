package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSecurityHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		mode           string
		expectedHeader string
		expectedValue  string
		shouldExist    bool
	}{
		{
			name:           "X-Content-Type-Options header",
			mode:           gin.TestMode,
			expectedHeader: "X-Content-Type-Options",
			expectedValue:  "nosniff",
			shouldExist:    true,
		},
		{
			name:           "X-Frame-Options header",
			mode:           gin.TestMode,
			expectedHeader: "X-Frame-Options",
			expectedValue:  "DENY",
			shouldExist:    true,
		},
		{
			name:           "X-XSS-Protection header",
			mode:           gin.TestMode,
			expectedHeader: "X-XSS-Protection",
			expectedValue:  "1; mode=block",
			shouldExist:    true,
		},
		{
			name:           "Content-Security-Policy header",
			mode:           gin.TestMode,
			expectedHeader: "Content-Security-Policy",
			expectedValue:  "default-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self' data:; connect-src 'self'",
			shouldExist:    true,
		},
		{
			name:           "Referrer-Policy header",
			mode:           gin.TestMode,
			expectedHeader: "Referrer-Policy",
			expectedValue:  "strict-origin-when-cross-origin",
			shouldExist:    true,
		},
		{
			name:           "Permissions-Policy header",
			mode:           gin.TestMode,
			expectedHeader: "Permissions-Policy",
			expectedValue:  "geolocation=(), microphone=(), camera=()",
			shouldExist:    true,
		},
		{
			name:           "HSTS header in test mode (should not exist)",
			mode:           gin.TestMode,
			expectedHeader: "Strict-Transport-Security",
			expectedValue:  "",
			shouldExist:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(tt.mode)

			r := gin.New()
			r.Use(SecurityHeaders())
			r.GET("/test", func(c *gin.Context) {
				c.String(http.StatusOK, "OK")
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if tt.shouldExist {
				got := w.Header().Get(tt.expectedHeader)
				if got != tt.expectedValue {
					t.Errorf("Header %s = %q, want %q", tt.expectedHeader, got, tt.expectedValue)
				}
			} else {
				if w.Header().Get(tt.expectedHeader) != "" {
					t.Errorf("Header %s should not exist in %s mode", tt.expectedHeader, tt.mode)
				}
			}
		})
	}
}

func TestSecurityHeadersInReleaseMode(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	defer gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(SecurityHeaders())
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// HSTS header should exist in release mode
	hsts := w.Header().Get("Strict-Transport-Security")
	expectedHSTS := "max-age=31536000; includeSubDomains"
	if hsts != expectedHSTS {
		t.Errorf("HSTS header = %q, want %q", hsts, expectedHSTS)
	}
}
