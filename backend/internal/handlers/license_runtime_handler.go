package handlers

import (
	"net/http"
	"strings"

	"nodepass-pro/backend/internal/license"
	"nodepass-pro/backend/internal/utils"

	"github.com/gin-gonic/gin"
)

// LicenseRuntimeHandler 运行时授权处理器。
type LicenseRuntimeHandler struct {
	manager *license.Manager
}

// NewLicenseRuntimeHandler 创建运行时授权处理器。
func NewLicenseRuntimeHandler(manager *license.Manager) *LicenseRuntimeHandler {
	return &LicenseRuntimeHandler{manager: manager}
}

// GetStatus GET /api/v1/license/status
func (h *LicenseRuntimeHandler) GetStatus(c *gin.Context) {
	if h.manager == nil {
		utils.Success(c, gin.H{
			"enabled": false,
			"valid":   true,
			"message": "license manager not initialized",
		})
		return
	}
	utils.Success(c, h.manager.Status())
}

// UpdateDomain PUT /api/v1/license/domain
func (h *LicenseRuntimeHandler) UpdateDomain(c *gin.Context) {
	if h.manager == nil {
		utils.Error(c, http.StatusServiceUnavailable, "LICENSE_MANAGER_UNAVAILABLE", "授权管理器未初始化")
		return
	}

	type request struct {
		Domain  string `json:"domain"`
		SiteURL string `json:"site_url"`
	}

	var req request
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	if strings.TrimSpace(req.Domain) == "" && strings.TrimSpace(req.SiteURL) == "" {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "domain 与 site_url 至少提供一个")
		return
	}

	status, err := h.manager.UpdateDomain(req.Domain, req.SiteURL)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "LICENSE_DOMAIN_UPDATE_FAILED", err.Error())
		return
	}

	utils.SuccessResponse(c, status, "授权域名更新成功")
}
