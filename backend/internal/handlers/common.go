package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"nodepass-pro/backend/internal/config"
	"nodepass-pro/backend/internal/services"
	"nodepass-pro/backend/internal/utils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func getUserContext(c *gin.Context) (uint, string, bool) {
	userIDValue, exists := c.Get("userID")
	if !exists {
		return 0, "", false
	}

	var userID uint
	switch value := userIDValue.(type) {
	case uint:
		userID = value
	case int:
		if value > 0 {
			userID = uint(value)
		}
	case int64:
		if value > 0 {
			userID = uint(value)
		}
	case float64:
		if value > 0 {
			userID = uint(value)
		}
	}
	if userID == 0 {
		return 0, "", false
	}

	roleValue, _ := c.Get("role")
	role, _ := roleValue.(string)
	return userID, role, true
}

func isAdminRole(role string) bool {
	return strings.EqualFold(strings.TrimSpace(role), "admin")
}

func parseUintParam(c *gin.Context, key string) (uint, bool) {
	raw := strings.TrimSpace(c.Param(key))
	if raw == "" {
		return 0, false
	}
	parsed, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, false
	}
	return uint(parsed), true
}

func parseUintQuery(c *gin.Context, key string) (*uint, error) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return nil, err
	}
	value := uint(parsed)
	return &value, nil
}

func parseOptionalBoolQuery(c *gin.Context, key string) (*bool, error) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseBool(raw)
	if err != nil {
		return nil, err
	}
	value := parsed
	return &value, nil
}

func parsePositiveIntQuery(c *gin.Context, key string, defaultValue int) (int, error) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return defaultValue, nil
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return 0, err
	}
	if parsed <= 0 {
		return 0, errors.New("必须为正整数")
	}
	return parsed, nil
}

func inferPanelURL(c *gin.Context) string {
	cfg := config.GlobalConfig
	trustForwarded := false
	if cfg != nil {
		trustForwarded = cfg.Server.TrustForwardedHeaders
	}

	scheme := ""
	host := ""

	// 只有在配置允许时才信任 X-Forwarded-* 头
	if trustForwarded {
		scheme = c.GetHeader("X-Forwarded-Proto")
		if idx := strings.Index(scheme, ","); idx >= 0 {
			scheme = strings.TrimSpace(scheme[:idx])
		}

		host = strings.TrimSpace(c.GetHeader("X-Forwarded-Host"))
		if idx := strings.Index(host, ","); idx >= 0 {
			host = strings.TrimSpace(host[:idx])
		}
	}

	// 回退到直接连接信息
	if scheme == "" {
		if c.Request.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	if host == "" {
		host = c.Request.Host
	}

	return scheme + "://" + host
}

func writeServiceError(c *gin.Context, err error, defaultCode string) {
	// 记录详细错误到日志（包含完整错误信息）
	zap.L().Error("服务错误",
		zap.Error(err),
		zap.String("path", c.Request.URL.Path),
		zap.String("method", c.Request.Method),
		zap.String("ip", c.ClientIP()))

	// 根据环境决定是否返回详细错误
	cfg := config.GlobalConfig
	isProduction := cfg != nil && cfg.Server.Mode == "release"

	// 返回给客户端的错误消息（生产环境脱敏）
	var clientMessage string

	switch {
	case errors.Is(err, services.ErrUnauthorized):
		clientMessage = "认证失败，请重新登录"
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", clientMessage)
	case errors.Is(err, services.ErrInvalidParams):
		if isProduction {
			clientMessage = "请求参数错误"
		} else {
			clientMessage = err.Error() // 开发环境返回详细信息
		}
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", clientMessage)
	case errors.Is(err, services.ErrForbidden):
		clientMessage = "无权限执行此操作"
		utils.Error(c, http.StatusForbidden, "FORBIDDEN", clientMessage)
	case errors.Is(err, services.ErrNotFound):
		clientMessage = "请求的资源不存在"
		utils.Error(c, http.StatusNotFound, "NOT_FOUND", clientMessage)
	case errors.Is(err, services.ErrConflict):
		if isProduction {
			clientMessage = "操作冲突，请稍后重试"
		} else {
			clientMessage = err.Error()
		}
		utils.Error(c, http.StatusConflict, "CONFLICT", clientMessage)
	case errors.Is(err, services.ErrQuotaExceeded):
		clientMessage = "配额已用尽"
		utils.Error(c, http.StatusBadRequest, "QUOTA_EXCEEDED", clientMessage)
	default:
		// 默认错误：生产环境返回通用消息
		clientMessage = "服务暂时不可用，请稍后重试"
		utils.Error(c, http.StatusInternalServerError, defaultCode, clientMessage)
	}
}
