package websocket

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"nodepass-pro/backend/internal/config"
	"nodepass-pro/backend/internal/services"

	gorilla "github.com/gorilla/websocket"
	"go.uber.org/zap"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = 30 * time.Second
	maxMessageSize = 8 * 1024
)

// MessageType WebSocket 消息类型。
type MessageType string

const (
	MessageTypeNodeStatusChanged MessageType = "node_status_changed"
	MessageTypeRuleStatusChanged MessageType = "rule_status_changed"
	MessageTypeTrafficAlert      MessageType = "traffic_alert"
	MessageTypeAnnouncement      MessageType = "announcement"
	MessageTypeConfigUpdated     MessageType = "config_updated"
)

type wsMessage struct {
	Type      MessageType `json:"type"`
	Data      any         `json:"data,omitempty"`
	Timestamp string      `json:"timestamp"`
}

type outboundMessage struct {
	userID  *uint
	payload []byte
}

// Hub WebSocket 连接管理中心。
type Hub struct {
	mu          sync.RWMutex
	clients     map[*Client]struct{}
	userClients map[uint]map[*Client]struct{}

	register   chan *Client
	unregister chan *Client
	broadcast  chan outboundMessage

	upgrader gorilla.Upgrader
}

// Client WebSocket 客户端连接。
type Client struct {
	hub       *Hub
	conn      *gorilla.Conn
	send      chan []byte
	userID    uint
	closeOnce sync.Once // 确保 send channel 只关闭一次
}

// NewHub 创建并启动 WebSocket Hub。
func NewHub() *Hub {
	hub := &Hub{
		clients:     make(map[*Client]struct{}),
		userClients: make(map[uint]map[*Client]struct{}),
		register:    make(chan *Client, 128),
		unregister:  make(chan *Client, 128),
		broadcast:   make(chan outboundMessage, 512),
		upgrader: gorilla.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     checkWebSocketOrigin,
			Subprotocols:    []string{"bearer"},
		},
	}
	go hub.run()
	return hub
}

// HandleConnection 处理 WebSocket 握手并注册连接。
func (h *Hub) HandleConnection(w http.ResponseWriter, r *http.Request, userID uint) error {
	if userID == 0 {
		return fmt.Errorf("%w: 用户信息缺失", services.ErrUnauthorized)
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return fmt.Errorf("升级 WebSocket 连接失败: %w", err)
	}

	client := &Client{
		hub:    h,
		conn:   conn,
		send:   make(chan []byte, 256),
		userID: userID,
	}

	h.register <- client
	go client.writePump()
	go client.readPump()

	return nil
}

// Broadcast 全局广播消息。
func (h *Hub) Broadcast(messageType MessageType, data any) error {
	payload, err := buildPayload(messageType, data)
	if err != nil {
		return err
	}
	h.broadcast <- outboundMessage{payload: payload}
	return nil
}

// SendToUser 向指定用户定向发送消息。
func (h *Hub) SendToUser(userID uint, messageType MessageType, data any) error {
	if userID == 0 {
		return fmt.Errorf("%w: userID 无效", services.ErrInvalidParams)
	}
	payload, err := buildPayload(messageType, data)
	if err != nil {
		return err
	}
	targetUserID := userID
	h.broadcast <- outboundMessage{userID: &targetUserID, payload: payload}
	return nil
}

// Count 返回当前在线连接数。
func (h *Hub) Count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.registerClient(client)
		case client := <-h.unregister:
			h.unregisterClient(client)
		case message := <-h.broadcast:
			h.dispatchMessage(message)
		}
	}
}

func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.clients[client] = struct{}{}
	if _, exists := h.userClients[client.userID]; !exists {
		h.userClients[client.userID] = make(map[*Client]struct{})
	}
	h.userClients[client.userID][client] = struct{}{}
}

func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.unregisterClientLocked(client)
}

func (h *Hub) unregisterClientLocked(client *Client) {
	if _, exists := h.clients[client]; !exists {
		return
	}

	delete(h.clients, client)
	if userSet, ok := h.userClients[client.userID]; ok {
		delete(userSet, client)
		if len(userSet) == 0 {
			delete(h.userClients, client.userID)
		}
	}

	// 使用 sync.Once 确保 channel 只关闭一次
	client.closeOnce.Do(func() {
		close(client.send)
	})
}

