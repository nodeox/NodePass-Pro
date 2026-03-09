package services

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"nodepass-license-unified/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestBatchUpdatePlanID(t *testing.T) {
	service, db, _, targetPlan, licenses := setupEnhancedBatchServiceTest(t)

	if err := service.BatchUpdate(&BatchUpdateRequest{
		LicenseIDs: []uint{licenses[0].ID, licenses[1].ID},
		Updates: map[string]interface{}{
			"plan_id": float64(targetPlan.ID),
		},
	}); err != nil {
		t.Fatalf("batch update plan_id failed: %v", err)
	}

	var after []models.License
	if err := db.Where("id IN ?", []uint{licenses[0].ID, licenses[1].ID}).Find(&after).Error; err != nil {
		t.Fatalf("query licenses failed: %v", err)
	}
	for _, item := range after {
		if item.PlanID != targetPlan.ID {
			t.Fatalf("expected plan_id=%d, got %d", targetPlan.ID, item.PlanID)
		}
	}
}

func TestBatchUpdateInvalidPlanID(t *testing.T) {
	service, _, _, _, licenses := setupEnhancedBatchServiceTest(t)

	err := service.BatchUpdate(&BatchUpdateRequest{
		LicenseIDs: []uint{licenses[0].ID},
		Updates: map[string]interface{}{
			"plan_id": float64(999999),
		},
	})
	if err == nil || !strings.Contains(err.Error(), "套餐不存在") {
		t.Fatalf("expected plan not found error, got %v", err)
	}
}

func TestBatchUpdateInvalidStatus(t *testing.T) {
	service, _, _, _, licenses := setupEnhancedBatchServiceTest(t)

	err := service.BatchUpdate(&BatchUpdateRequest{
		LicenseIDs: []uint{licenses[0].ID},
		Updates: map[string]interface{}{
			"status": "disabled",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "status 仅支持 active/revoked/expired") {
		t.Fatalf("expected invalid status error, got %v", err)
	}
}

func TestBatchUpdateInvalidMetadataJSON(t *testing.T) {
	service, _, _, _, licenses := setupEnhancedBatchServiceTest(t)

	err := service.BatchUpdate(&BatchUpdateRequest{
		LicenseIDs: []uint{licenses[0].ID},
		Updates: map[string]interface{}{
			"metadata_json": `{"region":`,
		},
	})
	if err == nil || !strings.Contains(err.Error(), "metadata_json 必须为合法 JSON") {
		t.Fatalf("expected invalid metadata_json error, got %v", err)
	}
}

func TestBatchActionsWriteAuditLogs(t *testing.T) {
	service, db, _, _, licenses := setupEnhancedBatchServiceTest(t)

	operatorID := uint(99)
	licenseIDs := []uint{licenses[0].ID, licenses[1].ID}

	if err := service.BatchUpdateWithOperator(&BatchUpdateRequest{
		LicenseIDs: licenseIDs,
		Updates: map[string]interface{}{
			"note": "batch-updated",
		},
	}, operatorID); err != nil {
		t.Fatalf("batch update with operator failed: %v", err)
	}
	if err := service.BatchRevokeWithOperator(licenseIDs, operatorID); err != nil {
		t.Fatalf("batch revoke with operator failed: %v", err)
	}
	if err := service.BatchRestoreWithOperator(licenseIDs, operatorID); err != nil {
		t.Fatalf("batch restore with operator failed: %v", err)
	}

	var logs []models.AdminAuditLog
	if err := db.Where("admin_id = ?", operatorID).Order("id asc").Find(&logs).Error; err != nil {
		t.Fatalf("query audit logs failed: %v", err)
	}
	if len(logs) != 3 {
		t.Fatalf("expected 3 audit logs, got %d", len(logs))
	}

	expectedActions := []string{
		AuditActionLicenseBatchUpdate,
		AuditActionLicenseBatchRevoke,
		AuditActionLicenseBatchRestore,
	}
	for i, action := range expectedActions {
		if logs[i].Action != action {
			t.Fatalf("expected action %s at index %d, got %s", action, i, logs[i].Action)
		}

		var payload map[string]interface{}
		if err := json.Unmarshal([]byte(logs[i].PayloadJSON), &payload); err != nil {
			t.Fatalf("unmarshal payload failed: %v", err)
		}
		if payload["rows_affected"] == nil {
			t.Fatalf("expected rows_affected in payload for action %s", action)
		}
	}
}

func setupEnhancedBatchServiceTest(t *testing.T) (*LicenseEnhancedService, *gorm.DB, models.LicensePlan, models.LicensePlan, []models.License) {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", sanitizeEnhancedBatchTestName(t.Name()))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}

	if err = db.AutoMigrate(&models.LicensePlan{}, &models.License{}, &models.AdminAuditLog{}); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}

	sourcePlan := models.LicensePlan{
		Code:         "BATCH-SRC-" + strings.ToUpper(sanitizeEnhancedBatchTestName(t.Name())),
		Name:         "Batch Source Plan",
		MaxMachines:  3,
		DurationDays: 365,
		Status:       "active",
	}
	if err = db.Create(&sourcePlan).Error; err != nil {
		t.Fatalf("create source plan failed: %v", err)
	}

	targetPlan := models.LicensePlan{
		Code:         "BATCH-TGT-" + strings.ToUpper(sanitizeEnhancedBatchTestName(t.Name())),
		Name:         "Batch Target Plan",
		MaxMachines:  20,
		DurationDays: 730,
		Status:       "active",
	}
	if err = db.Create(&targetPlan).Error; err != nil {
		t.Fatalf("create target plan failed: %v", err)
	}

	exp := time.Now().Add(180 * 24 * time.Hour)
	licenses := []models.License{
		{
			Key:       "NP-BATCH-A-" + strings.ToUpper(sanitizeEnhancedBatchTestName(t.Name())),
			PlanID:    sourcePlan.ID,
			Customer:  "Batch Customer A",
			Status:    "active",
			ExpiresAt: &exp,
			CreatedBy: 1,
		},
		{
			Key:       "NP-BATCH-B-" + strings.ToUpper(sanitizeEnhancedBatchTestName(t.Name())),
			PlanID:    sourcePlan.ID,
			Customer:  "Batch Customer B",
			Status:    "active",
			ExpiresAt: &exp,
			CreatedBy: 1,
		},
	}
	if err = db.Create(&licenses).Error; err != nil {
		t.Fatalf("create licenses failed: %v", err)
	}

	service := NewLicenseEnhancedService(db, NewUnifiedService(db))
	return service, db, sourcePlan, targetPlan, licenses
}

func sanitizeEnhancedBatchTestName(name string) string {
	replacer := strings.NewReplacer("/", "_", " ", "_", "-", "_")
	return strings.ToLower(replacer.Replace(name))
}
