package services

import (
	"fmt"
	"testing"
	"time"

	"nodepass-pro/backend/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTrafficTestDB(t *testing.T) *gorm.DB {
	dsn := fmt.Sprintf("file:%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("初始化测试数据库失败: %v", err)
	}

	if err = db.AutoMigrate(
		&models.User{},
		&models.TrafficRecord{},
		&models.Tunnel{},
		&models.NodeInstance{},
	); err != nil {
		t.Fatalf("自动迁移失败: %v", err)
	}

	return db
}

func createTestUser(t *testing.T, db *gorm.DB, trafficQuota, trafficUsed int64) *models.User {
	user := &models.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hashed",
		Role:         "user",
		Status:       "normal",
		TrafficQuota: trafficQuota,
		TrafficUsed:  trafficUsed,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}
	return user
}

func TestTrafficService_GetQuota(t *testing.T) {
	db := setupTrafficTestDB(t)
	service := NewTrafficService(db)

	tests := []struct {
		name               string
		trafficQuota       int64
		trafficUsed        int64
		expectUsagePercent float64
		expectOverLimit    bool
	}{
		{
			name:               "未使用流量",
			trafficQuota:       1073741824, // 1GB
			trafficUsed:        0,
			expectUsagePercent: 0,
			expectOverLimit:    false,
		},
		{
			name:               "使用 50% 流量",
			trafficQuota:       1073741824,
			trafficUsed:        536870912, // 0.5GB
			expectUsagePercent: 50,
			expectOverLimit:    false,
		},
		{
			name:               "使用 100% 流量",
			trafficQuota:       1073741824,
			trafficUsed:        1073741824,
			expectUsagePercent: 100,
			expectOverLimit:    true,
		},
		{
			name:               "超出配额",
			trafficQuota:       1073741824,
			trafficUsed:        2147483648, // 2GB
			expectUsagePercent: 200,
			expectOverLimit:    true,
		},
		{
			name:               "配额为 0",
			trafficQuota:       0,
			trafficUsed:        0,
			expectUsagePercent: 0,
			expectOverLimit:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := createTestUser(t, db, tt.trafficQuota, tt.trafficUsed)
			defer db.Delete(user)

			result, err := service.GetQuota(user.ID)
			if err != nil {
				t.Errorf("不期望错误，但得到: %v", err)
			}

			if result.TrafficQuota != tt.trafficQuota {
				t.Errorf("TrafficQuota = %d, 期望 %d", result.TrafficQuota, tt.trafficQuota)
			}
			if result.TrafficUsed != tt.trafficUsed {
				t.Errorf("TrafficUsed = %d, 期望 %d", result.TrafficUsed, tt.trafficUsed)
			}
			if result.UsagePercent != tt.expectUsagePercent {
				t.Errorf("UsagePercent = %.2f, 期望 %.2f", result.UsagePercent, tt.expectUsagePercent)
			}
			if result.IsOverLimit != tt.expectOverLimit {
				t.Errorf("IsOverLimit = %v, 期望 %v", result.IsOverLimit, tt.expectOverLimit)
			}
		})
	}
}

func TestTrafficService_GetUsage(t *testing.T) {
	db := setupTrafficTestDB(t)
	service := NewTrafficService(db)

	user := createTestUser(t, db, 1073741824, 0)

	// 创建测试流量记录
	now := time.Now().UTC()
	records := []models.TrafficRecord{
		{
			UserID:            user.ID,
			Hour:              now.Add(-2 * time.Hour),
			TrafficIn:         1000,
			TrafficOut:        2000,
			CalculatedTraffic: 3000,
		},
		{
			UserID:            user.ID,
			Hour:              now.Add(-1 * time.Hour),
			TrafficIn:         1500,
			TrafficOut:        2500,
			CalculatedTraffic: 4000,
		},
		{
			UserID:            user.ID,
			Hour:              now,
			TrafficIn:         2000,
			TrafficOut:        3000,
			CalculatedTraffic: 5000,
		},
	}

	for _, record := range records {
		if err := db.Create(&record).Error; err != nil {
			t.Fatalf("创建流量记录失败: %v", err)
		}
	}

	tests := []struct {
		name              string
		startTime         time.Time
		endTime           time.Time
		expectTrafficIn   int64
		expectTrafficOut  int64
		expectCalculated  int64
		expectRecordCount int64
		expectError       bool
	}{
		{
			name:              "查询所有记录",
			startTime:         now.Add(-3 * time.Hour),
			endTime:           now.Add(1 * time.Hour),
			expectTrafficIn:   4500,
			expectTrafficOut:  7500,
			expectCalculated:  12000,
			expectRecordCount: 3,
			expectError:       false,
		},
		{
			name:              "查询最近 1 小时",
			startTime:         now.Add(-1 * time.Hour),
			endTime:           now.Add(1 * time.Hour),
			expectTrafficIn:   3500,
			expectTrafficOut:  5500,
			expectCalculated:  9000,
			expectRecordCount: 2,
			expectError:       false,
		},
		{
			name:        "时间范围无效",
			startTime:   now,
			endTime:     now.Add(-1 * time.Hour),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.GetUsage(user.ID, tt.startTime, tt.endTime)

			if tt.expectError {
				if err == nil {
					t.Error("期望返回错误，但没有错误")
				}
				return
			}

			if err != nil {
				t.Errorf("不期望错误，但得到: %v", err)
			}

			if result.TotalTrafficIn != tt.expectTrafficIn {
				t.Errorf("TotalTrafficIn = %d, 期望 %d", result.TotalTrafficIn, tt.expectTrafficIn)
			}
			if result.TotalTrafficOut != tt.expectTrafficOut {
				t.Errorf("TotalTrafficOut = %d, 期望 %d", result.TotalTrafficOut, tt.expectTrafficOut)
			}
			if result.TotalCalculated != tt.expectCalculated {
				t.Errorf("TotalCalculated = %d, 期望 %d", result.TotalCalculated, tt.expectCalculated)
			}
			if result.RecordCount != tt.expectRecordCount {
				t.Errorf("RecordCount = %d, 期望 %d", result.RecordCount, tt.expectRecordCount)
			}
		})
	}
}

