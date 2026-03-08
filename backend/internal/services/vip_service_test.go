package services

import (
	"fmt"
	"testing"
	"time"

	"nodepass-pro/backend/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupVIPTestDB(t *testing.T) *gorm.DB {
	dsn := fmt.Sprintf("file:%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("初始化测试数据库失败: %v", err)
	}

	if err = db.AutoMigrate(
		&models.User{},
		&models.VIPLevel{},
	); err != nil {
		t.Fatalf("自动迁移失败: %v", err)
	}

	// 创建默认 VIP 等级
	levels := []models.VIPLevel{
		{
			Level:                   0,
			Name:                    "免费版",
			TrafficQuota:            1073741824, // 1GB
			MaxRules:                5,
			MaxBandwidth:            100,
			MaxSelfHostedEntryNodes: 0,
			MaxSelfHostedExitNodes:  0,
		},
		{
			Level:                   1,
			Name:                    "基础版",
			TrafficQuota:            10737418240, // 10GB
			MaxRules:                20,
			MaxBandwidth:            500,
			MaxSelfHostedEntryNodes: 2,
			MaxSelfHostedExitNodes:  1,
		},
		{
			Level:                   2,
			Name:                    "专业版",
			TrafficQuota:            53687091200, // 50GB
			MaxRules:                100,
			MaxBandwidth:            1000,
			MaxSelfHostedEntryNodes: 10,
			MaxSelfHostedExitNodes:  5,
		},
	}

	for _, level := range levels {
		if err := db.Create(&level).Error; err != nil {
			t.Fatalf("创建 VIP 等级失败: %v", err)
		}
	}

	return db
}

func createVIPTestUser(t *testing.T, db *gorm.DB, vipLevel int, expiresAt *time.Time) *models.User {
	user := &models.User{
		Username:     "vipuser",
		Email:        "vip@example.com",
		PasswordHash: "hashed",
		Role:         "user",
		Status:       "normal",
		VipLevel:     vipLevel,
		VipExpiresAt: expiresAt,
		TrafficQuota: 1073741824,
		TrafficUsed:  0,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}
	return user
}

func createVIPAdmin(t *testing.T, db *gorm.DB) *models.User {
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
	return admin
}

func TestVIPService_ListLevels(t *testing.T) {
	db := setupVIPTestDB(t)
	service := NewVIPService(db)

	levels, err := service.ListLevels()
	if err != nil {
		t.Errorf("不期望错误，但得到: %v", err)
	}

	if len(levels) != 3 {
		t.Errorf("VIP 等级数量 = %d, 期望 3", len(levels))
	}

	// 验证排序（按 level 升序）
	for i := 0; i < len(levels)-1; i++ {
		if levels[i].Level >= levels[i+1].Level {
			t.Error("VIP 等级未按 level 升序排列")
		}
	}
}

func TestVIPService_CreateLevel(t *testing.T) {
	db := setupVIPTestDB(t)
	service := NewVIPService(db)
	admin := createVIPAdmin(t, db)

	description := "企业版描述"
	price := 99.99
	durationDays := 30

	tests := []struct {
		name        string
		adminID     uint
		req         *VIPLevelCreateRequest
		expectError bool
		errorMsg    string
	}{
		{
			name:    "成功创建 VIP 等级",
			adminID: admin.ID,
			req: &VIPLevelCreateRequest{
				Level:                   3,
				Name:                    "企业版",
				Description:             &description,
				TrafficQuota:            107374182400, // 100GB
				MaxRules:                500,
				MaxBandwidth:            5000,
				MaxSelfHostedEntryNodes: 50,
				MaxSelfHostedExitNodes:  20,
				Price:                   &price,
				DurationDays:            &durationDays,
			},
			expectError: false,
		},
		{
			name:        "空请求体",
			adminID:     admin.ID,
			req:         nil,
			expectError: true,
			errorMsg:    "请求体不能为空",
		},
		{
			name:    "名称为空",
			adminID: admin.ID,
			req: &VIPLevelCreateRequest{
				Level:        4,
				Name:         "",
				TrafficQuota: 1073741824,
				MaxRules:     10,
				MaxBandwidth: 100,
			},
			expectError: true,
			errorMsg:    "name",
		},
		{
			name:    "流量配额为负数",
			adminID: admin.ID,
			req: &VIPLevelCreateRequest{
				Level:        5,
				Name:         "测试版",
				TrafficQuota: -1,
				MaxRules:     10,
				MaxBandwidth: 100,
			},
			expectError: true,
			errorMsg:    "traffic_quota",
		},
		{
			name:    "重复的等级",
			adminID: admin.ID,
			req: &VIPLevelCreateRequest{
				Level:        0, // 已存在
				Name:         "重复等级",
				TrafficQuota: 1073741824,
				MaxRules:     10,
				MaxBandwidth: 100,
			},
			expectError: true,
			errorMsg:    "已存在",
		},
		{
			name: "非管理员操作",
			adminID: func() uint {
				user := createVIPTestUser(t, db, 0, nil)
				return user.ID
			}(),
			req: &VIPLevelCreateRequest{
				Level:        6,
				Name:         "测试版",
				TrafficQuota: 1073741824,
				MaxRules:     10,
				MaxBandwidth: 100,
			},
			expectError: true,
			errorMsg:    "仅管理员",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			level, err := service.CreateLevel(tt.adminID, tt.req)

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
				if level == nil {
					t.Error("VIP 等级为 nil")
				} else {
					if level.Level != tt.req.Level {
						t.Errorf("Level = %d, 期望 %d", level.Level, tt.req.Level)
					}
					if level.Name != tt.req.Name {
						t.Errorf("Name = %q, 期望 %q", level.Name, tt.req.Name)
					}
					if level.TrafficQuota != tt.req.TrafficQuota {
						t.Errorf("TrafficQuota = %d, 期望 %d", level.TrafficQuota, tt.req.TrafficQuota)
					}
				}
			}
		})
	}
}

