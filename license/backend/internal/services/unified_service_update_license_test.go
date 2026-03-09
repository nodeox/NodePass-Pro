package services

import (
	"strings"
	"testing"
	"time"

	"nodepass-license-unified/internal/models"
)

func TestUpdateLicenseAllEditableFields(t *testing.T) {
	service, db, license := setupUnifiedActivationServiceTest(t, 3)

	plan2 := models.LicensePlan{
		Code:         "UPD-PLAN-2",
		Name:         "Update Plan 2",
		Description:  "for update tests",
		MaxMachines:  20,
		DurationDays: 730,
		Status:       "active",
	}
	if err := db.Create(&plan2).Error; err != nil {
		t.Fatalf("create second plan failed: %v", err)
	}

	newKey := "NP-EDIT-ABCD1234-EFGH5678-IJKL9012"
	newCustomer := "Edited Customer"
	newStatus := "revoked"
	newExpire := time.Now().Add(180 * 24 * time.Hour).Round(time.Second)
	newMax := 12
	newMetadata := `{"tier":"enterprise","region":"cn"}`
	newNote := "manually edited"

	updated, err := service.UpdateLicense(license.ID, &UpdateLicenseRequest{
		Key:          &newKey,
		PlanID:       &plan2.ID,
		Customer:     &newCustomer,
		Status:       &newStatus,
		ExpiresAt:    &newExpire,
		MaxMachines:  &newMax,
		MetadataJSON: &newMetadata,
		Note:         &newNote,
	})
	if err != nil {
		t.Fatalf("update license failed: %v", err)
	}

	if updated.Key != newKey {
		t.Fatalf("expected key %s, got %s", newKey, updated.Key)
	}
	if updated.PlanID != plan2.ID {
		t.Fatalf("expected plan_id %d, got %d", plan2.ID, updated.PlanID)
	}
	if updated.Customer != newCustomer {
		t.Fatalf("expected customer %s, got %s", newCustomer, updated.Customer)
	}
	if updated.Status != "revoked" {
		t.Fatalf("expected status revoked, got %s", updated.Status)
	}
	if updated.ExpiresAt == nil || updated.ExpiresAt.Unix() != newExpire.Unix() {
		t.Fatalf("expected expires_at %v, got %v", newExpire, updated.ExpiresAt)
	}
	if updated.MaxMachines == nil || *updated.MaxMachines != newMax {
		t.Fatalf("expected max_machines %d, got %+v", newMax, updated.MaxMachines)
	}
	if updated.MetadataJSON != newMetadata {
		t.Fatalf("expected metadata %s, got %s", newMetadata, updated.MetadataJSON)
	}
	if updated.Note != newNote {
		t.Fatalf("expected note %s, got %s", newNote, updated.Note)
	}
}

func TestUpdateLicenseClearNullableFields(t *testing.T) {
	service, _, license := setupUnifiedActivationServiceTest(t, 7)

	updated, err := service.UpdateLicense(license.ID, &UpdateLicenseRequest{
		ClearExpires: true,
		ClearMax:     true,
	})
	if err != nil {
		t.Fatalf("clear nullable fields failed: %v", err)
	}

	if updated.ExpiresAt != nil {
		t.Fatalf("expected expires_at nil, got %v", updated.ExpiresAt)
	}
	if updated.MaxMachines != nil {
		t.Fatalf("expected max_machines nil, got %+v", updated.MaxMachines)
	}
}

func TestUpdateLicenseValidation(t *testing.T) {
	service, db, license := setupUnifiedActivationServiceTest(t, 3)

	invalidStatus := "disabled"
	_, err := service.UpdateLicense(license.ID, &UpdateLicenseRequest{
		Status: &invalidStatus,
	})
	if err == nil || !strings.Contains(err.Error(), "status 仅支持 active/revoked/expired") {
		t.Fatalf("expected invalid status error, got %v", err)
	}

	zero := 0
	_, err = service.UpdateLicense(license.ID, &UpdateLicenseRequest{
		MaxMachines: &zero,
	})
	if err == nil || !strings.Contains(err.Error(), "max_machines 必须大于 0") {
		t.Fatalf("expected invalid max_machines error, got %v", err)
	}

	var planCount int64
	if err = db.Model(&models.LicensePlan{}).Count(&planCount).Error; err != nil {
		t.Fatalf("count plans failed: %v", err)
	}
	invalidPlanID := uint(planCount + 1000)
	_, err = service.UpdateLicense(license.ID, &UpdateLicenseRequest{
		PlanID: &invalidPlanID,
	})
	if err == nil || !strings.Contains(err.Error(), "套餐不存在") {
		t.Fatalf("expected plan not found error, got %v", err)
	}

	invalidMetadata := `{"tier":`
	_, err = service.UpdateLicense(license.ID, &UpdateLicenseRequest{
		MetadataJSON: &invalidMetadata,
	})
	if err == nil || !strings.Contains(err.Error(), "metadata_json 必须为合法 JSON") {
		t.Fatalf("expected invalid metadata_json error, got %v", err)
	}

	_, err = service.UpdateLicense(license.ID, &UpdateLicenseRequest{})
	if err == nil || !strings.Contains(err.Error(), "没有可更新字段") {
		t.Fatalf("expected no fields error, got %v", err)
	}
}
