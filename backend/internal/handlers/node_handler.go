package handlers

import (
	"net/http"
	"strings"

	"nodepass-panel/backend/internal/services"
	"nodepass-panel/backend/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// NodeHandler 节点管理处理器。
type NodeHandler struct {
	nodeService *services.NodeService
}

// NewNodeHandler 创建节点处理器。
func NewNodeHandler(db *gorm.DB) *NodeHandler {
	return &NodeHandler{
		nodeService: services.NewNodeService(db),
	}
}

// CreateNode POST /api/v1/nodes
func (h *NodeHandler) CreateNode(c *gin.Context) {
	userID, role, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	type requestPayload struct {
		services.CreateNodeRequest
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

	payload.PanelURL = strings.TrimSpace(payload.PanelURL)
	if payload.PanelURL == "" {
		payload.PanelURL = inferPanelURL(c)
	}
	if strings.TrimSpace(payload.HubURL) == "" {
		payload.HubURL = payload.PanelURL
	}

	result, err := h.nodeService.CreateNode(targetUserID, &payload.CreateNodeRequest)
	if err != nil {
		writeServiceError(c, err, "CREATE_NODE_FAILED")
		return
	}

	utils.SuccessResponse(c, result, "节点创建成功")
}

// ListNodes GET /api/v1/nodes
func (h *NodeHandler) ListNodes(c *gin.Context) {
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
	pageSize, err := parsePositiveIntQuery(c, "pageSize", 20)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "pageSize 参数错误")
		return
	}
	isSelfHosted, err := parseOptionalBoolQuery(c, "is_self_hosted")
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "is_self_hosted 参数错误")
		return
	}
	isPublic, err := parseOptionalBoolQuery(c, "is_public")
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "is_public 参数错误")
		return
	}

	filters := services.ListNodeFilters{
		Status:       strings.TrimSpace(c.Query("status")),
		IsSelfHosted: isSelfHosted,
		IsPublic:     isPublic,
		Region:       strings.TrimSpace(c.Query("region")),
		Page:         page,
		PageSize:     pageSize,
	}

	scopeUserID := userID
	if isAdminRole(role) {
		scopeUserID = 0
		queryUserID, parseErr := parseUintQuery(c, "user_id")
		if parseErr != nil {
			utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "user_id 参数错误")
			return
		}
		filters.UserID = queryUserID
	}

	result, err := h.nodeService.ListNodes(scopeUserID, filters)
	if err != nil {
		writeServiceError(c, err, "LIST_NODES_FAILED")
		return
	}

	utils.Success(c, result)
}

// GetNode GET /api/v1/nodes/:id
func (h *NodeHandler) GetNode(c *gin.Context) {
	userID, role, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	nodeID, ok := parseUintParam(c, "id")
	if !ok {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "节点 ID 无效")
		return
	}

	scopeUserID := userID
	if isAdminRole(role) {
		scopeUserID = 0
	}

	node, err := h.nodeService.GetNode(scopeUserID, nodeID)
	if err != nil {
		writeServiceError(c, err, "GET_NODE_FAILED")
		return
	}

	utils.Success(c, node)
}

// UpdateNode PUT /api/v1/nodes/:id
func (h *NodeHandler) UpdateNode(c *gin.Context) {
	userID, role, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	nodeID, ok := parseUintParam(c, "id")
	if !ok {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "节点 ID 无效")
		return
	}

	var req services.UpdateNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	scopeUserID := userID
	if isAdminRole(role) {
		scopeUserID = 0
	}

	node, err := h.nodeService.UpdateNode(scopeUserID, nodeID, &req)
	if err != nil {
		writeServiceError(c, err, "UPDATE_NODE_FAILED")
		return
	}

	utils.SuccessResponse(c, node, "节点更新成功")
}

// DeleteNode DELETE /api/v1/nodes/:id
func (h *NodeHandler) DeleteNode(c *gin.Context) {
	userID, role, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	nodeID, ok := parseUintParam(c, "id")
	if !ok {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "节点 ID 无效")
		return
	}

	scopeUserID := userID
	if isAdminRole(role) {
		scopeUserID = 0
	}

	if err := h.nodeService.DeleteNode(scopeUserID, nodeID); err != nil {
		writeServiceError(c, err, "DELETE_NODE_FAILED")
		return
	}

	utils.SuccessResponse(c, nil, "节点删除成功")
}

// GetQuota GET /api/v1/nodes/quota
func (h *NodeHandler) GetQuota(c *gin.Context) {
	userID, role, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	targetUserID := userID
	if isAdminRole(role) {
		queryUserID, err := parseUintQuery(c, "user_id")
		if err != nil {
			utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "user_id 参数错误")
			return
		}
		if queryUserID != nil {
			targetUserID = *queryUserID
		}
	}

	quota, err := h.nodeService.GetQuota(targetUserID)
	if err != nil {
		writeServiceError(c, err, "GET_QUOTA_FAILED")
		return
	}

	utils.Success(c, quota)
}
