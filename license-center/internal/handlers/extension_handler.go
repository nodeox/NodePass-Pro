package handlers

import (
	"net/http"
	"strconv"

	"nodepass-license-center/internal/services"
	"nodepass-license-center/internal/utils"

	"github.com/gin-gonic/gin"
)

// ExtensionHandler 扩展功能处理器
type ExtensionHandler struct {
	extensionService *services.ExtensionService
	webhookService   *services.WebhookService
}

// NewExtensionHandler 创建扩展功能处理器
func NewExtensionHandler(extensionService *services.ExtensionService, webhookService *services.WebhookService) *ExtensionHandler {
	return &ExtensionHandler{
		extensionService: extensionService,
		webhookService:   webhookService,
	}
}

// TransferLicense 转移授权码
func (h *ExtensionHandler) TransferLicense(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	if id == 0 {
		utils.Error(c, http.StatusBadRequest, "ERROR", "参数无效")
		return
	}

	var req struct {
		ToCustomer string `json:"to_customer" binding:"required"`
		Reason     string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "ERROR", "参数错误")
		return
	}

	operatorID := c.GetUint("admin_id")
	if err := h.extensionService.TransferLicense(uint(id), req.ToCustomer, req.Reason, operatorID); err != nil {
		utils.Error(c, http.StatusInternalServerError, "ERROR", err.Error())
		return
	}

	utils.Success(c, nil, "转移成功")
}

// BatchUpdateLicenses 批量更新授权码
func (h *ExtensionHandler) BatchUpdateLicenses(c *gin.Context) {
	var req struct {
		LicenseIDs []uint                 `json:"license_ids" binding:"required"`
		Updates    map[string]interface{} `json:"updates" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "ERROR", "参数错误")
		return
	}

	if err := h.extensionService.BatchUpdateLicenses(req.LicenseIDs, req.Updates); err != nil {
		utils.Error(c, http.StatusInternalServerError, "ERROR", err.Error())
		return
	}

	utils.Success(c, nil, "批量更新成功")
}

// BatchRevokeLicenses 批量吊销授权码
func (h *ExtensionHandler) BatchRevokeLicenses(c *gin.Context) {
	var req struct {
		LicenseIDs []uint `json:"license_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "ERROR", "参数错误")
		return
	}

	if err := h.extensionService.BatchRevokeLicenses(req.LicenseIDs); err != nil {
		utils.Error(c, http.StatusInternalServerError, "ERROR", err.Error())
		return
	}

	utils.Success(c, nil, "批量吊销成功")
}

// BatchRestoreLicenses 批量恢复授权码
func (h *ExtensionHandler) BatchRestoreLicenses(c *gin.Context) {
	var req struct {
		LicenseIDs []uint `json:"license_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "ERROR", "参数错误")
		return
	}

	if err := h.extensionService.BatchRestoreLicenses(req.LicenseIDs); err != nil {
		utils.Error(c, http.StatusInternalServerError, "ERROR", err.Error())
		return
	}

	utils.Success(c, nil, "批量恢复成功")
}

// BatchDeleteLicenses 批量删除授权码
func (h *ExtensionHandler) BatchDeleteLicenses(c *gin.Context) {
	var req struct {
		LicenseIDs []uint `json:"license_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "ERROR", "参数错误")
		return
	}

	if err := h.extensionService.BatchDeleteLicenses(req.LicenseIDs); err != nil {
		utils.Error(c, http.StatusInternalServerError, "ERROR", err.Error())
		return
	}

	utils.Success(c, nil, "批量删除成功")
}

// ListTags 查询标签
func (h *ExtensionHandler) ListTags(c *gin.Context) {
	tags, err := h.extensionService.ListTags()
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "ERROR", "查询失败")
		return
	}

	utils.Success(c, tags, "ok")
}

