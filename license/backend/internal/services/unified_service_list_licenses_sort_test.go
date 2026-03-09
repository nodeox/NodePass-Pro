package services

import (
	"testing"
	"time"

	"nodepass-license-unified/internal/models"
)

func TestListLicensesSortByFields(t *testing.T) {
	service, db, base := setupUnifiedActivationServiceTest(t, 3)

	createdAtA := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	createdAtB := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)
	createdAtC := time.Date(2026, 1, 3, 10, 0, 0, 0, time.UTC)

	expiresAtA := time.Date(2026, 6, 3, 0, 0, 0, 0, time.UTC)
	expiresAtB := time.Date(2026, 6, 2, 0, 0, 0, 0, time.UTC)
	expiresAtC := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	if err := db.Model(&models.License{}).Where("id = ?", base.ID).Updates(map[string]interface{}{
		"status":     "revoked",
		"created_at": createdAtA,
		"updated_at": createdAtA,
		"expires_at": expiresAtA,
	}).Error; err != nil {
		t.Fatalf("update base license failed: %v", err)
	}

	second := models.License{
		Key:         "NP-SORT-B",
		PlanID:      base.PlanID,
		Customer:    "Sort Test B",
		Status:      "active",
		ExpiresAt:   &expiresAtB,
		MaxMachines: base.MaxMachines,
		CreatedBy:   1,
		CreatedAt:   createdAtB,
		UpdatedAt:   createdAtB,
	}
	if err := db.Create(&second).Error; err != nil {
		t.Fatalf("create second license failed: %v", err)
	}

	third := models.License{
		Key:         "NP-SORT-C",
		PlanID:      base.PlanID,
		Customer:    "Sort Test C",
		Status:      "expired",
		ExpiresAt:   &expiresAtC,
		MaxMachines: base.MaxMachines,
		CreatedBy:   1,
		CreatedAt:   createdAtC,
		UpdatedAt:   createdAtC,
	}
	if err := db.Create(&third).Error; err != nil {
		t.Fatalf("create third license failed: %v", err)
	}

	createdAsc, err := service.ListLicenses(LicenseFilter{
		Page:      1,
		PageSize:  10,
		SortBy:    "created_at",
		SortOrder: "asc",
	})
	if err != nil {
		t.Fatalf("list licenses created_at asc failed: %v", err)
	}
	assertLicenseKeysOrder(t, createdAsc.Items, []string{base.Key, second.Key, third.Key})

	expiresDesc, err := service.ListLicenses(LicenseFilter{
		Page:      1,
		PageSize:  10,
		SortBy:    "expires_at",
		SortOrder: "desc",
	})
	if err != nil {
		t.Fatalf("list licenses expires_at desc failed: %v", err)
	}
	assertLicenseKeysOrder(t, expiresDesc.Items, []string{base.Key, second.Key, third.Key})

	statusAsc, err := service.ListLicenses(LicenseFilter{
		Page:      1,
		PageSize:  10,
		SortBy:    "status",
		SortOrder: "asc",
	})
	if err != nil {
		t.Fatalf("list licenses status asc failed: %v", err)
	}
	assertLicenseKeysOrder(t, statusAsc.Items, []string{second.Key, third.Key, base.Key})
}

func TestListLicensesInvalidSortFallbackToDefault(t *testing.T) {
	service, db, base := setupUnifiedActivationServiceTest(t, 3)

	second := models.License{
		Key:       "NP-SORT-DEF-B",
		PlanID:    base.PlanID,
		Customer:  "Default Sort B",
		Status:    "active",
		CreatedBy: 1,
	}
	if err := db.Create(&second).Error; err != nil {
		t.Fatalf("create second license failed: %v", err)
	}

	third := models.License{
		Key:       "NP-SORT-DEF-C",
		PlanID:    base.PlanID,
		Customer:  "Default Sort C",
		Status:    "active",
		CreatedBy: 1,
	}
	if err := db.Create(&third).Error; err != nil {
		t.Fatalf("create third license failed: %v", err)
	}

	result, err := service.ListLicenses(LicenseFilter{
		Page:      1,
		PageSize:  10,
		SortBy:    "invalid",
		SortOrder: "asc",
	})
	if err != nil {
		t.Fatalf("list licenses with invalid sort failed: %v", err)
	}
	assertLicenseKeysOrder(t, result.Items, []string{third.Key, second.Key, base.Key})
}

func assertLicenseKeysOrder(t *testing.T, items []models.License, expected []string) {
	t.Helper()
	if len(items) < len(expected) {
		t.Fatalf("license count mismatch, expected at least %d, got %d", len(expected), len(items))
	}
	for i, key := range expected {
		if items[i].Key != key {
			t.Fatalf("license order mismatch at index %d, expected %s, got %s", i, key, items[i].Key)
		}
	}
}
