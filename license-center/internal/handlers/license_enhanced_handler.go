package handlers

import (
	"net/http"
	"strconv"

	"nodepass-license-center/internal/services"
	"nodepass-license-center/internal/utils"

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

// BatchUpdate 批量更新授权码
func (h *LicenseEnhancedHandler) BatchUpdate(c *gin.Context) {
	var req services.BatchUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", err.Error())
		return
	}
	if err := h.service.BatchUpdate(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "UPDATE_FAILED", err.Error())
		return
	}
	utils.Success(c, gin.H{"count": len(req.LicenseIDs)}, "批量更新成功")
}

// BatchTransfer 批量转移客户
func (h *LicenseEnhancedHandler) BatchTransfer(c *gin.Context) {
	var req services.BatchTransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", err.Error())
		return
	}
	if err := h.service.BatchTransfer(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "TRANSFER_FAILED", err.Error())
		return
	}
	utils.Success(c, gin.H{"count": len(req.LicenseIDs), "new_customer": req.NewCustomer}, "批量转移成功")
}

// BatchRevoke 批量吊销
func (h *LicenseEnhancedHandler) BatchRevoke(c *gin.Context) {
	var req struct {
		LicenseIDs []uint `json:"license_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", err.Error())
		return
	}
	if err := h.service.BatchRevoke(req.LicenseIDs); err != nil {
		utils.Error(c, http.StatusBadRequest, "REVOKE_FAILED", err.Error())
		return
	}
	utils.Success(c, gin.H{"count": len(req.LicenseIDs)}, "批量吊销成功")
}

// BatchRestore 批量恢复
func (h *LicenseEnhancedHandler) BatchRestore(c *gin.Context) {
	var req struct {
		LicenseIDs []uint `json:"license_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", err.Error())
		return
	}
	if err := h.service.BatchRestore(req.LicenseIDs); err != nil {
		utils.Error(c, http.StatusBadRequest, "RESTORE_FAILED", err.Error())
		return
	}
	utils.Success(c, gin.H{"count": len(req.LicenseIDs)}, "批量恢复成功")
}

// BatchDelete 批量删除
func (h *LicenseEnhancedHandler) BatchDelete(c *gin.Context) {
	var req struct {
		LicenseIDs []uint `json:"license_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", err.Error())
		return
	}
	if err := h.service.BatchDelete(req.LicenseIDs); err != nil {
		utils.Error(c, http.StatusBadRequest, "DELETE_FAILED", err.Error())
		return
	}
	utils.Success(c, gin.H{"count": len(req.LicenseIDs)}, "批量删除成功")
}

// AdvancedSearch 高级搜索
func (h *LicenseEnhancedHandler) AdvancedSearch(c *gin.Context) {
	var req services.AdvancedSearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", err.Error())
		return
	}
	result, err := h.service.AdvancedSearch(&req)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "SEARCH_FAILED", err.Error())
		return
	}
	utils.Success(c, result, "ok")
}

// SaveSearch 保存搜索条件
func (h *LicenseEnhancedHandler) SaveSearch(c *gin.Context) {
	var req services.SavedSearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", err.Error())
		return
	}
	item, err := h.service.SaveSearch(&req, getAdminID(c))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "SAVE_FAILED", err.Error())
		return
	}
	utils.Success(c, item, "保存成功")
}

// ListSavedSearches 查询保存的搜索
func (h *LicenseEnhancedHandler) ListSavedSearches(c *gin.Context) {
	items, err := h.service.ListSavedSearches(getAdminID(c))
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "QUERY_FAILED", err.Error())
		return
	}
	utils.Success(c, items, "ok")
}

// GetSavedSearch 获取保存的搜索
func (h *LicenseEnhancedHandler) GetSavedSearch(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "id 无效")
		return
	}
	item, err := h.service.GetSavedSearch(uint(id))
	if err != nil {
		utils.Error(c, http.StatusNotFound, "NOT_FOUND", "搜索不存在")
		return
	}
	utils.Success(c, item, "ok")
}

// DeleteSavedSearch 删除保存的搜索
func (h *LicenseEnhancedHandler) DeleteSavedSearch(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "id 无效")
		return
	}
	if err := h.service.DeleteSavedSearch(uint(id), getAdminID(c)); err != nil {
		utils.Error(c, http.StatusBadRequest, "DELETE_FAILED", err.Error())
		return
	}
	utils.Success(c, gin.H{"id": id}, "删除成功")
}

// GetStatistics 获取统计信息
func (h *LicenseEnhancedHandler) GetStatistics(c *gin.Context) {
	stats, err := h.service.GetStatistics()
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "QUERY_FAILED", err.Error())
		return
	}
	utils.Success(c, stats, "ok")
}

// GetExpiringLicenses 获取即将过期的授权码
func (h *LicenseEnhancedHandler) GetExpiringLicenses(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))
	licenses, err := h.service.GetExpiringLicenses(days)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "QUERY_FAILED", err.Error())
		return
	}
	utils.Success(c, licenses, "ok")
}
