package middleware

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"nodepass-pro/backend/internal/database"
	"nodepass-pro/backend/internal/models"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AuditLogger 审计日志中间件。
func AuditLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !isWriteMethod(c.Request.Method) {
			c.Next()
			return
		}

		startAt := time.Now()
		c.Next()

		if database.DB == nil {
			return
		}

		action, resourceType := inferActionAndResource(c)
		resourceID := inferResourceID(c)
		userID := inferUserID(c)

		ip := c.ClientIP()
		userAgent := c.Request.UserAgent()
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		detailMap := map[string]any{
			"method":      c.Request.Method,
			"path":        path,
			"status":      c.Writer.Status(),
			"duration_ms": time.Since(startAt).Milliseconds(),
		}
		detailBytes, _ := json.Marshal(detailMap)
		detailText := string(detailBytes)

		logRecord := &models.AuditLog{
			UserID:    userID,
			Action:    action,
			Details:   &detailText,
			IPAddress: &ip,
			UserAgent: &userAgent,
		}
		if resourceType != "" {
			resourceTypeCopy := resourceType
			logRecord.ResourceType = &resourceTypeCopy
		}
		if resourceID != nil {
			logRecord.ResourceID = resourceID
		}

		if err := database.DB.Create(logRecord).Error; err != nil {
			zap.L().Warn("写入审计日志失败", zap.Error(err))
		}
	}
}

func isWriteMethod(method string) bool {
	switch strings.ToUpper(method) {
	case "POST", "PUT", "DELETE":
		return true
	default:
		return false
	}
}

func inferActionAndResource(c *gin.Context) (string, string) {
	path := c.FullPath()
	if path == "" {
		path = c.Request.URL.Path
	}

	segments := routeSegments(path)
	resource := inferResource(segments)
	method := strings.ToUpper(c.Request.Method)

	if method == "POST" {
		if len(segments) > 0 {
			last := segments[len(segments)-1]
			if last != resource && !strings.HasPrefix(last, ":") {
				return resource + "." + sanitizeSegment(last), resource
			}
		}
		return resource + ".create", resource
	}
	if method == "PUT" {
		return resource + ".update", resource
	}
	if method == "DELETE" {
		return resource + ".delete", resource
	}
	return strings.ToLower(method), resource
}

func inferResourceID(c *gin.Context) *uint {
	priorityKeys := []string{"id", "user_id", "node_id", "rule_id", "pair_id", "resource_id"}
	for _, key := range priorityKeys {
		value := strings.TrimSpace(c.Param(key))
		if value == "" {
			continue
		}
		if parsed, ok := parseUint(value); ok {
			return &parsed
		}
	}

	for _, param := range c.Params {
		if parsed, ok := parseUint(strings.TrimSpace(param.Value)); ok {
			return &parsed
		}
	}

	return nil
}

func inferUserID(c *gin.Context) *uint {
	userID, ok := getContextUserID(c)
	if !ok {
		return nil
	}
	return &userID
}

func inferResource(segments []string) string {
	if len(segments) == 0 {
		return "unknown"
	}

	if len(segments) >= 3 && segments[0] == "api" && segments[1] == "v1" {
		return sanitizeSegment(segments[2])
	}

	return sanitizeSegment(segments[0])
}

func routeSegments(path string) []string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	segments := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			continue
		}
		segments = append(segments, part)
	}
	return segments
}

func sanitizeSegment(segment string) string {
	return strings.ToLower(strings.TrimPrefix(strings.TrimSpace(segment), ":"))
}

func parseUint(value string) (uint, bool) {
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, false
	}
	return uint(parsed), true
}
