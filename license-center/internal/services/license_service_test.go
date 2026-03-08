package services

import (
	"testing"
	"time"

	"nodepass-license-center/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupLicenseServiceTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("无法连接数据库: %v", err)
	}

	if err := db.AutoMigrate(
		&models.LicensePlan{},
		&models.LicenseKey{},
		&models.MachineBinding{},
		&models.VerifyLog{},
		&models.DomainBinding{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}

	return db
}

func createTestPlan(t *testing.T, db *gorm.DB, code string) *models.LicensePlan {
	plan := &models.LicensePlan{
		Name:                 "Test Plan",
		Code:                 code,
		Description:          "Test plan description",
		IsEnabled:            true,
		MaxMachines:          5,
		DurationDays:         365,
		MinPanelVersion:      "1.0.0",
		MaxPanelVersion:      "2.0.0",
		MinBackendVersion:    "1.0.0",
		MaxBackendVersion:    "2.0.0",
		MinFrontendVersion:   "1.0.0",
		MaxFrontendVersion:   "2.0.0",
		MinNodeclientVersion: "1.0.0",
		MaxNodeclientVersion: "2.0.0",
	}
	if err := db.Create(plan).Error; err != nil {
		t.Fatalf("创建测试套餐失败: %v", err)
	}
	return plan
}

func createTestLicense(t *testing.T, db *gorm.DB, planID uint, key string) *models.LicenseKey {
	expiresAt := time.Now().Add(365 * 24 * time.Hour)
	license := &models.LicenseKey{
		Key:         key,
		PlanID:      planID,
		Customer:    "Test Customer",
		Status:      "active",
		ExpiresAt:   &expiresAt,
		MaxMachines: 5,
		CreatedBy:   1,
	}
	if err := db.Create(license).Error; err != nil {
		t.Fatalf("创建测试授权码失败: %v", err)
	}
	return license
}

func TestLicenseService_Verify(t *testing.T) {
	db := setupLicenseServiceTestDB(t)
	domainService := NewDomainBindingService(db)
	service := NewLicenseService(db, domainService)

	plan := createTestPlan(t, db, "standard")
	license := createTestLicense(t, db, plan.ID, "TEST-LICENSE-KEY-001")

	tests := []struct {
		name           string
		req            *VerifyRequest
		expectedValid  bool
		expectedMsg    string
	}{
		{
			name: "成功验证授权",
			req: &VerifyRequest{
				LicenseKey:  license.Key,
				MachineID:   "machine-001",
				MachineName: "Test Machine",
				Action:      "install",
				Versions: VerifyVersionInfo{
					Panel:      "1.5.0",
					Backend:    "1.5.0",
					Frontend:   "1.5.0",
					Nodeclient: "1.5.0",
				},
				Domain:  "example.com",
				SiteURL: "https://example.com",
			},
			expectedValid: true,
			expectedMsg:   "ok",
		},
		{
			name: "授权码为空",
			req: &VerifyRequest{
				LicenseKey: "",
				MachineID:  "machine-001",
			},
			expectedValid: false,
			expectedMsg:   "license_key/machine_id 不能为空",
		},
		{
			name: "机器ID为空",
			req: &VerifyRequest{
				LicenseKey: license.Key,
				MachineID:  "",
			},
			expectedValid: false,
			expectedMsg:   "license_key/machine_id 不能为空",
		},
		{
			name: "授权码不存在",
			req: &VerifyRequest{
				LicenseKey: "INVALID-KEY",
				MachineID:  "machine-001",
			},
			expectedValid: false,
			expectedMsg:   "授权码不存在",
		},
		{
			name: "版本不在允许范围内",
			req: &VerifyRequest{
				LicenseKey:  license.Key,
				MachineID:   "machine-002",
				MachineName: "Test Machine 2",
				Action:      "install",
				Versions: VerifyVersionInfo{
					Panel:      "3.0.0", // 超出范围
					Backend:    "1.5.0",
					Frontend:   "1.5.0",
					Nodeclient: "1.5.0",
				},
				Domain:  "example.com",
				SiteURL: "https://example.com",
			},
			expectedValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.Verify(tt.req, "127.0.0.1", "test-agent")
			if err != nil {
				t.Fatalf("验证失败: %v", err)
			}

			if result.Valid != tt.expectedValid {
				t.Errorf("期望 Valid = %v, 得到 %v", tt.expectedValid, result.Valid)
			}

			if tt.expectedMsg != "" && result.Message != tt.expectedMsg {
				t.Errorf("期望 Message = %s, 得到 %s", tt.expectedMsg, result.Message)
			}
		})
	}
}

