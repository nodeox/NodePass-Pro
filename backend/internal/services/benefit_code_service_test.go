package services

import (
	"fmt"
	"testing"
	"time"

	"nodepass-pro/backend/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupBenefitCodeTestDB(t *testing.T) *gorm.DB {
	dsn := fmt.Sprintf("file:%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("初始化测试数据库失败: %v", err)
	}

	if err = db.AutoMigrate(
		&models.User{},
		&models.VIPLevel{},
		&models.BenefitCode{},
	); err != nil {
		t.Fatalf("自动迁移失败: %v", err)
	}

	// 创建 VIP 等级
	levels := []models.VIPLevel{
		{Level: 0, Name: "免费版", TrafficQuota: 1073741824, MaxRules: 5, MaxBandwidth: 100},
		{Level: 1, Name: "基础版", TrafficQuota: 10737418240, MaxRules: 20, MaxBandwidth: 500},
		{Level: 2, Name: "专业版", TrafficQuota: 53687091200, MaxRules: 100, MaxBandwidth: 1000},
	}
	for _, level := range levels {
		if err := db.Create(&level).Error; err != nil {
			t.Fatalf("创建 VIP 等级失败: %v", err)
		}
	}

	return db
}

func createBenefitCodeAdmin(t *testing.T, db *gorm.DB) *models.User {
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

func createBenefitCodeUser(t *testing.T, db *gorm.DB) *models.User {
	user := &models.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hashed",
		Role:         "user",
		Status:       "normal",
		VipLevel:     0,
		TrafficQuota: 1073741824,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	return user
}

func TestBenefitCodeService_Generate(t *testing.T) {
	db := setupBenefitCodeTestDB(t)
	service := NewBenefitCodeService(db)
	admin := createBenefitCodeAdmin(t, db)

	futureTime := time.Now().Add(30 * 24 * time.Hour)

	tests := []struct {
		name         string
		adminID      uint
		vipLevel     int
		durationDays int
		count        int
		expiresAt    *time.Time
		expectError  bool
		errorMsg     string
	}{
		{
			name:         "成功生成 10 个权益码",
			adminID:      admin.ID,
			vipLevel:     1,
			durationDays: 30,
			count:        10,
			expiresAt:    &futureTime,
			expectError:  false,
		},
		{
			name:         "生成 1 个权益码",
			adminID:      admin.ID,
			vipLevel:     2,
			durationDays: 90,
			count:        1,
			expiresAt:    nil,
			expectError:  false,
		},
		{
			name:         "数量为 0",
			adminID:      admin.ID,
			vipLevel:     1,
			durationDays: 30,
			count:        0,
			expectError:  true,
			errorMsg:     "必须大于 0",
		},
		{
			name:         "数量超过限制",
			adminID:      admin.ID,
			vipLevel:     1,
			durationDays: 30,
			count:        1001,
			expectError:  true,
			errorMsg:     "最多生成 1000",
		},
		{
			name:         "持续天数为 0",
			adminID:      admin.ID,
			vipLevel:     1,
			durationDays: 0,
			count:        10,
			expectError:  true,
			errorMsg:     "必须大于 0",
		},
		{
			name:         "VIP 等级不存在",
			adminID:      admin.ID,
			vipLevel:     99,
			durationDays: 30,
			count:        10,
			expectError:  true,
			errorMsg:     "不存在",
		},
		{
			name:         "过期时间早于当前时间",
			adminID:      admin.ID,
			vipLevel:     1,
			durationDays: 30,
			count:        10,
			expiresAt:    func() *time.Time { t := time.Now().Add(-1 * time.Hour); return &t }(),
			expectError:  true,
			errorMsg:     "不能早于当前时间",
		},
		{
			name:         "非管理员操作",
			adminID:      createBenefitCodeUser(t, db).ID,
			vipLevel:     1,
			durationDays: 30,
			count:        10,
			expectError:  true,
			errorMsg:     "仅管理员",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codes, err := service.Generate(tt.adminID, tt.vipLevel, tt.durationDays, tt.count, tt.expiresAt)

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
				if len(codes) != tt.count {
					t.Errorf("生成权益码数量 = %d, 期望 %d", len(codes), tt.count)
				}

				// 验证权益码格式和唯一性
				seen := make(map[string]bool)
				for _, code := range codes {
					if code.Code == "" {
						t.Error("权益码为空")
					}
					if len(code.Code) != 17 {
						t.Errorf("权益码长度 = %d, 期望 17", len(code.Code))
					}
					if seen[code.Code] {
						t.Errorf("权益码重复: %s", code.Code)
					}
					seen[code.Code] = true

					if code.VipLevel != tt.vipLevel {
						t.Errorf("VipLevel = %d, 期望 %d", code.VipLevel, tt.vipLevel)
					}
					if code.DurationDays != tt.durationDays {
						t.Errorf("DurationDays = %d, 期望 %d", code.DurationDays, tt.durationDays)
					}
					if code.Status != "unused" {
						t.Errorf("Status = %q, 期望 %q", code.Status, "unused")
					}
					if !code.IsEnabled {
						t.Error("IsEnabled 应该为 true")
					}
				}
			}
		})
	}
}

