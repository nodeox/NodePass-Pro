package handlers

import (
	"net/http"
	"os"
	"strconv"

	"nodepass-license-unified/internal/services"
	"nodepass-license-unified/internal/utils"

	"github.com/gin-gonic/gin"
)

// LicenseEnhancedHandler 授权码增强处理器
type LicenseEnhancedHandler struct {
	service *services.LicenseEnhancedService
}

// NewLicenseEnhancedHandler 创建增强处理器
func NewLicenseEnhancedHandler(service *services.LicenseEnhancedService) *LicenseEnhancedHandler {
	return &LicenseEnhancedHandler{service: service}
}

// ExportCSV 导出授权码到 CSV
func (h *LicenseEnhancedHandler) ExportCSV(c *gin.Context) {
	var req services.ExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "参数错误")
		return
	}

	filePath, err := h.service.ExportToCSV(&req)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "EXPORT_FAILED", err.Error())
		return
	}
	defer func() {
		_ = os.Remove(filePath)
	}()

	c.FileAttachment(filePath, "licenses_export.csv")
}

// RenewLicenses 批量续期
func (h *LicenseEnhancedHandler) RenewLicenses(c *gin.Context) {
	var req services.RenewalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "参数错误")
		return
	}

	if err := h.service.RenewLicenses(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "RENEW_FAILED", err.Error())
		return
	}

	utils.Success(c, gin.H{
		"renewed_count": len(req.LicenseIDs),
	}, "续期成功")
}

// CloneLicense 克隆授权码
func (h *LicenseEnhancedHandler) CloneLicense(c *gin.Context) {
	var req services.CloneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "参数错误")
		return
	}

	createdBy := getAdminIDFromContext(c)

	licenses, err := h.service.CloneLicense(&req, createdBy)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "CLONE_FAILED", err.Error())
		return
	}

	utils.Success(c, licenses, "克隆成功")
}

// BatchUpdate 批量更新
func (h *LicenseEnhancedHandler) BatchUpdate(c *gin.Context) {
	var req services.BatchUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "参数错误")
		return
	}

	if err := h.service.BatchUpdateWithOperator(&req, getAdminIDFromContext(c)); err != nil {
		utils.Error(c, http.StatusBadRequest, "UPDATE_FAILED", err.Error())
		return
	}

	utils.Success(c, gin.H{
		"updated_count": len(req.LicenseIDs),
	}, "更新成功")
}

// BatchRevoke 批量吊销
func (h *LicenseEnhancedHandler) BatchRevoke(c *gin.Context) {
	var req struct {
		LicenseIDs []uint `json:"license_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "参数错误")
		return
	}

	if err := h.service.BatchRevokeWithOperator(req.LicenseIDs, getAdminIDFromContext(c)); err != nil {
		utils.Error(c, http.StatusBadRequest, "REVOKE_FAILED", err.Error())
		return
	}

	utils.Success(c, gin.H{
		"revoked_count": len(req.LicenseIDs),
	}, "吊销成功")
}

// BatchRestore 批量恢复
func (h *LicenseEnhancedHandler) BatchRestore(c *gin.Context) {
	var req struct {
		LicenseIDs []uint `json:"license_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "参数错误")
		return
	}

	if err := h.service.BatchRestoreWithOperator(req.LicenseIDs, getAdminIDFromContext(c)); err != nil {
		utils.Error(c, http.StatusBadRequest, "RESTORE_FAILED", err.Error())
		return
	}

	utils.Success(c, gin.H{
		"restored_count": len(req.LicenseIDs),
	}, "恢复成功")
}

// GetUsageReport 获取使用报告
func (h *LicenseEnhancedHandler) GetUsageReport(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "id 无效")
		return
	}

	report, err := h.service.GetUsageReport(uint(id))
	if err != nil {
		utils.Error(c, http.StatusNotFound, "NOT_FOUND", err.Error())
		return
	}

	utils.Success(c, report, "ok")
}

func getAdminIDFromContext(c *gin.Context) uint {
	value, exists := c.Get("admin_id")
	if !exists {
		return 0
	}

	switch v := value.(type) {
	case uint:
		return v
	case uint64:
		return uint(v)
	case int:
		if v > 0 {
			return uint(v)
		}
	case int64:
		if v > 0 {
			return uint(v)
		}
	}
	return 0
}

// GetCustomerReport 获取客户报告
func (h *LicenseEnhancedHandler) GetCustomerReport(c *gin.Context) {
	customer := c.Query("customer")
	if customer == "" {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "客户名称不能为空")
		return
	}

	report, err := h.service.GetCustomerReport(customer)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "QUERY_FAILED", err.Error())
		return
	}

	utils.Success(c, report, "ok")
}
