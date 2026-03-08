package handlers

import (
	"encoding/json"
	"io"
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
	type requestEntry struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	type batchPayload struct {
		Items []requestEntry `json:"items"`
	}

	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数读取失败")
		return
	}
	if len(rawBody) == 0 {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求体不能为空")
		return
	}

	entries := make([]services.SystemConfigEntry, 0)
	var batchReq batchPayload
	if unmarshalErr := json.Unmarshal(rawBody, &batchReq); unmarshalErr == nil && len(batchReq.Items) > 0 {
		for _, item := range batchReq.Items {
			entries = append(entries, services.SystemConfigEntry{
				Key:   item.Key,
				Value: item.Value,
			})
		}
	} else {
		var listReq []requestEntry
		if unmarshalErr := json.Unmarshal(rawBody, &listReq); unmarshalErr == nil && len(listReq) > 0 {
			for _, item := range listReq {
				entries = append(entries, services.SystemConfigEntry{
					Key:   item.Key,
					Value: item.Value,
				})
			}
		} else {
			var singleReq requestEntry
			if unmarshalErr := json.Unmarshal(rawBody, &singleReq); unmarshalErr != nil {
				utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+unmarshalErr.Error())
				return
			}
			entries = append(entries, services.SystemConfigEntry{
				Key:   singleReq.Key,
				Value: singleReq.Value,
			})
		}
	}

	if err := h.systemService.UpdateConfigs(entries); err != nil {
		writeServiceError(c, err, "UPDATE_SYSTEM_CONFIG_FAILED")
		return
	}

	if h.hub != nil {
		changedKeys := make([]string, 0, len(entries))
		for _, item := range entries {
			key := strings.TrimSpace(item.Key)
			if key != "" {
				changedKeys = append(changedKeys, key)
			}
		}
		_ = h.hub.Broadcast(panelws.MessageTypeConfigUpdated, gin.H{"keys": changedKeys})
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
