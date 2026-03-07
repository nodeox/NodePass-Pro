package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"nodepass-panel/backend/internal/services"
	"nodepass-panel/backend/internal/utils"

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
	scheme := c.GetHeader("X-Forwarded-Proto")
	if scheme == "" {
		if c.Request.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	return scheme + "://" + c.Request.Host
}

func writeServiceError(c *gin.Context, err error, defaultCode string) {
	switch {
	case errors.Is(err, services.ErrUnauthorized):
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
	case errors.Is(err, services.ErrInvalidParams):
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	case errors.Is(err, services.ErrForbidden):
		utils.Error(c, http.StatusForbidden, "FORBIDDEN", err.Error())
	case errors.Is(err, services.ErrNotFound):
		utils.Error(c, http.StatusNotFound, "NOT_FOUND", err.Error())
	case errors.Is(err, services.ErrConflict):
		utils.Error(c, http.StatusConflict, "CONFLICT", err.Error())
	case errors.Is(err, services.ErrQuotaExceeded):
		utils.Error(c, http.StatusBadRequest, "QUOTA_EXCEEDED", err.Error())
	default:
		zap.L().Error("处理请求失败", zap.Error(err))
		utils.Error(c, http.StatusInternalServerError, defaultCode, "服务器内部错误")
	}
}
