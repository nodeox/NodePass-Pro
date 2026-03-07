package handlers

import (
	"net/http"
	"strings"
	"time"

	"nodepass-pro/backend/internal/services"
	"nodepass-pro/backend/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// TrafficHandler 流量统计与配额管理处理器。
type TrafficHandler struct {
	trafficService *services.TrafficService
}

// NewTrafficHandler 创建流量处理器。
func NewTrafficHandler(db *gorm.DB) *TrafficHandler {
	return &TrafficHandler{
		trafficService: services.NewTrafficService(db),
	}
}

// GetQuota GET /api/v1/traffic/quota
func (h *TrafficHandler) GetQuota(c *gin.Context) {
	userID, _, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	result, err := h.trafficService.GetQuota(userID)
	if err != nil {
		writeServiceError(c, err, "GET_QUOTA_FAILED")
		return
	}

	utils.Success(c, result)
}

// GetUsage GET /api/v1/traffic/usage
func (h *TrafficHandler) GetUsage(c *gin.Context) {
	userID, _, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	startTime, err := parseTimeQuery(c.Query("start_time"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "start_time 参数错误")
		return
	}
	endTime, err := parseTimeQuery(c.Query("end_time"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "end_time 参数错误")
		return
	}
	if startTime == nil {
		defaultStart := time.Now().UTC().AddDate(0, 0, -30)
		startTime = &defaultStart
	}
	if endTime == nil {
		defaultEnd := time.Now().UTC()
		endTime = &defaultEnd
	}

	usage, err := h.trafficService.GetUsage(userID, *startTime, *endTime)
	if err != nil {
		writeServiceError(c, err, "GET_USAGE_FAILED")
		return
	}

	utils.Success(c, usage)
}

// GetRecords GET /api/v1/traffic/records
func (h *TrafficHandler) GetRecords(c *gin.Context) {
	userID, _, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

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

	ruleID, err := parseUintQuery(c, "rule_id")
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "rule_id 参数错误")
		return
	}
	nodeID, err := parseUintQuery(c, "node_id")
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "node_id 参数错误")
		return
	}
	startTime, err := parseTimeQuery(c.Query("start_time"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "start_time 参数错误")
		return
	}
	endTime, err := parseTimeQuery(c.Query("end_time"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "end_time 参数错误")
		return
	}

	result, err := h.trafficService.GetRecords(userID, services.TrafficRecordFilters{
		RuleID:    ruleID,
		NodeID:    nodeID,
		StartTime: startTime,
		EndTime:   endTime,
		Page:      page,
		PageSize:  pageSize,
	})
	if err != nil {
		writeServiceError(c, err, "GET_RECORDS_FAILED")
		return
	}

	utils.Success(c, result)
}

// ResetQuota POST /api/v1/traffic/quota/reset (admin only)
func (h *TrafficHandler) ResetQuota(c *gin.Context) {
	adminUserID, role, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}
	if !isAdminRole(role) {
		utils.Error(c, http.StatusForbidden, "FORBIDDEN", "仅管理员可执行此操作")
		return
	}

	type requestPayload struct {
		TargetUserID uint `json:"target_user_id" binding:"required"`
	}
	var req requestPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	if err := h.trafficService.ResetQuota(adminUserID, req.TargetUserID); err != nil {
		writeServiceError(c, err, "RESET_QUOTA_FAILED")
		return
	}

	utils.SuccessResponse(c, nil, "流量配额重置成功")
}

func parseTimeQuery(raw string) (*time.Time, error) {
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
			result := parsed.UTC()
			return &result, nil
		}
	}

	return nil, services.ErrInvalidParams
}
