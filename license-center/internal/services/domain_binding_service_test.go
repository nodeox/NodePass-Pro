package services

import (
	"testing"
	"time"

	"nodepass-license-center/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupDomainBindingServiceTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("无法连接数据库: %v", err)
	}

	if err := db.AutoMigrate(
		&models.LicensePlan{},
		&models.LicenseKey{},
		&models.DomainBinding{},
		&models.Alert{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}

	return db
}

func createTestLicenseForDomain(t *testing.T, db *gorm.DB, key string, boundDomain string) *models.LicenseKey {
	plan := &models.LicensePlan{
		Name:         "Test Plan",
		Code:         "test",
		IsEnabled:    true,
		MaxMachines:  5,
		DurationDays: 365,
	}
	if err := db.Create(plan).Error; err != nil {
		t.Fatalf("创建测试套餐失败: %v", err)
	}

	expiresAt := time.Now().Add(365 * 24 * time.Hour)
	license := &models.LicenseKey{
		Key:          key,
		PlanID:       plan.ID,
		Customer:     "Test Customer",
		Status:       "active",
		ExpiresAt:    &expiresAt,
		MaxMachines:  5,
		BoundDomain:  boundDomain,
		DomainLocked: false,
		CreatedBy:    1,
	}
	if err := db.Create(license).Error; err != nil {
		t.Fatalf("创建测试授权码失败: %v", err)
	}
	return license
}

func TestDomainBindingService_VerifyDomain(t *testing.T) {
	db := setupDomainBindingServiceTestDB(t)
	webhookService := NewWebhookService(db)
	alertService := NewAlertService(db, webhookService)
	service := NewDomainBindingService(db, webhookService, alertService)

	tests := []struct {
		name        string
		setupLicense func() *models.LicenseKey
		requestDomain string
		config      DomainBindingConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "域名验证功能未启用",
			setupLicense: func() *models.LicenseKey {
				return createTestLicenseForDomain(t, db, "TEST-001", "")
			},
			requestDomain: "example.com",
			config: DomainBindingConfig{
				Enabled: false,
			},
			expectError: false,
		},
		{
			name: "测试域名允许访问",
			setupLicense: func() *models.LicenseKey {
				return createTestLicenseForDomain(t, db, "TEST-002", "")
			},
			requestDomain: "localhost",
			config: DomainBindingConfig{
				Enabled:          true,
				AllowTestDomains: true,
				TestDomains:      []string{"localhost", "127.0.0.1"},
			},
			expectError: false,
		},
		{
			name: "首次自动绑定域名",
			setupLicense: func() *models.LicenseKey {
				return createTestLicenseForDomain(t, db, "TEST-003", "")
			},
			requestDomain: "example.com",
			config: DomainBindingConfig{
				Enabled:               true,
				AutoBindOnFirstVerify: true,
				AllowTestDomains:      false,
			},
			expectError: false,
		},
		{
			name: "域名匹配成功",
			setupLicense: func() *models.LicenseKey {
				return createTestLicenseForDomain(t, db, "TEST-004", "example.com")
			},
			requestDomain: "example.com",
			config: DomainBindingConfig{
				Enabled: true,
			},
			expectError: false,
		},
		{
			name: "域名不匹配",
			setupLicense: func() *models.LicenseKey {
				license := createTestLicenseForDomain(t, db, "TEST-005", "example.com")
				license.DomainLocked = true
				db.Save(license)
				return license
			},
			requestDomain: "different.com",
			config: DomainBindingConfig{
				Enabled:           true,
				AllowDomainChange: false,
			},
			expectError: true,
			errorMsg:    "域名不匹配",
		},
		{
			name: "无效的域名",
			setupLicense: func() *models.LicenseKey {
				return createTestLicenseForDomain(t, db, "TEST-006", "")
			},
			requestDomain: "",
			config: DomainBindingConfig{
				Enabled: true,
			},
			expectError: true,
			errorMsg:    "无效的域名",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			license := tt.setupLicense()
			err := service.VerifyDomain(license, tt.requestDomain, "127.0.0.1", tt.config)

			if tt.expectError {
				if err == nil {
					t.Error("期望返回错误,但没有错误")
				} else if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("期望错误消息包含 %s, 得到 %s", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("验证域名失败: %v", err)
			}
		})
	}
}

