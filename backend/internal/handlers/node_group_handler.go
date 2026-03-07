package handlers

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"nodepass-pro/backend/internal/services"
	"nodepass-pro/backend/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// NodeGroupHandler 节点分组处理器。
type NodeGroupHandler struct {
	service *services.NodeGroupService
}

// NewNodeGroupHandler 创建节点分组处理器。
func NewNodeGroupHandler(db *gorm.DB) *NodeGroupHandler {
	return &NodeGroupHandler{service: services.NewNodeGroupService(db)}
}

// Create POST /api/v1/node-groups
func (h *NodeGroupHandler) Create(c *gin.Context) {
	userID := getNodeGroupUserID(c)
	if userID == 0 {
		utils.ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	var req services.CreateNodeGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	group, err := h.service.Create(userID, &req)
	if err != nil {
		writeNodeGroupServiceError(c, err, "CREATE_NODE_GROUP_FAILED")
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, group, "节点组创建成功")
}

// List GET /api/v1/node-groups
func (h *NodeGroupHandler) List(c *gin.Context) {
	userID := getNodeGroupUserID(c)
	if userID == 0 {
		utils.ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	params := &services.ListNodeGroupParams{
		Page:     1,
		PageSize: 20,
	}

	if raw := strings.TrimSpace(c.Query("type")); raw != "" {
		params.Type = &raw
	}

	enabledRaw := strings.TrimSpace(c.Query("enabled"))
	if enabledRaw == "" {
		enabledRaw = strings.TrimSpace(c.Query("is_enabled"))
	}
	if enabledRaw != "" {
		enabled, err := strconv.ParseBool(enabledRaw)
		if err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", "enabled 参数错误")
			return
		}
		params.IsEnabled = &enabled
	}

	if raw := strings.TrimSpace(c.Query("page")); raw != "" {
		page, err := strconv.Atoi(raw)
		if err != nil || page <= 0 {
			utils.ErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", "page 参数错误")
			return
		}
		params.Page = page
	}

	pageSizeRaw := strings.TrimSpace(c.Query("page_size"))
	if pageSizeRaw == "" {
		pageSizeRaw = strings.TrimSpace(c.Query("pageSize"))
	}
	if pageSizeRaw != "" {
		pageSize, err := strconv.Atoi(pageSizeRaw)
		if err != nil || pageSize <= 0 {
			utils.ErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", "page_size 参数错误")
			return
		}
		params.PageSize = pageSize
	}

	items, total, err := h.service.List(userID, params)
	if err != nil {
		writeNodeGroupServiceError(c, err, "LIST_NODE_GROUPS_FAILED")
		return
	}

	utils.SuccessResponse(c, http.StatusOK, gin.H{
		"items":     items,
		"total":     total,
		"page":      params.Page,
		"page_size": params.PageSize,
	}, "获取节点组列表成功")
}

// Get GET /api/v1/node-groups/:id
func (h *NodeGroupHandler) Get(c *gin.Context) {
	userID := getNodeGroupUserID(c)
	if userID == 0 {
		utils.ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	id, ok := parseNodeGroupID(c)
	if !ok {
		return
	}

	group, err := h.service.Get(userID, id)
	if err != nil {
		writeNodeGroupServiceError(c, err, "GET_NODE_GROUP_FAILED")
		return
	}

	utils.SuccessResponse(c, http.StatusOK, group, "获取节点组成功")
}

// Update PUT /api/v1/node-groups/:id
func (h *NodeGroupHandler) Update(c *gin.Context) {
	userID := getNodeGroupUserID(c)
	if userID == 0 {
		utils.ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	id, ok := parseNodeGroupID(c)
	if !ok {
		return
	}

	var req services.UpdateNodeGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	group, err := h.service.Update(userID, id, &req)
	if err != nil {
		writeNodeGroupServiceError(c, err, "UPDATE_NODE_GROUP_FAILED")
		return
	}

	utils.SuccessResponse(c, http.StatusOK, group, "节点组更新成功")
}

// Delete DELETE /api/v1/node-groups/:id
func (h *NodeGroupHandler) Delete(c *gin.Context) {
	userID := getNodeGroupUserID(c)
	if userID == 0 {
		utils.ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	id, ok := parseNodeGroupID(c)
	if !ok {
		return
	}

	if err := h.service.Delete(userID, id); err != nil {
		writeNodeGroupServiceError(c, err, "DELETE_NODE_GROUP_FAILED")
		return
	}

	utils.SuccessResponse(c, http.StatusOK, nil, "节点组删除成功")
}

// Toggle POST /api/v1/node-groups/:id/toggle
func (h *NodeGroupHandler) Toggle(c *gin.Context) {
	userID := getNodeGroupUserID(c)
	if userID == 0 {
		utils.ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	id, ok := parseNodeGroupID(c)
	if !ok {
		return
	}

	group, err := h.service.Toggle(userID, id)
	if err != nil {
		writeNodeGroupServiceError(c, err, "TOGGLE_NODE_GROUP_FAILED")
		return
	}

	utils.SuccessResponse(c, http.StatusOK, group, "节点组状态切换成功")
}

// GetStats GET /api/v1/node-groups/:id/stats
func (h *NodeGroupHandler) GetStats(c *gin.Context) {
	userID := getNodeGroupUserID(c)
	if userID == 0 {
		utils.ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	id, ok := parseNodeGroupID(c)
	if !ok {
		return
	}

	stats, err := h.service.GetStats(userID, id)
	if err != nil {
		writeNodeGroupServiceError(c, err, "GET_NODE_GROUP_STATS_FAILED")
		return
	}

	utils.SuccessResponse(c, http.StatusOK, stats, "获取节点组统计成功")
}

// GenerateDeployCommand POST /api/v1/node-groups/:id/generate-deploy-command
func (h *NodeGroupHandler) GenerateDeployCommand(c *gin.Context) {
	userID := getNodeGroupUserID(c)
	if userID == 0 {
		utils.ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	id, ok := parseNodeGroupID(c)
	if !ok {
		return
	}

	req := &services.DeployNodeRequest{}
	if err := c.ShouldBindJSON(req); err != nil && !errors.Is(err, io.EOF) {
		utils.ErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	result, err := h.service.GenerateDeployCommand(userID, id, req)
	if err != nil {
		writeNodeGroupServiceError(c, err, "GENERATE_DEPLOY_COMMAND_FAILED")
		return
	}

	utils.SuccessResponse(c, http.StatusOK, result, "部署命令生成成功")
}

// ListNodes GET /api/v1/node-groups/:id/nodes
func (h *NodeGroupHandler) ListNodes(c *gin.Context) {
	userID := getNodeGroupUserID(c)
	if userID == 0 {
		utils.ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	id, ok := parseNodeGroupID(c)
	if !ok {
		return
	}

	items, err := h.service.ListNodes(userID, id)
	if err != nil {
		writeNodeGroupServiceError(c, err, "LIST_NODE_GROUP_NODES_FAILED")
		return
	}

	utils.SuccessResponse(c, http.StatusOK, items, "获取节点实例列表成功")
}

// AddNode POST /api/v1/node-groups/:id/nodes
func (h *NodeGroupHandler) AddNode(c *gin.Context) {
	userID := getNodeGroupUserID(c)
	if userID == 0 {
		utils.ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	id, ok := parseNodeGroupID(c)
	if !ok {
		return
	}

	var req services.AddNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	item, err := h.service.AddNode(userID, id, &req)
	if err != nil {
		writeNodeGroupServiceError(c, err, "ADD_NODE_GROUP_NODE_FAILED")
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, item, "节点实例创建成功")
}

// CreateRelation POST /api/v1/node-groups/:id/relations
func (h *NodeGroupHandler) CreateRelation(c *gin.Context) {
	userID := getNodeGroupUserID(c)
	if userID == 0 {
		utils.ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	entryGroupID, ok := parseNodeGroupID(c)
	if !ok {
		return
	}

	var req struct {
		ExitGroupID uint `json:"exit_group_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}
	if req.ExitGroupID == 0 {
		utils.ErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", "exit_group_id 必须大于 0")
		return
	}

	relation, err := h.service.CreateRelation(userID, entryGroupID, req.ExitGroupID)
	if err != nil {
		writeNodeGroupServiceError(c, err, "CREATE_NODE_GROUP_RELATION_FAILED")
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, relation, "节点组关联创建成功")
}

// ListRelations GET /api/v1/node-groups/:id/relations
func (h *NodeGroupHandler) ListRelations(c *gin.Context) {
	userID := getNodeGroupUserID(c)
	if userID == 0 {
		utils.ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	groupID, ok := parseNodeGroupID(c)
	if !ok {
		return
	}

	items, err := h.service.ListRelations(userID, groupID)
	if err != nil {
		writeNodeGroupServiceError(c, err, "LIST_NODE_GROUP_RELATIONS_FAILED")
		return
	}

	utils.SuccessResponse(c, http.StatusOK, items, "获取节点组关联列表成功")
}

// DeleteRelation DELETE /api/v1/node-group-relations/:id
func (h *NodeGroupHandler) DeleteRelation(c *gin.Context) {
	userID := getNodeGroupUserID(c)
	if userID == 0 {
		utils.ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	relationID, ok := parseNodeGroupID(c)
	if !ok {
		return
	}

	if err := h.service.DeleteRelation(userID, relationID); err != nil {
		writeNodeGroupServiceError(c, err, "DELETE_NODE_GROUP_RELATION_FAILED")
		return
	}

	utils.SuccessResponse(c, http.StatusOK, nil, "节点组关联删除成功")
}

// ToggleRelation POST /api/v1/node-group-relations/:id/toggle
func (h *NodeGroupHandler) ToggleRelation(c *gin.Context) {
	userID := getNodeGroupUserID(c)
	if userID == 0 {
		utils.ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	relationID, ok := parseNodeGroupID(c)
	if !ok {
		return
	}

	if err := h.service.ToggleRelation(userID, relationID); err != nil {
		writeNodeGroupServiceError(c, err, "TOGGLE_NODE_GROUP_RELATION_FAILED")
		return
	}

	utils.SuccessResponse(c, http.StatusOK, gin.H{"id": relationID}, "节点组关联状态切换成功")
}

func getNodeGroupUserID(c *gin.Context) uint {
	userID := c.GetUint("user_id")
	if userID > 0 {
		return userID
	}
	// 兼容当前中间件的 userID 键。
	return c.GetUint("userID")
}

func parseNodeGroupID(c *gin.Context) (uint, bool) {
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

func writeNodeGroupServiceError(c *gin.Context, err error, defaultCode string) {
	switch {
	case errors.Is(err, services.ErrUnauthorized):
		utils.ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
	case errors.Is(err, services.ErrInvalidParams):
		utils.ErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	case errors.Is(err, services.ErrForbidden):
		utils.ErrorResponse(c, http.StatusForbidden, "FORBIDDEN", err.Error())
	case errors.Is(err, services.ErrNotFound):
		utils.ErrorResponse(c, http.StatusNotFound, "NOT_FOUND", err.Error())
	case errors.Is(err, services.ErrConflict):
		utils.ErrorResponse(c, http.StatusConflict, "CONFLICT", err.Error())
	case errors.Is(err, services.ErrQuotaExceeded):
		utils.ErrorResponse(c, http.StatusBadRequest, "QUOTA_EXCEEDED", err.Error())
	default:
		utils.ErrorResponse(c, http.StatusInternalServerError, defaultCode, "服务器内部错误")
	}
}
