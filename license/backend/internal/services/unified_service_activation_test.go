package services

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"nodepass-license-unified/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestUnbindActivationByIDAndRebind(t *testing.T) {
	service, db, license := setupUnifiedActivationServiceTest(t, 1)

	verifyA, err := service.Verify(&VerifyRequest{
		LicenseKey:    license.Key,
		MachineID:     "machine-a",
		Hostname:      "host-a",
		Product:       "nodeclient",
		ClientVersion: "1.0.0",
		Channel:       "stable",
	}, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("verify machine-a failed: %v", err)
	}
	if !verifyA.Verified {
		t.Fatalf("expected machine-a verified, got status=%s", verifyA.Status)
	}

	verifyB, err := service.Verify(&VerifyRequest{
		LicenseKey:    license.Key,
		MachineID:     "machine-b",
		Hostname:      "host-b",
		Product:       "nodeclient",
		ClientVersion: "1.0.0",
		Channel:       "stable",
	}, "127.0.0.2", "test-agent")
	if err != nil {
		t.Fatalf("verify machine-b failed: %v", err)
	}
	if verifyB.Verified || verifyB.Status != "machine_limit_exceeded" {
		t.Fatalf("expected machine-b exceed limit, got verified=%v status=%s", verifyB.Verified, verifyB.Status)
	}

	activations, err := service.ListActivations(license.ID)
	if err != nil {
		t.Fatalf("list activations failed: %v", err)
	}
	if len(activations) != 1 {
		t.Fatalf("expected 1 activation, got %d", len(activations))
	}

	if err = service.UnbindActivationByID(license.ID, activations[0].ID); err != nil {
		t.Fatalf("unbind activation failed: %v", err)
	}

	verifyBAgain, err := service.Verify(&VerifyRequest{
		LicenseKey:    license.Key,
		MachineID:     "machine-b",
		Hostname:      "host-b",
		Product:       "nodeclient",
		ClientVersion: "1.0.0",
		Channel:       "stable",
	}, "127.0.0.2", "test-agent")
	if err != nil {
		t.Fatalf("verify machine-b after unbind failed: %v", err)
	}
	if !verifyBAgain.Verified {
		t.Fatalf("expected machine-b verified after unbind, got status=%s", verifyBAgain.Status)
	}

	after, err := service.ListActivations(license.ID)
	if err != nil {
		t.Fatalf("list activations after rebind failed: %v", err)
	}
	if len(after) != 1 || after[0].MachineID != "machine-b" {
		t.Fatalf("expected only machine-b activation, got %+v", after)
	}

	if err = service.UnbindActivationByID(license.ID, activations[0].ID); err == nil {
		t.Fatalf("expected unbind non-exist activation to fail")
	}

	var count int64
	if err = db.Model(&models.LicenseActivation{}).Where("license_id = ?", license.ID).Count(&count).Error; err != nil {
		t.Fatalf("count activations failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected activation count 1, got %d", count)
	}
}

func TestClearActivations(t *testing.T) {
	service, _, license := setupUnifiedActivationServiceTest(t, 5)

	_, err := service.Verify(&VerifyRequest{
		LicenseKey:    license.Key,
		MachineID:     "machine-1",
		Hostname:      "host-1",
		Product:       "nodeclient",
		ClientVersion: "1.0.0",
		Channel:       "stable",
	}, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("verify machine-1 failed: %v", err)
	}
	_, err = service.Verify(&VerifyRequest{
		LicenseKey:    license.Key,
		MachineID:     "machine-2",
		Hostname:      "host-2",
		Product:       "nodeclient",
		ClientVersion: "1.0.0",
		Channel:       "stable",
	}, "127.0.0.2", "test-agent")
	if err != nil {
		t.Fatalf("verify machine-2 failed: %v", err)
	}

	cleared, err := service.ClearActivations(license.ID)
	if err != nil {
		t.Fatalf("clear activations failed: %v", err)
	}
	if cleared != 2 {
		t.Fatalf("expected clear count 2, got %d", cleared)
	}

	items, err := service.ListActivations(license.ID)
	if err != nil {
		t.Fatalf("list activations failed: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected no activation after clear, got %d", len(items))
	}

	_, err = service.ClearActivations(999999)
	if err == nil || !strings.Contains(err.Error(), "授权不存在") {
		t.Fatalf("expected non-exist license clear fail, got err=%v", err)
	}
}

