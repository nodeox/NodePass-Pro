package handlers

import (
	"net/http"

	"nodepass-pro/backend/internal/models"
	"nodepass-pro/backend/internal/services"
	"nodepass-pro/backend/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// NotificationChannelHandler 通知渠道处理器
type NotificationChannelHandler struct {
	channelService *services.NotificationChannelService
}

// NewNotificationChannelHandler 创建通知渠道处理器
func NewNotificationChannelHandler(db *gorm.DB) *NotificationChannelHandler {
	return &NotificationChannelHandler{
		channelService: services.NewNotificationChannelService(db),
	}
}

// Create POST /api/v1/notification-channels
// 创建通知渠道
func (h *NotificationChannelHandler) Create(c *gin.Context) {
	var channel models.NotificationChannel
	if err := c.ShouldBindJSON(&channel); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	// 验证必填字段
	if channel.Name == "" || channel.Type == "" || channel.Config == "" {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "名称、类型和配置不能为空")
		return
	}

	// 验证类型
	validTypes := map[string]bool{
		"email":    true,
		"telegram": true,
		"webhook":  true,
		"slack":    true,
	}
	if !validTypes[channel.Type] {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "不支持的通知渠道类型")
		return
	}

	if err := h.channelService.CreateChannel(&channel); err != nil {
		writeServiceError(c, err, "CREATE_CHANNEL_FAILED")
		return
	}

	utils.SuccessResponse(c, channel, "创建通知渠道成功")
}

// Update PUT /api/v1/notification-channels/:id
// 更新通知渠道
func (h *NotificationChannelHandler) Update(c *gin.Context) {
	id, err := parseAlertUintParam(c, "id")
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "无效的渠道 ID")
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	if err := h.channelService.UpdateChannel(id, updates); err != nil {
		writeServiceError(c, err, "UPDATE_CHANNEL_FAILED")
		return
	}

	utils.SuccessResponse(c, nil, "更新通知渠道成功")
}

// Delete DELETE /api/v1/notification-channels/:id
// 删除通知渠道
func (h *NotificationChannelHandler) Delete(c *gin.Context) {
	id, err := parseAlertUintParam(c, "id")
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "无效的渠道 ID")
		return
	}

	if err := h.channelService.DeleteChannel(id); err != nil {
		writeServiceError(c, err, "DELETE_CHANNEL_FAILED")
		return
	}

	utils.SuccessResponse(c, nil, "删除通知渠道成功")
}

// Get GET /api/v1/notification-channels/:id
// 获取通知渠道详情
func (h *NotificationChannelHandler) Get(c *gin.Context) {
	id, err := parseAlertUintParam(c, "id")
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "无效的渠道 ID")
		return
	}

	channel, err := h.channelService.GetChannel(id)
	if err != nil {
		writeServiceError(c, err, "GET_CHANNEL_FAILED")
		return
	}

	utils.SuccessResponse(c, channel, "获取通知渠道成功")
}

// List GET /api/v1/notification-channels
// 获取通知渠道列表
func (h *NotificationChannelHandler) List(c *gin.Context) {
	enabledStr := c.Query("is_enabled")
	var isEnabled *bool
	if enabledStr != "" {
		val := enabledStr == "true" || enabledStr == "1"
		isEnabled = &val
	}

	channels, err := h.channelService.ListChannels(isEnabled)
	if err != nil {
		writeServiceError(c, err, "LIST_CHANNELS_FAILED")
		return
	}

	utils.SuccessResponse(c, gin.H{
		"list":  channels,
		"total": len(channels),
	}, "获取通知渠道列表成功")
}

// Toggle POST /api/v1/notification-channels/:id/toggle
// 启用/禁用通知渠道
func (h *NotificationChannelHandler) Toggle(c *gin.Context) {
	id, err := parseAlertUintParam(c, "id")
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "无效的渠道 ID")
		return
	}

	// 获取当前渠道
	channel, err := h.channelService.GetChannel(id)
	if err != nil {
		writeServiceError(c, err, "GET_CHANNEL_FAILED")
		return
	}

	// 切换状态
	newStatus := !channel.IsEnabled
	if err := h.channelService.UpdateChannel(id, map[string]interface{}{
		"is_enabled": newStatus,
	}); err != nil {
		writeServiceError(c, err, "TOGGLE_CHANNEL_FAILED")
		return
	}

	message := "通知渠道已启用"
	if !newStatus {
		message = "通知渠道已禁用"
	}

	utils.SuccessResponse(c, gin.H{"is_enabled": newStatus}, message)
}

// Test POST /api/v1/notification-channels/:id/test
// 测试通知渠道
func (h *NotificationChannelHandler) Test(c *gin.Context) {
	id, err := parseAlertUintParam(c, "id")
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "无效的渠道 ID")
		return
	}

	if err := h.channelService.TestChannel(id); err != nil {
		writeServiceError(c, err, "TEST_CHANNEL_FAILED")
		return
	}

	utils.SuccessResponse(c, nil, "测试通知已发送")
}
