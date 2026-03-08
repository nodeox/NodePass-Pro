package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// SignatureConfig 签名配置
type SignatureConfig struct {
	Secret        string
	TimeWindow    int64 // 时间窗口（秒）
	NonceStore    NonceStore
	SkipPaths     []string
	HeaderPrefix  string
}

// NonceStore Nonce 存储接口
type NonceStore interface {
	Exists(nonce string) bool
	Add(nonce string, ttl time.Duration) error
}

// MemoryNonceStore 内存 Nonce 存储
type MemoryNonceStore struct {
	store map[string]time.Time
	mu    sync.RWMutex
}

// NewMemoryNonceStore 创建内存 Nonce 存储
func NewMemoryNonceStore() *MemoryNonceStore {
	store := &MemoryNonceStore{
		store: make(map[string]time.Time),
	}
	go store.cleanup()
	return store
}

// Exists 检查 Nonce 是否存在
func (s *MemoryNonceStore) Exists(nonce string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	expiry, exists := s.store[nonce]
	if !exists {
		return false
	}
	return time.Now().Before(expiry)
}

// Add 添加 Nonce
func (s *MemoryNonceStore) Add(nonce string, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.store[nonce] = time.Now().Add(ttl)
	return nil
}

// cleanup 清理过期 Nonce
func (s *MemoryNonceStore) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for nonce, expiry := range s.store {
			if now.After(expiry) {
				delete(s.store, nonce)
			}
		}
		s.mu.Unlock()
	}
}

// SignatureMiddleware 签名验证中间件
func SignatureMiddleware(config SignatureConfig) gin.HandlerFunc {
	if config.TimeWindow == 0 {
		config.TimeWindow = 300 // 默认 5 分钟
	}
	if config.HeaderPrefix == "" {
		config.HeaderPrefix = "X-License"
	}
	if config.NonceStore == nil {
		config.NonceStore = NewMemoryNonceStore()
	}

	return func(c *gin.Context) {
		// 跳过指定路径
		for _, path := range config.SkipPaths {
			if c.Request.URL.Path == path {
				c.Next()
				return
			}
		}

		// 获取签名相关头部
		timestamp := c.GetHeader(config.HeaderPrefix + "-Timestamp")
		nonce := c.GetHeader(config.HeaderPrefix + "-Nonce")
		signature := c.GetHeader(config.HeaderPrefix + "-Signature")

		if timestamp == "" || nonce == "" || signature == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success":   false,
				"message":   "缺少签名参数",
				"timestamp": time.Now().UTC(),
			})
			c.Abort()
			return
		}

		// 验证时间戳
		ts, err := strconv.ParseInt(timestamp, 10, 64)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success":   false,
				"message":   "时间戳格式错误",
				"timestamp": time.Now().UTC(),
			})
			c.Abort()
			return
		}

		now := time.Now().Unix()
		if now-ts > config.TimeWindow || ts-now > config.TimeWindow {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success":   false,
				"message":   "请求已过期",
				"timestamp": time.Now().UTC(),
			})
			c.Abort()
			return
		}

		// 验证 Nonce（防重放）
		if config.NonceStore.Exists(nonce) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success":   false,
				"message":   "重复的请求",
				"timestamp": time.Now().UTC(),
			})
			c.Abort()
			return
		}

		// 读取请求体
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success":   false,
				"message":   "读取请求体失败",
				"timestamp": time.Now().UTC(),
			})
			c.Abort()
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

		// 计算签名
		expectedSignature := calculateSignature(config.Secret, timestamp, nonce, string(body))
		if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success":   false,
				"message":   "签名验证失败",
				"timestamp": time.Now().UTC(),
			})
			c.Abort()
			return
		}

		// 存储 Nonce
		_ = config.NonceStore.Add(nonce, time.Duration(config.TimeWindow)*time.Second)

		c.Next()
	}
}

// calculateSignature 计算签名
func calculateSignature(secret, timestamp, nonce, body string) string {
	message := fmt.Sprintf("%s:%s:%s", timestamp, nonce, body)
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}