func TestVIPService_UpdateLevel(t *testing.T) {
	db := setupVIPTestDB(t)
	service := NewVIPService(db)
	admin := createVIPAdmin(t, db)

	// 获取一个现有等级
	var existingLevel models.VIPLevel
	if err := db.First(&existingLevel, "level = ?", 1).Error; err != nil {
		t.Fatalf("查询现有等级失败: %v", err)
	}

	newName := "基础版 Plus"
	newQuota := int64(21474836480) // 20GB
	newMaxRules := 30

	tests := []struct {
		name        string
		adminID     uint
		levelID     uint
		req         *VIPLevelUpdateRequest
		expectError bool
		errorMsg    string
	}{
		{
			name:    "成功更新 VIP 等级",
			adminID: admin.ID,
			levelID: existingLevel.ID,
			req: &VIPLevelUpdateRequest{
				Name:         &newName,
				TrafficQuota: &newQuota,
				MaxRules:     &newMaxRules,
			},
			expectError: false,
		},
		{
			name:        "空请求体",
			adminID:     admin.ID,
			levelID:     existingLevel.ID,
			req:         nil,
			expectError: true,
			errorMsg:    "请求体不能为空",
		},
		{
			name:    "等级不存在",
			adminID: admin.ID,
			levelID: 99999,
			req: &VIPLevelUpdateRequest{
				Name: &newName,
			},
			expectError: true,
			errorMsg:    "不存在",
		},
		{
			name:    "流量配额为负数",
			adminID: admin.ID,
			levelID: existingLevel.ID,
			req: &VIPLevelUpdateRequest{
				TrafficQuota: func() *int64 { v := int64(-1); return &v }(),
			},
			expectError: true,
			errorMsg:    "不能为负数",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			level, err := service.UpdateLevel(tt.adminID, tt.levelID, tt.req)

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
				if level == nil {
					t.Error("VIP 等级为 nil")
				}
			}
		})
	}
}

func TestVIPService_GetMyLevel(t *testing.T) {
	db := setupVIPTestDB(t)
	service := NewVIPService(db)

	futureTime := time.Now().Add(30 * 24 * time.Hour)
	user := createVIPTestUser(t, db, 1, &futureTime)

	result, err := service.GetMyLevel(user.ID)
	if err != nil {
		t.Errorf("不期望错误，但得到: %v", err)
	}

	if result == nil {
		t.Fatal("结果为 nil")
	}

	if result.UserID != user.ID {
		t.Errorf("UserID = %d, 期望 %d", result.UserID, user.ID)
	}
	if result.VIPLevel != 1 {
		t.Errorf("VIPLevel = %d, 期望 1", result.VIPLevel)
	}
	if result.VIPExpiresAt == nil {
		t.Error("VIPExpiresAt 为 nil")
	}
	if result.LevelDetail == nil {
		t.Error("LevelDetail 为 nil")
	} else {
		if result.LevelDetail.Level != 1 {
			t.Errorf("LevelDetail.Level = %d, 期望 1", result.LevelDetail.Level)
		}
	}
}

func TestVIPService_UpgradeUser(t *testing.T) {
	db := setupVIPTestDB(t)
	service := NewVIPService(db)
	admin := createVIPAdmin(t, db)

	user := createVIPTestUser(t, db, 0, nil)

	tests := []struct {
		name         string
		adminID      uint
		userID       uint
		targetLevel  int
		durationDays int
		expectError  bool
		errorMsg     string
	}{
		{
			name:         "成功升级到基础版",
			adminID:      admin.ID,
			userID:       user.ID,
			targetLevel:  1,
			durationDays: 30,
			expectError:  false,
		},
		{
			name:         "升级到不存在的等级",
			adminID:      admin.ID,
			userID:       user.ID,
			targetLevel:  99,
			durationDays: 30,
			expectError:  true,
			errorMsg:     "不存在",
		},
		{
			name:         "用户不存在",
			adminID:      admin.ID,
			userID:       99999,
			targetLevel:  1,
			durationDays: 30,
			expectError:  true,
			errorMsg:     "用户不存在",
		},
		{
			name:         "非管理员操作",
			adminID:      user.ID,
			userID:       user.ID,
			targetLevel:  2,
			durationDays: 30,
			expectError:  true,
			errorMsg:     "仅管理员",
		},
		{
			name:         "持续天数为负数",
			adminID:      admin.ID,
			userID:       user.ID,
			targetLevel:  1,
			durationDays: -1,
			expectError:  true,
			errorMsg:     "durationDays",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.UpgradeUser(tt.adminID, tt.userID, tt.targetLevel, tt.durationDays)

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

				// 验证用户已升级
				var updatedUser models.User
				if err := db.First(&updatedUser, tt.userID).Error; err != nil {
					t.Fatalf("查询用户失败: %v", err)
				}

				if updatedUser.VipLevel != tt.targetLevel {
					t.Errorf("VipLevel = %d, 期望 %d", updatedUser.VipLevel, tt.targetLevel)
				}

				if updatedUser.VipExpiresAt == nil {
					t.Error("VipExpiresAt 为 nil")
				} else {
					expectedExpiry := time.Now().Add(time.Duration(tt.durationDays) * 24 * time.Hour)
					diff := updatedUser.VipExpiresAt.Sub(expectedExpiry)
					if diff < -time.Minute || diff > time.Minute {
						t.Errorf("VipExpiresAt 时间不正确，差异: %v", diff)
					}
				}
			}
		})
	}
}

