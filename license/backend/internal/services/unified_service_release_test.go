package services

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"nodepass-license-unified/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCreateReleaseWithPackage(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:release_package_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err = db.AutoMigrate(&models.ProductRelease{}); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}

	service := NewUnifiedService(db)
	release, err := service.CreateReleaseWithPackage(&CreateReleaseRequest{
		Product:      "nodeclient",
		Version:      "1.2.3",
		Channel:      "stable",
		IsMandatory:  true,
		ReleaseNotes: "release package test",
	}, &ReleasePackageInfo{
		FileName:   "nodeclient-1.2.3-linux-amd64.tar.gz",
		FilePath:   "/tmp/nodeclient-1.2.3-linux-amd64.tar.gz",
		FileSize:   1024,
		FileSHA256: "ABCDEF1234",
	})
	if err != nil {
		t.Fatalf("create release with package failed: %v", err)
	}

	if release.FileName == "" || release.FilePath == "" || release.FileSize != 1024 {
		t.Fatalf("release package fields not persisted: %+v", release)
	}
	if release.FileSHA256 != strings.ToLower("ABCDEF1234") {
		t.Fatalf("expected lowercase sha256, got %s", release.FileSHA256)
	}
}

func TestCreateReleaseWithPackageValidation(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:release_package_validation_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err = db.AutoMigrate(&models.ProductRelease{}); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}

	service := NewUnifiedService(db)
	_, err = service.CreateReleaseWithPackage(&CreateReleaseRequest{
		Product: "nodeclient",
		Version: "1.2.3",
	}, &ReleasePackageInfo{
		FileName: "",
		FilePath: "/tmp/test.bin",
		FileSize: 100,
	})
	if err == nil || !strings.Contains(err.Error(), "安装包信息不完整") {
		t.Fatalf("expected invalid package error, got %v", err)
	}
}

func TestUpdateRelease(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:release_update_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err = db.AutoMigrate(&models.ProductRelease{}); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}

	service := NewUnifiedService(db)
	base, err := service.CreateRelease(&CreateReleaseRequest{
		Product:      "nodeclient",
		Version:      "1.0.0",
		Channel:      "stable",
		IsMandatory:  false,
		ReleaseNotes: "initial",
	})
	if err != nil {
		t.Fatalf("create release failed: %v", err)
	}

	version := "1.0.1"
	isMandatory := true
	isActive := false
	notes := "updated notes"
	publishedAt := time.Now().UTC().Add(2 * time.Hour)
	updated, err := service.UpdateRelease(base.ID, &UpdateReleaseRequest{
		Version:      &version,
		IsMandatory:  &isMandatory,
		IsActive:     &isActive,
		ReleaseNotes: &notes,
		PublishedAt:  &publishedAt,
	})
	if err != nil {
		t.Fatalf("update release failed: %v", err)
	}
	if updated.Version != "1.0.1" || !updated.IsMandatory || updated.IsActive {
		t.Fatalf("update fields not applied: %+v", updated)
	}
	if updated.ReleaseNotes != "updated notes" {
		t.Fatalf("release notes not updated: %s", updated.ReleaseNotes)
	}
	if updated.PublishedAt == nil || updated.PublishedAt.Unix() != publishedAt.Unix() {
		t.Fatalf("published_at not updated")
	}
}

func TestReplaceReleasePackage(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:release_replace_package_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err = db.AutoMigrate(&models.ProductRelease{}); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}

	service := NewUnifiedService(db)
	release, err := service.CreateReleaseWithPackage(&CreateReleaseRequest{
		Product: "nodeclient",
		Version: "1.0.0",
	}, &ReleasePackageInfo{
		FileName: "old.bin",
		FilePath: "/tmp/old.bin",
		FileSize: 100,
	})
	if err != nil {
		t.Fatalf("create release failed: %v", err)
	}

	replaced, oldPath, err := service.ReplaceReleasePackage(release.ID, &ReleasePackageInfo{
		FileName:   "new.bin",
		FilePath:   "/tmp/new.bin",
		FileSize:   200,
		FileSHA256: "ABC123",
	})
	if err != nil {
		t.Fatalf("replace package failed: %v", err)
	}
	if oldPath != "/tmp/old.bin" {
		t.Fatalf("unexpected old path: %s", oldPath)
	}
	if replaced.FileName != "new.bin" || replaced.FilePath != "/tmp/new.bin" || replaced.FileSize != 200 {
		t.Fatalf("replace fields not applied: %+v", replaced)
	}
	if replaced.FileSHA256 != "abc123" {
		t.Fatalf("expected lowercase hash, got %s", replaced.FileSHA256)
	}
}