func TestDomainBindingService_BindDomain(t *testing.T) {
	db := setupDomainBindingServiceTestDB(t)
	webhookService := NewWebhookService(db)
	alertService := NewAlertService(db, webhookService)
	service := NewDomainBindingService(db, webhookService, alertService)

	license := createTestLicenseForDomain(t, db, "TEST-BIND-001", "")

	tests := []struct {
		name        string
		domain      string
		expectError bool
	}{
		{
			name:        "成功绑定域名",
			domain:      "example.com",
			expectError: false,
		},
		{
			name:        "绑定空域名",
			domain:      "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.bindDomain(license, tt.domain, "127.0.0.1", 0, "测试绑定")

			if tt.expectError {
				if err == nil {
					t.Error("期望返回错误,但没有错误")
				}
				return
			}

			if err != nil {
				t.Fatalf("绑定域名失败: %v", err)
			}

			// 验证绑定结果
			var updated models.LicenseKey
			if err := db.First(&updated, license.ID).Error; err != nil {
				t.Fatalf("查询授权码失败: %v", err)
			}

			if updated.BoundDomain != tt.domain {
				t.Errorf("期望 BoundDomain = %s, 得到 %s", tt.domain, updated.BoundDomain)
			}
		})
	}
}

func TestDomainBindingService_UnbindDomain(t *testing.T) {
	db := setupDomainBindingServiceTestDB(t)
	webhookService := NewWebhookService(db)
	alertService := NewAlertService(db, webhookService)
	service := NewDomainBindingService(db, webhookService, alertService)

	license := createTestLicenseForDomain(t, db, "TEST-UNBIND-001", "example.com")

	err := service.UnbindDomain(license.ID, 1, "测试解绑")
	if err != nil {
		t.Fatalf("解绑域名失败: %v", err)
	}

	// 验证解绑结果
	var updated models.LicenseKey
	if err := db.First(&updated, license.ID).Error; err != nil {
		t.Fatalf("查询授权码失败: %v", err)
	}

	if updated.BoundDomain != "" {
		t.Errorf("期望 BoundDomain 为空, 得到 %s", updated.BoundDomain)
	}
}

func TestDomainBindingService_ChangeDomain(t *testing.T) {
	db := setupDomainBindingServiceTestDB(t)
	webhookService := NewWebhookService(db)
	alertService := NewAlertService(db, webhookService)
	service := NewDomainBindingService(db, webhookService, alertService)

	tests := []struct {
		name        string
		setupLicense func() *models.LicenseKey
		newDomain   string
		config      DomainBindingConfig
		expectError bool
	}{
		{
			name: "成功更换域名",
			setupLicense: func() *models.LicenseKey {
				return createTestLicenseForDomain(t, db, "TEST-CHANGE-001", "old.com")
			},
			newDomain: "new.com",
			config: DomainBindingConfig{
				AllowDomainChange:    true,
				DomainChangeCooldown: 0,
			},
			expectError: false,
		},
		{
			name: "域名已锁定",
			setupLicense: func() *models.LicenseKey {
				license := createTestLicenseForDomain(t, db, "TEST-CHANGE-002", "locked.com")
				license.DomainLocked = true
				db.Save(license)
				return license
			},
			newDomain: "new.com",
			config: DomainBindingConfig{
				AllowDomainChange: true,
			},
			expectError: true,
		},
		{
			name: "不允许更换域名",
			setupLicense: func() *models.LicenseKey {
				return createTestLicenseForDomain(t, db, "TEST-CHANGE-003", "fixed.com")
			},
			newDomain: "new.com",
			config: DomainBindingConfig{
				AllowDomainChange: false,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			license := tt.setupLicense()
			err := service.ChangeDomain(license.ID, tt.newDomain, 1, "测试更换", tt.config)

			if tt.expectError {
				if err == nil {
					t.Error("期望返回错误,但没有错误")
				}
				return
			}

			if err != nil {
				t.Fatalf("更换域名失败: %v", err)
			}

			// 验证更换结果
			var updated models.LicenseKey
			if err := db.First(&updated, license.ID).Error; err != nil {
				t.Fatalf("查询授权码失败: %v", err)
			}

			if updated.BoundDomain != tt.newDomain {
				t.Errorf("期望 BoundDomain = %s, 得到 %s", tt.newDomain, updated.BoundDomain)
			}
		})
	}
}

