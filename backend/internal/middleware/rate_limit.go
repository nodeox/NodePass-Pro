package middleware

import (
	"net/http"
	"sync"
	"time"

	"nodepass-pro/backend/internal/utils"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// IPRateLimiter 基于 IP 的令牌桶限流器（使用 sync.Map 优化并发性能）。
type IPRateLimiter struct {
	visitors        sync.Map // map[string]*visitor
	qps             rate.Limit
	burst           int
	cleanupInterval time.Duration
	ttl             time.Duration
	lastCleanupAt   time.Time
	cleanupMu       sync.Mutex
}

// NewIPRateLimiter 创建新的限流器。
func NewIPRateLimiter(qps float64, burst int) *IPRateLimiter {
	if qps <= 0 {
		qps = 10
	}
	if burst <= 0 {
		burst = 20
	}

	limiter := &IPRateLimiter{
		qps:             rate.Limit(qps),
		burst:           burst,
		cleanupInterval: 1 * time.Minute,
		ttl:             5 * time.Minute,
		lastCleanupAt:   time.Now(),
	}

	// 启动后台清理 goroutine
	go limiter.cleanupLoop()

	return limiter
}

// RateLimit 返回限流中间件。
func RateLimit(qps float64, burst int) gin.HandlerFunc {
	limiter := NewIPRateLimiter(qps, burst)

	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		if !limiter.allow(clientIP) {
			utils.Error(c, http.StatusTooManyRequests, "RATE_LIMITED", "请求过于频繁，请稍后再试")
			c.Abort()
			return
		}
		c.Next()
	}
}

func (l *IPRateLimiter) allow(ip string) bool {
	now := time.Now()

	// 从 sync.Map 获取或创建 visitor
	v, _ := l.visitors.LoadOrStore(ip, &visitor{
		limiter:  rate.NewLimiter(l.qps, l.burst),
		lastSeen: now,
	})

	vis := v.(*visitor)
	vis.lastSeen = now

	return vis.limiter.Allow()
}

// cleanupLoop 后台清理过期的访客记录
func (l *IPRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(l.cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		l.cleanup()
	}
}

func (l *IPRateLimiter) cleanup() {
	l.cleanupMu.Lock()
	defer l.cleanupMu.Unlock()

	now := time.Now()
	if now.Sub(l.lastCleanupAt) < l.cleanupInterval {
		return
	}

	// 遍历并删除过期的访客
	l.visitors.Range(func(key, value interface{}) bool {
		v := value.(*visitor)
		if now.Sub(v.lastSeen) > l.ttl {
			l.visitors.Delete(key)
		}
		return true
	})

	l.lastCleanupAt = now
}

