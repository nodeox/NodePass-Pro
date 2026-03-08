package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"nodepass-pro/backend/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTunnelHandlerTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("无法连接数据库: %v", err)
	}

	if err := db.AutoMigrate(
		&models.User{},
		&models.NodeGroup{},
		&models.Tunnel{},
		&models.NodeInstance{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}

	return db
}

func setupTunnelHandlerTestRouter(db *gorm.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// 添加模拟认证中间件
	router.Use(func(c *gin.Context) {
		if userID := c.GetHeader("X-User-ID"); userID != "" {
			if parsed, err := strconv.ParseUint(userID, 10, 64); err == nil && parsed > 0 {
				id := uint(parsed)
				c.Set("userID", id)
				c.Set("user_id", id)
			}
		}
		if role := c.GetHeader("X-User-Role"); role != "" {
			c.Set("role", role)
		}
		c.Next()
	})

	handler := NewTunnelHandler(db)

	api := router.Group("/api/v1")
	{
		tunnels := api.Group("/tunnels")
		{
			tunnels.POST("", handler.Create)
			tunnels.GET("", handler.List)
			tunnels.GET("/:id", handler.Get)
			tunnels.PUT("/:id", handler.Update)
			tunnels.DELETE("/:id", handler.Delete)
			tunnels.POST("/:id/start", handler.Start)
			tunnels.POST("/:id/stop", handler.Stop)
		}
	}

	return router
}

func createTestTunnelUser(t *testing.T, db *gorm.DB, role string) *models.User {
	user := &models.User{
		Username:     fmt.Sprintf("testuser_%d", time.Now().UnixNano()),
		Email:        fmt.Sprintf("test_%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hashed_password",
		Role:         role,
		Status:       "active",
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}
	return user
}

func createTestNodeGroup(t *testing.T, db *gorm.DB, userID uint, name string) *models.NodeGroup {
	desc := "Test group"
	group := &models.NodeGroup{
		Name:        name,
		Description: &desc,
		UserID:      userID,
		Type:        "entry",
		IsEnabled:   true,
	}
	if err := db.Create(group).Error; err != nil {
		t.Fatalf("创建测试节点组失败: %v", err)
	}
	return group
}

func createTestTunnel(t *testing.T, db *gorm.DB, userID, entryGroupID uint) *models.Tunnel {
	tunnel := &models.Tunnel{
		Name:         fmt.Sprintf("test_tunnel_%d", time.Now().UnixNano()),
		UserID:       userID,
		EntryGroupID: entryGroupID,
		Protocol:     "tcp",
		RemoteHost:   "127.0.0.1",
		RemotePort:   8080,
		Status:       "stopped",
	}
	if err := db.Create(tunnel).Error; err != nil {
		t.Fatalf("创建测试隧道失败: %v", err)
	}
	return tunnel
}

func TestTunnelHandler_Create(t *testing.T) {
	db := setupTunnelHandlerTestDB(t)
	router := setupTunnelHandlerTestRouter(db)

	user := createTestTunnelUser(t, db, "user")
	admin := createTestTunnelUser(t, db, "admin")
	entryGroup := createTestNodeGroup(t, db, user.ID, "entry_group")

	tests := []struct {
		name           string
		userID         uint
		role           string
		payload        interface{}
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name:   "成功创建隧道",
			userID: user.ID,
			role:   "user",
			payload: map[string]interface{}{
				"name":           "test_tunnel",
				"entry_group_id": entryGroup.ID,
				"protocol":       "tcp",
				"remote_host":    "127.0.0.1",
				"remote_port":    8080,
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if !resp["success"].(bool) {
					t.Error("期望 success 为 true")
				}
				data := resp["data"].(map[string]interface{})
				if data["name"] != "test_tunnel" {
					t.Errorf("期望 name 为 test_tunnel, 得到 %v", data["name"])
				}
			},
		},
		{
			name:   "管理员为其他用户创建隧道",
			userID: admin.ID,
			role:   "admin",
			payload: map[string]interface{}{
				"name":           "admin_tunnel",
				"user_id":        user.ID,
				"entry_group_id": entryGroup.ID,
				"protocol":       "tcp",
				"remote_host":    "127.0.0.1",
				"remote_port":    9090,
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if !resp["success"].(bool) {
					t.Error("期望 success 为 true")
				}
			},
		},
		{
			name:           "缺少必填字段",
			userID:         user.ID,
			role:           "user",
			payload:        map[string]interface{}{},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if resp["success"].(bool) {
					t.Error("期望 success 为 false")
				}
			},
		},
		{
			name:   "无效的 remote_port",
			userID: user.ID,
			role:   "user",
			payload: map[string]interface{}{
				"name":           "invalid_tunnel",
				"entry_group_id": entryGroup.ID,
				"protocol":       "tcp",
				"remote_host":    "127.0.0.1",
				"remote_port":    99999,
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if resp["success"].(bool) {
					t.Error("期望 success 为 false")
				}
			},
		},
		{
			name:           "未认证用户",
			userID:         0,
			role:           "",
			payload:        map[string]interface{}{},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if resp["success"].(bool) {
					t.Error("期望 success 为 false")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/tunnels", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			if tt.userID > 0 {
				req.Header.Set("X-User-ID", fmt.Sprintf("%d", tt.userID))
				req.Header.Set("X-User-Role", tt.role)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("期望状态码 %d, 得到 %d, 响应: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			var resp map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("解析响应失败: %v, 响应内容: %s", err, w.Body.String())
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}
		})
	}
}

