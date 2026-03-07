package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"nodepass-pro/backend/internal/config"
	"nodepass-pro/backend/internal/utils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// TelegramWebhookAuth 验证 Telegram webhook secret token。
// Telegram 官方通过 X-Telegram-Bot-Api-Secret-Token 头进行鉴权。
func TelegramWebhookAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		cfg := config.GlobalConfig
		if cfg == nil {
			zap.L().Error("Telegram 配置未初始化，拒绝 webhook 请求")
			utils.Error(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Telegram 服务未初始化")
			c.Abort()
			return
		}

		expectedToken := strings.TrimSpace(cfg.Telegram.SecretToken)
		if expectedToken == "" {
			zap.L().Error("Telegram webhook secret_token 未配置，拒绝 webhook 请求")
			utils.Error(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Telegram webhook 鉴权未配置")
			c.Abort()
			return
		}

		receivedToken := strings.TrimSpace(c.GetHeader("X-Telegram-Bot-Api-Secret-Token"))
		if receivedToken == "" {
			zap.L().Warn("Telegram webhook 缺少 secret_token 头",
				zap.String("ip", c.ClientIP()))
			utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "缺少 Telegram webhook 鉴权头")
			c.Abort()
			return
		}
		if subtle.ConstantTimeCompare([]byte(receivedToken), []byte(expectedToken)) != 1 {
			zap.L().Warn("Telegram webhook secret_token 校验失败",
				zap.String("ip", c.ClientIP()))
			utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Telegram webhook 鉴权失败")
			c.Abort()
			return
		}

		c.Next()
	}
}
