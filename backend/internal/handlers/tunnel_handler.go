package handlers

import (
	"errors"
	"net/http"
	"strings"

	"nodepass-pro/backend/internal/models"
	"nodepass-pro/backend/internal/services"
	"nodepass-pro/backend/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// TunnelHandler 隧道处理器。
type TunnelHandler struct {
	service *services.TunnelService
}

// NewTunnelHandler 创建隧道处理器。
func NewTunnelHandler(db *gorm.DB) *TunnelHandler {
	return &TunnelHandler{service: services.NewTunnelService(db)}
}

// Create POST /api/v1/tunnels
func (h *TunnelHandler) Create(c *gin.Context) {
	userID, role, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	type payload struct {
		services.CreateTunnelRequest
		UserID *uint `json:"user_id"`
	}
	var req payload
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	// 普通用户只能为自己创建隧道
	targetUserID := userID
	if req.UserID != nil && isAdminRole(role) {
		// 管理员可以为指定用户创建隧道
		targetUserID = *req.UserID
	}

	tunnel, err := h.service.Create(targetUserID, &req.CreateTunnelRequest)
	if err != nil {
		writeTunnelServiceError(c, err, "CREATE_TUNNEL_FAILED")
		return
	}
	utils.SuccessResponse(c, tunnel, "隧道创建成功")
}

// List GET /api/v1/tunnels
func (h *TunnelHandler) List(c *gin.Context) {
	userID, role, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	page, err := parsePositiveIntQuery(c, "page", 1)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "page 参数错误")
		return
	}
	// 兼容前端发送的 page_size 和 pageSize 两种参数名
	pageSizeRawKey := "page_size"
	if strings.TrimSpace(c.Query("page_size")) == "" && strings.TrimSpace(c.Query("pageSize")) != "" {
		pageSizeRawKey = "pageSize"
	}
	pageSize, err := parsePositiveIntQuery(c, pageSizeRawKey, 20)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "page_size 参数错误")
		return
	}

	params := &services.ListTunnelParams{
		Page:     page,
		PageSize: pageSize,
	}
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		params.Status = &status
	}

	scopeUserID := userID
	if isAdminRole(role) {
		scopeUserID = 0
		queryUserID, parseErr := parseUintQuery(c, "user_id")
		if parseErr != nil {
			utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "user_id 参数错误")
			return
		}
		if queryUserID != nil {
			scopeUserID = *queryUserID
		}
	}

	items, total, err := h.service.List(scopeUserID, params)
	if err != nil {
		writeTunnelServiceError(c, err, "LIST_TUNNELS_FAILED")
		return
	}
	utils.SuccessResponse(c, http.StatusOK, gin.H{
		"items":     items,
		"total":     total,
		"page":      params.Page,
		"page_size": params.PageSize,
	}, "获取隧道列表成功")
}

// Get GET /api/v1/tunnels/:id
func (h *TunnelHandler) Get(c *gin.Context) {
	userID, role, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}
	tunnelID, ok := parseUintParam(c, "id")
	if !ok {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "隧道 ID 无效")
		return
	}
	scopeUserID := userID
	if isAdminRole(role) {
		scopeUserID = 0
	}

	tunnel, err := h.service.Get(scopeUserID, tunnelID)
	if err != nil {
		writeTunnelServiceError(c, err, "GET_TUNNEL_FAILED")
		return
	}
	utils.Success(c, tunnel)
}

// Update PUT /api/v1/tunnels/:id
func (h *TunnelHandler) Update(c *gin.Context) {
	userID, role, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}
	tunnelID, ok := parseUintParam(c, "id")
	if !ok {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "隧道 ID 无效")
		return
	}

	var req services.UpdateTunnelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	scopeUserID := userID
	if isAdminRole(role) {
		scopeUserID = 0
	}

	tunnel, err := h.service.Update(scopeUserID, tunnelID, &req)
	if err != nil {
		writeTunnelServiceError(c, err, "UPDATE_TUNNEL_FAILED")
		return
	}
	utils.SuccessResponse(c, tunnel, "隧道更新成功")
}

// Delete DELETE /api/v1/tunnels/:id
func (h *TunnelHandler) Delete(c *gin.Context) {
	userID, role, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}
	tunnelID, ok := parseUintParam(c, "id")
	if !ok {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "隧道 ID 无效")
		return
	}
	scopeUserID := userID
	if isAdminRole(role) {
		scopeUserID = 0
	}

	if err := h.service.Delete(scopeUserID, tunnelID); err != nil {
		writeTunnelServiceError(c, err, "DELETE_TUNNEL_FAILED")
		return
	}
	utils.SuccessResponse(c, nil, "隧道删除成功")
}

// Start POST /api/v1/tunnels/:id/start
func (h *TunnelHandler) Start(c *gin.Context) {
	h.changeStatus(c, "start")
}

// Stop POST /api/v1/tunnels/:id/stop
func (h *TunnelHandler) Stop(c *gin.Context) {
	h.changeStatus(c, "stop")
}

func (h *TunnelHandler) changeStatus(c *gin.Context, action string) {
	userID, role, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}
	tunnelID, ok := parseUintParam(c, "id")
	if !ok {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "隧道 ID 无效")
		return
	}
	scopeUserID := userID
	if isAdminRole(role) {
		scopeUserID = 0
	}

	var (
		tunnel *models.Tunnel
		err    error
	)
	switch action {
	case "start":
		err = h.service.Start(scopeUserID, tunnelID)
	case "stop":
		err = h.service.Stop(scopeUserID, tunnelID)
	default:
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "不支持的动作")
		return
	}
	if err != nil {
		writeTunnelServiceError(c, err, "CHANGE_TUNNEL_STATUS_FAILED")
		return
	}
	tunnel, err = h.service.Get(scopeUserID, tunnelID)
	if err != nil {
		writeServiceError(c, err, "GET_TUNNEL_FAILED")
		return
	}
	utils.SuccessResponse(c, tunnel, "隧道状态更新成功")
}

func writeTunnelServiceError(c *gin.Context, err error, defaultCode string) {
	message := cleanTunnelServiceErrorMessage(err)
	switch {
	case errors.Is(err, services.ErrUnauthorized):
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", message)
	case errors.Is(err, services.ErrInvalidParams):
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", message)
	case errors.Is(err, services.ErrForbidden):
		utils.Error(c, http.StatusForbidden, "FORBIDDEN", message)
	case errors.Is(err, services.ErrNotFound):
		utils.Error(c, http.StatusNotFound, "NOT_FOUND", message)
	case errors.Is(err, services.ErrConflict):
		utils.Error(c, http.StatusConflict, "CONFLICT", message)
	default:
		writeServiceError(c, err, defaultCode)
	}
}

func cleanTunnelServiceErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	message := strings.TrimSpace(err.Error())
	message = strings.TrimPrefix(message, "invalid params: ")
	message = strings.TrimPrefix(message, "not found: ")
	message = strings.TrimPrefix(message, "forbidden: ")
	message = strings.TrimPrefix(message, "unauthorized: ")
	message = strings.TrimPrefix(message, "conflict: ")
	message = strings.TrimSpace(message)
	if message == "" {
		return "请求处理失败"
	}
	return message
}
