package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"nodepass-pro/backend/internal/models"
	"nodepass-pro/backend/internal/services"
	"nodepass-pro/backend/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AlertHandler 告警处理器
type AlertHandler struct {
	alertService *services.AlertService
}

// NewAlertHandler 创建告警处理器
func NewAlertHandler(db *gorm.DB) *AlertHandler {
	return &AlertHandler{
		alertService: services.NewAlertService(db),
	}
}

// List GET /api/v1/alerts
// 获取告警列表
func (h *AlertHandler) List(c *gin.Context) {
	// 解析查询参数
	statusStr := c.QueryArray("status")
	levelStr := c.QueryArray("level")
	resourceType := strings.TrimSpace(c.Query("resource_type"))

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 转换状态
	var statuses []models.AlertStatus
	for _, s := range statusStr {
		statuses = append(statuses, models.AlertStatus(s))
	}

	// 转换级别
	var levels []models.AlertLevel
	for _, l := range levelStr {
		levels = append(levels, models.AlertLevel(l))
	}

	// 查询告警
	alerts, total, err := h.alertService.ListAlerts(statuses, levels, resourceType, page, pageSize)
	if err != nil {
		writeServiceError(c, err, "LIST_ALERTS_FAILED")
		return
	}

	utils.SuccessResponse(c, gin.H{
		"list":      alerts,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}, "获取告警列表成功")
}

// Get GET /api/v1/alerts/:id
// 获取告警详情
func (h *AlertHandler) Get(c *gin.Context) {
	id, err := parseAlertUintParam(c, "id")
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "无效的告警 ID")
		return
	}

	alert, err := h.alertService.GetAlert(id)
	if err != nil {
		writeServiceError(c, err, "GET_ALERT_FAILED")
		return
	}

	utils.SuccessResponse(c, alert, "获取告警详情成功")
}

// Acknowledge POST /api/v1/alerts/:id/acknowledge
// 确认告警
func (h *AlertHandler) Acknowledge(c *gin.Context) {
	userID, _, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	id, err := parseAlertUintParam(c, "id")
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "无效的告警 ID")
		return
	}

	if err := h.alertService.AcknowledgeAlert(id, userID); err != nil {
		writeServiceError(c, err, "ACKNOWLEDGE_ALERT_FAILED")
		return
	}

	utils.SuccessResponse(c, nil, "告警已确认")
}

// Resolve POST /api/v1/alerts/:id/resolve
// 解决告警
func (h *AlertHandler) Resolve(c *gin.Context) {
	userID, _, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	id, err := parseAlertUintParam(c, "id")
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "无效的告警 ID")
		return
	}

	type requestPayload struct {
		Notes string `json:"notes"`
	}
	var req requestPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		// 允许不提供 notes
		req.Notes = ""
	}

	if err := h.alertService.ResolveAlert(id, userID, req.Notes); err != nil {
		writeServiceError(c, err, "RESOLVE_ALERT_FAILED")
		return
	}

	utils.SuccessResponse(c, nil, "告警已解决")
}

// Silence POST /api/v1/alerts/:id/silence
// 静默告警
func (h *AlertHandler) Silence(c *gin.Context) {
	id, err := parseAlertUintParam(c, "id")
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "无效的告警 ID")
		return
	}

	type requestPayload struct {
		Duration int `json:"duration" binding:"required"` // 秒
	}
	var req requestPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	if req.Duration < 60 || req.Duration > 86400*7 {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "静默时间必须在 1 分钟到 7 天之间")
		return
	}

	duration := time.Duration(req.Duration) * time.Second
	if err := h.alertService.SilenceAlert(id, duration); err != nil {
		writeServiceError(c, err, "SILENCE_ALERT_FAILED")
		return
	}

	utils.SuccessResponse(c, nil, "告警已静默")
}

// Stats GET /api/v1/alerts/stats
// 获取告警统计
func (h *AlertHandler) Stats(c *gin.Context) {
	stats, err := h.alertService.GetAlertStats()
	if err != nil {
		writeServiceError(c, err, "GET_ALERT_STATS_FAILED")
		return
	}

	utils.SuccessResponse(c, stats, "获取告警统计成功")
}

// GetFiring GET /api/v1/alerts/firing
// 获取正在触发的告警
func (h *AlertHandler) GetFiring(c *gin.Context) {
	alerts, err := h.alertService.GetFiringAlerts()
	if err != nil {
		writeServiceError(c, err, "GET_FIRING_ALERTS_FAILED")
		return
	}

	utils.SuccessResponse(c, gin.H{
		"list":  alerts,
		"total": len(alerts),
	}, "获取触发中的告警成功")
}

// parseAlertUintParam 解析 uint 参数
func parseAlertUintParam(c *gin.Context, key string) (uint, error) {
	str := strings.TrimSpace(c.Param(key))
	if str == "" {
		return 0, fmt.Errorf("参数 %s 不能为空", key)
	}
	val, err := strconv.ParseUint(str, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("参数 %s 格式错误", key)
	}
	return uint(val), nil
}
