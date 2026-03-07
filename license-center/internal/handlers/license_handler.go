package handlers

import (
	"net/http"
	"strconv"
	"time"

	"nodepass-license-center/internal/services"
	"nodepass-license-center/internal/utils"

	"github.com/gin-gonic/gin"
)

// LicenseHandler 授权管理处理器。
type LicenseHandler struct {
	service *services.LicenseService
}

// NewLicenseHandler 创建授权处理器。
func NewLicenseHandler(service *services.LicenseService) *LicenseHandler {
	return &LicenseHandler{service: service}
}

// Verify 授权校验（公开接口）。
func (h *LicenseHandler) Verify(c *gin.Context) {
	var req services.VerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", err.Error())
		return
	}
	result, err := h.service.Verify(&req, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "VERIFY_FAILED", err.Error())
		return
	}
	utils.Success(c, result, "ok")
}

// ListPlans 查询套餐。
func (h *LicenseHandler) ListPlans(c *gin.Context) {
	items, err := h.service.ListPlans()
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "QUERY_FAILED", err.Error())
		return
	}
	utils.Success(c, items, "ok")
}

// CreatePlan 创建套餐。
func (h *LicenseHandler) CreatePlan(c *gin.Context) {
	var req services.CreatePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", err.Error())
		return
	}
	plan, err := h.service.CreatePlan(&req)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "CREATE_FAILED", err.Error())
		return
	}
	utils.Success(c, plan, "创建成功")
}

// UpdatePlan 更新套餐。
func (h *LicenseHandler) UpdatePlan(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "id 无效")
		return
	}
	var req services.CreatePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", err.Error())
		return
	}
	plan, err := h.service.UpdatePlan(uint(id), &req)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "UPDATE_FAILED", err.Error())
		return
	}
	utils.Success(c, plan, "更新成功")
}

// DeletePlan 删除套餐。
func (h *LicenseHandler) DeletePlan(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "id 无效")
		return
	}
	if err := h.service.DeletePlan(uint(id)); err != nil {
		utils.Error(c, http.StatusBadRequest, "DELETE_FAILED", err.Error())
		return
	}
	utils.Success(c, gin.H{"id": id}, "删除成功")
}

// GenerateLicenses 生成授权码。
func (h *LicenseHandler) GenerateLicenses(c *gin.Context) {
	var req services.GenerateLicenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", err.Error())
		return
	}
	req.CreatedBy = getAdminID(c)
	items, err := h.service.GenerateLicenses(&req)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "GENERATE_FAILED", err.Error())
		return
	}
	utils.Success(c, items, "生成成功")
}

// ListLicenses 查询授权码。
func (h *LicenseHandler) ListLicenses(c *gin.Context) {
	planID, _ := strconv.ParseUint(c.Query("plan_id"), 10, 64)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	result, err := h.service.ListLicenses(services.LicenseFilter{
		Status:   c.Query("status"),
		Customer: c.Query("customer"),
		PlanID:   uint(planID),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "QUERY_FAILED", err.Error())
		return
	}
	utils.Success(c, result, "ok")
}

// GetLicense 获取授权详情。
func (h *LicenseHandler) GetLicense(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "id 无效")
		return
	}
	item, err := h.service.GetLicense(uint(id))
	if err != nil {
		utils.Error(c, http.StatusNotFound, "NOT_FOUND", "授权码不存在")
		return
	}
	utils.Success(c, item, "ok")
}

// UpdateLicense 更新授权码。
func (h *LicenseHandler) UpdateLicense(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "id 无效")
		return
	}

	payload := map[string]interface{}{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", err.Error())
		return
	}

	if raw, ok := payload["expires_at"]; ok {
		switch v := raw.(type) {
		case string:
			if v == "" {
				payload["expires_at"] = nil
			} else if parsed, parseErr := time.Parse(time.RFC3339, v); parseErr == nil {
				payload["expires_at"] = parsed
			}
		}
	}

	allowed := map[string]bool{
		"status": true, "expires_at": true, "max_machines": true,
		"note": true, "metadata_json": true, "customer": true,
	}
	for k := range payload {
		if !allowed[k] {
			delete(payload, k)
		}
	}

	item, err := h.service.UpdateLicense(uint(id), payload)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "UPDATE_FAILED", err.Error())
		return
	}
	utils.Success(c, item, "更新成功")
}

// RevokeLicense 吊销授权。
func (h *LicenseHandler) RevokeLicense(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "id 无效")
		return
	}
	if err := h.service.RevokeLicense(uint(id)); err != nil {
		utils.Error(c, http.StatusInternalServerError, "UPDATE_FAILED", err.Error())
		return
	}
	utils.Success(c, gin.H{"id": id, "status": "revoked"}, "已吊销")
}

// RestoreLicense 恢复授权。
func (h *LicenseHandler) RestoreLicense(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "id 无效")
		return
	}
	if err := h.service.RestoreLicense(uint(id)); err != nil {
		utils.Error(c, http.StatusInternalServerError, "UPDATE_FAILED", err.Error())
		return
	}
	utils.Success(c, gin.H{"id": id, "status": "active"}, "已恢复")
}

// DeleteLicense 删除授权码。
func (h *LicenseHandler) DeleteLicense(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "id 无效")
		return
	}
	if err := h.service.DeleteLicense(uint(id)); err != nil {
		utils.Error(c, http.StatusBadRequest, "DELETE_FAILED", err.Error())
		return
	}
	utils.Success(c, gin.H{"id": id}, "删除成功")
}

// ListActivations 查询授权机器绑定。
func (h *LicenseHandler) ListActivations(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "id 无效")
		return
	}
	items, err := h.service.ListActivations(uint(id))
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "QUERY_FAILED", err.Error())
		return
	}
	utils.Success(c, items, "ok")
}

// UnbindActivation 解绑机器。
func (h *LicenseHandler) UnbindActivation(c *gin.Context) {
	licenseID, err1 := strconv.ParseUint(c.Param("id"), 10, 64)
	activationID, err2 := strconv.ParseUint(c.Param("activationId"), 10, 64)
	if err1 != nil || err2 != nil || licenseID == 0 || activationID == 0 {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "参数无效")
		return
	}
	if err := h.service.UnbindActivation(uint(licenseID), uint(activationID)); err != nil {
		utils.Error(c, http.StatusInternalServerError, "UPDATE_FAILED", err.Error())
		return
	}
	utils.Success(c, gin.H{"license_id": licenseID, "activation_id": activationID}, "解绑成功")
}

// ListVerifyLogs 查询验证日志。
func (h *LicenseHandler) ListVerifyLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	result, err := h.service.ListVerifyLogs(page, pageSize)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "QUERY_FAILED", err.Error())
		return
	}
	utils.Success(c, result, "ok")
}

// Stats 查询统计。
func (h *LicenseHandler) Stats(c *gin.Context) {
	stats, err := h.service.Stats()
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "QUERY_FAILED", err.Error())
		return
	}
	utils.Success(c, stats, "ok")
}
