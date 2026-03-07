package handlers

import (
	"net/http"
	"strings"

	"nodepass-pro/backend/internal/services"
	"nodepass-pro/backend/internal/utils"
	panelws "nodepass-pro/backend/internal/websocket"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SystemHandler 系统配置与统计处理器。
type SystemHandler struct {
	systemService *services.SystemService
	hub           *panelws.Hub
}

// NewSystemHandler 创建系统处理器。
func NewSystemHandler(db *gorm.DB, hub *panelws.Hub) *SystemHandler {
	return &SystemHandler{
		systemService: services.NewSystemService(db),
		hub:           hub,
	}
}

// GetConfig GET /api/v1/system/config
func (h *SystemHandler) GetConfig(c *gin.Context) {
	configMap, err := h.systemService.GetConfig()
	if err != nil {
		writeServiceError(c, err, "GET_SYSTEM_CONFIG_FAILED")
		return
	}
	utils.Success(c, configMap)
}

// UpdateConfig PUT /api/v1/system/config
func (h *SystemHandler) UpdateConfig(c *gin.Context) {
	type requestPayload struct {
		Key   string `json:"key" binding:"required"`
		Value string `json:"value"`
	}
	var req requestPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	key := strings.TrimSpace(req.Key)
	if err := h.systemService.UpdateConfig(key, req.Value); err != nil {
		writeServiceError(c, err, "UPDATE_SYSTEM_CONFIG_FAILED")
		return
	}

	if h.hub != nil {
		_ = h.hub.Broadcast(panelws.MessageTypeConfigUpdated, gin.H{
			"key": key,
		})
	}

	utils.SuccessResponse(c, nil, "系统配置更新成功")
}

// GetStats GET /api/v1/system/stats
func (h *SystemHandler) GetStats(c *gin.Context) {
	stats, err := h.systemService.GetStats()
	if err != nil {
		writeServiceError(c, err, "GET_SYSTEM_STATS_FAILED")
		return
	}
	utils.Success(c, stats)
}
