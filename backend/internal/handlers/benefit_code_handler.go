package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"nodepass-pro/backend/internal/services"
	"nodepass-pro/backend/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// BenefitCodeHandler 权益码处理器。
type BenefitCodeHandler struct {
	benefitService *services.BenefitCodeService
}

// NewBenefitCodeHandler 创建权益码处理器。
func NewBenefitCodeHandler(db *gorm.DB) *BenefitCodeHandler {
	return &BenefitCodeHandler{
		benefitService: services.NewBenefitCodeService(db),
	}
}

// Generate POST /api/v1/benefit-codes/generate (admin)
func (h *BenefitCodeHandler) Generate(c *gin.Context) {
	adminID, role, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}
	if !isAdminRole(role) {
		utils.Error(c, http.StatusForbidden, "FORBIDDEN", "仅管理员可执行此操作")
		return
	}

	type requestPayload struct {
		VIPLevel     int        `json:"vip_level" binding:"required"`
		DurationDays int        `json:"duration_days" binding:"required"`
		Count        int        `json:"count" binding:"required"`
		ExpiresAt    *time.Time `json:"expires_at"`
	}
	var req requestPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	codes, err := h.benefitService.Generate(adminID, req.VIPLevel, req.DurationDays, req.Count, req.ExpiresAt)
	if err != nil {
		writeServiceError(c, err, "GENERATE_BENEFIT_CODES_FAILED")
		return
	}

	utils.Success(c, gin.H{
		"list":  codes,
		"total": len(codes),
	})
}

// List GET /api/v1/benefit-codes (admin)
func (h *BenefitCodeHandler) List(c *gin.Context) {
	_, role, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}
	if !isAdminRole(role) {
		utils.Error(c, http.StatusForbidden, "FORBIDDEN", "仅管理员可执行此操作")
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

	var vipLevel *int
	rawVIPLevel := strings.TrimSpace(c.Query("vip_level"))
	if rawVIPLevel != "" {
		parsed, parseErr := strconv.Atoi(rawVIPLevel)
		if parseErr != nil {
			utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "vip_level 参数错误")
			return
		}
		vipLevel = &parsed
	}

	result, err := h.benefitService.List(services.BenefitCodeListFilters{
		Status:   strings.TrimSpace(c.Query("status")),
		VIPLevel: vipLevel,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		writeServiceError(c, err, "LIST_BENEFIT_CODES_FAILED")
		return
	}
	utils.Success(c, result)
}

// Redeem POST /api/v1/benefit-codes/redeem
func (h *BenefitCodeHandler) Redeem(c *gin.Context) {
	userID, _, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	type requestPayload struct {
		Code string `json:"code" binding:"required"`
	}
	var req requestPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	result, err := h.benefitService.Redeem(userID, req.Code)
	if err != nil {
		writeServiceError(c, err, "REDEEM_BENEFIT_CODE_FAILED")
		return
	}

	utils.SuccessResponse(c, result, "权益码兑换成功")
}

// BatchDelete POST /api/v1/benefit-codes/batch-delete (admin)
func (h *BenefitCodeHandler) BatchDelete(c *gin.Context) {
	adminID, role, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}
	if !isAdminRole(role) {
		utils.Error(c, http.StatusForbidden, "FORBIDDEN", "仅管理员可执行此操作")
		return
	}

	type requestPayload struct {
		IDs []uint `json:"ids" binding:"required"`
	}
	var req requestPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	affected, err := h.benefitService.BatchDelete(adminID, req.IDs)
	if err != nil {
		writeServiceError(c, err, "BATCH_DELETE_BENEFIT_CODES_FAILED")
		return
	}

	utils.Success(c, gin.H{
		"deleted": affected,
	})
}
