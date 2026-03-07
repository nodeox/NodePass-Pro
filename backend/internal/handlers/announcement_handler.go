package handlers

import (
	"net/http"
	"strings"
	"time"

	"nodepass-panel/backend/internal/services"
	"nodepass-panel/backend/internal/utils"
	panelws "nodepass-panel/backend/internal/websocket"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AnnouncementHandler 公告处理器。
type AnnouncementHandler struct {
	announcementService *services.AnnouncementService
	hub                 *panelws.Hub
}

// NewAnnouncementHandler 创建公告处理器。
func NewAnnouncementHandler(db *gorm.DB, hub *panelws.Hub) *AnnouncementHandler {
	return &AnnouncementHandler{
		announcementService: services.NewAnnouncementService(db),
		hub:                 hub,
	}
}

// List GET /api/v1/announcements
func (h *AnnouncementHandler) List(c *gin.Context) {
	onlyEnabled := true
	if raw := strings.TrimSpace(c.Query("only_enabled")); raw != "" {
		parsed, err := parseOptionalBoolQuery(c, "only_enabled")
		if err != nil {
			utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "only_enabled 参数错误")
			return
		}
		if parsed != nil {
			onlyEnabled = *parsed
		}
	}

	result, err := h.announcementService.List(onlyEnabled)
	if err != nil {
		writeServiceError(c, err, "LIST_ANNOUNCEMENTS_FAILED")
		return
	}

	utils.Success(c, gin.H{
		"list":  result,
		"total": len(result),
	})
}

// Create POST /api/v1/announcements
func (h *AnnouncementHandler) Create(c *gin.Context) {
	adminUserID, _, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	var req services.AnnouncementCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	item, err := h.announcementService.Create(adminUserID, &req)
	if err != nil {
		writeServiceError(c, err, "CREATE_ANNOUNCEMENT_FAILED")
		return
	}
	h.broadcastAnnouncement("created", item)

	utils.SuccessResponse(c, item, "公告创建成功")
}

// Update PUT /api/v1/announcements/:id
func (h *AnnouncementHandler) Update(c *gin.Context) {
	adminUserID, _, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	id, ok := parseUintParam(c, "id")
	if !ok {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "公告 ID 无效")
		return
	}

	var req services.AnnouncementUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	item, err := h.announcementService.Update(adminUserID, id, &req)
	if err != nil {
		writeServiceError(c, err, "UPDATE_ANNOUNCEMENT_FAILED")
		return
	}
	h.broadcastAnnouncement("updated", item)

	utils.SuccessResponse(c, item, "公告更新成功")
}

// Delete DELETE /api/v1/announcements/:id
func (h *AnnouncementHandler) Delete(c *gin.Context) {
	adminUserID, _, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	id, ok := parseUintParam(c, "id")
	if !ok {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "公告 ID 无效")
		return
	}

	if err := h.announcementService.Delete(adminUserID, id); err != nil {
		writeServiceError(c, err, "DELETE_ANNOUNCEMENT_FAILED")
		return
	}

	if h.hub != nil {
		_ = h.hub.Broadcast(panelws.MessageTypeAnnouncement, gin.H{
			"operation": "deleted",
			"id":        id,
			"timestamp": time.Now().Format(time.RFC3339),
		})
	}

	utils.SuccessResponse(c, nil, "公告删除成功")
}

func (h *AnnouncementHandler) broadcastAnnouncement(operation string, item interface{}) {
	if h.hub == nil {
		return
	}
	_ = h.hub.Broadcast(panelws.MessageTypeAnnouncement, gin.H{
		"operation": operation,
		"data":      item,
	})
}
