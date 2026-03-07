package handlers

import (
	"net/http"
	"strings"

	"nodepass-panel/backend/internal/services"
	"nodepass-panel/backend/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// NodeAgentHandler 节点客户端 API 处理器（使用 node token 认证）。
type NodeAgentHandler struct {
	configService *services.ConfigDistributionService
}

// NewNodeAgentHandler 创建节点客户端处理器。
func NewNodeAgentHandler(db *gorm.DB) *NodeAgentHandler {
	return &NodeAgentHandler{
		configService: services.NewConfigDistributionService(db),
	}
}

// Register POST /api/v1/nodes/register
func (h *NodeAgentHandler) Register(c *gin.Context) {
	type requestPayload struct {
		Token    string `json:"token" binding:"required"`
		Hostname string `json:"hostname"`
		Version  string `json:"version"`
	}

	var req requestPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	result, err := h.configService.HandleNodeRegister(req.Token, req.Hostname, req.Version)
	if err != nil {
		writeServiceError(c, err, "NODE_REGISTER_FAILED")
		return
	}

	utils.Success(c, result)
}

// Heartbeat POST /api/v1/nodes/heartbeat
func (h *NodeAgentHandler) Heartbeat(c *gin.Context) {
	type requestPayload struct {
		Token                string                        `json:"token" binding:"required"`
		CurrentConfigVersion int                           `json:"current_config_version"`
		SystemInfo           *services.HeartbeatSystemInfo `json:"system_info"`
		RulesStatus          []services.RuleRuntimeStatus  `json:"rules_status"`
	}

	var req requestPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	result, err := h.configService.HandleHeartbeat(
		req.Token,
		req.CurrentConfigVersion,
		req.SystemInfo,
		req.RulesStatus,
	)
	if err != nil {
		writeServiceError(c, err, "HEARTBEAT_FAILED")
		return
	}

	utils.Success(c, result)
}

// PullConfig GET /api/v1/nodes/:id/config
func (h *NodeAgentHandler) PullConfig(c *gin.Context) {
	nodeID, ok := parseUintParam(c, "id")
	if !ok {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "节点 ID 无效")
		return
	}

	token := extractNodeToken(c)
	if token == "" {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未提供节点 token")
		return
	}

	node, err := h.configService.AuthenticateNode(token)
	if err != nil {
		writeServiceError(c, err, "UNAUTHORIZED")
		return
	}
	if node.ID != nodeID {
		utils.Error(c, http.StatusForbidden, "FORBIDDEN", "无权拉取该节点配置")
		return
	}

	config, err := h.configService.HandleConfigPull(nodeID)
	if err != nil {
		writeServiceError(c, err, "CONFIG_PULL_FAILED")
		return
	}

	utils.Success(c, config)
}

// ReportTraffic POST /api/v1/nodes/traffic/report
func (h *NodeAgentHandler) ReportTraffic(c *gin.Context) {
	type requestPayload struct {
		Token   string                         `json:"token" binding:"required"`
		Records []services.TrafficReportRecord `json:"records"`
	}

	var req requestPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	accepted, err := h.configService.HandleTrafficReport(req.Token, req.Records)
	if err != nil {
		writeServiceError(c, err, "TRAFFIC_REPORT_FAILED")
		return
	}

	utils.Success(c, gin.H{
		"accepted": accepted,
	})
}

func extractNodeToken(c *gin.Context) string {
	token := strings.TrimSpace(c.GetHeader("X-Node-Token"))
	if token != "" {
		return token
	}

	authorization := strings.TrimSpace(c.GetHeader("Authorization"))
	if authorization != "" {
		parts := strings.SplitN(authorization, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			token = strings.TrimSpace(parts[1])
		}
	}
	if token != "" {
		return token
	}

	token = strings.TrimSpace(c.Query("token"))
	return token
}
