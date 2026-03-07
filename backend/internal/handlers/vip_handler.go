package handlers

import (
	"net/http"

	"nodepass-pro/backend/internal/services"
	"nodepass-pro/backend/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// VIPHandler VIP 管理处理器。
type VIPHandler struct {
	vipService *services.VIPService
}

// NewVIPHandler 创建 VIP 处理器。
func NewVIPHandler(db *gorm.DB) *VIPHandler {
	return &VIPHandler{
		vipService: services.NewVIPService(db),
	}
}

// ListLevels GET /api/v1/vip/levels
func (h *VIPHandler) ListLevels(c *gin.Context) {
	levels, err := h.vipService.ListLevels()
	if err != nil {
		writeServiceError(c, err, "LIST_VIP_LEVELS_FAILED")
		return
	}
	utils.Success(c, gin.H{
		"list":  levels,
		"total": len(levels),
	})
}

// CreateLevel POST /api/v1/vip/levels (admin)
func (h *VIPHandler) CreateLevel(c *gin.Context) {
	adminID, role, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}
	if !isAdminRole(role) {
		utils.Error(c, http.StatusForbidden, "FORBIDDEN", "仅管理员可执行此操作")
		return
	}

	var req services.VIPLevelCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	level, err := h.vipService.CreateLevel(adminID, &req)
	if err != nil {
		writeServiceError(c, err, "CREATE_VIP_LEVEL_FAILED")
		return
	}
	utils.SuccessResponse(c, level, "VIP 等级创建成功")
}

// UpdateLevel PUT /api/v1/vip/levels/:id (admin)
func (h *VIPHandler) UpdateLevel(c *gin.Context) {
	adminID, role, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}
	if !isAdminRole(role) {
		utils.Error(c, http.StatusForbidden, "FORBIDDEN", "仅管理员可执行此操作")
		return
	}

	levelID, ok := parseUintParam(c, "id")
	if !ok {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "VIP 等级 ID 无效")
		return
	}

	var req services.VIPLevelUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	level, err := h.vipService.UpdateLevel(adminID, levelID, &req)
	if err != nil {
		writeServiceError(c, err, "UPDATE_VIP_LEVEL_FAILED")
		return
	}
	utils.SuccessResponse(c, level, "VIP 等级更新成功")
}

// GetMyLevel GET /api/v1/vip/my-level
func (h *VIPHandler) GetMyLevel(c *gin.Context) {
	userID, _, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	result, err := h.vipService.GetMyLevel(userID)
	if err != nil {
		writeServiceError(c, err, "GET_MY_VIP_LEVEL_FAILED")
		return
	}
	utils.Success(c, result)
}

// UpgradeUser POST /api/v1/users/:id/vip/upgrade (admin)
func (h *VIPHandler) UpgradeUser(c *gin.Context) {
	adminID, role, ok := getUserContext(c)
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

	type requestPayload struct {
		Level        int `json:"level" binding:"required"`
		DurationDays int `json:"duration_days" binding:"required"`
	}
	var req requestPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	user, err := h.vipService.UpgradeUser(adminID, targetUserID, req.Level, req.DurationDays)
	if err != nil {
		writeServiceError(c, err, "UPGRADE_USER_VIP_FAILED")
		return
	}
	utils.SuccessResponse(c, user, "用户 VIP 升级成功")
}
