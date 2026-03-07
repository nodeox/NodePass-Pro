package handlers

import (
	"net/http"

	"nodepass-panel/backend/internal/utils"
	panelws "nodepass-panel/backend/internal/websocket"

	"github.com/gin-gonic/gin"
)

// WebSocketHandler WebSocket 处理器。
type WebSocketHandler struct {
	hub *panelws.Hub
}

// NewWebSocketHandler 创建 WebSocket 处理器。
func NewWebSocketHandler(hub *panelws.Hub) *WebSocketHandler {
	return &WebSocketHandler{hub: hub}
}

// Handle GET /ws
func (h *WebSocketHandler) Handle(c *gin.Context) {
	if h.hub == nil {
		utils.Error(c, http.StatusServiceUnavailable, "WEBSOCKET_UNAVAILABLE", "WebSocket Hub 未初始化")
		return
	}

	if err := h.hub.HandleConnection(c.Writer, c.Request); err != nil {
		writeServiceError(c, err, "WEBSOCKET_CONNECT_FAILED")
		return
	}
}
