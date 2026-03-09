package handlers

import (
	"net/http"
	"strconv"

	"nodepass-license-center/internal/models"
	"nodepass-license-center/internal/services"
	"nodepass-license-center/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// VersionHandler 版本管理处理器
type VersionHandler struct {
	service *services.VersionService
}

// NewVersionHandler 创建版本管理处理器
func NewVersionHandler(db *gorm.DB) *VersionHandler {
	return &VersionHandler{
		service: services.NewVersionService(db),
	}
}

// GetSystemVersionInfo 获取系统版本信息
// GET /api/versions/system
func (h *VersionHandler) GetSystemVersionInfo(c *gin.Context) {
	info, err := h.service.GetSystemVersionInfo()
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "GET_VERSION_FAILED", err.Error())
		return
	}
	utils.Success(c, info, "ok")
}

// GetComponentVersion 获取组件版本
// GET /api/versions/components/:component
func (h *VersionHandler) GetComponentVersion(c *gin.Context) {
	component := models.ComponentType(c.Param("component"))

	version, err := h.service.GetComponentVersion(component)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "GET_VERSION_FAILED", err.Error())
		return
	}

	if version == nil {
		utils.Error(c, http.StatusNotFound, "NOT_FOUND", "未找到组件版本")
		return
	}

	utils.Success(c, version, "ok")
}

// UpdateComponentVersion 更新组件版本
// POST /api/versions/components
func (h *VersionHandler) UpdateComponentVersion(c *gin.Context) {
	var req models.ComponentVersion
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", err.Error())
		return
	}

	if err := h.service.UpdateComponentVersion(&req); err != nil {
		utils.Error(c, http.StatusInternalServerError, "UPDATE_FAILED", err.Error())
		return
	}

	utils.Success(c, req, "更新成功")
}

// ListComponentVersions 列出组件版本历史
// GET /api/versions/components/:component/history
func (h *VersionHandler) ListComponentVersions(c *gin.Context) {
	component := models.ComponentType(c.Param("component"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	versions, err := h.service.ListComponentVersions(component, limit)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "QUERY_FAILED", err.Error())
		return
	}

	utils.Success(c, versions, "ok")
}

// CheckCompatibility 检查版本兼容性
// GET /api/versions/compatibility/:version
func (h *VersionHandler) CheckCompatibility(c *gin.Context) {
	version := c.Param("version")

	info, err := h.service.CheckCompatibility(version)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "CHECK_FAILED", err.Error())
		return
	}

	utils.Success(c, info, "ok")
}

// CreateCompatibilityConfig 创建兼容性配置
// POST /api/versions/compatibility
func (h *VersionHandler) CreateCompatibilityConfig(c *gin.Context) {
	var req models.VersionCompatibility
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", err.Error())
		return
	}

	if err := h.service.CreateCompatibilityConfig(&req); err != nil {
		utils.Error(c, http.StatusInternalServerError, "CREATE_FAILED", err.Error())
		return
	}

	utils.Success(c, req, "创建成功")
}

// ListCompatibilityConfigs 列出兼容性配置
// GET /api/versions/compatibility
func (h *VersionHandler) ListCompatibilityConfigs(c *gin.Context) {
	configs, err := h.service.ListCompatibilityConfigs()
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "QUERY_FAILED", err.Error())
		return
	}

	utils.Success(c, configs, "ok")
}
