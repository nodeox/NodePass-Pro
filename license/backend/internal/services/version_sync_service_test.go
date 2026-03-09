package services

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"nodepass-license-unified/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestGetVersionSyncConfigInit(t *testing.T) {
	service := setupVersionSyncServiceTest(t)

	cfg, err := service.GetVersionSyncConfig()
	if err != nil {
		t.Fatalf("get config failed: %v", err)
	}
	if cfg.IntervalMinutes != 60 {
		t.Fatalf("expected default interval 60, got %d", cfg.IntervalMinutes)
	}
	if cfg.Product != "nodeclient" || cfg.Channel != "stable" {
		t.Fatalf("unexpected default product/channel: %+v", cfg)
	}
}

func TestListVersionSyncConfigsInit(t *testing.T) {
	service := setupVersionSyncServiceTest(t)

	items, err := service.ListVersionSyncConfigs()
	if err != nil {
		t.Fatalf("list configs failed: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 configs, got %d", len(items))
	}
	expected := []string{"backend", "frontend", "nodeclient"}
	for i, product := range expected {
		if items[i].Product != product {
			t.Fatalf("expected product[%d]=%s, got %s", i, product, items[i].Product)
		}
	}
}

func TestUpdateVersionSyncConfigEnabledValidation(t *testing.T) {
	service := setupVersionSyncServiceTest(t)
	enabled := true

	_, err := service.UpdateVersionSyncConfig(&UpdateVersionSyncConfigRequest{
		Enabled: &enabled,
	})
	if err == nil {
		t.Fatalf("expected validation error when enabled without repo config")
	}
}

func TestManualSyncVersionMirrorImportsReleases(t *testing.T) {
	service := setupVersionSyncServiceTest(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/nodeox/nodeclient/releases" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"tag_name":"v1.2.3","name":"v1.2.3","body":"stable release","html_url":"https://example.com/r1","prerelease":false,"draft":false,"published_at":"2026-03-01T12:00:00Z"},
			{"tag_name":"v1.2.4-rc1","name":"v1.2.4-rc1","body":"preview","html_url":"https://example.com/r2","prerelease":true,"draft":false,"published_at":"2026-03-02T12:00:00Z"},
			{"tag_name":"v1.2.5","name":"v1.2.5","body":"draft","html_url":"https://example.com/r3","prerelease":false,"draft":true,"published_at":"2026-03-03T12:00:00Z"}
		]`))
	}))
	defer server.Close()

	enabled := true
	product := "backend"
	owner := "nodeox"
	repo := "nodeclient"
	channel := "stable"
	apiBaseURL := server.URL
	includePrerelease := false

	_, err := service.UpdateVersionSyncConfig(&UpdateVersionSyncConfigRequest{
		Product:           &product,
		Enabled:           &enabled,
		GitHubOwner:       &owner,
		GitHubRepo:        &repo,
		Channel:           &channel,
		APIBaseURL:        &apiBaseURL,
		IncludePrerelease: &includePrerelease,
	})
	if err != nil {
		t.Fatalf("update sync config failed: %v", err)
	}

	result, err := service.ManualSyncVersionMirrorByProduct(product)
	if err != nil {
		t.Fatalf("manual sync failed: %v", err)
	}
	if result.Product != product {
		t.Fatalf("expected result product=%s, got %s", product, result.Product)
	}
	if result.FetchedCount != 3 {
		t.Fatalf("expected fetched_count=3, got %d", result.FetchedCount)
	}
	if result.ImportedCount != 1 {
		t.Fatalf("expected imported_count=1, got %d", result.ImportedCount)
	}
	if result.SkippedCount != 2 {
		t.Fatalf("expected skipped_count=2, got %d", result.SkippedCount)
	}

	var releases []models.ProductRelease
	if err = service.db.Find(&releases).Error; err != nil {
		t.Fatalf("query releases failed: %v", err)
	}
	if len(releases) != 1 {
		t.Fatalf("expected 1 mirrored release, got %d", len(releases))
	}
	if releases[0].Version != "1.2.3" {
		t.Fatalf("expected normalized version 1.2.3, got %s", releases[0].Version)
	}

	secondResult, err := service.ManualSyncVersionMirrorByProduct(product)
	if err != nil {
		t.Fatalf("second manual sync failed: %v", err)
	}
	if secondResult.ImportedCount != 0 {
		t.Fatalf("expected second sync imported_count=0, got %d", secondResult.ImportedCount)
	}
}

func setupVersionSyncServiceTest(t *testing.T) *UnifiedService {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", sanitizeVersionSyncTestName(t.Name()))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err = db.AutoMigrate(&models.VersionSyncConfig{}, &models.ProductRelease{}, &models.AdminAuditLog{}); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}
	return NewUnifiedService(db)
}

func sanitizeVersionSyncTestName(name string) string {
	replacer := strings.NewReplacer("/", "_", " ", "_", "-", "_")
	return strings.ToLower(replacer.Replace(name))
}