func TestTrafficService_UpdateQuota(t *testing.T) {
	db := setupTrafficTestDB(t)
	service := NewTrafficService(db)

	// 创建管理员用户
	admin := &models.User{
		Username:     "admin",
		Email:        "admin@example.com",
		PasswordHash: "hashed",
		Role:         "admin",
		Status:       "normal",
	}
	if err := db.Create(admin).Error; err != nil {
		t.Fatalf("创建管理员失败: %v", err)
	}

	user := createTestUser(t, db, 1073741824, 0)

	tests := []struct {
		name        string
		adminID     uint
		userID      uint
		newQuota    int64
		expectError bool
		errorMsg    string
	}{
		{
			name:        "成功更新配额",
			adminID:     admin.ID,
			userID:      user.ID,
			newQuota:    2147483648, // 2GB
			expectError: false,
		},
		{
			name:        "配额为 0",
			adminID:     admin.ID,
			userID:      user.ID,
			newQuota:    0,
			expectError: false,
		},
		{
			name:        "配额为负数",
			adminID:     admin.ID,
			userID:      user.ID,
			newQuota:    -1,
			expectError: true,
			errorMsg:    "不能为负数",
		},
		{
			name:        "用户不存在",
			adminID:     admin.ID,
			userID:      99999,
			newQuota:    1073741824,
			expectError: true,
			errorMsg:    "用户不存在",
		},
		{
			name:        "非管理员操作",
			adminID:     user.ID, // 普通用户
			userID:      user.ID,
			newQuota:    2147483648,
			expectError: true,
			errorMsg:    "仅管理员",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.UpdateQuota(tt.adminID, tt.userID, tt.newQuota)

			if tt.expectError {
				if err == nil {
					t.Error("期望返回错误，但没有错误")
				}
				if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("错误信息 = %q, 期望包含 %q", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("不期望错误，但得到: %v", err)
				}

				// 验证配额已更新
				var updatedUser models.User
				if err := db.First(&updatedUser, tt.userID).Error; err != nil {
					t.Fatalf("查询用户失败: %v", err)
				}
				if updatedUser.TrafficQuota != tt.newQuota {
					t.Errorf("TrafficQuota = %d, 期望 %d", updatedUser.TrafficQuota, tt.newQuota)
				}
			}
		})
	}
}

func TestTrafficService_ResetQuota(t *testing.T) {
	db := setupTrafficTestDB(t)
	service := NewTrafficService(db)

	// 创建管理员
	admin := &models.User{
		Username:     "admin",
		Email:        "admin@example.com",
		PasswordHash: "hashed",
		Role:         "admin",
		Status:       "normal",
	}
	if err := db.Create(admin).Error; err != nil {
		t.Fatalf("创建管理员失败: %v", err)
	}

	// 创建多个用户
	users := []*models.User{
		{
			Username:     "user1",
			Email:        "user1@example.com",
			PasswordHash: "hashed",
			Role:         "user",
			Status:       "normal",
			TrafficQuota: 1073741824,
			TrafficUsed:  536870912,
		},
		{
			Username:     "user2",
			Email:        "user2@example.com",
			PasswordHash: "hashed",
			Role:         "user",
			Status:       "normal",
			TrafficQuota: 2147483648,
			TrafficUsed:  1073741824,
		},
	}

	for _, user := range users {
		if err := db.Create(user).Error; err != nil {
			t.Fatalf("创建用户失败: %v", err)
		}
	}

	tests := []struct {
		name        string
		adminID     uint
		userID      uint
		expectError bool
		errorMsg    string
	}{
		{
			name:        "重置单个用户",
			adminID:     admin.ID,
			userID:      users[0].ID,
			expectError: false,
		},
		{
			name:        "目标用户无效",
			adminID:     admin.ID,
			userID:      0,
			expectError: true,
			errorMsg:    "无效",
		},
		{
			name:        "非管理员操作",
			adminID:     users[0].ID,
			userID:      users[1].ID,
			expectError: true,
			errorMsg:    "仅管理员",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ResetQuota(tt.adminID, tt.userID)

			if tt.expectError {
				if err == nil {
					t.Error("期望返回错误，但没有错误")
				}
				if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("错误信息 = %q, 期望包含 %q", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("不期望错误，但得到: %v", err)
				}

				// 验证指定用户流量已重置
				var user models.User
				if err := db.First(&user, tt.userID).Error; err != nil {
					t.Fatalf("查询用户失败: %v", err)
				}
				if user.TrafficUsed != 0 {
					t.Errorf("TrafficUsed = %d, 期望 0", user.TrafficUsed)
				}
			}
		})
	}
}

