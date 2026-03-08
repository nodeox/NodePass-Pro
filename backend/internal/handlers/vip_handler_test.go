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

func setupVIPHandlerTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("无法连接数据库: %v", err)
	}

	if err := db.AutoMigrate(
		&models.User{},
		&models.VIPLevel{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}

	return db
}

func setupVIPHandlerTestRouter(db *gorm.DB) *gin.Engine {
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

	handler := NewVIPHandler(db)

	api := router.Group("/api/v1")
	{
		vip := api.Group("/vip")
		{
			vip.GET("/levels", handler.ListLevels)
			vip.POST("/levels", handler.CreateLevel)
			vip.PUT("/levels/:id", handler.UpdateLevel)
			vip.GET("/my-level", handler.GetMyLevel)
		}
		api.POST("/users/:id/vip/upgrade", handler.UpgradeUser)
	}

	return router
}

func createTestVIPUser(t *testing.T, db *gorm.DB, role string) *models.User {
	user := &models.User{
		Username:     fmt.Sprintf("testuser_%d", time.Now().UnixNano()),
		Email:        fmt.Sprintf("test_%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hashed_password",
		Role:         role,
		Status:       "active",
		VipLevel:     0,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}
	return user
}

func createTestVIPLevel(t *testing.T, db *gorm.DB, level int, name string) *models.VIPLevel {
	price := 99.0
	durationDays := 30
	vipLevel := &models.VIPLevel{
		Level:                   level,
		Name:                    name,
		TrafficQuota:            100 * 1024 * 1024 * 1024, // 100GB
		MaxRules:                50,
		MaxBandwidth:            100,
		MaxSelfHostedEntryNodes: 5,
		MaxSelfHostedExitNodes:  5,
		AccessibleNodeLevel:     1,
		TrafficMultiplier:       1.0,
		Price:                   &price,
		DurationDays:            &durationDays,
	}
	if err := db.Create(vipLevel).Error; err != nil {
		t.Fatalf("创建测试 VIP 等级失败: %v", err)
	}
	return vipLevel
}

func TestVIPHandler_ListLevels(t *testing.T) {
	db := setupVIPHandlerTestDB(t)
	router := setupVIPHandlerTestRouter(db)

	// 创建测试 VIP 等级
	createTestVIPLevel(t, db, 1, "VIP1")
	createTestVIPLevel(t, db, 2, "VIP2")

	tests := []struct {
		name           string
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name:           "成功获取 VIP 等级列表",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if !resp["success"].(bool) {
					t.Error("期望 success 为 true")
				}
				data := resp["data"].(map[string]interface{})
				list := data["list"].([]interface{})
				if len(list) != 2 {
					t.Errorf("期望 2 个 VIP 等级, 得到 %d", len(list))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/vip/levels", nil)

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

func TestVIPHandler_CreateLevel(t *testing.T) {
	db := setupVIPHandlerTestDB(t)
	router := setupVIPHandlerTestRouter(db)

	admin := createTestVIPUser(t, db, "admin")
	user := createTestVIPUser(t, db, "user")

	tests := []struct {
		name           string
		userID         uint
		role           string
		payload        interface{}
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name:   "管理员成功创建 VIP 等级",
			userID: admin.ID,
			role:   "admin",
			payload: map[string]interface{}{
				"level":                       1,
				"name":                        "VIP1",
				"traffic_quota":               107374182400,
				"max_rules":                   50,
				"max_bandwidth":               1000,
				"max_self_hosted_entry_nodes": 5,
				"max_self_hosted_exit_nodes":  5,
				"accessible_node_level":       2,
				"traffic_multiplier":          1.0,
				"price":                       99.0,
				"duration_days":               30,
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if !resp["success"].(bool) {
					t.Error("期望 success 为 true")
				}
				data := resp["data"].(map[string]interface{})
				if data["name"] != "VIP1" {
					t.Errorf("期望 name 为 VIP1, 得到 %v", data["name"])
				}
			},
		},
		{
			name:           "普通用户无权创建 VIP 等级",
			userID:         user.ID,
			role:           "user",
			payload:        map[string]interface{}{},
			expectedStatus: http.StatusForbidden,
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
		{
			name:           "缺少必填字段",
			userID:         admin.ID,
			role:           "admin",
			payload:        map[string]interface{}{},
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
			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/vip/levels", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			if tt.userID > 0 {
				req.Header.Set("X-User-ID", fmt.Sprintf("%d", tt.userID))
				req.Header.Set("X-User-Role", tt.role)
			}

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

func TestVIPHandler_UpdateLevel(t *testing.T) {
	db := setupVIPHandlerTestDB(t)
	router := setupVIPHandlerTestRouter(db)

	admin := createTestVIPUser(t, db, "admin")
	user := createTestVIPUser(t, db, "user")
	vipLevel := createTestVIPLevel(t, db, 1, "VIP1")

	tests := []struct {
		name           string
		userID         uint
		role           string
		levelID        uint
		payload        interface{}
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name:    "管理员成功更新 VIP 等级",
			userID:  admin.ID,
			role:    "admin",
			levelID: vipLevel.ID,
			payload: map[string]interface{}{
				"name":          "VIP1_Updated",
				"max_rules":     20,
				"max_bandwidth": 200,
				"price":         199.0,
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if !resp["success"].(bool) {
					t.Error("期望 success 为 true")
				}
				data := resp["data"].(map[string]interface{})
				if data["name"] != "VIP1_Updated" {
					t.Errorf("期望 name 为 VIP1_Updated, 得到 %v", data["name"])
				}
			},
		},
		{
			name:           "普通用户无权更新 VIP 等级",
			userID:         user.ID,
			role:           "user",
			levelID:        vipLevel.ID,
			payload:        map[string]interface{}{},
			expectedStatus: http.StatusForbidden,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if resp["success"].(bool) {
					t.Error("期望 success 为 false")
				}
			},
		},
		{
			name:           "VIP 等级不存在",
			userID:         admin.ID,
			role:           "admin",
			levelID:        99999,
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
			url := fmt.Sprintf("/api/v1/vip/levels/%d", tt.levelID)
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

func TestVIPHandler_GetMyLevel(t *testing.T) {
	db := setupVIPHandlerTestDB(t)
	router := setupVIPHandlerTestRouter(db)

	user := createTestVIPUser(t, db, "user")
	vipLevel := createTestVIPLevel(t, db, 1, "VIP1")

	// 设置用户 VIP 等级
	user.VipLevel = vipLevel.Level
	db.Save(user)

	tests := []struct {
		name           string
		userID         uint
		role           string
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name:           "成功获取用户 VIP 等级",
			userID:         user.ID,
			role:           "user",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if !resp["success"].(bool) {
					t.Error("期望 success 为 true")
				}
			},
		},
		{
			name:           "未认证用户",
			userID:         0,
			role:           "",
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
			req := httptest.NewRequest(http.MethodGet, "/api/v1/vip/my-level", nil)
			if tt.userID > 0 {
				req.Header.Set("X-User-ID", fmt.Sprintf("%d", tt.userID))
				req.Header.Set("X-User-Role", tt.role)
			}

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

func TestVIPHandler_UpgradeUser(t *testing.T) {
	db := setupVIPHandlerTestDB(t)
	router := setupVIPHandlerTestRouter(db)

	admin := createTestVIPUser(t, db, "admin")
	user := createTestVIPUser(t, db, "user")
	targetUser := createTestVIPUser(t, db, "user")
	vipLevel := createTestVIPLevel(t, db, 1, "VIP1")

	tests := []struct {
		name           string
		userID         uint
		role           string
		targetUserID   uint
		payload        interface{}
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name:         "管理员成功升级用户 VIP",
			userID:       admin.ID,
			role:         "admin",
			targetUserID: targetUser.ID,
			payload: map[string]interface{}{
				"level":         vipLevel.Level,
				"duration_days": 30,
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if !resp["success"].(bool) {
					t.Error("期望 success 为 true")
				}
			},
		},
		{
			name:         "普通用户无权升级 VIP",
			userID:       user.ID,
			role:         "user",
			targetUserID: targetUser.ID,
			payload: map[string]interface{}{
				"level":         1,
				"duration_days": 30,
			},
			expectedStatus: http.StatusForbidden,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if resp["success"].(bool) {
					t.Error("期望 success 为 false")
				}
			},
		},
		{
			name:           "缺少必填字段",
			userID:         admin.ID,
			role:           "admin",
			targetUserID:   targetUser.ID,
			payload:        map[string]interface{}{},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if resp["success"].(bool) {
					t.Error("期望 success 为 false")
				}
			},
		},
		{
			name:         "目标用户不存在",
			userID:       admin.ID,
			role:         "admin",
			targetUserID: 99999,
			payload: map[string]interface{}{
				"level":         1,
				"duration_days": 30,
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
			body, _ := json.Marshal(tt.payload)
			url := fmt.Sprintf("/api/v1/users/%d/vip/upgrade", tt.targetUserID)
			req := httptest.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
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