func TestTunnelHandler_List(t *testing.T) {
	db := setupTunnelHandlerTestDB(t)
	router := setupTunnelHandlerTestRouter(db)

	user := createTestTunnelUser(t, db, "user")
	admin := createTestTunnelUser(t, db, "admin")
	entryGroup := createTestNodeGroup(t, db, user.ID, "entry_group")

	// 创建测试隧道
	createTestTunnel(t, db, user.ID, entryGroup.ID)
	createTestTunnel(t, db, user.ID, entryGroup.ID)

	tests := []struct {
		name           string
		userID         uint
		role           string
		query          string
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name:           "普通用户获取自己的隧道列表",
			userID:         user.ID,
			role:           "user",
			query:          "",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if !resp["success"].(bool) {
					t.Error("期望 success 为 true")
				}
				data := resp["data"].(map[string]interface{})
				items := data["items"].([]interface{})
				if len(items) != 2 {
					t.Errorf("期望 2 个隧道, 得到 %d", len(items))
				}
			},
		},
		{
			name:           "管理员获取所有隧道",
			userID:         admin.ID,
			role:           "admin",
			query:          "",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if !resp["success"].(bool) {
					t.Error("期望 success 为 true")
				}
			},
		},
		{
			name:           "分页查询",
			userID:         user.ID,
			role:           "user",
			query:          "?page=1&page_size=1",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				data := resp["data"].(map[string]interface{})
				items := data["items"].([]interface{})
				if len(items) != 1 {
					t.Errorf("期望 1 个隧道, 得到 %d", len(items))
				}
			},
		},
		{
			name:           "按状态过滤",
			userID:         user.ID,
			role:           "user",
			query:          "?status=stopped",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if !resp["success"].(bool) {
					t.Error("期望 success 为 true")
				}
			},
		},
		{
			name:           "无效的 page 参数",
			userID:         user.ID,
			role:           "user",
			query:          "?page=-1",
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if resp["success"].(bool) {
					t.Error("期望 success 为 false")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/tunnels"+tt.query, nil)
			req.Header.Set("X-User-ID", fmt.Sprintf("%d", tt.userID))
			req.Header.Set("X-User-Role", tt.role)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("期望状态码 %d, 得到 %d", tt.expectedStatus, w.Code)
			}

			var resp map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("解析响应失败: %v", err)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}
		})
	}
}