func TestDeleteReleaseWithOperatorAudit(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:release_delete_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err = db.AutoMigrate(&models.ProductRelease{}, &models.AdminAuditLog{}); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}

	service := NewUnifiedService(db)
	release, err := service.CreateReleaseWithPackage(&CreateReleaseRequest{
		Product: "nodeclient",
		Version: "2.0.0",
		Channel: "stable",
	}, &ReleasePackageInfo{
		FileName: "nodeclient-2.0.0.tar.gz",
		FilePath: "/tmp/nodeclient-2.0.0.tar.gz",
		FileSize: 1234,
	})
	if err != nil {
		t.Fatalf("create release failed: %v", err)
	}

	oldPath, err := service.DeleteReleaseWithOperator(release.ID, 9)
	if err != nil {
		t.Fatalf("delete release failed: %v", err)
	}
	if oldPath != "/tmp/nodeclient-2.0.0.tar.gz" {
		t.Fatalf("unexpected old path: %s", oldPath)
	}

	var count int64
	if err = db.Model(&models.ProductRelease{}).Where("id = ?", release.ID).Count(&count).Error; err != nil {
		t.Fatalf("query release count failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected release not visible after soft delete, count=%d", count)
	}

	var unscopedCount int64
	if err = db.Unscoped().Model(&models.ProductRelease{}).Where("id = ?", release.ID).Count(&unscopedCount).Error; err != nil {
		t.Fatalf("query unscoped release count failed: %v", err)
	}
	if unscopedCount != 1 {
		t.Fatalf("expected release exists in recycle bin, count=%d", unscopedCount)
	}

	var logs []models.AdminAuditLog
	if err = db.Where("admin_id = ? AND action = ?", 9, AuditActionReleaseDelete).Find(&logs).Error; err != nil {
		t.Fatalf("query audit logs failed: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 audit log, got %d", len(logs))
	}

	var payload map[string]interface{}
	if err = json.Unmarshal([]byte(logs[0].PayloadJSON), &payload); err != nil {
		t.Fatalf("unmarshal payload failed: %v", err)
	}
	if payload["release_id"] == nil || payload["version"] != "2.0.0" {
		t.Fatalf("unexpected audit payload: %+v", payload)
	}
}

func TestRestoreReleaseWithOperatorAudit(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:release_restore_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err = db.AutoMigrate(&models.ProductRelease{}, &models.AdminAuditLog{}); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}

	service := NewUnifiedService(db)
	release, err := service.CreateRelease(&CreateReleaseRequest{
		Product: "backend",
		Version: "3.0.0",
		Channel: "stable",
	})
	if err != nil {
		t.Fatalf("create release failed: %v", err)
	}
	if _, err = service.DeleteReleaseWithOperator(release.ID, 10); err != nil {
		t.Fatalf("delete release failed: %v", err)
	}

	restored, err := service.RestoreReleaseWithOperator(release.ID, 11)
	if err != nil {
		t.Fatalf("restore release failed: %v", err)
	}
	if restored.ID != release.ID {
		t.Fatalf("unexpected restored id: %d", restored.ID)
	}

	var visibleCount int64
	if err = db.Model(&models.ProductRelease{}).Where("id = ?", release.ID).Count(&visibleCount).Error; err != nil {
		t.Fatalf("query release count failed: %v", err)
	}
	if visibleCount != 1 {
		t.Fatalf("expected release visible after restore, count=%d", visibleCount)
	}

	var logs []models.AdminAuditLog
	if err = db.Where("admin_id = ? AND action = ?", 11, AuditActionReleaseRestore).Find(&logs).Error; err != nil {
		t.Fatalf("query restore audit logs failed: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 restore audit log, got %d", len(logs))
	}
}

func TestListDeletedReleases(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:release_recycle_list_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err = db.AutoMigrate(&models.ProductRelease{}); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}

	service := NewUnifiedService(db)
	r1, err := service.CreateRelease(&CreateReleaseRequest{Product: "nodeclient", Version: "1.0.0", Channel: "stable"})
	if err != nil {
		t.Fatalf("create r1 failed: %v", err)
	}
	r2, err := service.CreateRelease(&CreateReleaseRequest{Product: "backend", Version: "1.0.0", Channel: "beta"})
	if err != nil {
		t.Fatalf("create r2 failed: %v", err)
	}
	if _, err = service.DeleteRelease(r1.ID); err != nil {
		t.Fatalf("delete r1 failed: %v", err)
	}
	if _, err = service.DeleteRelease(r2.ID); err != nil {
		t.Fatalf("delete r2 failed: %v", err)
	}

	items, err := service.ListDeletedReleases("nodeclient", "stable")
	if err != nil {
		t.Fatalf("list deleted releases failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 deleted release, got %d", len(items))
	}
	if items[0].ID != r1.ID {
		t.Fatalf("unexpected deleted release id: %d", items[0].ID)
	}
}

func TestPurgeReleaseWithOperatorAudit(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:release_purge_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err = db.AutoMigrate(&models.ProductRelease{}, &models.AdminAuditLog{}); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}

	service := NewUnifiedService(db)
	release, err := service.CreateReleaseWithPackage(&CreateReleaseRequest{
		Product: "frontend",
		Version: "4.0.0",
		Channel: "stable",
	}, &ReleasePackageInfo{
		FileName: "frontend-4.0.0.zip",
		FilePath: "/tmp/frontend-4.0.0.zip",
		FileSize: 8888,
	})
	if err != nil {
		t.Fatalf("create release failed: %v", err)
	}
	if _, err = service.DeleteReleaseWithOperator(release.ID, 20); err != nil {
		t.Fatalf("soft delete release failed: %v", err)
	}

	oldPath, err := service.PurgeReleaseWithOperator(release.ID, 21)
	if err != nil {
		t.Fatalf("purge release failed: %v", err)
	}
	if oldPath != "/tmp/frontend-4.0.0.zip" {
		t.Fatalf("unexpected old path: %s", oldPath)
	}

	var unscopedCount int64
	if err = db.Unscoped().Model(&models.ProductRelease{}).Where("id = ?", release.ID).Count(&unscopedCount).Error; err != nil {
		t.Fatalf("query unscoped release count failed: %v", err)
	}
	if unscopedCount != 0 {
		t.Fatalf("expected release purged, count=%d", unscopedCount)
	}

	var logs []models.AdminAuditLog
	if err = db.Where("admin_id = ? AND action = ?", 21, AuditActionReleasePurge).Find(&logs).Error; err != nil {
		t.Fatalf("query purge audit logs failed: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 purge audit log, got %d", len(logs))
	}
}
