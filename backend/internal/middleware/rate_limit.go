package middleware

import (
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"nodepass-pro/backend/internal/utils"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type visitor struct {
	limiter  *rate.Limiter
	lastSeen int64 // 使用 int64 存储 Unix 纳秒时间戳，配合 atomic 操作
}

// KeyRateLimiter 基于 key 的令牌桶限流器（使用 sync.Map 优化并发性能）。
type KeyRateLimiter struct {
	visitors        sync.Map // map[string]*visitor
	qps             rate.Limit
	burst           int
	cleanupInterval time.Duration
	ttl             time.Duration
	lastCleanupAt   time.Time
	cleanupMu       sync.Mutex
	stopChan        chan struct{} // 用于停止清理 goroutine
	stopped         bool
	stopMu          sync.Mutex
}

// NewKeyRateLimiter 创建新的限流器。
func NewKeyRateLimiter(qps float64, burst int) *KeyRateLimiter {
	if qps <= 0 {
		qps = 10
	}
	if burst <= 0 {
		burst = 20
	}

	limiter := &KeyRateLimiter{
		qps:             rate.Limit(qps),
		burst:           burst,
		cleanupInterval: 1 * time.Minute,
		ttl:             5 * time.Minute,
		lastCleanupAt:   time.Now(),
		stopChan:        make(chan struct{}),
		stopped:         false,
	}

	// 启动后台清理 goroutine
	go limiter.cleanupLoop()

	return limiter
}

// Stop 停止限流器的后台清理 goroutine
func (l *KeyRateLimiter) Stop() {
	l.stopMu.Lock()
	defer l.stopMu.Unlock()

	if !l.stopped {
		close(l.stopChan)
		l.stopped = true
	}
}

// RateLimit 返回限流中间件。
func RateLimit(qps float64, burst int) gin.HandlerFunc {
	return RateLimitBy(qps, burst, func(c *gin.Context) string {
		return c.ClientIP()
	})
}

// RateLimitBy 返回可自定义 key 的限流中间件。
func RateLimitBy(qps float64, burst int, keyExtractor func(c *gin.Context) string) gin.HandlerFunc {
	limiter := NewKeyRateLimiter(qps, burst)

	return func(c *gin.Context) {
		key := ""
		if keyExtractor != nil {
			key = strings.TrimSpace(keyExtractor(c))
		}
		if key == "" {
			key = strings.TrimSpace(c.ClientIP())
		}
		if key == "" {
			key = "anonymous"
		}

		if !limiter.allow(key) {
			utils.Error(c, http.StatusTooManyRequests, "RATE_LIMITED", "请求过于频繁，请稍后再试")
			c.Abort()
			return
		}
		c.Next()
	}
}

func (l *KeyRateLimiter) allow(key string) bool {
	now := time.Now()

	// 从 sync.Map 获取或创建 visitor
	v, _ := l.visitors.LoadOrStore(key, &visitor{
		limiter:  rate.NewLimiter(l.qps, l.burst),
		lastSeen: now.UnixNano(),
	})

	vis := v.(*visitor)
	// 使用原子操作更新 lastSeen，避免数据竞争
	atomic.StoreInt64(&vis.lastSeen, now.UnixNano())

	return vis.limiter.Allow()
}

// cleanupLoop 后台清理过期的访客记录
func (l *KeyRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(l.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			l.cleanup()
		case <-l.stopChan:
			return
		}
	}
}

func (l *KeyRateLimiter) cleanup() {
	l.cleanupMu.Lock()
	defer l.cleanupMu.Unlock()

	now := time.Now()
	if now.Sub(l.lastCleanupAt) < l.cleanupInterval {
		return
	}

	// 遍历并删除过期的访客
	nowNano := now.UnixNano()
	ttlNano := l.ttl.Nanoseconds()
	l.visitors.Range(func(key, value interface{}) bool {
		v := value.(*visitor)
		// 使用原子操作读取 lastSeen，避免数据竞争
		lastSeenNano := atomic.LoadInt64(&v.lastSeen)
		if nowNano-lastSeenNano > ttlNano {
			l.visitors.Delete(key)
		}
		return true
	})

	l.lastCleanupAt = now
}