func TestBenefitCodeService_Redeem(t *testing.T) {
	db := setupBenefitCodeTestDB(t)
	service := NewBenefitCodeService(db)
	admin := createBenefitCodeAdmin(t, db)
	user := createBenefitCodeUser(t, db)

	// 生成测试权益码
	validCodes, err := service.Generate(admin.ID, 1, 30, 3, nil)
	if err != nil {
		t.Fatalf("生成权益码失败: %v", err)
	}

	// 创建已使用的权益码
	usedCode := &models.BenefitCode{
		Code:         "USED-CODE-12345",
		VipLevel:     1,
		DurationDays: 30,
		Status:       "used",
		IsEnabled:    true,
		UsedBy:       &user.ID,
		UsedAt:       func() *time.Time { t := time.Now(); return &t }(),
	}
	if err := db.Create(usedCode).Error; err != nil {
		t.Fatalf("创建已使用权益码失败: %v", err)
	}

	// 创建已过期的权益码
	expiredTime := time.Now().Add(-1 * time.Hour)
	expiredCode := &models.BenefitCode{
		Code:         "EXPIRED-CODE-123",
		VipLevel:     1,
		DurationDays: 30,
		Status:       "unused",
		IsEnabled:    true,
		ExpiresAt:    &expiredTime,
	}
	if err := db.Create(expiredCode).Error; err != nil {
		t.Fatalf("创建过期权益码失败: %v", err)
	}

	// 创建已禁用的权益码
	disabledCode := &models.BenefitCode{
		Code:         "DISABLED-CODE-12",
		VipLevel:     1,
		DurationDays: 30,
		Status:       "unused",
		IsEnabled:    false,
	}
	if err := db.Create(disabledCode).Error; err != nil {
		t.Fatalf("创建禁用权益码失败: %v", err)
	}
	if err := db.Model(disabledCode).Update("is_enabled", false).Error; err != nil {
		t.Fatalf("设置禁用权益码状态失败: %v", err)
	}

	tests := []struct {
		name        string
		userID      uint
		code        string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "成功兑换权益码",
			userID:      user.ID,
			code:        validCodes[0].Code,
			expectError: false,
		},
		{
			name:        "权益码不存在",
			userID:      user.ID,
			code:        "NONEXISTENT-CODE",
			expectError: true,
			errorMsg:    "权益码不存在",
		},
		{
			name:        "权益码已使用",
			userID:      user.ID,
			code:        usedCode.Code,
			expectError: true,
			errorMsg:    "权益码已使用",
		},
		{
			name:        "权益码已过期",
			userID:      user.ID,
			code:        expiredCode.Code,
			expectError: true,
			errorMsg:    "已过期",
		},
		{
			name:        "权益码已禁用",
			userID:      user.ID,
			code:        disabledCode.Code,
			expectError: true,
			errorMsg:    "已禁用",
		},
		{
			name:        "空权益码",
			userID:      user.ID,
			code:        "",
			expectError: true,
			errorMsg:    "不能为空",
		},
		{
			name:        "用户不存在",
			userID:      99999,
			code:        validCodes[1].Code,
			expectError: true,
			errorMsg:    "用户不存在",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.Redeem(tt.userID, tt.code)

			if tt.expectError {
				if err == nil {
					t.Error("期望返回错误，但没有错误")
				}
				if tt.errorMsg != "" && err != nil && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("错误信息 = %q, 期望包含 %q", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("不期望错误，但得到: %v", err)
				}
				if result == nil {
					t.Fatal("兑换结果为 nil")
				}

				// 验证兑换结果
				if result.Code != tt.code {
					t.Errorf("Code = %q, 期望 %q", result.Code, tt.code)
				}
				if result.AppliedLevel != 1 {
					t.Errorf("AppliedLevel = %d, 期望 1", result.AppliedLevel)
				}
				if result.VIPExpiresAt == nil {
					t.Error("VIPExpiresAt 为 nil")
				}

				// 验证用户已升级
				var updatedUser models.User
				if err := db.First(&updatedUser, tt.userID).Error; err != nil {
					t.Fatalf("查询用户失败: %v", err)
				}
				if updatedUser.VipLevel != 1 {
					t.Errorf("用户 VipLevel = %d, 期望 1", updatedUser.VipLevel)
				}

				// 验证权益码状态已更新
				var updatedCode models.BenefitCode
				if err := db.Where("code = ?", tt.code).First(&updatedCode).Error; err != nil {
					t.Fatalf("查询权益码失败: %v", err)
				}
				if updatedCode.Status != "used" {
					t.Errorf("权益码 Status = %q, 期望 %q", updatedCode.Status, "used")
				}
				if updatedCode.UsedBy == nil || *updatedCode.UsedBy != tt.userID {
					t.Error("UsedBy 未正确设置")
				}
				if updatedCode.UsedAt == nil {
					t.Error("UsedAt 未设置")
				}
			}
		})
	}
}

