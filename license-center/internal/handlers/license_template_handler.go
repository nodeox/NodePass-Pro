package handlers

import (
	"net/http"
	"strconv"

	"nodepass-license-center/internal/services"
	"nodepass-license-center/internal/utils"

	"github.com/gin-gonic/gin"
)

// LicenseTemplateHandler 授权码模板处理器
type LicenseTemplateHandler struct {
	service *services.LicenseTemplateService
}

// NewLicenseTemplateHandler 创建模板处理器
func NewLicenseTemplateHandler(service *services.LicenseTemplateService) *LicenseTemplateHandler {
	return &LicenseTemplateHandler{service: service}
}

// ListTemplates 查询模板列表
func (h *LicenseTemplateHandler) ListTemplates(c *gin.Context) {
	items, err := h.service.ListTemplates()
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "QUERY_FAILED", err.Error())
		return
	}
	utils.Success(c, items, "ok")
}

// GetTemplate 获取模板详情
func (h *LicenseTemplateHandler) GetTemplate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "id 无效")
		return
	}
	item, err := h.service.GetTemplate(uint(id))
	if err != nil {
		utils.Error(c, http.StatusNotFound, "NOT_FOUND", "模板不存在")
		return
	}
	utils.Success(c, item, "ok")
}

// CreateTemplate 创建模板
func (h *LicenseTemplateHandler) CreateTemplate(c *gin.Context) {
	var req services.CreateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", err.Error())
		return
	}
	item, err := h.service.CreateTemplate(&req, getAdminID(c))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "CREATE_FAILED", err.Error())
		return
	}
	utils.Success(c, item, "创建成功")
}

// UpdateTemplate 更新模板
func (h *LicenseTemplateHandler) UpdateTemplate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "id 无效")
		return
	}
	var req services.CreateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", err.Error())
		return
	}
	item, err := h.service.UpdateTemplate(uint(id), &req)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "UPDATE_FAILED", err.Error())
		return
	}
	utils.Success(c, item, "更新成功")
}

// DeleteTemplate 删除模板
func (h *LicenseTemplateHandler) DeleteTemplate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "id 无效")
		return
	}
	if err := h.service.DeleteTemplate(uint(id)); err != nil {
		utils.Error(c, http.StatusBadRequest, "DELETE_FAILED", err.Error())
		return
	}
	utils.Success(c, gin.H{"id": id}, "删除成功")
}

// GenerateFromTemplate 从模板生成授权码
func (h *LicenseTemplateHandler) GenerateFromTemplate(c *gin.Context) {
	var req services.GenerateFromTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", err.Error())
		return
	}
	req.CreatedBy = getAdminID(c)
	items, err := h.service.GenerateFromTemplate(&req)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "GENERATE_FAILED", err.Error())
		return
	}
	utils.Success(c, items, "生成成功")
}

// ToggleTemplate 启用/禁用模板
func (h *LicenseTemplateHandler) ToggleTemplate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "id 无效")
		return
	}
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", err.Error())
		return
	}
	if err := h.service.ToggleTemplate(uint(id), req.Enabled); err != nil {
		utils.Error(c, http.StatusBadRequest, "UPDATE_FAILED", err.Error())
		return
	}
	utils.Success(c, gin.H{"id": id, "enabled": req.Enabled}, "更新成功")
}