// CreateTag 创建标签
func (h *ExtensionHandler) CreateTag(c *gin.Context) {
	var req struct {
		Name  string `json:"name" binding:"required"`
		Color string `json:"color"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "ERROR", "参数错误")
		return
	}

	tag, err := h.extensionService.CreateTag(req.Name, req.Color)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "ERROR", err.Error())
		return
	}

	utils.Success(c, tag, "创建成功")
}

// UpdateTag 更新标签
func (h *ExtensionHandler) UpdateTag(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	if id == 0 {
		utils.Error(c, http.StatusBadRequest, "ERROR", "参数无效")
		return
	}

	var req struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "ERROR", "参数错误")
		return
	}

	if err := h.extensionService.UpdateTag(uint(id), req.Name, req.Color); err != nil {
		utils.Error(c, http.StatusInternalServerError, "ERROR", err.Error())
		return
	}

	utils.Success(c, nil, "更新成功")
}

// DeleteTag 删除标签
func (h *ExtensionHandler) DeleteTag(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	if id == 0 {
		utils.Error(c, http.StatusBadRequest, "ERROR", "参数无效")
		return
	}

	if err := h.extensionService.DeleteTag(uint(id)); err != nil {
		utils.Error(c, http.StatusInternalServerError, "ERROR", err.Error())
		return
	}

	utils.Success(c, nil, "删除成功")
}

// AddTagsToLicense 为授权码添加标签
func (h *ExtensionHandler) AddTagsToLicense(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	if id == 0 {
		utils.Error(c, http.StatusBadRequest, "ERROR", "参数无效")
		return
	}

	var req struct {
		TagIDs []uint `json:"tag_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "ERROR", "参数错误")
		return
	}

	if err := h.extensionService.AddTagsToLicense(uint(id), req.TagIDs); err != nil {
		utils.Error(c, http.StatusInternalServerError, "ERROR", err.Error())
		return
	}

	utils.Success(c, nil, "添加成功")
}

// RemoveTagsFromLicense 从授权码移除标签
func (h *ExtensionHandler) RemoveTagsFromLicense(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	if id == 0 {
		utils.Error(c, http.StatusBadRequest, "ERROR", "参数无效")
		return
	}

	var req struct {
		TagIDs []uint `json:"tag_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "ERROR", "参数错误")
		return
	}

	if err := h.extensionService.RemoveTagsFromLicense(uint(id), req.TagIDs); err != nil {
		utils.Error(c, http.StatusInternalServerError, "ERROR", err.Error())
		return
	}

	utils.Success(c, nil, "移除成功")
}

// GetLicenseTags 获取授权码的标签
func (h *ExtensionHandler) GetLicenseTags(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	if id == 0 {
		utils.Error(c, http.StatusBadRequest, "ERROR", "参数无效")
		return
	}

	tags, err := h.extensionService.GetLicenseTags(uint(id))
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "ERROR", "查询失败")
		return
	}

	utils.Success(c, tags, "ok")
}

// ListWebhooks 查询 Webhook
func (h *ExtensionHandler) ListWebhooks(c *gin.Context) {
	webhooks, err := h.webhookService.ListWebhooks()
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "ERROR", "查询失败")
		return
	}

	utils.Success(c, webhooks, "ok")
}

// CreateWebhook 创建 Webhook
func (h *ExtensionHandler) CreateWebhook(c *gin.Context) {
	var req struct {
		Name      string   `json:"name" binding:"required"`
		URL       string   `json:"url" binding:"required"`
		Secret    string   `json:"secret"`
		Events    []string `json:"events" binding:"required"`
		IsEnabled bool     `json:"is_enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "ERROR", "参数错误")
		return
	}

	webhook, err := h.webhookService.CreateWebhook(req.Name, req.URL, req.Secret, req.Events, req.IsEnabled)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "ERROR", err.Error())
		return
	}

	utils.Success(c, webhook, "创建成功")
}

// UpdateWebhook 更新 Webhook
func (h *ExtensionHandler) UpdateWebhook(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	if id == 0 {
		utils.Error(c, http.StatusBadRequest, "ERROR", "参数无效")
		return
	}

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "ERROR", "参数错误")
		return
	}

	if err := h.webhookService.UpdateWebhook(uint(id), req); err != nil {
		utils.Error(c, http.StatusInternalServerError, "ERROR", err.Error())
		return
	}

	utils.Success(c, nil, "更新成功")
}

// DeleteWebhook 删除 Webhook
func (h *ExtensionHandler) DeleteWebhook(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	if id == 0 {
		utils.Error(c, http.StatusBadRequest, "ERROR", "参数无效")
		return
	}

	if err := h.webhookService.DeleteWebhook(uint(id)); err != nil {
		utils.Error(c, http.StatusInternalServerError, "ERROR", err.Error())
		return
	}

	utils.Success(c, nil, "删除成功")
}

// ListWebhookLogs 查询 Webhook 日志
func (h *ExtensionHandler) ListWebhookLogs(c *gin.Context) {
	webhookID, _ := strconv.ParseUint(c.DefaultQuery("webhook_id", "0"), 10, 32)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	result, err := h.webhookService.ListWebhookLogs(uint(webhookID), page, pageSize)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "ERROR", "查询失败")
		return
	}

	utils.Success(c, result, "ok")
}