func TestVerifyConcurrentMachineLimit(t *testing.T) {
	service, _, license := setupUnifiedActivationServiceTest(t, 1)

	reqs := []VerifyRequest{
		{
			LicenseKey:    license.Key,
			MachineID:     "machine-concurrent-a",
			Hostname:      "host-concurrent-a",
			Product:       "nodeclient",
			ClientVersion: "1.0.0",
			Channel:       "stable",
		},
		{
			LicenseKey:    license.Key,
			MachineID:     "machine-concurrent-b",
			Hostname:      "host-concurrent-b",
			Product:       "nodeclient",
			ClientVersion: "1.0.0",
			Channel:       "stable",
		},
	}

	results := make([]*VerifyResult, len(reqs))
	errs := make([]error, len(reqs))

	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range reqs {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			results[i], errs[i] = service.Verify(&reqs[i], fmt.Sprintf("127.0.0.%d", i+10), "test-agent")
		}()
	}

	close(start)
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("verify request %d failed: %v", i, err)
		}
	}

	verifiedCount := 0
	limitExceededCount := 0
	for _, item := range results {
		if item.Verified {
			verifiedCount++
		}
		if item.Status == "machine_limit_exceeded" {
			limitExceededCount++
		}
	}
	if verifiedCount != 1 || limitExceededCount != 1 {
		t.Fatalf("expected one verified and one limited, got verified=%d limited=%d", verifiedCount, limitExceededCount)
	}

	activations, err := service.ListActivations(license.ID)
	if err != nil {
		t.Fatalf("list activations failed: %v", err)
	}
	if len(activations) != 1 {
		t.Fatalf("expected exactly 1 activation after concurrent verify, got %d", len(activations))
	}
}

func TestClearActivationsWithOperatorWritesAuditLog(t *testing.T) {
	service, db, license := setupUnifiedActivationServiceTest(t, 5)

	_, err := service.Verify(&VerifyRequest{
		LicenseKey:    license.Key,
		MachineID:     "machine-audit-1",
		Hostname:      "host-audit-1",
		Product:       "nodeclient",
		ClientVersion: "1.0.0",
		Channel:       "stable",
	}, "127.0.0.31", "test-agent")
	if err != nil {
		t.Fatalf("verify machine-audit-1 failed: %v", err)
	}
	_, err = service.Verify(&VerifyRequest{
		LicenseKey:    license.Key,
		MachineID:     "machine-audit-2",
		Hostname:      "host-audit-2",
		Product:       "nodeclient",
		ClientVersion: "1.0.0",
		Channel:       "stable",
	}, "127.0.0.32", "test-agent")
	if err != nil {
		t.Fatalf("verify machine-audit-2 failed: %v", err)
	}

	operatorID := uint(7)
	cleared, err := service.ClearActivationsWithOperator(license.ID, operatorID)
	if err != nil {
		t.Fatalf("clear activations with operator failed: %v", err)
	}
	if cleared != 2 {
		t.Fatalf("expected cleared count 2, got %d", cleared)
	}

	var logs []models.AdminAuditLog
	if err = db.Where("admin_id = ? AND action = ?", operatorID, AuditActionLicenseClearActivations).
		Order("id desc").
		Find(&logs).Error; err != nil {
		t.Fatalf("query audit logs failed: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 audit log, got %d", len(logs))
	}

	var payload map[string]interface{}
	if err = json.Unmarshal([]byte(logs[0].PayloadJSON), &payload); err != nil {
		t.Fatalf("unmarshal audit payload failed: %v", err)
	}
	if int(payload["license_id"].(float64)) != int(license.ID) {
		t.Fatalf("expected payload license_id=%d, got %+v", license.ID, payload["license_id"])
	}
	if int(payload["cleared_count"].(float64)) != 2 {
		t.Fatalf("expected payload cleared_count=2, got %+v", payload["cleared_count"])
	}
}

func setupUnifiedActivationServiceTest(t *testing.T, maxMachines int) (*UnifiedService, *gorm.DB, models.License) {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", sanitizeUnifiedActivationTestName(t.Name()))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}

	if err = db.AutoMigrate(
		&models.LicensePlan{},
		&models.License{},
		&models.LicenseActivation{},
		&models.AdminAuditLog{},
		&models.ProductRelease{},
		&models.VersionPolicy{},
		&models.VerifyLog{},
	); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}

	plan := models.LicensePlan{
		Code:         "ACT-" + strings.ToUpper(sanitizeUnifiedActivationTestName(t.Name())),
		Name:         "Activation Plan",
		Description:  "for activation tests",
		MaxMachines:  10,
		DurationDays: 365,
		Status:       "active",
	}
	if err = db.Create(&plan).Error; err != nil {
		t.Fatalf("create plan failed: %v", err)
	}

	exp := time.Now().Add(365 * 24 * time.Hour)
	license := models.License{
		Key:         "NP-ACT-" + strings.ToUpper(sanitizeUnifiedActivationTestName(t.Name())),
		PlanID:      plan.ID,
		Customer:    "Activation Test",
		Status:      "active",
		ExpiresAt:   &exp,
		MaxMachines: &maxMachines,
		CreatedBy:   1,
	}
	if err = db.Create(&license).Error; err != nil {
		t.Fatalf("create license failed: %v", err)
	}

	return NewUnifiedService(db), db, license
}

func sanitizeUnifiedActivationTestName(name string) string {
	replacer := strings.NewReplacer("/", "_", " ", "_", "-", "_")
	return strings.ToLower(replacer.Replace(name))
}
