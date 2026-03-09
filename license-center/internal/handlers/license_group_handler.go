package handlers

import (
	"net/http"
	"strconv"

	"nodepass-license-center/internal/services"
	"nodepass-license-center/internal/utils"

	"github.com/gin-gonic/gin"
)

// LicenseGroupHandler 授权码分组处理器
type LicenseGroupHandler struct {
	service *services.LicenseGroupService
}

// NewLicenseGroupHandler 创建分组处理器
func NewLicenseGroupHandler(service *services.LicenseGroupService) *LicenseGroupHandler {
	return &LicenseGroupHandler{service: service}
}

// ListGroups 查询分组列表
func (h *LicenseGroupHandler) ListGroups(c *gin.Context) {
	items, err := h.service.ListGroups()
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "QUERY_FAILED", err.Error())
		return
	}
	utils.Success(c, items, "ok")
}

// GetGroup 获取分组详情
func (h *LicenseGroupHandler) GetGroup(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "id 无效")
		return
	}
	item, err := h.service.GetGroup(uint(id))
	if err != nil {
		utils.Error(c, http.StatusNotFound, "NOT_FOUND", "分组不存在")
		return
	}
	utils.Success(c, item, "ok")
}

// CreateGroup 创建分组
func (h *LicenseGroupHandler) CreateGroup(c *gin.Context) {
	var req services.CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", err.Error())
		return
	}
	item, err := h.service.CreateGroup(&req, getAdminID(c))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "CREATE_FAILED", err.Error())
		return
	}
	utils.Success(c, item, "创建成功")
}

// UpdateGroup 更新分组
func (h *LicenseGroupHandler) UpdateGroup(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "id 无效")
		return
	}
	var req services.CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", err.Error())
		return
	}
	item, err := h.service.UpdateGroup(uint(id), &req)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "UPDATE_FAILED", err.Error())
		return
	}
	utils.Success(c, item, "更新成功")
}

// DeleteGroup 删除分组
func (h *LicenseGroupHandler) DeleteGroup(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "id 无效")
		return
	}
	if err := h.service.DeleteGroup(uint(id)); err != nil {
		utils.Error(c, http.StatusBadRequest, "DELETE_FAILED", err.Error())
		return
	}
	utils.Success(c, gin.H{"id": id}, "删除成功")
}

// AddLicensesToGroup 添加授权码到分组
func (h *LicenseGroupHandler) AddLicensesToGroup(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "id 无效")
		return
	}
	var req struct {
		LicenseIDs []uint `json:"license_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", err.Error())
		return
	}
	if err := h.service.AddLicensesToGroup(uint(id), req.LicenseIDs); err != nil {
		utils.Error(c, http.StatusBadRequest, "ADD_FAILED", err.Error())
		return
	}
	utils.Success(c, gin.H{"group_id": id, "count": len(req.LicenseIDs)}, "添加成功")
}

// RemoveLicensesFromGroup 从分组移除授权码
func (h *LicenseGroupHandler) RemoveLicensesFromGroup(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "id 无效")
		return
	}
	var req struct {
		LicenseIDs []uint `json:"license_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", err.Error())
		return
	}
	if err := h.service.RemoveLicensesFromGroup(uint(id), req.LicenseIDs); err != nil {
		utils.Error(c, http.StatusBadRequest, "REMOVE_FAILED", err.Error())
		return
	}
	utils.Success(c, gin.H{"group_id": id, "count": len(req.LicenseIDs)}, "移除成功")
}

// GetGroupLicenses 获取分组内的授权码
func (h *LicenseGroupHandler) GetGroupLicenses(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "id 无效")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	result, err := h.service.GetGroupLicenses(uint(id), page, pageSize)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "QUERY_FAILED", err.Error())
		return
	}
	utils.Success(c, result, "ok")
}

// GetGroupStats 获取分组统计信息
func (h *LicenseGroupHandler) GetGroupStats(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "id 无效")
		return
	}
	stats, err := h.service.GetGroupStats(uint(id))
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "QUERY_FAILED", err.Error())
		return
	}
	utils.Success(c, stats, "ok")
}

// GetLicenseGroups 获取授权码所属的分组列表
func (h *LicenseGroupHandler) GetLicenseGroups(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "id 无效")
		return
	}
	groups, err := h.service.GetLicenseGroups(uint(id))
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "QUERY_FAILED", err.Error())
		return
	}
	utils.Success(c, groups, "ok")
}