func TestLicenseService_GenerateLicenses(t *testing.T) {
	db := setupLicenseServiceTestDB(t)
	domainService := NewDomainBindingService(db)
	service := NewLicenseService(db, domainService)

	plan := createTestPlan(t, db, "standard")

	tests := []struct {
		name          string
		req           *GenerateLicenseRequest
		expectedCount int
		expectError   bool
	}{
		{
			name: "成功生成单个授权码",
			req: &GenerateLicenseRequest{
				PlanID:    plan.ID,
				Customer:  "Test Customer",
				Count:     1,
				Prefix:    "TEST",
				CreatedBy: 1,
			},
			expectedCount: 1,
			expectError:   false,
		},
		{
			name: "成功生成多个授权码",
			req: &GenerateLicenseRequest{
				PlanID:    plan.ID,
				Customer:  "Test Customer",
				Count:     5,
				Prefix:    "BATCH",
				CreatedBy: 1,
			},
			expectedCount: 5,
			expectError:   false,
		},
		{
			name: "生成数量为0",
			req: &GenerateLicenseRequest{
				PlanID:    plan.ID,
				Customer:  "Test Customer",
				Count:     0,
				CreatedBy: 1,
			},
			expectedCount: 0,
			expectError:   true,
		},
		{
			name: "套餐不存在",
			req: &GenerateLicenseRequest{
				PlanID:    99999,
				Customer:  "Test Customer",
				Count:     1,
				CreatedBy: 1,
			},
			expectedCount: 0,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			licenses, err := service.GenerateLicenses(tt.req)

			if tt.expectError {
				if err == nil {
					t.Error("期望返回错误,但没有错误")
				}
				return
			}

			if err != nil {
				t.Fatalf("生成授权码失败: %v", err)
			}

			if len(licenses) != tt.expectedCount {
				t.Errorf("期望生成 %d 个授权码, 得到 %d", tt.expectedCount, len(licenses))
			}

			// 验证授权码格式
			for _, license := range licenses {
				if license.Key == "" {
					t.Error("授权码不能为空")
				}
				if license.PlanID != tt.req.PlanID {
					t.Errorf("期望 PlanID = %d, 得到 %d", tt.req.PlanID, license.PlanID)
				}
				if license.Customer != tt.req.Customer {
					t.Errorf("期望 Customer = %s, 得到 %s", tt.req.Customer, license.Customer)
				}
			}
		})
	}
}

func TestLicenseService_ListLicenses(t *testing.T) {
	db := setupLicenseServiceTestDB(t)
	domainService := NewDomainBindingService(db)
	service := NewLicenseService(db, domainService)

	plan := createTestPlan(t, db, "standard")
	createTestLicense(t, db, plan.ID, "LICENSE-001")
	createTestLicense(t, db, plan.ID, "LICENSE-002")
	createTestLicense(t, db, plan.ID, "LICENSE-003")

	tests := []struct {
		name          string
		filter        *LicenseFilter
		expectedCount int
	}{
		{
			name: "获取所有授权码",
			filter: &LicenseFilter{
				Page:     1,
				PageSize: 10,
			},
			expectedCount: 3,
		},
		{
			name: "分页查询",
			filter: &LicenseFilter{
				Page:     1,
				PageSize: 2,
			},
			expectedCount: 2,
		},
		{
			name: "按状态过滤",
			filter: &LicenseFilter{
				Status:   "active",
				Page:     1,
				PageSize: 10,
			},
			expectedCount: 3,
		},
		{
			name: "按套餐过滤",
			filter: &LicenseFilter{
				PlanID:   plan.ID,
				Page:     1,
				PageSize: 10,
			},
			expectedCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.ListLicenses(tt.filter)
			if err != nil {
				t.Fatalf("获取授权码列表失败: %v", err)
			}

			if len(result.Items) != tt.expectedCount {
				t.Errorf("期望 %d 个授权码, 得到 %d", tt.expectedCount, len(result.Items))
			}
		})
	}
}