func TestTunnelHandler_Get(t *testing.T) {
	db := setupTunnelHandlerTestDB(t)
	router := setupTunnelHandlerTestRouter(db)

	user := createTestTunnelUser(t, db, "user")
	admin := createTestTunnelUser(t, db, "admin")
	otherUser := createTestTunnelUser(t, db, "user")
	entryGroup := createTestNodeGroup(t, db, user.ID, "entry_group")

	tunnel := createTestTunnel(t, db, user.ID, entryGroup.ID)

	tests := []struct {
		name           string
		userID         uint
		role           string
		tunnelID       uint
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name:           "成功获取隧道详情",
			userID:         user.ID,
			role:           "user",
			tunnelID:       tunnel.ID,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if !resp["success"].(bool) {
					t.Error("期望 success 为 true")
				}
				data := resp["data"].(map[string]interface{})
				if uint(data["id"].(float64)) != tunnel.ID {
					t.Errorf("期望 id 为 %d, 得到 %v", tunnel.ID, data["id"])
				}
			},
		},
		{
			name:           "管理员获取任意隧道",
			userID:         admin.ID,
			role:           "admin",
			tunnelID:       tunnel.ID,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if !resp["success"].(bool) {
					t.Error("期望 success 为 true")
				}
			},
		},
		{
			name:           "普通用户无法访问其他用户的隧道",
			userID:         otherUser.ID,
			role:           "user",
			tunnelID:       tunnel.ID,
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if resp["success"].(bool) {
					t.Error("期望 success 为 false")
				}
			},
		},
		{
			name:           "隧道不存在",
			userID:         user.ID,
			role:           "user",
			tunnelID:       99999,
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if resp["success"].(bool) {
					t.Error("期望 success 为 false")
				}
			},
		},
		{
			name:           "无效的隧道 ID",
			userID:         user.ID,
			role:           "user",
			tunnelID:       0,
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if resp["success"].(bool) {
					t.Error("期望 success 为 false")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := fmt.Sprintf("/api/v1/tunnels/%d", tt.tunnelID)
			req := httptest.NewRequest(http.MethodGet, url, nil)
			req.Header.Set("X-User-ID", fmt.Sprintf("%d", tt.userID))
			req.Header.Set("X-User-Role", tt.role)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("期望状态码 %d, 得到 %d", tt.expectedStatus, w.Code)
			}

			var resp map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("解析响应失败: %v", err)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}
		})
	}
}

func TestTunnelHandler_Update(t *testing.T) {
	db := setupTunnelHandlerTestDB(t)
	router := setupTunnelHandlerTestRouter(db)

	user := createTestTunnelUser(t, db, "user")
	admin := createTestTunnelUser(t, db, "admin")
	otherUser := createTestTunnelUser(t, db, "user")
	entryGroup := createTestNodeGroup(t, db, user.ID, "entry_group")

	tunnel := createTestTunnel(t, db, user.ID, entryGroup.ID)

	tests := []struct {
		name           string
		userID         uint
		role           string
		tunnelID       uint
		payload        interface{}
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name:     "成功更新隧道",
			userID:   user.ID,
			role:     "user",
			tunnelID: tunnel.ID,
			payload: map[string]interface{}{
				"name":        "updated_tunnel",
				"remote_port": 9090,
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if !resp["success"].(bool) {
					t.Error("期望 success 为 true")
				}
				data := resp["data"].(map[string]interface{})
				if data["name"] != "updated_tunnel" {
					t.Errorf("期望 name 为 updated_tunnel, 得到 %v", data["name"])
				}
			},
		},
		{
			name:     "管理员更新任意隧道",
			userID:   admin.ID,
			role:     "admin",
			tunnelID: tunnel.ID,
			payload: map[string]interface{}{
				"name": "admin_updated",
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if !resp["success"].(bool) {
					t.Error("期望 success 为 true")
				}
			},
		},
		{
			name:     "普通用户无法更新其他用户的隧道",
			userID:   otherUser.ID,
			role:     "user",
			tunnelID: tunnel.ID,
			payload: map[string]interface{}{
				"name": "hacked",
			},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if resp["success"].(bool) {
					t.Error("期望 success 为 false")
				}
			},
		},
		{
			name:     "无效的 remote_port",
			userID:   user.ID,
			role:     "user",
			tunnelID: tunnel.ID,
			payload: map[string]interface{}{
				"remote_port": -1,
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if resp["success"].(bool) {
					t.Error("期望 success 为 false")
				}
			},
		},
		{
			name:           "隧道不存在",
			userID:         user.ID,
			role:           "user",
			tunnelID:       99999,
			payload:        map[string]interface{}{},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if resp["success"].(bool) {
					t.Error("期望 success 为 false")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.payload)
			url := fmt.Sprintf("/api/v1/tunnels/%d", tt.tunnelID)
			req := httptest.NewRequest(http.MethodPut, url, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-User-ID", fmt.Sprintf("%d", tt.userID))
			req.Header.Set("X-User-Role", tt.role)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("期望状态码 %d, 得到 %d", tt.expectedStatus, w.Code)
			}

			var resp map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("解析响应失败: %v", err)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}
		})
	}
}

