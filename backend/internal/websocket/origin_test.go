package websocket

import (
	"net/http"
	"net/url"
	"testing"

	"nodepass-pro/backend/internal/config"
)

func TestIsLocalhost(t *testing.T) {
	tests := []struct {
		name     string
		hostname string
		expected bool
	}{
		{"localhost", "localhost", true},
		{"127.0.0.1", "127.0.0.1", true},
		{"::1", "::1", true},
		{"[::1]", "[::1]", true},
		{"Localhost uppercase", "LOCALHOST", true},
		{"example.com", "example.com", false},
		{"localhost.evil.com", "localhost.evil.com", false},
		{"127.0.0.1.evil.com", "127.0.0.1.evil.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isLocalhost(tt.hostname)
			if result != tt.expected {
				t.Errorf("isLocalhost(%q) = %v, expected %v", tt.hostname, result, tt.expected)
			}
		})
	}
}

func TestMatchOrigin(t *testing.T) {
	tests := []struct {
		name     string
		origin   string
		allowed  string
		expected bool
	}{
		{
			name:     "精确匹配完整 URL",
			origin:   "https://example.com",
			allowed:  "https://example.com",
			expected: true,
		},
		{
			name:     "精确匹配带端口",
			origin:   "http://localhost:5173",
			allowed:  "http://localhost:5173",
			expected: true,
		},
		{
			name:     "scheme 不匹配",
			origin:   "http://example.com",
			allowed:  "https://example.com",
			expected: false,
		},
		{
			name:     "端口不匹配",
			origin:   "http://localhost:5173",
			allowed:  "http://localhost:8080",
			expected: false,
		},
		{
			name:     "主机名精确匹配",
			origin:   "https://example.com",
			allowed:  "example.com",
			expected: true,
		},
		{
			name:     "通配符匹配子域名",
			origin:   "https://sub.example.com",
			allowed:  "*.example.com",
			expected: true,
		},
		{
			name:     "通配符匹配根域名",
			origin:   "https://example.com",
			allowed:  "*.example.com",
			expected: true,
		},
		{
			name:     "通配符不匹配其他域名",
			origin:   "https://evil.com",
			allowed:  "*.example.com",
			expected: false,
		},
		{
			name:     "通配符不匹配部分域名",
			origin:   "https://notexample.com",
			allowed:  "*.example.com",
			expected: false,
		},
		{
			name:     "大小写不敏感",
			origin:   "https://Example.COM",
			allowed:  "example.com",
			expected: true,
		},
		{
			name:     "空允许来源",
			origin:   "https://example.com",
			allowed:  "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originURL, err := url.Parse(tt.origin)
			if err != nil {
				t.Fatalf("解析 origin 失败: %v", err)
			}

			result := matchOrigin(originURL, tt.allowed)
			if result != tt.expected {
				t.Errorf("matchOrigin(%q, %q) = %v, expected %v",
					tt.origin, tt.allowed, result, tt.expected)
			}
		})
	}
}

func TestCheckWebSocketOrigin(t *testing.T) {
	// 保存原始配置
	originalConfig := config.GlobalConfig
	defer func() {
		config.GlobalConfig = originalConfig
	}()

	tests := []struct {
		name     string
		origin   string
		referer  string
		mode     string
		allowed  []string
		expected bool
	}{
		{
			name:     "开发模式 - localhost",
			origin:   "http://localhost:5173",
			mode:     "debug",
			expected: true,
		},
		{
			name:     "开发模式 - 127.0.0.1",
			origin:   "http://127.0.0.1:5173",
			mode:     "debug",
			expected: true,
		},
		{
			name:     "开发模式 - 无 Origin 头（拒绝）",
			origin:   "",
			mode:     "debug",
			expected: false,
		},
		{
			name:     "生产模式 - 无 Origin 头",
			origin:   "",
			mode:     "release",
			expected: false,
		},
		{
			name:     "生产模式 - 配置的允许来源",
			origin:   "https://panel.example.com",
			mode:     "release",
			allowed:  []string{"panel.example.com"},
			expected: true,
		},
		{
			name:     "生产模式 - 通配符匹配",
			origin:   "https://sub.example.com",
			mode:     "release",
			allowed:  []string{"*.example.com"},
			expected: true,
		},
		{
			name:     "生产模式 - 不在允许列表",
			origin:   "https://evil.com",
			mode:     "release",
			allowed:  []string{"example.com"},
			expected: false,
		},
		{
			name:     "从 Referer 提取 Origin",
			origin:   "",
			referer:  "https://example.com/some/path",
			mode:     "release",
			allowed:  []string{"example.com"},
			expected: true,
		},
		{
			name:     "无效的 Origin 格式",
			origin:   "not-a-valid-url",
			mode:     "release",
			allowed:  []string{"example.com"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 设置测试配置
			config.GlobalConfig = &config.Config{
				Server: config.ServerConfig{
					Mode:           tt.mode,
					AllowedOrigins: tt.allowed,
				},
			}

			// 创建测试请求
			req := &http.Request{
				Header: http.Header{},
			}
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			if tt.referer != "" {
				req.Header.Set("Referer", tt.referer)
			}

			result := checkWebSocketOrigin(req)
			if result != tt.expected {
				t.Errorf("checkWebSocketOrigin() = %v, expected %v (origin=%q, mode=%q, allowed=%v)",
					result, tt.expected, tt.origin, tt.mode, tt.allowed)
			}
		})
	}
}