func TestLicenseService_GetLicense(t *testing.T) {
	db := setupLicenseServiceTestDB(t)
	domainService := NewDomainBindingService(db)
	service := NewLicenseService(db, domainService)

	plan := createTestPlan(t, db, "standard")
	license := createTestLicense(t, db, plan.ID, "LICENSE-001")

	tests := []struct {
		name        string
		licenseID   uint
		expectError bool
	}{
		{
			name:        "成功获取授权码",
			licenseID:   license.ID,
			expectError: false,
		},
		{
			name:        "授权码不存在",
			licenseID:   99999,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.GetLicense(tt.licenseID)

			if tt.expectError {
				if err == nil {
					t.Error("期望返回错误,但没有错误")
				}
				return
			}

			if err != nil {
				t.Fatalf("获取授权码失败: %v", err)
			}

			if result.ID != tt.licenseID {
				t.Errorf("期望 ID = %d, 得到 %d", tt.licenseID, result.ID)
			}
		})
	}
}

func TestLicenseService_RevokeLicense(t *testing.T) {
	db := setupLicenseServiceTestDB(t)
	domainService := NewDomainBindingService(db)
	service := NewLicenseService(db, domainService)

	plan := createTestPlan(t, db, "standard")
	license := createTestLicense(t, db, plan.ID, "LICENSE-001")

	tests := []struct {
		name        string
		licenseID   uint
		expectError bool
	}{
		{
			name:        "成功吊销授权码",
			licenseID:   license.ID,
			expectError: false,
		},
		{
			name:        "授权码不存在",
			licenseID:   99999,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.RevokeLicense(tt.licenseID)

			if tt.expectError {
				if err == nil {
					t.Error("期望返回错误,但没有错误")
				}
				return
			}

			if err != nil {
				t.Fatalf("吊销授权码失败: %v", err)
			}

			// 验证状态已更新
			var updated models.LicenseKey
			if err := db.First(&updated, tt.licenseID).Error; err != nil {
				t.Fatalf("查询授权码失败: %v", err)
			}

			if updated.Status != "revoked" {
				t.Errorf("期望状态为 revoked, 得到 %s", updated.Status)
			}
		})
	}
}

func TestLicenseService_CreatePlan(t *testing.T) {
	db := setupLicenseServiceTestDB(t)
	domainService := NewDomainBindingService(db)
	service := NewLicenseService(db, domainService)

	tests := []struct {
		name        string
		req         *CreatePlanRequest
		expectError bool
	}{
		{
			name: "成功创建套餐",
			req: &CreatePlanRequest{
				Name:                 "Premium Plan",
				Code:                 "premium",
				Description:          "Premium plan description",
				IsEnabled:            true,
				MaxMachines:          10,
				DurationDays:         365,
				MinPanelVersion:      "1.0.0",
				MaxPanelVersion:      "2.0.0",
				MinBackendVersion:    "1.0.0",
				MaxBackendVersion:    "2.0.0",
				MinFrontendVersion:   "1.0.0",
				MaxFrontendVersion:   "2.0.0",
				MinNodeclientVersion: "1.0.0",
				MaxNodeclientVersion: "2.0.0",
			},
			expectError: false,
		},
		{
			name: "套餐代码重复",
			req: &CreatePlanRequest{
				Name:         "Duplicate Plan",
				Code:         "premium", // 重复的代码
				IsEnabled:    true,
				MaxMachines:  5,
				DurationDays: 365,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, err := service.CreatePlan(tt.req)

			if tt.expectError {
				if err == nil {
					t.Error("期望返回错误,但没有错误")
				}
				return
			}

			if err != nil {
				t.Fatalf("创建套餐失败: %v", err)
			}

			if plan.Name != tt.req.Name {
				t.Errorf("期望 Name = %s, 得到 %s", tt.req.Name, plan.Name)
			}
			if plan.Code != tt.req.Code {
				t.Errorf("期望 Code = %s, 得到 %s", tt.req.Code, plan.Code)
			}
		})
	}
}

func TestLicenseService_ListPlans(t *testing.T) {
	db := setupLicenseServiceTestDB(t)
	domainService := NewDomainBindingService(db)
	service := NewLicenseService(db, domainService)

	createTestPlan(t, db, "basic")
	createTestPlan(t, db, "standard")
	createTestPlan(t, db, "premium")

	plans, err := service.ListPlans()
	if err != nil {
		t.Fatalf("获取套餐列表失败: %v", err)
	}

	if len(plans) != 3 {
		t.Errorf("期望 3 个套餐, 得到 %d", len(plans))
	}
}
