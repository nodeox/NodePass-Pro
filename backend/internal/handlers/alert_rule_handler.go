package handlers

import (
	"net/http"

	"nodepass-pro/backend/internal/models"
	"nodepass-pro/backend/internal/services"
	"nodepass-pro/backend/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AlertRuleHandler 告警规则处理器
type AlertRuleHandler struct {
	ruleService *services.AlertRuleService
}

// NewAlertRuleHandler 创建告警规则处理器
func NewAlertRuleHandler(db *gorm.DB) *AlertRuleHandler {
	return &AlertRuleHandler{
		ruleService: services.NewAlertRuleService(db),
	}
}

// Create POST /api/v1/alert-rules
// 创建告警规则
func (h *AlertRuleHandler) Create(c *gin.Context) {
	var rule models.AlertRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	// 验证必填字段
	if rule.Name == "" || rule.Type == "" || rule.Condition == "" {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "名称、类型和条件不能为空")
		return
	}

	if err := h.ruleService.CreateAlertRule(&rule); err != nil {
		writeServiceError(c, err, "CREATE_ALERT_RULE_FAILED")
		return
	}

	utils.SuccessResponse(c, rule, "创建告警规则成功")
}

// Update PUT /api/v1/alert-rules/:id
// 更新告警规则
func (h *AlertRuleHandler) Update(c *gin.Context) {
	id, err := parseAlertUintParam(c, "id")
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "无效的规则 ID")
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	if err := h.ruleService.UpdateAlertRule(id, updates); err != nil {
		writeServiceError(c, err, "UPDATE_ALERT_RULE_FAILED")
		return
	}

	utils.SuccessResponse(c, nil, "更新告警规则成功")
}

// Delete DELETE /api/v1/alert-rules/:id
// 删除告警规则
func (h *AlertRuleHandler) Delete(c *gin.Context) {
	id, err := parseAlertUintParam(c, "id")
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "无效的规则 ID")
		return
	}

	if err := h.ruleService.DeleteAlertRule(id); err != nil {
		writeServiceError(c, err, "DELETE_ALERT_RULE_FAILED")
		return
	}

	utils.SuccessResponse(c, nil, "删除告警规则成功")
}

// Get GET /api/v1/alert-rules/:id
// 获取告警规则详情
func (h *AlertRuleHandler) Get(c *gin.Context) {
	id, err := parseAlertUintParam(c, "id")
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "无效的规则 ID")
		return
	}

	rule, err := h.ruleService.GetAlertRule(id)
	if err != nil {
		writeServiceError(c, err, "GET_ALERT_RULE_FAILED")
		return
	}

	utils.SuccessResponse(c, rule, "获取告警规则成功")
}

// List GET /api/v1/alert-rules
// 获取告警规则列表
func (h *AlertRuleHandler) List(c *gin.Context) {
	enabledStr := c.Query("is_enabled")
	var isEnabled *bool
	if enabledStr != "" {
		val := enabledStr == "true" || enabledStr == "1"
		isEnabled = &val
	}

	rules, err := h.ruleService.ListAlertRules(isEnabled)
	if err != nil {
		writeServiceError(c, err, "LIST_ALERT_RULES_FAILED")
		return
	}

	utils.SuccessResponse(c, gin.H{
		"list":  rules,
		"total": len(rules),
	}, "获取告警规则列表成功")
}

// Toggle POST /api/v1/alert-rules/:id/toggle
// 启用/禁用告警规则
func (h *AlertRuleHandler) Toggle(c *gin.Context) {
	id, err := parseAlertUintParam(c, "id")
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "无效的规则 ID")
		return
	}

	// 获取当前规则
	rule, err := h.ruleService.GetAlertRule(id)
	if err != nil {
		writeServiceError(c, err, "GET_ALERT_RULE_FAILED")
		return
	}

	// 切换状态
	newStatus := !rule.IsEnabled
	if err := h.ruleService.UpdateAlertRule(id, map[string]interface{}{
		"is_enabled": newStatus,
	}); err != nil {
		writeServiceError(c, err, "TOGGLE_ALERT_RULE_FAILED")
		return
	}

	message := "告警规则已启用"
	if !newStatus {
		message = "告警规则已禁用"
	}

	utils.SuccessResponse(c, gin.H{"is_enabled": newStatus}, message)
}
