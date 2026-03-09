package handlers

import (
	"net/http"
	"strconv"
	"time"

	"nodepass-pro/backend/internal/models"
	"nodepass-pro/backend/internal/services"
	"nodepass-pro/backend/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// NodePerformanceHandler 节点性能监控处理器
type NodePerformanceHandler struct {
	service *services.NodePerformanceService
}

// NewNodePerformanceHandler 创建节点性能监控处理器
func NewNodePerformanceHandler(db *gorm.DB) *NodePerformanceHandler {
	return &NodePerformanceHandler{
		service: services.NewNodePerformanceService(db),
	}
}

// RecordMetric POST /api/v1/node-instances/:id/performance/metrics
func (h *NodePerformanceHandler) RecordMetric(c *gin.Context) {
	nodeInstanceID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAM", "无效的节点实例 ID")
		return
	}

	var metric models.NodePerformanceMetric
	if err := c.ShouldBindJSON(&metric); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	metric.NodeInstanceID = uint(nodeInstanceID)

	if err := h.service.RecordMetric(&metric); err != nil {
		utils.Error(c, http.StatusInternalServerError, "RECORD_FAILED", "记录性能指标失败: "+err.Error())
		return
	}

	// 检查并触发告警
	if err := h.service.CheckAndTriggerAlerts(&metric); err != nil {
		// 告警失败不影响指标记录
		utils.SuccessResponse(c, metric, "性能指标记录成功（告警检查失败）")
		return
	}

	utils.SuccessResponse(c, metric, "性能指标记录成功")
}

// GetLatestMetric GET /api/v1/node-instances/:id/performance/latest
func (h *NodePerformanceHandler) GetLatestMetric(c *gin.Context) {
	nodeInstanceID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAM", "无效的节点实例 ID")
		return
	}

	metric, err := h.service.GetLatestMetric(uint(nodeInstanceID))
	if err != nil {
		utils.Error(c, http.StatusNotFound, "NOT_FOUND", "性能指标不存在")
		return
	}

	utils.Success(c, metric)
}

// GetMetrics GET /api/v1/node-instances/:id/performance/metrics
func (h *NodePerformanceHandler) GetMetrics(c *gin.Context) {
	nodeInstanceID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAM", "无效的节点实例 ID")
		return
	}

	// 解析时间范围
	var startTime, endTime time.Time
	if startStr := c.Query("start_time"); startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			startTime = t
		}
	}
	if endStr := c.Query("end_time"); endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			endTime = t
		}
	}

	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	metrics, err := h.service.GetMetrics(uint(nodeInstanceID), startTime, endTime, limit)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "QUERY_FAILED", "查询性能指标失败: "+err.Error())
		return
	}

	utils.Success(c, gin.H{
		"metrics": metrics,
		"total":   len(metrics),
	})
}

// GetMetricsStats GET /api/v1/node-instances/:id/performance/stats
func (h *NodePerformanceHandler) GetMetricsStats(c *gin.Context) {
	nodeInstanceID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAM", "无效的节点实例 ID")
		return
	}

	// 默认查询最近 24 小时
	duration := 24 * time.Hour
	if durationStr := c.Query("duration"); durationStr != "" {
		if d, err := time.ParseDuration(durationStr); err == nil {
			duration = d
		}
	}

	stats, err := h.service.GetMetricsStats(uint(nodeInstanceID), duration)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "QUERY_FAILED", "查询性能统计失败: "+err.Error())
		return
	}

	utils.Success(c, stats)
}

// CreateAlert POST /api/v1/node-instances/:id/performance/alert
func (h *NodePerformanceHandler) CreateAlert(c *gin.Context) {
	nodeInstanceID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAM", "无效的节点实例 ID")
		return
	}

	var alert models.NodePerformanceAlert
	if err := c.ShouldBindJSON(&alert); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	alert.NodeInstanceID = uint(nodeInstanceID)

	if err := h.service.CreateAlert(&alert); err != nil {
		utils.Error(c, http.StatusInternalServerError, "CREATE_FAILED", "创建性能告警配置失败: "+err.Error())
		return
	}

	utils.SuccessResponse(c, alert, "性能告警配置创建成功")
}

// GetAlert GET /api/v1/node-instances/:id/performance/alert
func (h *NodePerformanceHandler) GetAlert(c *gin.Context) {
	nodeInstanceID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAM", "无效的节点实例 ID")
		return
	}

	alert, err := h.service.GetAlert(uint(nodeInstanceID))
	if err != nil {
		utils.Error(c, http.StatusNotFound, "NOT_FOUND", "性能告警配置不存在")
		return
	}

	utils.Success(c, alert)
}

// UpdateAlert PUT /api/v1/node-instances/:id/performance/alert
func (h *NodePerformanceHandler) UpdateAlert(c *gin.Context) {
	nodeInstanceID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAM", "无效的节点实例 ID")
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	if err := h.service.UpdateAlert(uint(nodeInstanceID), updates); err != nil {
		utils.Error(c, http.StatusInternalServerError, "UPDATE_FAILED", "更新性能告警配置失败: "+err.Error())
		return
	}

	utils.SuccessResponse(c, nil, "性能告警配置更新成功")
}

// DeleteAlert DELETE /api/v1/node-instances/:id/performance/alert
func (h *NodePerformanceHandler) DeleteAlert(c *gin.Context) {
	nodeInstanceID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAM", "无效的节点实例 ID")
		return
	}

	if err := h.service.DeleteAlert(uint(nodeInstanceID)); err != nil {
		utils.Error(c, http.StatusInternalServerError, "DELETE_FAILED", "删除性能告警配置失败: "+err.Error())
		return
	}

	utils.SuccessResponse(c, nil, "性能告警配置删除成功")
}

// GetAlertRecords GET /api/v1/node-instances/:id/performance/alert-records
func (h *NodePerformanceHandler) GetAlertRecords(c *gin.Context) {
	nodeInstanceID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAM", "无效的节点实例 ID")
		return
	}

	var resolved *bool
	if resolvedStr := c.Query("resolved"); resolvedStr != "" {
		if r, err := strconv.ParseBool(resolvedStr); err == nil {
			resolved = &r
		}
	}

	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	records, err := h.service.GetAlertRecords(uint(nodeInstanceID), resolved, limit)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "QUERY_FAILED", "查询告警记录失败: "+err.Error())
		return
	}

	utils.Success(c, gin.H{
		"records": records,
		"total":   len(records),
	})
}

// ResolveAlert POST /api/v1/node-instances/performance/alert-records/:alert_id/resolve
func (h *NodePerformanceHandler) ResolveAlert(c *gin.Context) {
	alertID, err := strconv.ParseUint(c.Param("alert_id"), 10, 32)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAM", "无效的告警 ID")
		return
	}

	if err := h.service.ResolveAlert(uint(alertID)); err != nil {
		utils.Error(c, http.StatusInternalServerError, "RESOLVE_FAILED", "解决告警失败: "+err.Error())
		return
	}

	utils.SuccessResponse(c, nil, "告警已解决")
}

// GetSummaries GET /api/v1/node-instances/:id/performance/summaries
func (h *NodePerformanceHandler) GetSummaries(c *gin.Context) {
	nodeInstanceID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAM", "无效的节点实例 ID")
		return
	}

	period := c.Query("period") // hourly, daily
	limit := 30
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	summaries, err := h.service.GetSummaries(uint(nodeInstanceID), period, limit)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "QUERY_FAILED", "查询性能汇总失败: "+err.Error())
		return
	}

	utils.Success(c, gin.H{
		"summaries": summaries,
		"total":     len(summaries),
	})
}