func (h *Hub) dispatchMessage(message outboundMessage) {
	h.mu.Lock()
	defer h.mu.Unlock()

	staleClients := make([]*Client, 0)
	sendToClient := func(client *Client) {
		select {
		case client.send <- message.payload:
		default:
			staleClients = append(staleClients, client)
		}
	}

	if message.userID != nil {
		if userSet, ok := h.userClients[*message.userID]; ok {
			for client := range userSet {
				sendToClient(client)
			}
		}
	} else {
		for client := range h.clients {
			sendToClient(client)
		}
	}

	for _, client := range staleClients {
		h.unregisterClientLocked(client)
	}
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		if err := c.conn.Close(); err != nil {
			zap.L().Debug("关闭 WebSocket 连接失败", zap.Error(err))
		}
	}()

	c.conn.SetReadLimit(maxMessageSize)
	if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		zap.L().Warn("设置读取超时失败", zap.Error(err))
	}
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			break
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		if err := c.conn.Close(); err != nil {
			zap.L().Debug("关闭 WebSocket 连接失败", zap.Error(err))
		}
	}()

	for {
		select {
		case payload, ok := <-c.send:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				zap.L().Warn("设置写入超时失败", zap.Error(err))
				return
			}
			if !ok {
				if err := c.conn.WriteMessage(gorilla.CloseMessage, []byte{}); err != nil {
					zap.L().Debug("发送关闭消息失败", zap.Error(err))
				}
				return
			}

			writer, err := c.conn.NextWriter(gorilla.TextMessage)
			if err != nil {
				return
			}
			if _, err = writer.Write(payload); err != nil {
				if closeErr := writer.Close(); closeErr != nil {
					zap.L().Debug("关闭 writer 失败", zap.Error(closeErr))
				}
				return
			}
			if err = writer.Close(); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				zap.L().Warn("设置写入超时失败", zap.Error(err))
				return
			}
			if err := c.conn.WriteMessage(gorilla.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func buildPayload(messageType MessageType, data any) ([]byte, error) {
	if strings.TrimSpace(string(messageType)) == "" {
		return nil, fmt.Errorf("%w: 消息类型不能为空", services.ErrInvalidParams)
	}

	payload, err := json.Marshal(wsMessage{
		Type:      messageType,
		Data:      data,
		Timestamp: time.Now().Format(time.RFC3339),
	})
	if err != nil {
		return nil, fmt.Errorf("序列化 WebSocket 消息失败: %w", err)
	}
	return payload, nil
}

// checkWebSocketOrigin 验证 WebSocket 连接的来源。
func checkWebSocketOrigin(r *http.Request) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))

	// 如果没有 Origin 头，尝试从 Referer 获取
	if origin == "" {
		referer := strings.TrimSpace(r.Header.Get("Referer"))
		if referer != "" {
			// 从 Referer 中提取 origin
			if parsedURL, err := url.Parse(referer); err == nil {
				origin = fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host)
			}
		}
	}

	// 如果仍然没有 origin，拒绝连接（即使在开发模式）
	cfg := config.GlobalConfig
	isDevelopment := cfg != nil && cfg.Server.Mode != "release"

	if origin == "" {
		// 开发模式下也要求 Origin，但给出更友好的日志
		if isDevelopment {
			zap.L().Warn("WebSocket 连接被拒绝：缺少 Origin 头（开发模式也需要 Origin）",
				zap.String("remote_addr", r.RemoteAddr),
				zap.String("hint", "请确保客户端发送 Origin 头"))
		} else {
			zap.L().Warn("WebSocket 连接被拒绝：缺少 Origin 头",
				zap.String("remote_addr", r.RemoteAddr))
		}
		return false
	}

	// 解析 origin URL
	originURL, err := url.Parse(origin)
	if err != nil {
		zap.L().Warn("WebSocket 连接被拒绝：无效的 Origin 格式",
			zap.String("origin", origin),
			zap.String("error", err.Error()))
		return false
	}

	// 开发环境：允许 localhost 和 127.0.0.1
	if isDevelopment {
		if isLocalhost(originURL.Hostname()) {
			return true
		}
	}

	// 生产环境或配置了允许来源：检查配置的允许来源列表
	if cfg != nil && cfg.Server.AllowedOrigins != nil {
		for _, allowed := range cfg.Server.AllowedOrigins {
			if matchOrigin(originURL, allowed) {
				return true
			}
		}
	}

	// 如果没有配置允许来源，且是生产环境，拒绝所有非本地连接
	zap.L().Warn("WebSocket 连接被拒绝：不允许的来源",
		zap.String("origin", origin),
		zap.String("remote_addr", r.RemoteAddr))
	return false
}

// isLocalhost 检查主机名是否为本地地址。
func isLocalhost(hostname string) bool {
	hostname = strings.ToLower(strings.TrimSpace(hostname))
	return hostname == "localhost" ||
		hostname == "127.0.0.1" ||
		hostname == "::1" ||
		hostname == "[::1]"
}

// matchOrigin 检查 origin 是否匹配允许的来源。
// 支持精确匹配和通配符匹配。
func matchOrigin(originURL *url.URL, allowed string) bool {
	allowed = strings.TrimSpace(allowed)
	if allowed == "" {
		return false
	}

	// 如果允许的来源是完整的 URL，进行精确匹配
	if strings.HasPrefix(allowed, "http://") || strings.HasPrefix(allowed, "https://") {
		allowedURL, err := url.Parse(allowed)
		if err != nil {
			return false
		}
		// 精确匹配 scheme 和 host
		return originURL.Scheme == allowedURL.Scheme &&
			strings.EqualFold(originURL.Host, allowedURL.Host)
	}

	// 如果允许的来源只是主机名（支持通配符）
	// 例如：example.com, *.example.com
	if strings.HasPrefix(allowed, "*.") {
		// 通配符匹配：*.example.com 匹配 sub.example.com
		domain := strings.TrimPrefix(allowed, "*.")
		hostname := strings.ToLower(originURL.Hostname())
		return strings.HasSuffix(hostname, "."+domain) || hostname == domain
	}

	// 精确主机名匹配
	return strings.EqualFold(originURL.Hostname(), allowed)
}
