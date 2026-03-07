package middleware

import (
	"testing"
)

func TestIsNodeAgentPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "新链路节点心跳路径",
			path:     "/api/v1/node-instances/heartbeat",
			expected: true,
		},
		{
			name:     "普通节点列表路径（应该需要 CSRF）",
			path:     "/api/v1/nodes",
			expected: false,
		},
		{
			name:     "节点详情路径（应该需要 CSRF）",
			path:     "/api/v1/nodes/123",
			expected: false,
		},
		{
			name:     "隧道路径（应该需要 CSRF）",
			path:     "/api/v1/tunnels",
			expected: false,
		},
		{
			name:     "流量配额路径（应该需要 CSRF）",
			path:     "/api/v1/traffic/quota",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNodeAgentPath(tt.path)
			if result != tt.expected {
				t.Errorf("isNodeAgentPath(%q) = %v, expected %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestGenerateCSRFToken(t *testing.T) {
	token1, err := generateCSRFToken()
	if err != nil {
		t.Fatalf("生成 CSRF 令牌失败: %v", err)
	}

	if len(token1) == 0 {
		t.Error("生成的令牌不应为空")
	}

	token2, err := generateCSRFToken()
	if err != nil {
		t.Fatalf("生成第二个 CSRF 令牌失败: %v", err)
	}

	if token1 == token2 {
		t.Error("连续生成的令牌应该不同")
	}
}