func TestBenefitCodeService_List(t *testing.T) {
	db := setupBenefitCodeTestDB(t)
	service := NewBenefitCodeService(db)
	admin := createBenefitCodeAdmin(t, db)

	// 生成测试权益码
	_, err := service.Generate(admin.ID, 1, 30, 15, nil)
	if err != nil {
		t.Fatalf("生成权益码失败: %v", err)
	}
	_, err = service.Generate(admin.ID, 2, 90, 10, nil)
	if err != nil {
		t.Fatalf("生成权益码失败: %v", err)
	}

	tests := []struct {
		name           string
		filters        BenefitCodeListFilters
		expectMinCount int
		expectMaxCount int
		expectError    bool
	}{
		{
			name: "默认分页",
			filters: BenefitCodeListFilters{
				Page:     1,
				PageSize: 20,
			},
			expectMinCount: 20,
			expectMaxCount: 20,
			expectError:    false,
		},
		{
			name: "第二页",
			filters: BenefitCodeListFilters{
				Page:     2,
				PageSize: 20,
			},
			expectMinCount: 5,
			expectMaxCount: 5,
			expectError:    false,
		},
		{
			name: "按 VIP 等级过滤",
			filters: BenefitCodeListFilters{
				VIPLevel: func() *int { v := 1; return &v }(),
				Page:     1,
				PageSize: 20,
			},
			expectMinCount: 15,
			expectMaxCount: 15,
			expectError:    false,
		},
		{
			name: "按状态过滤",
			filters: BenefitCodeListFilters{
				Status:   "unused",
				Page:     1,
				PageSize: 50,
			},
			expectMinCount: 25,
			expectMaxCount: 25,
			expectError:    false,
		},
		{
			name: "小页面大小",
			filters: BenefitCodeListFilters{
				Page:     1,
				PageSize: 10,
			},
			expectMinCount: 10,
			expectMaxCount: 10,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.List(tt.filters)

			if tt.expectError {
				if err == nil {
					t.Error("期望返回错误，但没有错误")
				}
				return
			}

			if err != nil {
				t.Errorf("不期望错误，但得到: %v", err)
			}

			if result == nil {
				t.Fatal("结果为 nil")
			}

			count := len(result.List)
			if count < tt.expectMinCount || count > tt.expectMaxCount {
				t.Errorf("权益码数量 = %d, 期望在 %d-%d 之间", count, tt.expectMinCount, tt.expectMaxCount)
			}

			if result.Total < int64(tt.expectMinCount) {
				t.Errorf("Total = %d, 期望至少 %d", result.Total, tt.expectMinCount)
			}
		})
	}
}

func TestBenefitCodeService_BatchDelete(t *testing.T) {
	db := setupBenefitCodeTestDB(t)
	service := NewBenefitCodeService(db)
	admin := createBenefitCodeAdmin(t, db)

	// 生成测试权益码
	codes, err := service.Generate(admin.ID, 1, 30, 5, nil)
	if err != nil {
		t.Fatalf("生成权益码失败: %v", err)
	}

	tests := []struct {
		name        string
		adminID     uint
		ids         []uint
		expectError bool
		errorMsg    string
	}{
		{
			name:        "成功删除多个权益码",
			adminID:     admin.ID,
			ids:         []uint{codes[0].ID, codes[1].ID, codes[2].ID},
			expectError: false,
		},
		{
			name:        "删除单个权益码",
			adminID:     admin.ID,
			ids:         []uint{codes[3].ID},
			expectError: false,
		},
		{
			name:        "空 ID 列表",
			adminID:     admin.ID,
			ids:         []uint{},
			expectError: true,
			errorMsg:    "不能为空",
		},
		{
			name:        "非管理员操作",
			adminID:     createBenefitCodeUser(t, db).ID,
			ids:         []uint{codes[4].ID},
			expectError: true,
			errorMsg:    "仅管理员",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.BatchDelete(tt.adminID, tt.ids)

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

				// 验证权益码已删除
				for _, id := range tt.ids {
					var code models.BenefitCode
					err := db.First(&code, id).Error
					if err == nil {
						t.Errorf("权益码 ID %d 未被删除", id)
					}
				}
			}
		})
	}
}

func TestBenefitCodeService_ValidateCode(t *testing.T) {
	db := setupBenefitCodeTestDB(t)
	service := NewBenefitCodeService(db)
	admin := createBenefitCodeAdmin(t, db)

	// 生成有效权益码
	validCodes, err := service.Generate(admin.ID, 1, 30, 1, nil)
	if err != nil {
		t.Fatalf("生成权益码失败: %v", err)
	}

	tests := []struct {
		name        string
		code        string
		expectValid bool
	}{
		{
			name:        "有效权益码",
			code:        validCodes[0].Code,
			expectValid: true,
		},
		{
			name:        "不存在的权益码",
			code:        "INVALID-CODE-123",
			expectValid: false,
		},
		{
			name:        "空权益码",
			code:        "",
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, err := service.validateCode(tt.code)

			if tt.expectValid {
				if err != nil {
					t.Errorf("不期望错误，但得到: %v", err)
				}
				if code == nil {
					t.Error("权益码为 nil")
				}
			} else {
				if err == nil {
					t.Error("期望返回错误，但没有错误")
				}
			}
		})
	}
}
