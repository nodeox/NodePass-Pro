package websocket

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"nodepass-panel/backend/internal/config"
	"nodepass-panel/backend/internal/services"

	"github.com/golang-jwt/jwt/v5"
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
	hub    *Hub
	conn   *gorilla.Conn
	send   chan []byte
	userID uint
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
			CheckOrigin: func(r *http.Request) bool {
				// 验证来源，防止 CSRF 攻击
				origin := r.Header.Get("Origin")
				if origin == "" {
					// 如果没有 Origin 头，检查 Referer
					origin = r.Header.Get("Referer")
				}

				// 允许本地开发环境
				if origin == "" ||
					strings.Contains(origin, "localhost") ||
					strings.Contains(origin, "127.0.0.1") {
					return true
				}

				// 生产环境应该检查配置的允许来源列表
				// TODO: 从配置文件读取允许的来源
				cfg := config.GlobalConfig
				if cfg != nil && cfg.Server.AllowedOrigins != nil {
					for _, allowed := range cfg.Server.AllowedOrigins {
						if strings.Contains(origin, allowed) {
							return true
						}
					}
				}

				zap.L().Warn("WebSocket 连接被拒绝：不允许的来源",
					zap.String("origin", origin),
					zap.String("remote_addr", r.RemoteAddr))
				return false
			},
		},
	}
	go hub.run()
	return hub
}

// HandleConnection 处理 WebSocket 握手并注册连接。
func (h *Hub) HandleConnection(w http.ResponseWriter, r *http.Request) error {
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	if token == "" {
		return fmt.Errorf("%w: 缺少 token", services.ErrUnauthorized)
	}

	userID, err := h.verifyJWT(token)
	if err != nil {
		return err
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

	close(client.send)
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

func (h *Hub) verifyJWT(tokenString string) (uint, error) {
	cfg := config.GlobalConfig
	if cfg == nil || strings.TrimSpace(cfg.JWT.Secret) == "" {
		return 0, fmt.Errorf("%w: JWT 配置无效", services.ErrUnauthorized)
	}

	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("不支持的签名算法: %v", token.Header["alg"])
		}
		return []byte(cfg.JWT.Secret), nil
	})
	if err != nil || token == nil || !token.Valid {
		return 0, fmt.Errorf("%w: token 无效或已过期", services.ErrUnauthorized)
	}

	rawUserID, exists := claims["user_id"]
	if !exists {
		return 0, fmt.Errorf("%w: token 缺少用户信息", services.ErrUnauthorized)
	}

	var userID uint
	switch value := rawUserID.(type) {
	case float64:
		if value > 0 {
			userID = uint(value)
		}
	case int64:
		if value > 0 {
			userID = uint(value)
		}
	case int:
		if value > 0 {
			userID = uint(value)
		}
	}
	if userID == 0 {
		return 0, fmt.Errorf("%w: token 用户信息无效", services.ErrUnauthorized)
	}

	return userID, nil
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
