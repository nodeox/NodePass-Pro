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

func setupNodeGroupHandlerTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("无法连接数据库: %v", err)
	}

	if err := db.AutoMigrate(
		&models.User{},
		&models.NodeGroup{},
		&models.NodeGroupRelation{},
		&models.Node{},
		&models.NodeGroupStats{},
		&models.NodeInstance{},
		&models.Tunnel{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}

	return db
}

func setupNodeGroupHandlerTestRouter(db *gorm.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// 添加模拟认证中间件
	router.Use(func(c *gin.Context) {
		if userID := c.GetHeader("X-User-ID"); userID != "" {
			if parsed, err := strconv.ParseUint(userID, 10, 64); err == nil && parsed > 0 {
				id := uint(parsed)
				c.Set("user_id", id)
				c.Set("userID", id)
			}
		}
		c.Next()
	})

	handler := NewNodeGroupHandler(db)

	api := router.Group("/api/v1")
	{
		groups := api.Group("/node-groups")
		{
			groups.POST("", handler.Create)
			groups.GET("", handler.List)
			groups.GET("/accessible-nodes", handler.ListAccessibleNodes)
			groups.GET("/:id", handler.Get)
			groups.PUT("/:id", handler.Update)
			groups.DELETE("/:id", handler.Delete)
			groups.POST("/:id/toggle", handler.Toggle)
			groups.GET("/:id/stats", handler.GetStats)
			groups.POST("/:id/generate-deploy-command", handler.GenerateDeployCommand)
			groups.GET("/:id/nodes", handler.ListNodes)
			groups.POST("/:id/nodes", handler.AddNode)
			groups.POST("/:id/relations", handler.CreateRelation)
			groups.GET("/:id/relations", handler.ListRelations)
		}
		api.DELETE("/node-group-relations/:id", handler.DeleteRelation)
		api.POST("/node-group-relations/:id/toggle", handler.ToggleRelation)
	}

	return router
}

