package handlers

import (
	"net/http"
	"strings"

	"nodepass-pro/backend/internal/models"
	"nodepass-pro/backend/internal/services"
	"nodepass-pro/backend/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// TunnelTemplateHandler 隧道模板处理器。
type TunnelTemplateHandler struct {
	service *services.TunnelTemplateService
}

// NewTunnelTemplateHandler 创建隧道模板处理器。
func NewTunnelTemplateHandler(db *gorm.DB) *TunnelTemplateHandler {
	return &TunnelTemplateHandler{service: services.NewTunnelTemplateService(db)}
}

// Create POST /api/v1/tunnel-templates
func (h *TunnelTemplateHandler) Create(c *gin.Context) {
	userID, _, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	var req services.CreateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	template, err := h.service.Create(userID, &req)
	if err != nil {
		writeServiceError(c, err, "CREATE_TEMPLATE_FAILED")
		return
	}
	utils.SuccessResponse(c, template, "模板创建成功")
}

// List GET /api/v1/tunnel-templates
func (h *TunnelTemplateHandler) List(c *gin.Context) {
	userID, _, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	page, err := parsePositiveIntQuery(c, "page", 1)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "page 参数错误")
		return
	}
	pageSize, err := parsePositiveIntQuery(c, "page_size", 20)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "page_size 参数错误")
		return
	}

	params := &services.ListTemplateParams{
		Page:     page,
		PageSize: pageSize,
	}

	if protocol := strings.TrimSpace(c.Query("protocol")); protocol != "" {
		params.Protocol = &protocol
	}

	if isPublicStr := strings.TrimSpace(c.Query("is_public")); isPublicStr != "" {
		isPublic := isPublicStr == "true" || isPublicStr == "1"
		params.IsPublic = &isPublic
	}

	items, total, err := h.service.List(userID, params)
	if err != nil {
		writeServiceError(c, err, "LIST_TEMPLATES_FAILED")
		return
	}

	utils.SuccessResponse(c, http.StatusOK, gin.H{
		"items":     items,
		"total":     total,
		"page":      params.Page,
		"page_size": params.PageSize,
	}, "获取模板列表成功")
}

// Get GET /api/v1/tunnel-templates/:id
func (h *TunnelTemplateHandler) Get(c *gin.Context) {
	userID, _, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	templateID, ok := parseUintParam(c, "id")
	if !ok {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "模板 ID 无效")
		return
	}

	template, err := h.service.Get(userID, templateID)
	if err != nil {
		writeServiceError(c, err, "GET_TEMPLATE_FAILED")
		return
	}
	utils.Success(c, template)
}

// Update PUT /api/v1/tunnel-templates/:id
func (h *TunnelTemplateHandler) Update(c *gin.Context) {
	userID, _, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	templateID, ok := parseUintParam(c, "id")
	if !ok {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "模板 ID 无效")
		return
	}

	var req services.UpdateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	template, err := h.service.Update(userID, templateID, &req)
	if err != nil {
		writeServiceError(c, err, "UPDATE_TEMPLATE_FAILED")
		return
	}
	utils.SuccessResponse(c, template, "模板更新成功")
}

// Delete DELETE /api/v1/tunnel-templates/:id
func (h *TunnelTemplateHandler) Delete(c *gin.Context) {
	userID, _, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	templateID, ok := parseUintParam(c, "id")
	if !ok {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "模板 ID 无效")
		return
	}

	if err := h.service.Delete(userID, templateID); err != nil {
		writeServiceError(c, err, "DELETE_TEMPLATE_FAILED")
		return
	}
	utils.SuccessResponse(c, nil, "模板删除成功")
}

// TunnelImportExportHandler 隧道导入导出处理器。
type TunnelImportExportHandler struct {
	db      *gorm.DB
	service *services.TunnelService
}

// NewTunnelImportExportHandler 创建隧道导入导出处理器。
func NewTunnelImportExportHandler(db *gorm.DB) *TunnelImportExportHandler {
	return &TunnelImportExportHandler{
		db:      db,
		service: services.NewTunnelService(db),
	}
}

