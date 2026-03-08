package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"nodepass-pro/backend/internal/config"
	"nodepass-pro/backend/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims JWT 载荷声明。
type JWTClaims struct {
	UserID uint   `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// AuthMiddleware JWT 认证中间件。
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, err := extractBearerToken(c.GetHeader("Authorization"))
		if err != nil {
			if isWebSocketUpgradeRequest(c) {
				tokenString = extractWebSocketProtocolToken(c.GetHeader("Sec-WebSocket-Protocol"))
				if tokenString == "" {
					utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未提供认证令牌")
					c.Abort()
					return
				}
			} else {
				utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
				c.Abort()
				return
			}
		}

		cfg := config.GlobalConfig
		if cfg == nil || strings.TrimSpace(cfg.JWT.Secret) == "" {
			utils.Error(c, http.StatusInternalServerError, "JWT_NOT_CONFIGURED", "JWT 密钥未配置")
			c.Abort()
			return
		}

		claims := &JWTClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("不支持的签名算法: %v", token.Header["alg"])
			}
			return []byte(cfg.JWT.Secret), nil
		})
		if err != nil || token == nil || !token.Valid {
			utils.Error(c, http.StatusUnauthorized, "TOKEN_INVALID", "无效或已过期的令牌")
			c.Abort()
			return
		}
		if claims.UserID == 0 {
			utils.Error(c, http.StatusUnauthorized, "TOKEN_INVALID", "令牌缺少用户信息")
			c.Abort()
			return
		}

		c.Set("userID", claims.UserID)
		c.Set("role", claims.Role)
		c.Next()
	}
}

func isWebSocketUpgradeRequest(c *gin.Context) bool {
	upgrade := strings.TrimSpace(strings.ToLower(c.GetHeader("Upgrade")))
	if upgrade == "websocket" {
		return true
	}

	connection := strings.TrimSpace(strings.ToLower(c.GetHeader("Connection")))
	return strings.Contains(connection, "upgrade")
}

func extractWebSocketProtocolToken(protocolHeader string) string {
	if strings.TrimSpace(protocolHeader) == "" {
		return ""
	}

	parts := strings.Split(protocolHeader, ",")
	trimmed := make([]string, 0, len(parts))
	for _, item := range parts {
		value := strings.TrimSpace(item)
		if value != "" {
			trimmed = append(trimmed, value)
		}
	}
	if len(trimmed) < 2 {
		return ""
	}
	if !strings.EqualFold(trimmed[0], "bearer") {
		return ""
	}
	return trimmed[1]
}

func extractBearerToken(authorization string) (string, error) {
	if strings.TrimSpace(authorization) == "" {
		return "", errors.New("未提供认证令牌")
	}

	parts := strings.SplitN(strings.TrimSpace(authorization), " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", errors.New("认证令牌格式错误")
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", errors.New("认证令牌为空")
	}

	return token, nil
}
