package middleware

import (
	"net/http"
	"sync"
	"time"

	"nodepass-panel/backend/internal/utils"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// IPRateLimiter 基于 IP 的令牌桶限流器。
type IPRateLimiter struct {
	mu              sync.Mutex
	visitors        map[string]*visitor
	qps             rate.Limit
	burst           int
	cleanupInterval time.Duration
	ttl             time.Duration
	lastCleanupAt   time.Time
}

// NewIPRateLimiter 创建新的限流器。
func NewIPRateLimiter(qps float64, burst int) *IPRateLimiter {
	if qps <= 0 {
		qps = 10
	}
	if burst <= 0 {
		burst = 20
	}

	return &IPRateLimiter{
		visitors:        make(map[string]*visitor),
		qps:             rate.Limit(qps),
		burst:           burst,
		cleanupInterval: 1 * time.Minute,
		ttl:             5 * time.Minute,
		lastCleanupAt:   time.Now(),
	}
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
	l.mu.Lock()
	defer l.mu.Unlock()

	l.cleanupIfNeeded(time.Now())
	v, exists := l.visitors[ip]
	if !exists {
		v = &visitor{
			limiter:  rate.NewLimiter(l.qps, l.burst),
			lastSeen: time.Now(),
		}
		l.visitors[ip] = v
	}

	v.lastSeen = time.Now()
	return v.limiter.Allow()
}

func (l *IPRateLimiter) cleanupIfNeeded(now time.Time) {
	if now.Sub(l.lastCleanupAt) < l.cleanupInterval {
		return
	}

	for ip, v := range l.visitors {
		if now.Sub(v.lastSeen) > l.ttl {
			delete(l.visitors, ip)
		}
	}
	l.lastCleanupAt = now
}
