package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"nodepass-pro/backend/internal/services"
	"nodepass-pro/backend/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// UserAdminHandler 用户管理处理器（管理员）。
type UserAdminHandler struct {
	userAdminService *services.UserAdminService
}

// NewUserAdminHandler 创建用户管理处理器。
func NewUserAdminHandler(db *gorm.DB) *UserAdminHandler {
	return &UserAdminHandler{
		userAdminService: services.NewUserAdminService(db),
	}
}

// ListUsers GET /api/v1/users (admin)
func (h *UserAdminHandler) ListUsers(c *gin.Context) {
	adminUserID, role, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}
	if !isAdminRole(role) {
		utils.Error(c, http.StatusForbidden, "FORBIDDEN", "仅管理员可执行此操作")
		return
	}

	page, err := parsePositiveIntQuery(c, "page", 1)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "page 参数错误")
		return
	}

	pageSizeRawKey := "pageSize"
	if strings.TrimSpace(c.Query("page_size")) != "" {
		pageSizeRawKey = "page_size"
	}
	pageSize, err := parsePositiveIntQuery(c, pageSizeRawKey, 20)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "pageSize 参数错误")
		return
	}

	vipLevel, err := parseOptionalInt(c.Query("vip_level"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "vip_level 参数错误")
		return
	}

	result, err := h.userAdminService.ListUsers(adminUserID, services.UserListFilters{
		Role:     c.Query("role"),
		Status:   c.Query("status"),
		VIPLevel: vipLevel,
		Keyword:  c.Query("keyword"),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		writeServiceError(c, err, "LIST_USERS_FAILED")
		return
	}

	utils.Success(c, result)
}

// UpdateRole PUT /api/v1/users/:id/role (admin)
func (h *UserAdminHandler) UpdateRole(c *gin.Context) {
	adminUserID, role, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}
	if !isAdminRole(role) {
		utils.Error(c, http.StatusForbidden, "FORBIDDEN", "仅管理员可执行此操作")
		return
	}

	targetUserID, ok := parseUintParam(c, "id")
	if !ok {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "目标用户 ID 无效")
		return
	}

	type rolePayload struct {
		Role string `json:"role" binding:"required"`
	}
	var req rolePayload
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	user, err := h.userAdminService.UpdateUserRole(adminUserID, targetUserID, req.Role)
	if err != nil {
		writeServiceError(c, err, "UPDATE_USER_ROLE_FAILED")
		return
	}

	utils.SuccessResponse(c, user, "用户角色更新成功")
}

// UpdateStatus PUT /api/v1/users/:id/status (admin)
func (h *UserAdminHandler) UpdateStatus(c *gin.Context) {
	adminUserID, role, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}
	if !isAdminRole(role) {
		utils.Error(c, http.StatusForbidden, "FORBIDDEN", "仅管理员可执行此操作")
		return
	}

	targetUserID, ok := parseUintParam(c, "id")
	if !ok {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "目标用户 ID 无效")
		return
	}

	type statusPayload struct {
		Status string `json:"status" binding:"required"`
	}
	var req statusPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	user, err := h.userAdminService.UpdateUserStatus(adminUserID, targetUserID, req.Status)
	if err != nil {
		writeServiceError(c, err, "UPDATE_USER_STATUS_FAILED")
		return
	}

	utils.SuccessResponse(c, user, "用户状态更新成功")
}

func parseOptionalInt(raw string) (*int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return nil, err
	}
	value := parsed
	return &value, nil
}