func createTestNodeGroupUser(t *testing.T, db *gorm.DB, role string) *models.User {
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

func createTestGroup(t *testing.T, db *gorm.DB, userID uint, name string, isPublic bool) *models.NodeGroup {
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

func TestNodeGroupHandler_Create(t *testing.T) {
	db := setupNodeGroupHandlerTestDB(t)
	router := setupNodeGroupHandlerTestRouter(db)

	user := createTestNodeGroupUser(t, db, "user")

	tests := []struct {
		name           string
		userID         uint
		payload        interface{}
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name:   "成功创建节点组",
			userID: user.ID,
			payload: map[string]interface{}{
				"name":        "test_group",
				"description": "Test description",
				"type":        "entry",
				"is_public":   false,
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if !resp["success"].(bool) {
					t.Error("期望 success 为 true")
				}
				data := resp["data"].(map[string]interface{})
				if data["name"] != "test_group" {
					t.Errorf("期望 name 为 test_group, 得到 %v", data["name"])
				}
			},
		},
		{
			name:           "缺少必填字段",
			userID:         user.ID,
			payload:        map[string]interface{}{},
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
			req := httptest.NewRequest(http.MethodPost, "/api/v1/node-groups", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			if tt.userID > 0 {
				req.Header.Set("X-User-ID", fmt.Sprintf("%d", tt.userID))
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

func TestNodeGroupHandler_List(t *testing.T) {
	db := setupNodeGroupHandlerTestDB(t)
	router := setupNodeGroupHandlerTestRouter(db)

	user := createTestNodeGroupUser(t, db, "user")

	// 创建测试节点组
	createTestGroup(t, db, user.ID, "group1", false)
	createTestGroup(t, db, user.ID, "group2", true)

	tests := []struct {
		name           string
		userID         uint
		query          string
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name:           "获取节点组列表",
			userID:         user.ID,
			query:          "",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if !resp["success"].(bool) {
					t.Error("期望 success 为 true")
				}
				data := resp["data"].(map[string]interface{})
				items := data["items"].([]interface{})
				if len(items) != 2 {
					t.Errorf("期望 2 个节点组, 得到 %d", len(items))
				}
			},
		},
		{
			name:           "分页查询",
			userID:         user.ID,
			query:          "?page=1&page_size=1",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				data := resp["data"].(map[string]interface{})
				items := data["items"].([]interface{})
				if len(items) != 1 {
					t.Errorf("期望 1 个节点组, 得到 %d", len(items))
				}
			},
		},
		{
			name:           "按类型过滤",
			userID:         user.ID,
			query:          "?type=entry",
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
			req := httptest.NewRequest(http.MethodGet, "/api/v1/node-groups"+tt.query, nil)
			req.Header.Set("X-User-ID", fmt.Sprintf("%d", tt.userID))

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

func TestNodeGroupHandler_Get(t *testing.T) {
	db := setupNodeGroupHandlerTestDB(t)
	router := setupNodeGroupHandlerTestRouter(db)

	user := createTestNodeGroupUser(t, db, "user")
	otherUser := createTestNodeGroupUser(t, db, "user")

	group := createTestGroup(t, db, user.ID, "test_group", false)

	tests := []struct {
		name           string
		userID         uint
		groupID        uint
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name:           "成功获取节点组详情",
			userID:         user.ID,
			groupID:        group.ID,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if !resp["success"].(bool) {
					t.Error("期望 success 为 true")
				}
				data := resp["data"].(map[string]interface{})
				if uint(data["id"].(float64)) != group.ID {
					t.Errorf("期望 id 为 %d, 得到 %v", group.ID, data["id"])
				}
			},
		},
		{
			name:           "普通用户无法访问其他用户的私有节点组",
			userID:         otherUser.ID,
			groupID:        group.ID,
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if resp["success"].(bool) {
					t.Error("期望 success 为 false")
				}
			},
		},
		{
			name:           "节点组不存在",
			userID:         user.ID,
			groupID:        99999,
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
			url := fmt.Sprintf("/api/v1/node-groups/%d", tt.groupID)
			req := httptest.NewRequest(http.MethodGet, url, nil)
			req.Header.Set("X-User-ID", fmt.Sprintf("%d", tt.userID))

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

func TestNodeGroupHandler_Update(t *testing.T) {
	db := setupNodeGroupHandlerTestDB(t)
	router := setupNodeGroupHandlerTestRouter(db)

	user := createTestNodeGroupUser(t, db, "user")
	otherUser := createTestNodeGroupUser(t, db, "user")

	group := createTestGroup(t, db, user.ID, "test_group", false)

	tests := []struct {
		name           string
		userID         uint
		groupID        uint
		payload        interface{}
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name:    "成功更新节点组",
			userID:  user.ID,
			groupID: group.ID,
			payload: map[string]interface{}{
				"name":        "updated_group",
				"description": "Updated description",
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if !resp["success"].(bool) {
					t.Error("期望 success 为 true")
				}
				data := resp["data"].(map[string]interface{})
				if data["name"] != "updated_group" {
					t.Errorf("期望 name 为 updated_group, 得到 %v", data["name"])
				}
			},
		},
		{
			name:    "普通用户无法更新其他用户的节点组",
			userID:  otherUser.ID,
			groupID: group.ID,
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
			name:           "节点组不存在",
			userID:         user.ID,
			groupID:        99999,
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
			url := fmt.Sprintf("/api/v1/node-groups/%d", tt.groupID)
			req := httptest.NewRequest(http.MethodPut, url, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-User-ID", fmt.Sprintf("%d", tt.userID))

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

func TestNodeGroupHandler_Delete(t *testing.T) {
	db := setupNodeGroupHandlerTestDB(t)
	router := setupNodeGroupHandlerTestRouter(db)

	user := createTestNodeGroupUser(t, db, "user")
	otherUser := createTestNodeGroupUser(t, db, "user")

	tests := []struct {
		name           string
		userID         uint
		setupGroup     func() uint
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name:   "成功删除节点组",
			userID: user.ID,
			setupGroup: func() uint {
				return createTestGroup(t, db, user.ID, "to_delete", false).ID
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if !resp["success"].(bool) {
					t.Error("期望 success 为 true")
				}
			},
		},
		{
			name:   "普通用户无法删除其他用户的节点组",
			userID: otherUser.ID,
			setupGroup: func() uint {
				return createTestGroup(t, db, user.ID, "protected", false).ID
			},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if resp["success"].(bool) {
					t.Error("期望 success 为 false")
				}
			},
		},
		{
			name:   "节点组不存在",
			userID: user.ID,
			setupGroup: func() uint {
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
			groupID := tt.setupGroup()
			url := fmt.Sprintf("/api/v1/node-groups/%d", groupID)
			req := httptest.NewRequest(http.MethodDelete, url, nil)
			req.Header.Set("X-User-ID", fmt.Sprintf("%d", tt.userID))

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

func TestNodeGroupHandler_Toggle(t *testing.T) {
	db := setupNodeGroupHandlerTestDB(t)
	router := setupNodeGroupHandlerTestRouter(db)

	user := createTestNodeGroupUser(t, db, "user")
	group := createTestGroup(t, db, user.ID, "test_group", false)

	tests := []struct {
		name           string
		userID         uint
		groupID        uint
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name:           "切换节点组状态",
			userID:         user.ID,
			groupID:        group.ID,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if !resp["success"].(bool) {
					t.Error("期望 success 为 true")
				}
			},
		},
		{
			name:           "节点组不存在",
			userID:         user.ID,
			groupID:        99999,
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
			url := fmt.Sprintf("/api/v1/node-groups/%d/toggle", tt.groupID)
			req := httptest.NewRequest(http.MethodPost, url, nil)
			req.Header.Set("X-User-ID", fmt.Sprintf("%d", tt.userID))

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

func TestNodeGroupHandler_CreateRelation(t *testing.T) {
	db := setupNodeGroupHandlerTestDB(t)
	router := setupNodeGroupHandlerTestRouter(db)

	user := createTestNodeGroupUser(t, db, "user")
	entryGroup := createTestGroup(t, db, user.ID, "entry_group", false)
	exitGroup := createTestGroup(t, db, user.ID, "exit_group", false)
	exitGroup.Type = models.NodeGroupTypeExit
	if err := db.Save(exitGroup).Error; err != nil {
		t.Fatalf("更新出口组类型失败: %v", err)
	}

	tests := []struct {
		name           string
		userID         uint
		entryGroupID   uint
		payload        interface{}
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name:         "成功创建节点组关联",
			userID:       user.ID,
			entryGroupID: entryGroup.ID,
			payload: map[string]interface{}{
				"exit_group_id": exitGroup.ID,
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if !resp["success"].(bool) {
					t.Error("期望 success 为 true")
				}
			},
		},
		{
			name:           "缺少 exit_group_id",
			userID:         user.ID,
			entryGroupID:   entryGroup.ID,
			payload:        map[string]interface{}{},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if resp["success"].(bool) {
					t.Error("期望 success 为 false")
				}
			},
		},
		{
			name:         "exit_group_id 为 0",
			userID:       user.ID,
			entryGroupID: entryGroup.ID,
			payload: map[string]interface{}{
				"exit_group_id": 0,
			},
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
			url := fmt.Sprintf("/api/v1/node-groups/%d/relations", tt.entryGroupID)
			req := httptest.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-User-ID", fmt.Sprintf("%d", tt.userID))

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