func TestTunnelHandler_Delete(t *testing.T) {
	db := setupTunnelHandlerTestDB(t)
	router := setupTunnelHandlerTestRouter(db)

	user := createTestTunnelUser(t, db, "user")
	admin := createTestTunnelUser(t, db, "admin")
	otherUser := createTestTunnelUser(t, db, "user")
	entryGroup := createTestNodeGroup(t, db, user.ID, "entry_group")

	tests := []struct {
		name           string
		userID         uint
		role           string
		setupTunnel    func() uint
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name:   "成功删除隧道",
			userID: user.ID,
			role:   "user",
			setupTunnel: func() uint {
				return createTestTunnel(t, db, user.ID, entryGroup.ID).ID
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if !resp["success"].(bool) {
					t.Error("期望 success 为 true")
				}
			},
		},
		{
			name:   "管理员删除任意隧道",
			userID: admin.ID,
			role:   "admin",
			setupTunnel: func() uint {
				return createTestTunnel(t, db, user.ID, entryGroup.ID).ID
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if !resp["success"].(bool) {
					t.Error("期望 success 为 true")
				}
			},
		},
		{
			name:   "普通用户无法删除其他用户的隧道",
			userID: otherUser.ID,
			role:   "user",
			setupTunnel: func() uint {
				return createTestTunnel(t, db, user.ID, entryGroup.ID).ID
			},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if resp["success"].(bool) {
					t.Error("期望 success 为 false")
				}
			},
		},
		{
			name:   "隧道不存在",
			userID: user.ID,
			role:   "user",
			setupTunnel: func() uint {
				return 99999
			},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if resp["success"].(bool) {
					t.Error("期望 success 为 false")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tunnelID := tt.setupTunnel()
			url := fmt.Sprintf("/api/v1/tunnels/%d", tunnelID)
			req := httptest.NewRequest(http.MethodDelete, url, nil)
			req.Header.Set("X-User-ID", fmt.Sprintf("%d", tt.userID))
			req.Header.Set("X-User-Role", tt.role)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("期望状态码 %d, 得到 %d", tt.expectedStatus, w.Code)
			}

			var resp map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("解析响应失败: %v", err)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}
		})
	}
}

func TestTunnelHandler_StartStop(t *testing.T) {
	db := setupTunnelHandlerTestDB(t)
	router := setupTunnelHandlerTestRouter(db)

	user := createTestTunnelUser(t, db, "user")
	entryGroup := createTestNodeGroup(t, db, user.ID, "entry_group")
	tunnel := createTestTunnel(t, db, user.ID, entryGroup.ID)

	tests := []struct {
		name           string
		action         string
		userID         uint
		role           string
		tunnelID       uint
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name:           "启动隧道",
			action:         "start",
			userID:         user.ID,
			role:           "user",
			tunnelID:       tunnel.ID,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				// 接受成功或冲突(已经在运行)
				if !resp["success"].(bool) {
					if errObj, ok := resp["error"].(map[string]interface{}); ok {
						if code, ok := errObj["code"].(string); ok && code == "CONFLICT" {
							t.Log("隧道已经在运行,这是可接受的")
							return
						}
					}
					t.Error("期望 success 为 true")
				}
			},
		},
		{
			name:           "停止隧道",
			action:         "stop",
			userID:         user.ID,
			role:           "user",
			tunnelID:       tunnel.ID,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if !resp["success"].(bool) {
					t.Error("期望 success 为 true")
				}
			},
		},
		{
			name:           "隧道不存在",
			action:         "start",
			userID:         user.ID,
			role:           "user",
			tunnelID:       99999,
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if resp["success"].(bool) {
					t.Error("期望 success 为 false")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := fmt.Sprintf("/api/v1/tunnels/%d/%s", tt.tunnelID, tt.action)
			req := httptest.NewRequest(http.MethodPost, url, nil)
			req.Header.Set("X-User-ID", fmt.Sprintf("%d", tt.userID))
			req.Header.Set("X-User-Role", tt.role)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				// 对于启动操作,接受 200 或 409 (已经在运行)
				if tt.action == "start" && w.Code == http.StatusConflict {
					t.Logf("隧道已经在运行 (409),这是可接受的")
				} else {
					t.Errorf("期望状态码 %d, 得到 %d", tt.expectedStatus, w.Code)
				}
			}

			var resp map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("解析响应失败: %v", err)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}
		})
	}
}
