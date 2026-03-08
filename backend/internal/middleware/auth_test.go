package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"nodepass-pro/backend/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 设置测试配置
	config.GlobalConfig = &config.Config{
		JWT: config.JWTConfig{
			Secret:     "test-secret-key-for-testing-only",
			ExpireTime: 24,
		},
	}

	tests := []struct {
		name           string
		setupRequest   func(*http.Request)
		expectedStatus int
		expectedError  string
	}{
		{
			name: "有效的 JWT token",
			setupRequest: func(req *http.Request) {
				token := generateTestToken(t, 1, "user")
				req.Header.Set("Authorization", "Bearer "+token)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "缺少 Authorization 头",
			setupRequest: func(req *http.Request) {
				// 不设置 Authorization 头
			},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "未提供认证令牌",
		},
		{
			name: "Authorization 格式错误",
			setupRequest: func(req *http.Request) {
				req.Header.Set("Authorization", "InvalidFormat token")
			},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "认证令牌格式错误",
		},
		{
			name: "空的 Bearer token",
			setupRequest: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer ")
			},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "认证令牌为空",
		},
		{
			name: "无效的 JWT token",
			setupRequest: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer invalid.jwt.token")
			},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "无效或已过期的令牌",
		},
		{
			name: "过期的 JWT token",
			setupRequest: func(req *http.Request) {
				token := generateExpiredToken(t, 1, "user")
				req.Header.Set("Authorization", "Bearer "+token)
			},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "无效或已过期的令牌",
		},
		{
			name: "缺少 UserID 的 token",
			setupRequest: func(req *http.Request) {
				token := generateTestToken(t, 0, "user") // UserID = 0
				req.Header.Set("Authorization", "Bearer "+token)
			},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "令牌缺少用户信息",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(AuthMiddleware())
			router.GET("/test", func(c *gin.Context) {
				userID, exists := c.Get("userID")
				if !exists {
					t.Error("userID 未设置到上下文")
				}
				role, exists := c.Get("role")
				if !exists {
					t.Error("role 未设置到上下文")
				}
				c.JSON(http.StatusOK, gin.H{
					"userID": userID,
					"role":   role,
				})
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			tt.setupRequest(req)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("状态码 = %d, 期望 %d", w.Code, tt.expectedStatus)
			}
		})
	}
}

func TestAuthMiddleware_WebSocket(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config.GlobalConfig = &config.Config{
		JWT: config.JWTConfig{
			Secret:     "test-secret-key-for-testing-only",
			ExpireTime: 24,
		},
	}

	tests := []struct {
		name           string
		setupRequest   func(*http.Request)
		expectedStatus int
	}{
		{
			name: "WebSocket 升级请求通过子协议携带 token",
			setupRequest: func(req *http.Request) {
				token := generateTestToken(t, 1, "user")
				req.Header.Set("Upgrade", "websocket")
				req.Header.Set("Connection", "Upgrade")
				req.Header.Set("Sec-WebSocket-Protocol", "bearer, "+token)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "WebSocket 升级请求缺少 token",
			setupRequest: func(req *http.Request) {
				req.Header.Set("Upgrade", "websocket")
				req.Header.Set("Connection", "Upgrade")
			},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(AuthMiddleware())
			router.GET("/ws", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "ok"})
			})

			req := httptest.NewRequest(http.MethodGet, "/ws", nil)
			tt.setupRequest(req)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("状态码 = %d, 期望 %d", w.Code, tt.expectedStatus)
			}
		})
	}
}

func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		name          string
		authorization string
		expectToken   string
		expectError   bool
	}{
		{
			name:          "有效的 Bearer token",
			authorization: "Bearer valid-token-123",
			expectToken:   "valid-token-123",
			expectError:   false,
		},
		{
			name:          "Bearer 大小写不敏感",
			authorization: "bearer valid-token-123",
			expectToken:   "valid-token-123",
			expectError:   false,
		},
		{
			name:          "空的 Authorization",
			authorization: "",
			expectToken:   "",
			expectError:   true,
		},
		{
			name:          "缺少 Bearer 前缀",
			authorization: "valid-token-123",
			expectToken:   "",
			expectError:   true,
		},
		{
			name:          "空的 token",
			authorization: "Bearer ",
			expectToken:   "",
			expectError:   true,
		},
		{
			name:          "带空格的 token",
			authorization: "Bearer  token-with-spaces  ",
			expectToken:   "token-with-spaces",
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := extractBearerToken(tt.authorization)
			if tt.expectError {
				if err == nil {
					t.Error("期望返回错误，但没有错误")
				}
			} else {
				if err != nil {
					t.Errorf("不期望错误，但得到: %v", err)
				}
				if token != tt.expectToken {
					t.Errorf("token = %q, 期望 %q", token, tt.expectToken)
				}
			}
		})
	}
}

func TestIsWebSocketUpgradeRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name     string
		headers  map[string]string
		expected bool
	}{
		{
			name: "标准 WebSocket 升级请求",
			headers: map[string]string{
				"Upgrade":    "websocket",
				"Connection": "Upgrade",
			},
			expected: true,
		},
		{
			name: "Upgrade 头大小写不敏感",
			headers: map[string]string{
				"Upgrade":    "WebSocket",
				"Connection": "Upgrade",
			},
			expected: true,
		},
		{
			name: "Connection 包含 upgrade",
			headers: map[string]string{
				"Connection": "keep-alive, Upgrade",
			},
			expected: true,
		},
		{
			name: "普通 HTTP 请求",
			headers: map[string]string{
				"Content-Type": "application/json",
			},
			expected: false,
		},
		{
			name:     "空头部",
			headers:  map[string]string{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			c.Request = req

			result := isWebSocketUpgradeRequest(c)
			if result != tt.expected {
				t.Errorf("isWebSocketUpgradeRequest() = %v, 期望 %v", result, tt.expected)
			}
		})
	}
}

func TestExtractWebSocketProtocolToken(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   string
	}{
		{
			name:   "有效 bearer 子协议",
			header: "bearer, token-123",
			want:   "token-123",
		},
		{
			name:   "大小写不敏感",
			header: "Bearer, token-123",
			want:   "token-123",
		},
		{
			name:   "缺少 bearer 前缀",
			header: "token-123",
			want:   "",
		},
		{
			name:   "空 header",
			header: "",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractWebSocketProtocolToken(tt.header)
			if got != tt.want {
				t.Fatalf("extractWebSocketProtocolToken() = %q, want %q", got, tt.want)
			}
		})
	}
}

// 辅助函数：生成测试用的 JWT token
func generateTestToken(t *testing.T, userID uint, role string) string {
	claims := &JWTClaims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(config.GlobalConfig.JWT.Secret))
	if err != nil {
		t.Fatalf("生成测试 token 失败: %v", err)
	}
	return tokenString
}

// 辅助函数：生成过期的 JWT token
func generateExpiredToken(t *testing.T, userID uint, role string) string {
	claims := &JWTClaims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)), // 1 小时前过期
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(config.GlobalConfig.JWT.Secret))
	if err != nil {
		t.Fatalf("生成过期 token 失败: %v", err)
	}
	return tokenString
}
