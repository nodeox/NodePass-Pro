package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestNewKeyRateLimiter(t *testing.T) {
	tests := []struct {
		name        string
		qps         float64
		burst       int
		expectQPS   float64
		expectBurst int
	}{
		{
			name:        "正常参数",
			qps:         5.0,
			burst:       10,
			expectQPS:   5.0,
			expectBurst: 10,
		},
		{
			name:        "QPS 为 0 使用默认值",
			qps:         0,
			burst:       10,
			expectQPS:   10.0,
			expectBurst: 10,
		},
		{
			name:        "Burst 为 0 使用默认值",
			qps:         5.0,
			burst:       0,
			expectQPS:   5.0,
			expectBurst: 20,
		},
		{
			name:        "负数参数使用默认值",
			qps:         -1,
			burst:       -1,
			expectQPS:   10.0,
			expectBurst: 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limiter := NewKeyRateLimiter(tt.qps, tt.burst)
			if limiter == nil {
				t.Fatal("限流器创建失败")
			}
			if float64(limiter.qps) != tt.expectQPS {
				t.Errorf("QPS = %v, 期望 %v", limiter.qps, tt.expectQPS)
			}
			if limiter.burst != tt.expectBurst {
				t.Errorf("Burst = %v, 期望 %v", limiter.burst, tt.expectBurst)
			}
		})
	}
}

func TestRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name          string
		qps           float64
		burst         int
		requestCount  int
		expectBlocked int
		requestDelay  time.Duration
	}{
		{
			name:          "低于限流阈值",
			qps:           10.0,
			burst:         20,
			requestCount:  5,
			expectBlocked: 0,
			requestDelay:  0,
		},
		{
			name:          "超过突发限制",
			qps:           1.0,
			burst:         5,
			requestCount:  10,
			expectBlocked: 5,
			requestDelay:  0,
		},
		{
			name:          "带延迟的请求",
			qps:           10.0,
			burst:         2,
			requestCount:  3,
			expectBlocked: 0,
			requestDelay:  150 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(RateLimit(tt.qps, tt.burst))
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "ok"})
			})

			blocked := 0
			for i := 0; i < tt.requestCount; i++ {
				if tt.requestDelay > 0 && i > 0 {
					time.Sleep(tt.requestDelay)
				}

				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				req.RemoteAddr = "192.168.1.1:12345"
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				if w.Code == http.StatusTooManyRequests {
					blocked++
				}
			}

			if blocked != tt.expectBlocked {
				t.Errorf("被限流请求数 = %d, 期望 %d", blocked, tt.expectBlocked)
			}
		})
	}
}

func TestRateLimitBy(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("按用户 ID 限流", func(t *testing.T) {
		router := gin.New()
		router.Use(RateLimitBy(1.0, 2, func(c *gin.Context) string {
			return c.GetHeader("X-User-ID")
		}))
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "ok"})
		})

		// 用户 1 发送 3 个请求，应该有 1 个被限流
		user1Blocked := 0
		for i := 0; i < 3; i++ {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("X-User-ID", "user1")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code == http.StatusTooManyRequests {
				user1Blocked++
			}
		}

		// 用户 2 发送 2 个请求，应该都通过
		user2Blocked := 0
		for i := 0; i < 2; i++ {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("X-User-ID", "user2")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code == http.StatusTooManyRequests {
				user2Blocked++
			}
		}

		if user1Blocked != 1 {
			t.Errorf("用户 1 被限流请求数 = %d, 期望 1", user1Blocked)
		}
		if user2Blocked != 0 {
			t.Errorf("用户 2 被限流请求数 = %d, 期望 0", user2Blocked)
		}
	})

	t.Run("空 key 使用 IP", func(t *testing.T) {
		router := gin.New()
		router.Use(RateLimitBy(1.0, 2, func(c *gin.Context) string {
			return "" // 返回空字符串
		}))
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "ok"})
		})

		blocked := 0
		for i := 0; i < 3; i++ {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = "192.168.1.1:12345"
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code == http.StatusTooManyRequests {
				blocked++
			}
		}

		if blocked != 1 {
			t.Errorf("被限流请求数 = %d, 期望 1", blocked)
		}
	})
}

func TestKeyRateLimiter_Allow(t *testing.T) {
	limiter := NewKeyRateLimiter(1.0, 2)

	// 前两个请求应该通过（burst = 2）
	if !limiter.allow("test-key") {
		t.Error("第 1 个请求应该通过")
	}
	if !limiter.allow("test-key") {
		t.Error("第 2 个请求应该通过")
	}

	// 第三个请求应该被限流
	if limiter.allow("test-key") {
		t.Error("第 3 个请求应该被限流")
	}

	// 等待令牌恢复
	time.Sleep(1100 * time.Millisecond)

	// 应该可以再次通过
	if !limiter.allow("test-key") {
		t.Error("等待后的请求应该通过")
	}
}

func TestKeyRateLimiter_Cleanup(t *testing.T) {
	limiter := &KeyRateLimiter{
		qps:             10,
		burst:           20,
		cleanupInterval: 100 * time.Millisecond,
		ttl:             200 * time.Millisecond,
		lastCleanupAt:   time.Now(),
	}

	// 添加一些访客
	limiter.allow("key1")
	limiter.allow("key2")
	limiter.allow("key3")

	// 验证访客存在
	count := 0
	limiter.visitors.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	if count != 3 {
		t.Errorf("访客数量 = %d, 期望 3", count)
	}

	// 等待 TTL 过期
	time.Sleep(300 * time.Millisecond)

	// 手动触发清理
	limiter.cleanup()

	// 验证过期访客被清理
	count = 0
	limiter.visitors.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	if count != 0 {
		t.Errorf("清理后访客数量 = %d, 期望 0", count)
	}
}

func BenchmarkRateLimit(b *testing.B) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RateLimit(1000.0, 2000))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = "192.168.1.1:12345"
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}
	})
}
