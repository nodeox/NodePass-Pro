package handlers

import (
	"net/http"
	"strings"

	"nodepass-panel/backend/internal/services"
	"nodepass-panel/backend/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RuleHandler 规则管理处理器。
type RuleHandler struct {
	ruleService *services.RuleService
}

// NewRuleHandler 创建规则处理器。
func NewRuleHandler(db *gorm.DB) *RuleHandler {
	return &RuleHandler{
		ruleService: services.NewRuleService(db),
	}
}

// CreateRule POST /api/v1/rules
func (h *RuleHandler) CreateRule(c *gin.Context) {
	userID, role, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	type requestPayload struct {
		services.CreateRuleRequest
		UserID *uint `json:"user_id"`
	}
	var payload requestPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	targetUserID := userID
	if payload.UserID != nil {
		if !isAdminRole(role) {
			utils.Error(c, http.StatusForbidden, "FORBIDDEN", "普通用户不能指定 user_id")
			return
		}
		targetUserID = *payload.UserID
	}
	if targetUserID == 0 {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "user_id 无效")
		return
	}

	rule, err := h.ruleService.CreateRule(targetUserID, &payload.CreateRuleRequest)
	if err != nil {
		writeServiceError(c, err, "CREATE_RULE_FAILED")
		return
	}

	utils.SuccessResponse(c, rule, "规则创建成功")
}

// ListRules GET /api/v1/rules
func (h *RuleHandler) ListRules(c *gin.Context) {
	userID, role, ok := getUserContext(c)
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

	filters := services.ListRuleFilters{
		Status:   strings.TrimSpace(c.Query("status")),
		Mode:     strings.TrimSpace(c.Query("mode")),
		Page:     page,
		PageSize: pageSize,
	}

	scopeUserID := userID
	if isAdminRole(role) {
		scopeUserID = 0
		queryUserID, parseErr := parseUintQuery(c, "user_id")
		if parseErr != nil {
			utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "user_id 参数错误")
			return
		}
		filters.UserID = queryUserID
	}

	result, err := h.ruleService.ListRules(scopeUserID, filters)
	if err != nil {
		writeServiceError(c, err, "LIST_RULES_FAILED")
		return
	}

	utils.Success(c, result)
}

// GetRule GET /api/v1/rules/:id
func (h *RuleHandler) GetRule(c *gin.Context) {
	userID, role, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	ruleID, ok := parseUintParam(c, "id")
	if !ok {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "规则 ID 无效")
		return
	}

	scopeUserID := userID
	if isAdminRole(role) {
		scopeUserID = 0
	}

	rule, err := h.ruleService.GetRule(scopeUserID, ruleID)
	if err != nil {
		writeServiceError(c, err, "GET_RULE_FAILED")
		return
	}

	utils.Success(c, rule)
}

// UpdateRule PUT /api/v1/rules/:id
func (h *RuleHandler) UpdateRule(c *gin.Context) {
	userID, role, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	ruleID, ok := parseUintParam(c, "id")
	if !ok {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "规则 ID 无效")
		return
	}

	var req services.UpdateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	scopeUserID := userID
	if isAdminRole(role) {
		scopeUserID = 0
	}

	rule, err := h.ruleService.UpdateRule(scopeUserID, ruleID, &req)
	if err != nil {
		writeServiceError(c, err, "UPDATE_RULE_FAILED")
		return
	}

	utils.SuccessResponse(c, rule, "规则更新成功")
}

// DeleteRule DELETE /api/v1/rules/:id
func (h *RuleHandler) DeleteRule(c *gin.Context) {
	userID, role, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	ruleID, ok := parseUintParam(c, "id")
	if !ok {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "规则 ID 无效")
		return
	}

	scopeUserID := userID
	if isAdminRole(role) {
		scopeUserID = 0
	}

	if err := h.ruleService.DeleteRule(scopeUserID, ruleID); err != nil {
		writeServiceError(c, err, "DELETE_RULE_FAILED")
		return
	}

	utils.SuccessResponse(c, nil, "规则删除成功")
}

// StartRule POST /api/v1/rules/:id/start
func (h *RuleHandler) StartRule(c *gin.Context) {
	userID, role, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	ruleID, ok := parseUintParam(c, "id")
	if !ok {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "规则 ID 无效")
		return
	}

	scopeUserID := userID
	if isAdminRole(role) {
		scopeUserID = 0
	}

	rule, err := h.ruleService.StartRule(scopeUserID, ruleID)
	if err != nil {
		writeServiceError(c, err, "START_RULE_FAILED")
		return
	}

	utils.SuccessResponse(c, rule, "规则启动成功")
}

// StopRule POST /api/v1/rules/:id/stop
func (h *RuleHandler) StopRule(c *gin.Context) {
	userID, role, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	ruleID, ok := parseUintParam(c, "id")
	if !ok {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "规则 ID 无效")
		return
	}

	scopeUserID := userID
	if isAdminRole(role) {
		scopeUserID = 0
	}

	rule, err := h.ruleService.StopRule(scopeUserID, ruleID)
	if err != nil {
		writeServiceError(c, err, "STOP_RULE_FAILED")
		return
	}

	utils.SuccessResponse(c, rule, "规则停止成功")
}

// RestartRule POST /api/v1/rules/:id/restart
func (h *RuleHandler) RestartRule(c *gin.Context) {
	userID, role, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	ruleID, ok := parseUintParam(c, "id")
	if !ok {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "规则 ID 无效")
		return
	}

	scopeUserID := userID
	if isAdminRole(role) {
		scopeUserID = 0
	}

	rule, err := h.ruleService.RestartRule(scopeUserID, ruleID)
	if err != nil {
		writeServiceError(c, err, "RESTART_RULE_FAILED")
		return
	}

	utils.SuccessResponse(c, rule, "规则重启成功")
}
