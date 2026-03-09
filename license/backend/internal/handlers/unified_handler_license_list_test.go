package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"nodepass-license-unified/internal/models"
	"nodepass-license-unified/internal/services"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type handlerErrorResp struct {
	Success bool `json:"success"`
	Error   struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func TestListLicensesRejectInvalidSortBy(t *testing.T) {
	router := setupUnifiedListRouterTest(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/licenses?sort_by=drop_table", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d, body=%s", resp.Code, resp.Body.String())
	}

	var payload handlerErrorResp
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if payload.Error.Code != "INVALID_PARAMS" {
		t.Fatalf("expected INVALID_PARAMS, got %s", payload.Error.Code)
	}
	if !strings.Contains(payload.Error.Message, "sort_by") {
		t.Fatalf("expected sort_by error message, got %s", payload.Error.Message)
	}
}

func TestListLicensesRejectInvalidSortOrder(t *testing.T) {
	router := setupUnifiedListRouterTest(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/licenses?sort_by=created_at&sort_order=sideways", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d, body=%s", resp.Code, resp.Body.String())
	}

	var payload handlerErrorResp
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if payload.Error.Code != "INVALID_PARAMS" {
		t.Fatalf("expected INVALID_PARAMS, got %s", payload.Error.Code)
	}
	if !strings.Contains(payload.Error.Message, "sort_order") {
		t.Fatalf("expected sort_order error message, got %s", payload.Error.Message)
	}
}

func TestListLicensesAcceptValidSortQuery(t *testing.T) {
	router := setupUnifiedListRouterTest(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/licenses?sort_by=created_at&sort_order=asc", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", resp.Code, resp.Body.String())
	}
}

func setupUnifiedListRouterTest(t *testing.T) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", sanitizeHandlerTestName(t.Name()))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}

	if err = db.AutoMigrate(&models.LicensePlan{}, &models.License{}); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}

	plan := models.LicensePlan{
		Code:         "PLAN-" + strings.ToUpper(sanitizeHandlerTestName(t.Name())),
		Name:         "Plan",
		MaxMachines:  3,
		DurationDays: 365,
		Status:       "active",
	}
	if err = db.Create(&plan).Error; err != nil {
		t.Fatalf("create plan failed: %v", err)
	}

	exp := time.Now().Add(30 * 24 * time.Hour)
	license := models.License{
		Key:       "NP-HANDLER-" + strings.ToUpper(sanitizeHandlerTestName(t.Name())),
		PlanID:    plan.ID,
		Customer:  "Handler Test",
		Status:    "active",
		ExpiresAt: &exp,
		CreatedBy: 1,
	}
	if err = db.Create(&license).Error; err != nil {
		t.Fatalf("create license failed: %v", err)
	}

	handler := NewUnifiedHandler(services.NewUnifiedService(db))
	router := gin.New()
	router.GET("/api/v1/licenses", handler.ListLicenses)
	return router
}
