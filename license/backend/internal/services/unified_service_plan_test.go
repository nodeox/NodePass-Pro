package services

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"nodepass-license-unified/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCreatePlanValidation(t *testing.T) {
	service, _ := setupUnifiedPlanServiceTest(t)

	_, err := service.CreatePlan(&CreatePlanRequest{
		Code:         "NP-TEST",
		Name:         "Test Plan",
		MaxMachines:  3,
		DurationDays: 365,
		Status:       "invalid",
	})
	if err == nil || !strings.Contains(err.Error(), "plan status 仅支持 active/disabled") {
		t.Fatalf("expected invalid status error, got %v", err)
	}

	_, err = service.CreatePlan(&CreatePlanRequest{
		Code:         "NP-TEST",
		Name:         "Test Plan",
		MaxMachines:  0,
		DurationDays: 365,
		Status:       "active",
	})
	if err == nil || !strings.Contains(err.Error(), "max_machines 必须大于 0") {
		t.Fatalf("expected invalid max_machines error, got %v", err)
	}
}

func TestUpdatePlanValidation(t *testing.T) {
	service, plan := setupUnifiedPlanServiceTest(t)

	_, err := service.UpdatePlan(plan.ID, &CreatePlanRequest{
		Code:         plan.Code,
		Name:         plan.Name,
		Description:  plan.Description,
		MaxMachines:  plan.MaxMachines,
		DurationDays: plan.DurationDays,
		Status:       "paused",
	})
	if err == nil || !strings.Contains(err.Error(), "plan status 仅支持 active/disabled") {
		t.Fatalf("expected invalid status error, got %v", err)
	}
}

func TestCreatePlanDefaultStatusActive(t *testing.T) {
	service, _ := setupUnifiedPlanServiceTest(t)

	created, err := service.CreatePlan(&CreatePlanRequest{
		Code:         "NP-DEFAULT-STATUS",
		Name:         "Default Status Plan",
		MaxMachines:  5,
		DurationDays: 180,
	})
	if err != nil {
		t.Fatalf("create plan failed: %v", err)
	}
	if created.Status != "active" {
		t.Fatalf("expected default status active, got %s", created.Status)
	}
}

func TestListPlansIncludeUsageStats(t *testing.T) {
	service, plan := setupUnifiedPlanServiceTest(t)

	exp1 := time.Now().UTC().Add(7 * 24 * time.Hour)
	license1 := models.License{
		Key:       "NP-STATS-1",
		PlanID:    plan.ID,
		Customer:  "stats-customer-1",
		Status:    "active",
		ExpiresAt: &exp1,
		CreatedBy: 1,
	}
	if err := service.db.Create(&license1).Error; err != nil {
		t.Fatalf("create active license failed: %v", err)
	}

	exp2 := time.Now().UTC().Add(14 * 24 * time.Hour)
	license2 := models.License{
		Key:       "NP-STATS-2",
		PlanID:    plan.ID,
		Customer:  "stats-customer-2",
		Status:    "revoked",
		ExpiresAt: &exp2,
		CreatedBy: 1,
	}
	if err := service.db.Create(&license2).Error; err != nil {
		t.Fatalf("create revoked license failed: %v", err)
	}

	now := time.Now().UTC()
	activations := []models.LicenseActivation{
		{
			LicenseID:  license1.ID,
			MachineID:  "m-stats-1",
			Hostname:   "host-1",
			IPAddress:  "127.0.0.1",
			LastSeenAt: now,
		},
		{
			LicenseID:  license1.ID,
			MachineID:  "m-stats-2",
			Hostname:   "host-2",
			IPAddress:  "127.0.0.2",
			LastSeenAt: now,
		},
	}
	if err := service.db.Create(&activations).Error; err != nil {
		t.Fatalf("create activations failed: %v", err)
	}

	items, err := service.ListPlans()
	if err != nil {
		t.Fatalf("list plans failed: %v", err)
	}

	var found *models.LicensePlan
	for idx := range items {
		if items[idx].ID == plan.ID {
			found = &items[idx]
			break
		}
	}
	if found == nil {
		t.Fatalf("seed plan not found in list")
	}

	if found.LicenseCount != 2 {
		t.Fatalf("expected license_count 2, got %d", found.LicenseCount)
	}
	if found.ActiveCount != 1 {
		t.Fatalf("expected active_license_count 1, got %d", found.ActiveCount)
	}
	if found.BindingCount != 2 {
		t.Fatalf("expected activation_count 2, got %d", found.BindingCount)
	}
}

func TestClonePlanDefaults(t *testing.T) {
	service, source := setupUnifiedPlanServiceTest(t)

	cloned, err := service.ClonePlan(source.ID, &ClonePlanRequest{})
	if err != nil {
		t.Fatalf("clone plan failed: %v", err)
	}
	if cloned.ID == source.ID {
		t.Fatalf("clone id should differ from source")
	}
	if cloned.Code == source.Code {
		t.Fatalf("clone code should differ from source")
	}
	if !strings.Contains(cloned.Code, "COPY-") {
		t.Fatalf("clone code should contain COPY marker, got %s", cloned.Code)
	}
	if cloned.Name != source.Name+" 副本" {
		t.Fatalf("unexpected clone name: %s", cloned.Name)
	}
	if cloned.MaxMachines != source.MaxMachines || cloned.DurationDays != source.DurationDays {
		t.Fatalf("clone should keep max_machines and duration_days from source")
	}
	if cloned.Status != source.Status {
		t.Fatalf("clone status should follow source, got %s", cloned.Status)
	}
}

func TestClonePlanOverrides(t *testing.T) {
	service, source := setupUnifiedPlanServiceTest(t)
	description := ""

	cloned, err := service.ClonePlan(source.ID, &ClonePlanRequest{
		Code:        "NP-CLONE-CUSTOM",
		Name:        "Custom Clone",
		Description: &description,
		Status:      "disabled",
	})
	if err != nil {
		t.Fatalf("clone plan failed: %v", err)
	}
	if cloned.Code != "NP-CLONE-CUSTOM" {
		t.Fatalf("unexpected clone code: %s", cloned.Code)
	}
	if cloned.Name != "Custom Clone" {
		t.Fatalf("unexpected clone name: %s", cloned.Name)
	}
	if cloned.Description != "" {
		t.Fatalf("expected empty description override, got %s", cloned.Description)
	}
	if cloned.Status != "disabled" {
		t.Fatalf("expected disabled status, got %s", cloned.Status)
	}
}

func setupUnifiedPlanServiceTest(t *testing.T) (*UnifiedService, models.LicensePlan) {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", sanitizeUnifiedPlanTestName(t.Name()))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}

	if err = db.AutoMigrate(&models.LicensePlan{}, &models.License{}, &models.LicenseActivation{}); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}

	plan := models.LicensePlan{
		Code:         "PLAN-" + strings.ToUpper(sanitizeUnifiedPlanTestName(t.Name())),
		Name:         "Plan",
		Description:  "for plan tests",
		MaxMachines:  3,
		DurationDays: 365,
		Status:       "active",
	}
	if err = db.Create(&plan).Error; err != nil {
		t.Fatalf("create seed plan failed: %v", err)
	}

	return NewUnifiedService(db), plan
}

func sanitizeUnifiedPlanTestName(name string) string {
	replacer := strings.NewReplacer("/", "_", " ", "_", "-", "_")
	return strings.ToLower(replacer.Replace(name))
}