func TestVIPService_CheckExpiration(t *testing.T) {
	db := setupVIPTestDB(t)
	service := NewVIPService(db)

	// 创建测试用户
	pastTime := time.Now().Add(-1 * time.Hour)
	futureTime := time.Now().Add(24 * time.Hour)

	users := []*models.User{
		{
			Username:     "expired1",
			Email:        "expired1@example.com",
			PasswordHash: "hashed",
			Role:         "user",
			Status:       "normal",
			VipLevel:     1,
			VipExpiresAt: &pastTime,
			TrafficQuota: 10737418240,
		},
		{
			Username:     "expired2",
			Email:        "expired2@example.com",
			PasswordHash: "hashed",
			Role:         "user",
			Status:       "normal",
			VipLevel:     2,
			VipExpiresAt: &pastTime,
			TrafficQuota: 53687091200,
		},
		{
			Username:     "active",
			Email:        "active@example.com",
			PasswordHash: "hashed",
			Role:         "user",
			Status:       "normal",
			VipLevel:     1,
			VipExpiresAt: &futureTime,
			TrafficQuota: 10737418240,
		},
		{
			Username:     "free",
			Email:        "free@example.com",
			PasswordHash: "hashed",
			Role:         "user",
			Status:       "normal",
			VipLevel:     0,
			VipExpiresAt: nil,
			TrafficQuota: 1073741824,
		},
	}

	for _, user := range users {
		if err := db.Create(user).Error; err != nil {
			t.Fatalf("创建用户失败: %v", err)
		}
	}

	// 执行到期检查
	affected, err := service.CheckExpiration()
	if err != nil {
		t.Errorf("到期检查失败: %v", err)
	}

	// 应该有 2 个用户被降级
	if affected != 2 {
		t.Errorf("降级用户数 = %d, 期望 2", affected)
	}

	// 验证过期用户已降级到免费版
	var expiredUsers []models.User
	if err := db.Where("username IN ?", []string{"expired1", "expired2"}).Find(&expiredUsers).Error; err != nil {
		t.Fatalf("查询用户失败: %v", err)
	}

	for _, user := range expiredUsers {
		if user.VipLevel != 0 {
			t.Errorf("用户 %s VipLevel = %d, 期望 0", user.Username, user.VipLevel)
		}
		if user.TrafficQuota != 1073741824 {
			t.Errorf("用户 %s TrafficQuota = %d, 期望 1073741824", user.Username, user.TrafficQuota)
		}
	}

	// 验证未过期用户未受影响
	var activeUser models.User
	if err := db.Where("username = ?", "active").First(&activeUser).Error; err != nil {
		t.Fatalf("查询用户失败: %v", err)
	}
	if activeUser.VipLevel != 1 {
		t.Errorf("活跃用户 VipLevel = %d, 期望 1", activeUser.VipLevel)
	}
}

func TestVIPService_GetLevelByLevel(t *testing.T) {
	db := setupVIPTestDB(t)
	service := NewVIPService(db)

	tests := []struct {
		name        string
		level       int
		expectError bool
		expectName  string
	}{
		{
			name:        "获取免费版",
			level:       0,
			expectError: false,
			expectName:  "免费版",
		},
		{
			name:        "获取基础版",
			level:       1,
			expectError: false,
			expectName:  "基础版",
		},
		{
			name:        "获取专业版",
			level:       2,
			expectError: false,
			expectName:  "专业版",
		},
		{
			name:        "等级不存在",
			level:       99,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			level, err := service.getLevelByLevel(tt.level)

			if tt.expectError {
				if err == nil {
					t.Error("期望返回错误，但没有错误")
				}
			} else {
				if err != nil {
					t.Errorf("不期望错误，但得到: %v", err)
				}
				if level == nil {
					t.Error("VIP 等级为 nil")
				} else {
					if level.Name != tt.expectName {
						t.Errorf("Name = %q, 期望 %q", level.Name, tt.expectName)
					}
				}
			}
		})
	}
}