func TestTrafficService_MonthlyReset(t *testing.T) {
	db := setupTrafficTestDB(t)
	service := NewTrafficService(db)

	// 创建测试用户
	users := []*models.User{
		{
			Username:     "user1",
			Email:        "user1@example.com",
			PasswordHash: "hashed",
			Role:         "user",
			Status:       "normal",
			TrafficQuota: 1073741824,
			TrafficUsed:  536870912,
		},
		{
			Username:     "user2",
			Email:        "user2@example.com",
			PasswordHash: "hashed",
			Role:         "user",
			Status:       "normal",
			TrafficQuota: 2147483648,
			TrafficUsed:  1073741824,
		},
		{
			Username:     "admin",
			Email:        "admin@example.com",
			PasswordHash: "hashed",
			Role:         "admin",
			Status:       "normal",
			TrafficQuota: 0,
			TrafficUsed:  0,
		},
	}

	for _, user := range users {
		if err := db.Create(user).Error; err != nil {
			t.Fatalf("创建用户失败: %v", err)
		}
	}

	// 执行月度重置
	err := service.MonthlyReset()
	if err != nil {
		t.Errorf("月度重置失败: %v", err)
	}

	// 验证所有用户流量已重置
	var allUsers []models.User
	if err := db.Find(&allUsers).Error; err != nil {
		t.Fatalf("查询用户失败: %v", err)
	}

	for _, user := range allUsers {
		if user.TrafficUsed != 0 {
			t.Errorf("用户 %s TrafficUsed = %d, 期望 0", user.Username, user.TrafficUsed)
		}
	}
}

func TestTrafficService_GetRecords(t *testing.T) {
	db := setupTrafficTestDB(t)
	service := NewTrafficService(db)

	user := createTestUser(t, db, 1073741824, 0)

	// 创建测试流量记录
	now := time.Now().UTC()
	for i := 0; i < 25; i++ {
		record := &models.TrafficRecord{
			UserID:            user.ID,
			Hour:              now.Add(time.Duration(-i) * time.Hour),
			TrafficIn:         int64(1000 * (i + 1)),
			TrafficOut:        int64(2000 * (i + 1)),
			CalculatedTraffic: int64(3000 * (i + 1)),
		}
		if err := db.Create(record).Error; err != nil {
			t.Fatalf("创建流量记录失败: %v", err)
		}
	}

	tests := []struct {
		name              string
		filters           TrafficRecordFilters
		expectRecordCount int
		expectError       bool
	}{
		{
			name: "默认分页",
			filters: TrafficRecordFilters{
				Page:     1,
				PageSize: 20,
			},
			expectRecordCount: 20,
			expectError:       false,
		},
		{
			name: "第二页",
			filters: TrafficRecordFilters{
				Page:     2,
				PageSize: 20,
			},
			expectRecordCount: 5,
			expectError:       false,
		},
		{
			name: "小页面大小",
			filters: TrafficRecordFilters{
				Page:     1,
				PageSize: 10,
			},
			expectRecordCount: 10,
			expectError:       false,
		},
		{
			name: "超大页面大小限制",
			filters: TrafficRecordFilters{
				Page:     1,
				PageSize: 300, // 应该被限制为 200
			},
			expectRecordCount: 25, // 总共只有 25 条
			expectError:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.GetRecords(user.ID, tt.filters)

			if tt.expectError {
				if err == nil {
					t.Error("期望返回错误，但没有错误")
				}
				return
			}

			if err != nil {
				t.Errorf("不期望错误，但得到: %v", err)
			}

			if len(result.List) != tt.expectRecordCount {
				t.Errorf("记录数 = %d, 期望 %d", len(result.List), tt.expectRecordCount)
			}

			if result.Total != 25 {
				t.Errorf("Total = %d, 期望 25", result.Total)
			}
		})
	}
}
