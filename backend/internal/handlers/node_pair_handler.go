package handlers

import (
	"net/http"

	"nodepass-panel/backend/internal/services"
	"nodepass-panel/backend/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// NodePairHandler 节点配对处理器。
type NodePairHandler struct {
	nodePairService *services.NodePairService
}

// NewNodePairHandler 创建节点配对处理器。
func NewNodePairHandler(db *gorm.DB) *NodePairHandler {
	return &NodePairHandler{
		nodePairService: services.NewNodePairService(db),
	}
}

// CreatePair POST /api/v1/node-pairs
func (h *NodePairHandler) CreatePair(c *gin.Context) {
	userID, role, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	type requestPayload struct {
		services.CreateNodePairRequest
		UserID *uint `json:"user_id"`
	}

	var payload requestPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	targetUserID := userID
	if payload.UserID != nil {
		if !isAdminRole(role) {
			utils.Error(c, http.StatusForbidden, "FORBIDDEN", "普通用户不能指定 user_id")
			return
		}
		targetUserID = *payload.UserID
	}
	if targetUserID == 0 {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "user_id 无效")
		return
	}

	pair, err := h.nodePairService.CreatePair(targetUserID, &payload.CreateNodePairRequest)
	if err != nil {
		writeServiceError(c, err, "CREATE_PAIR_FAILED")
		return
	}

	utils.SuccessResponse(c, pair, "节点配对创建成功")
}

// ListPairs GET /api/v1/node-pairs
func (h *NodePairHandler) ListPairs(c *gin.Context) {
	userID, role, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	scopeUserID := userID
	if isAdminRole(role) {
		scopeUserID = 0
		if queryUserID, err := parseUintQuery(c, "user_id"); err == nil && queryUserID != nil {
			scopeUserID = *queryUserID
		} else if err != nil {
			utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "user_id 参数错误")
			return
		}
	}

	pairs, err := h.nodePairService.ListPairs(scopeUserID)
	if err != nil {
		writeServiceError(c, err, "LIST_PAIRS_FAILED")
		return
	}

	utils.Success(c, gin.H{
		"list":  pairs,
		"total": len(pairs),
	})
}

// UpdatePair PUT /api/v1/node-pairs/:id
func (h *NodePairHandler) UpdatePair(c *gin.Context) {
	userID, role, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	pairID, ok := parseUintParam(c, "id")
	if !ok {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "节点配对 ID 无效")
		return
	}

	var req services.UpdateNodePairRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	scopeUserID := userID
	if isAdminRole(role) {
		scopeUserID = 0
	}

	pair, err := h.nodePairService.UpdatePair(scopeUserID, pairID, &req)
	if err != nil {
		writeServiceError(c, err, "UPDATE_PAIR_FAILED")
		return
	}

	utils.SuccessResponse(c, pair, "节点配对更新成功")
}

// DeletePair DELETE /api/v1/node-pairs/:id
func (h *NodePairHandler) DeletePair(c *gin.Context) {
	userID, role, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	pairID, ok := parseUintParam(c, "id")
	if !ok {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "节点配对 ID 无效")
		return
	}

	scopeUserID := userID
	if isAdminRole(role) {
		scopeUserID = 0
	}

	if err := h.nodePairService.DeletePair(scopeUserID, pairID); err != nil {
		writeServiceError(c, err, "DELETE_PAIR_FAILED")
		return
	}

	utils.SuccessResponse(c, nil, "节点配对删除成功")
}

// TogglePair PUT /api/v1/node-pairs/:id/toggle
func (h *NodePairHandler) TogglePair(c *gin.Context) {
	userID, role, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	pairID, ok := parseUintParam(c, "id")
	if !ok {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "节点配对 ID 无效")
		return
	}

	scopeUserID := userID
	if isAdminRole(role) {
		scopeUserID = 0
	}

	pair, err := h.nodePairService.TogglePair(scopeUserID, pairID)
	if err != nil {
		writeServiceError(c, err, "TOGGLE_PAIR_FAILED")
		return
	}

	utils.Success(c, pair)
}
