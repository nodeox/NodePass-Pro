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

// NodeInstanceHandler 节点实例处理器。
type NodeInstanceHandler struct {
	service *services.NodeInstanceService
}

// NewNodeInstanceHandler 创建节点实例处理器。
func NewNodeInstanceHandler(db *gorm.DB) *NodeInstanceHandler {
	return &NodeInstanceHandler{service: services.NewNodeInstanceService(db)}
}

// Get GET /api/v1/node-instances/:id
func (h *NodeInstanceHandler) Get(c *gin.Context) {
	userID := getNodeInstanceUserID(c)
	if userID == 0 {
		utils.ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	id, ok := parseNodeInstanceID(c)
	if !ok {
		return
	}

	instance, err := h.service.Get(userID, id)
	if err != nil {
		writeServiceError(c, err, "GET_NODE_INSTANCE_FAILED")
		return
	}

	utils.SuccessResponse(c, http.StatusOK, instance, "获取节点实例成功")
}

// Update PUT /api/v1/node-instances/:id
func (h *NodeInstanceHandler) Update(c *gin.Context) {
	userID := getNodeInstanceUserID(c)
	if userID == 0 {
		utils.ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	id, ok := parseNodeInstanceID(c)
	if !ok {
		return
	}

	var req services.UpdateNodeInstanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	instance, err := h.service.Update(userID, id, &req)
	if err != nil {
		writeServiceError(c, err, "UPDATE_NODE_INSTANCE_FAILED")
		return
	}

	utils.SuccessResponse(c, http.StatusOK, instance, "节点实例更新成功")
}

// Delete DELETE /api/v1/node-instances/:id
func (h *NodeInstanceHandler) Delete(c *gin.Context) {
	userID := getNodeInstanceUserID(c)
	if userID == 0 {
		utils.ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	id, ok := parseNodeInstanceID(c)
	if !ok {
		return
	}

	if err := h.service.Delete(userID, id); err != nil {
		writeServiceError(c, err, "DELETE_NODE_INSTANCE_FAILED")
		return
	}

	utils.SuccessResponse(c, http.StatusOK, nil, "节点实例删除成功")
}

// Restart POST /api/v1/node-instances/:id/restart
func (h *NodeInstanceHandler) Restart(c *gin.Context) {
	userID := getNodeInstanceUserID(c)
	if userID == 0 {
		utils.ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	id, ok := parseNodeInstanceID(c)
	if !ok {
		return
	}

	instance, err := h.service.Restart(userID, id)
	if err != nil {
		writeServiceError(c, err, "RESTART_NODE_INSTANCE_FAILED")
		return
	}

	utils.SuccessResponse(c, http.StatusOK, instance, "节点实例已重启")
}

// Heartbeat POST /api/v1/node-instances/heartbeat
// 注意：此接口应注册在公开路由，不需要 JWT。
func (h *NodeInstanceHandler) Heartbeat(c *gin.Context) {
	var req services.HeartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	resp, err := h.service.Heartbeat(&req)
	if err != nil {
		writeServiceError(c, err, "NODE_INSTANCE_HEARTBEAT_FAILED")
		return
	}

	utils.SuccessResponse(c, http.StatusOK, resp, "心跳上报成功")
}

func getNodeInstanceUserID(c *gin.Context) uint {
	userID := c.GetUint("user_id")
	if userID > 0 {
		return userID
	}
	// 兼容当前中间件设置的 userID 键。
	return c.GetUint("userID")
}

func parseNodeInstanceID(c *gin.Context) (uint, bool) {
	rawID := strings.TrimSpace(c.Param("id"))
	if rawID == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", "id 不能为空")
		return 0, false
	}
	parsed, err := strconv.ParseUint(rawID, 10, 64)
	if err != nil || parsed == 0 {
		utils.ErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", "id 参数无效")
		return 0, false
	}
	return uint(parsed), true
}
