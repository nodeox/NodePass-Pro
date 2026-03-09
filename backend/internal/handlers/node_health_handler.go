package handlers

import (
	"net/http"
	"strconv"
	"time"

	"nodepass-pro/backend/internal/models"
	"nodepass-pro/backend/internal/services"
	"nodepass-pro/backend/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// NodeHealthHandler 节点健康检查处理器
type NodeHealthHandler struct {
	service *services.NodeHealthService
}

// NewNodeHealthHandler 创建节点健康检查处理器
func NewNodeHealthHandler(db *gorm.DB) *NodeHealthHandler {
	return &NodeHealthHandler{
		service: services.NewNodeHealthService(db),
	}
}

// CreateHealthCheck POST /api/v1/node-instances/:id/health-check
func (h *NodeHealthHandler) CreateHealthCheck(c *gin.Context) {
	nodeInstanceID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAM", "无效的节点实例 ID")
		return
	}

	var req models.NodeHealthCheck
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	req.NodeInstanceID = uint(nodeInstanceID)

	if err := h.service.CreateHealthCheck(&req); err != nil {
		utils.Error(c, http.StatusInternalServerError, "CREATE_FAILED", "创建健康检查配置失败: "+err.Error())
		return
	}

	utils.SuccessResponse(c, req, "健康检查配置创建成功")
}

// GetHealthCheck GET /api/v1/node-instances/:id/health-check
func (h *NodeHealthHandler) GetHealthCheck(c *gin.Context) {
	nodeInstanceID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAM", "无效的节点实例 ID")
		return
	}

	check, err := h.service.GetHealthCheck(uint(nodeInstanceID))
	if err != nil {
		utils.Error(c, http.StatusNotFound, "NOT_FOUND", "健康检查配置不存在")
		return
	}

	utils.Success(c, check)
}

// UpdateHealthCheck PUT /api/v1/node-instances/:id/health-check
func (h *NodeHealthHandler) UpdateHealthCheck(c *gin.Context) {
	nodeInstanceID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAM", "无效的节点实例 ID")
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	if err := h.service.UpdateHealthCheck(uint(nodeInstanceID), updates); err != nil {
		utils.Error(c, http.StatusInternalServerError, "UPDATE_FAILED", "更新健康检查配置失败: "+err.Error())
		return
	}

	utils.SuccessResponse(c, nil, "健康检查配置更新成功")
}

// DeleteHealthCheck DELETE /api/v1/node-instances/:id/health-check
func (h *NodeHealthHandler) DeleteHealthCheck(c *gin.Context) {
	nodeInstanceID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAM", "无效的节点实例 ID")
		return
	}

	if err := h.service.DeleteHealthCheck(uint(nodeInstanceID)); err != nil {
		utils.Error(c, http.StatusInternalServerError, "DELETE_FAILED", "删除健康检查配置失败: "+err.Error())
		return
	}

	utils.SuccessResponse(c, nil, "健康检查配置删除成功")
}

// PerformHealthCheck POST /api/v1/node-instances/:id/health-check/perform
func (h *NodeHealthHandler) PerformHealthCheck(c *gin.Context) {
	nodeInstanceID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAM", "无效的节点实例 ID")
		return
	}

	record, err := h.service.PerformHealthCheck(uint(nodeInstanceID))
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "CHECK_FAILED", "执行健康检查失败: "+err.Error())
		return
	}

	utils.SuccessResponse(c, record, "健康检查完成")
}

// GetQualityScore GET /api/v1/node-instances/:id/quality-score
func (h *NodeHealthHandler) GetQualityScore(c *gin.Context) {
	nodeInstanceID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAM", "无效的节点实例 ID")
		return
	}

	score, err := h.service.GetQualityScore(uint(nodeInstanceID))
	if err != nil {
		utils.Error(c, http.StatusNotFound, "NOT_FOUND", "质量评分不存在")
		return
	}

	utils.Success(c, score)
}

// GetHealthRecords GET /api/v1/node-instances/:id/health-records
func (h *NodeHealthHandler) GetHealthRecords(c *gin.Context) {
	nodeInstanceID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAM", "无效的节点实例 ID")
		return
	}

	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	records, err := h.service.GetHealthRecords(uint(nodeInstanceID), limit)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "QUERY_FAILED", "查询健康检查记录失败: "+err.Error())
		return
	}

	utils.Success(c, gin.H{
		"records": records,
		"total":   len(records),
	})
}

// GetHealthStats GET /api/v1/node-instances/:id/health-stats
func (h *NodeHealthHandler) GetHealthStats(c *gin.Context) {
	nodeInstanceID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAM", "无效的节点实例 ID")
		return
	}

	// 默认查询最近 24 小时
	duration := 24 * time.Hour
	if durationStr := c.Query("duration"); durationStr != "" {
		if d, err := time.ParseDuration(durationStr); err == nil {
			duration = d
		}
	}

	stats, err := h.service.GetHealthStats(uint(nodeInstanceID), duration)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "QUERY_FAILED", "查询健康统计失败: "+err.Error())
		return
	}

	utils.Success(c, stats)
}

// ListQualityScores GET /api/v1/node-instances/quality-scores
func (h *NodeHealthHandler) ListQualityScores(c *gin.Context) {
	userID, _, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	// 获取用户的所有节点实例的质量评分
	var scores []models.NodeQualityScore
	query := h.service.GetDB().
		Joins("JOIN node_instances ON node_instances.id = node_quality_scores.node_instance_id").
		Joins("JOIN node_groups ON node_groups.id = node_instances.node_group_id").
		Where("node_groups.user_id = ?", userID).
		Order("node_quality_scores.overall_score DESC")

	if err := query.Find(&scores).Error; err != nil {
		utils.Error(c, http.StatusInternalServerError, "QUERY_FAILED", "查询质量评分失败: "+err.Error())
		return
	}

	utils.Success(c, gin.H{
		"scores": scores,
		"total":  len(scores),
	})
}
