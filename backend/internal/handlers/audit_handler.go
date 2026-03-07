package handlers

import (
	"net/http"
	"strings"
	"time"

	"nodepass-panel/backend/internal/services"
	"nodepass-panel/backend/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AuditHandler 审计日志处理器。
type AuditHandler struct {
	auditService *services.AuditService
}

// NewAuditHandler 创建审计处理器。
func NewAuditHandler(db *gorm.DB) *AuditHandler {
	return &AuditHandler{
		auditService: services.NewAuditService(db),
	}
}

// List GET /api/v1/audit-logs
func (h *AuditHandler) List(c *gin.Context) {
	page, err := parsePositiveIntQuery(c, "page", 1)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "page 参数错误")
		return
	}
	pageSize, err := parsePositiveIntQuery(c, "pageSize", 20)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "pageSize 参数错误")
		return
	}

	userID, err := parseUintQuery(c, "user_id")
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "user_id 参数错误")
		return
	}

	startTime, err := parseOptionalDateTime(c.Query("start_time"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "start_time 参数错误")
		return
	}
	endTime, err := parseOptionalDateTime(c.Query("end_time"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "end_time 参数错误")
		return
	}

	result, listErr := h.auditService.List(services.AuditListFilters{
		UserID:       userID,
		Action:       strings.TrimSpace(c.Query("action")),
		ResourceType: strings.TrimSpace(c.Query("resource_type")),
		StartTime:    startTime,
		EndTime:      endTime,
		Page:         page,
		PageSize:     pageSize,
	})
	if listErr != nil {
		writeServiceError(c, listErr, "LIST_AUDIT_LOGS_FAILED")
		return
	}

	utils.Success(c, result)
}

func parseOptionalDateTime(raw string) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
	}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, raw)
		if err == nil {
			value := parsed.UTC()
			return &value, nil
		}
	}
	return nil, services.ErrInvalidParams
}