// Export POST /api/v1/tunnels/export
func (h *TunnelImportExportHandler) Export(c *gin.Context) {
	userID, _, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	var req struct {
		TunnelIDs []uint                     `json:"tunnel_ids" binding:"required"`
		Format    services.TunnelExportFormat `json:"format" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	data, err := h.service.ExportTunnels(userID, req.TunnelIDs, req.Format)
	if err != nil {
		writeServiceError(c, err, "EXPORT_TUNNELS_FAILED")
		return
	}

	utils.SuccessResponse(c, gin.H{
		"data":   data,
		"format": req.Format,
	}, "导出成功")
}

// ExportAll POST /api/v1/tunnels/export-all
func (h *TunnelImportExportHandler) ExportAll(c *gin.Context) {
	userID, _, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	var req struct {
		Format services.TunnelExportFormat `json:"format" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	data, err := h.service.BatchExportAllTunnels(userID, req.Format)
	if err != nil {
		writeServiceError(c, err, "EXPORT_ALL_TUNNELS_FAILED")
		return
	}

	utils.SuccessResponse(c, gin.H{
		"data":   data,
		"format": req.Format,
	}, "导出成功")
}

// Import POST /api/v1/tunnels/import
func (h *TunnelImportExportHandler) Import(c *gin.Context) {
	userID, _, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	var req services.TunnelImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	result, err := h.service.ImportTunnels(userID, &req)
	if err != nil {
		// 如果是部分导入成功，返回结果而不是错误
		if result != nil && result.Success > 0 {
			utils.SuccessResponse(c, result, "部分导入成功")
			return
		}
		writeServiceError(c, err, "IMPORT_TUNNELS_FAILED")
		return
	}

	utils.SuccessResponse(c, result, "导入成功")
}

// ApplyTemplate POST /api/v1/tunnels/apply-template
func (h *TunnelImportExportHandler) ApplyTemplate(c *gin.Context) {
	userID, _, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	var req struct {
		TemplateID   uint   `json:"template_id" binding:"required"`
		Name         string `json:"name" binding:"required"`
		Description  *string `json:"description"`
		EntryGroupID uint   `json:"entry_group_id" binding:"required"`
		ExitGroupID  *uint  `json:"exit_group_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	templateService := services.NewTunnelTemplateService(h.db)
	template, err := templateService.Get(userID, req.TemplateID)
	if err != nil {
		writeServiceError(c, err, "GET_TEMPLATE_FAILED")
		return
	}

	templateConfig, err := template.GetConfig()
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "PARSE_TEMPLATE_FAILED", "解析模板配置失败")
		return
	}

	// 构建隧道配置
	tunnelConfig := &models.TunnelConfig{
		LoadBalanceStrategy: models.LoadBalanceStrategy(templateConfig.LoadBalanceStrategy),
		IPType:              templateConfig.IPType,
		EnableProxyProtocol: templateConfig.EnableProxyProtocol,
		ForwardTargets:      templateConfig.ForwardTargets,
		HealthCheckInterval: templateConfig.HealthCheckInterval,
		HealthCheckTimeout:  templateConfig.HealthCheckTimeout,
		ProtocolConfig:      templateConfig.ProtocolConfig,
	}

	createReq := &services.CreateTunnelRequest{
		Name:         req.Name,
		Description:  req.Description,
		EntryGroupID: req.EntryGroupID,
		ExitGroupID:  req.ExitGroupID,
		Protocol:     template.Protocol,
		ListenHost:   templateConfig.ListenHost,
		ListenPort:   templateConfig.ListenPort,
		RemoteHost:   templateConfig.RemoteHost,
		RemotePort:   templateConfig.RemotePort,
		Config:       tunnelConfig,
	}

	tunnel, err := h.service.Create(userID, createReq)
	if err != nil {
		writeServiceError(c, err, "APPLY_TEMPLATE_FAILED")
		return
	}

	// 增加模板使用次数
	_ = templateService.IncrementUsage(template.ID)

	utils.SuccessResponse(c, tunnel, "应用模板成功")
}
