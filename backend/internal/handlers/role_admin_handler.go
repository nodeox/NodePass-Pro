package handlers

import (
	"net/http"

	"nodepass-pro/backend/internal/services"
	"nodepass-pro/backend/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RoleAdminHandler 角色管理处理器（管理员）。
type RoleAdminHandler struct {
	roleAdminService *services.RoleAdminService
}

// NewRoleAdminHandler 创建角色管理处理器。
func NewRoleAdminHandler(db *gorm.DB) *RoleAdminHandler {
	return &RoleAdminHandler{
		roleAdminService: services.NewRoleAdminService(db),
	}
}

// ListRoles GET /api/v1/roles
func (h *RoleAdminHandler) ListRoles(c *gin.Context) {
	adminUserID, role, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}
	if !isAdminRole(role) {
		utils.Error(c, http.StatusForbidden, "FORBIDDEN", "仅管理员可执行此操作")
		return
	}

	includeDisabled, err := parseOptionalBoolQuery(c, "include_disabled")
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "include_disabled 参数错误")
		return
	}

	items, err := h.roleAdminService.ListRoles(
		adminUserID,
		includeDisabled != nil && *includeDisabled,
		c.Query("keyword"),
	)
	if err != nil {
		writeServiceError(c, err, "LIST_ROLES_FAILED")
		return
	}

	utils.Success(c, gin.H{
		"list":  items,
		"total": len(items),
	})
}

// GetRole GET /api/v1/roles/:id
func (h *RoleAdminHandler) GetRole(c *gin.Context) {
	adminUserID, role, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}
	if !isAdminRole(role) {
		utils.Error(c, http.StatusForbidden, "FORBIDDEN", "仅管理员可执行此操作")
		return
	}

	roleID, ok := parseUintParam(c, "id")
	if !ok {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "角色 ID 无效")
		return
	}

	item, err := h.roleAdminService.GetRole(adminUserID, roleID)
	if err != nil {
		writeServiceError(c, err, "GET_ROLE_FAILED")
		return
	}

	utils.Success(c, item)
}

// CreateRole POST /api/v1/roles
func (h *RoleAdminHandler) CreateRole(c *gin.Context) {
	adminUserID, role, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}
	if !isAdminRole(role) {
		utils.Error(c, http.StatusForbidden, "FORBIDDEN", "仅管理员可执行此操作")
		return
	}

	type createRolePayload struct {
		Code        string   `json:"code" binding:"required"`
		Name        string   `json:"name" binding:"required"`
		Description *string  `json:"description"`
		IsEnabled   *bool    `json:"is_enabled"`
		Permissions []string `json:"permissions"`
	}

	var req createRolePayload
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	item, err := h.roleAdminService.CreateRole(adminUserID, services.CreateRolePayload{
		Code:        req.Code,
		Name:        req.Name,
		Description: req.Description,
		IsEnabled:   req.IsEnabled,
		Permissions: req.Permissions,
	})
	if err != nil {
		writeServiceError(c, err, "CREATE_ROLE_FAILED")
		return
	}

	utils.SuccessResponse(c, item, "角色创建成功")
}

// UpdateRole PUT /api/v1/roles/:id
func (h *RoleAdminHandler) UpdateRole(c *gin.Context) {
	adminUserID, role, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}
	if !isAdminRole(role) {
		utils.Error(c, http.StatusForbidden, "FORBIDDEN", "仅管理员可执行此操作")
		return
	}

	roleID, ok := parseUintParam(c, "id")
	if !ok {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "角色 ID 无效")
		return
	}

	type updateRolePayload struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		IsEnabled   *bool   `json:"is_enabled"`
	}

	var req updateRolePayload
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	item, err := h.roleAdminService.UpdateRole(adminUserID, roleID, services.UpdateRolePayload{
		Name:        req.Name,
		Description: req.Description,
		IsEnabled:   req.IsEnabled,
	})
	if err != nil {
		writeServiceError(c, err, "UPDATE_ROLE_FAILED")
		return
	}

	utils.SuccessResponse(c, item, "角色更新成功")
}

// UpdateRolePermissions PUT /api/v1/roles/:id/permissions
func (h *RoleAdminHandler) UpdateRolePermissions(c *gin.Context) {
	adminUserID, role, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}
	if !isAdminRole(role) {
		utils.Error(c, http.StatusForbidden, "FORBIDDEN", "仅管理员可执行此操作")
		return
	}

	roleID, ok := parseUintParam(c, "id")
	if !ok {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "角色 ID 无效")
		return
	}

	type updateRolePermissionsPayload struct {
		Permissions []string `json:"permissions"`
	}

	var req updateRolePermissionsPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	item, err := h.roleAdminService.UpdateRolePermissions(adminUserID, roleID, req.Permissions)
	if err != nil {
		writeServiceError(c, err, "UPDATE_ROLE_PERMISSIONS_FAILED")
		return
	}

	utils.SuccessResponse(c, item, "角色权限更新成功")
}

// DeleteRole DELETE /api/v1/roles/:id
func (h *RoleAdminHandler) DeleteRole(c *gin.Context) {
	adminUserID, role, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}
	if !isAdminRole(role) {
		utils.Error(c, http.StatusForbidden, "FORBIDDEN", "仅管理员可执行此操作")
		return
	}

	roleID, ok := parseUintParam(c, "id")
	if !ok {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "角色 ID 无效")
		return
	}

	if err := h.roleAdminService.DeleteRole(adminUserID, roleID); err != nil {
		writeServiceError(c, err, "DELETE_ROLE_FAILED")
		return
	}

	utils.SuccessResponse(c, nil, "角色删除成功")
}

// ListAvailablePermissions GET /api/v1/roles/permissions
func (h *RoleAdminHandler) ListAvailablePermissions(c *gin.Context) {
	adminUserID, role, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}
	if !isAdminRole(role) {
		utils.Error(c, http.StatusForbidden, "FORBIDDEN", "仅管理员可执行此操作")
		return
	}

	permissions, err := h.roleAdminService.ListAvailablePermissions(adminUserID)
	if err != nil {
		writeServiceError(c, err, "LIST_PERMISSIONS_FAILED")
		return
	}

	utils.Success(c, gin.H{
		"list":  permissions,
		"total": len(permissions),
	})
}
