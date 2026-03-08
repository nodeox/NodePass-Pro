package handlers

import (
	"net/http"
	"strconv"

	"nodepass-license-center/internal/services"
	"nodepass-license-center/internal/utils"

	"github.com/gin-gonic/gin"
)

// MonitoringHandler 监控处理器
type MonitoringHandler struct {
	monitoringService *services.MonitoringService
	alertService      *services.AlertService
}

// NewMonitoringHandler 创建监控处理器
func NewMonitoringHandler(monitoringService *services.MonitoringService, alertService *services.AlertService) *MonitoringHandler {
	return &MonitoringHandler{
		monitoringService: monitoringService,
		alertService:      alertService,
	}
}

// GetDashboard 获取仪表盘数据
func (h *MonitoringHandler) GetDashboard(c *gin.Context) {
	stats, err := h.monitoringService.GetDashboardStats(c.Request.Context())
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "ERROR", "获取统计数据失败")
		return
	}

	utils.Success(c, stats, "ok")
}

// GetVerifyTrend 获取验证趋势
func (h *MonitoringHandler) GetVerifyTrend(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))
	if days <= 0 || days > 90 {
		days = 7
	}

	trend, err := h.monitoringService.GetVerifyTrend(days)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "ERROR", "获取验证趋势失败")
		return
	}

	utils.Success(c, trend, "ok")
}

// GetTopCustomers 获取 Top 客户
func (h *MonitoringHandler) GetTopCustomers(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if limit <= 0 || limit > 100 {
		limit = 10
	}

	customers, err := h.monitoringService.GetTopCustomers(limit)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "ERROR", "获取客户统计失败")
		return
	}

	utils.Success(c, customers, "ok")
}

// ListAlerts 查询告警
func (h *MonitoringHandler) ListAlerts(c *gin.Context) {
	var isRead *bool
	if c.Query("is_read") != "" {
		val := c.Query("is_read") == "true"
		isRead = &val
	}

	level := c.Query("level")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	result, err := h.alertService.ListAlerts(isRead, level, page, pageSize)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "ERROR", "查询告警失败")
		return
	}

	utils.Success(c, result, "ok")
}

// MarkAlertRead 标记告警已读
func (h *MonitoringHandler) MarkAlertRead(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	if id == 0 {
		utils.Error(c, http.StatusBadRequest, "ERROR", "参数无效")
		return
	}

	if err := h.alertService.MarkAlertRead(uint(id)); err != nil {
		utils.Error(c, http.StatusInternalServerError, "ERROR", "标记失败")
		return
	}

	utils.Success(c, nil, "ok")
}

// MarkAllAlertsRead 标记所有告警已读
func (h *MonitoringHandler) MarkAllAlertsRead(c *gin.Context) {
	if err := h.alertService.MarkAllAlertsRead(); err != nil {
		utils.Error(c, http.StatusInternalServerError, "ERROR", "标记失败")
		return
	}

	utils.Success(c, nil, "ok")
}

// DeleteAlert 删除告警
func (h *MonitoringHandler) DeleteAlert(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	if id == 0 {
		utils.Error(c, http.StatusBadRequest, "ERROR", "参数无效")
		return
	}

	if err := h.alertService.DeleteAlert(uint(id)); err != nil {
		utils.Error(c, http.StatusInternalServerError, "ERROR", "删除失败")
		return
	}

	utils.Success(c, nil, "ok")
}

// GetAlertStats 获取告警统计
func (h *MonitoringHandler) GetAlertStats(c *gin.Context) {
	stats, err := h.alertService.GetAlertStats()
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "ERROR", "获取统计失败")
		return
	}

	utils.Success(c, stats, "ok")
}
