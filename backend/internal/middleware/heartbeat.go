package middleware

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"nodepass-pro/backend/internal/utils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// HeartbeatRateLimit 心跳接口限流（按 IP 限流，避免 DoS 攻击）。
// 注意：不按 node_id 限流，因为 node_id 可以从公开 API 获取，
// 攻击者可以伪造 node_id 消耗限流配额，影响真实节点心跳。
// 改为按 IP 限流，配合 token 验证，可以有效防止 DoS 攻击。
func HeartbeatRateLimit(qps float64, burst int) gin.HandlerFunc {
	return RateLimitBy(qps, burst, func(c *gin.Context) string {
		// 只按 IP 限流，不按 node_id
		// 这样可以防止攻击者通过伪造 node_id 来消耗特定节点的限流配额
		return "heartbeat:ip:" + strings.TrimSpace(c.ClientIP())
	})
}

// HeartbeatReplayProtection 心跳接口防重放中间件。
// 使用时间戳和 nonce 防止重放攻击。
func HeartbeatReplayProtection() gin.HandlerFunc {
	// 使用 sync.Map 存储最近的 nonce
	var recentNonces sync.Map

	// 后台清理过期 nonce
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			// 清理 10 分钟前的 nonce
			cutoff := time.Now().Add(-10 * time.Minute).Unix()
			recentNonces.Range(func(key, value interface{}) bool {
				if ts, ok := value.(int64); ok && ts < cutoff {
					recentNonces.Delete(key)
				}
				return true
			})
		}
	}()

	return func(c *gin.Context) {
		// 获取时间戳
		timestampStr := strings.TrimSpace(c.GetHeader("X-Timestamp"))
		if timestampStr == "" {
			zap.L().Warn("心跳请求缺少时间戳头",
				zap.String("ip", c.ClientIP()))
			utils.Error(c, http.StatusUnauthorized, "REPLAY_ATTACK", "缺少时间戳头")
			c.Abort()
			return
		}

		timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
		if err != nil {
			utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "时间戳格式错误")
			c.Abort()
			return
		}

		// 验证时间戳（允许 5 分钟误差）
		now := time.Now().Unix()
		if timestamp < now-300 || timestamp > now+300 {
			zap.L().Warn("心跳请求时间戳过期",
				zap.Int64("timestamp", timestamp),
				zap.Int64("now", now),
				zap.String("ip", c.ClientIP()))
			utils.Error(c, http.StatusUnauthorized, "REPLAY_ATTACK", "请求已过期")
			c.Abort()
			return
		}

		// 获取 nonce
		nonce := strings.TrimSpace(c.GetHeader("X-Nonce"))
		if nonce == "" {
			zap.L().Warn("心跳请求缺少 nonce",
				zap.String("ip", c.ClientIP()))
			utils.Error(c, http.StatusUnauthorized, "REPLAY_ATTACK", "缺少 nonce")
			c.Abort()
			return
		}
		if len(nonce) > 128 {
			utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "nonce 长度非法")
			c.Abort()
			return
		}
		// 检查 nonce 是否已使用
		if _, exists := recentNonces.LoadOrStore(nonce, timestamp); exists {
			zap.L().Warn("检测到重放攻击",
				zap.String("nonce", nonce),
				zap.String("ip", c.ClientIP()))
			utils.Error(c, http.StatusUnauthorized, "REPLAY_ATTACK", "请求已被使用")
			c.Abort()
			return
		}

		c.Next()
	}
}