func TestDomainBindingService_LockDomain(t *testing.T) {
	db := setupDomainBindingServiceTestDB(t)
	webhookService := NewWebhookService(db)
	alertService := NewAlertService(db, webhookService)
	service := NewDomainBindingService(db, webhookService, alertService)

	license := createTestLicenseForDomain(t, db, "TEST-LOCK-001", "example.com")

	err := service.LockDomain(license.ID, 1, "测试锁定")
	if err != nil {
		t.Fatalf("锁定域名失败: %v", err)
	}

	// 验证锁定结果
	var updated models.LicenseKey
	if err := db.First(&updated, license.ID).Error; err != nil {
		t.Fatalf("查询授权码失败: %v", err)
	}

	if !updated.DomainLocked {
		t.Error("期望 DomainLocked 为 true")
	}
}

func TestDomainBindingService_UnlockDomain(t *testing.T) {
	db := setupDomainBindingServiceTestDB(t)
	webhookService := NewWebhookService(db)
	alertService := NewAlertService(db, webhookService)
	service := NewDomainBindingService(db, webhookService, alertService)

	license := createTestLicenseForDomain(t, db, "TEST-UNLOCK-001", "example.com")
	license.DomainLocked = true
	db.Save(license)

	err := service.UnlockDomain(license.ID, 1, "测试解锁")
	if err != nil {
		t.Fatalf("解锁域名失败: %v", err)
	}

	// 验证解锁结果
	var updated models.LicenseKey
	if err := db.First(&updated, license.ID).Error; err != nil {
		t.Fatalf("查询授权码失败: %v", err)
	}

	if updated.DomainLocked {
		t.Error("期望 DomainLocked 为 false")
	}
}

func TestDomainBindingService_ListBindings(t *testing.T) {
	db := setupDomainBindingServiceTestDB(t)
	webhookService := NewWebhookService(db)
	alertService := NewAlertService(db, webhookService)
	service := NewDomainBindingService(db, webhookService, alertService)

	license1 := createTestLicenseForDomain(t, db, "TEST-LIST-001", "example1.com")
	license2 := createTestLicenseForDomain(t, db, "TEST-LIST-002", "example2.com")

	// 创建绑定记录
	service.bindDomain(license1, "example1.com", "127.0.0.1", 0, "测试")
	service.bindDomain(license2, "example2.com", "127.0.0.1", 0, "测试")

	bindings, err := service.ListBindings(license1.ID)
	if err != nil {
		t.Fatalf("获取绑定列表失败: %v", err)
	}

	if len(bindings) == 0 {
		t.Error("期望至少有一个绑定记录")
	}
}

func TestIsTestDomain(t *testing.T) {
	testDomains := []string{"localhost", "127.0.0.1", "*.test", "*.local"}

	tests := []struct {
		domain   string
		expected bool
	}{
		{"localhost", true},
		{"127.0.0.1", true},
		{"example.test", true},
		{"app.local", true},
		{"example.com", false},
		{"production.io", false},
	}

	for _, tt := range tests {
		t.Run(tt.domain, func(t *testing.T) {
			result := isTestDomain(tt.domain, testDomains)
			if result != tt.expected {
				t.Errorf("isTestDomain(%s) = %v, 期望 %v", tt.domain, result, tt.expected)
			}
		})
	}
}

func TestExtractDomain(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"example.com", "example.com"},
		{"https://example.com", "example.com"},
		{"http://example.com:8080", "example.com"},
		{"https://sub.example.com/path", "sub.example.com"},
		{"", ""},
		{"invalid", "invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := extractDomain(tt.input)
			if result != tt.expected {
				t.Errorf("extractDomain(%s) = %s, 期望 %s", tt.input, result, tt.expected)
			}
		})
	}
}

// 辅助函数
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) >= len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
