package handlers

import (
	"bytes"
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

func TestPaymentCallbackInvalidSignature(t *testing.T) {
	router, db, order := setupCommercialCallbackRouterTest(t)

	amount := int64(1000)
	reqBody := services.PaymentCallbackRequest{
		OrderNo:      order.OrderNo,
		PaymentTxnID: "txn-invalid-signature",
		Status:       services.OrderStatusPaid,
		AmountCents:  &amount,
	}
	raw, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("marshal request failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/commercial/payments/callback/alipay", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(services.CallbackHeaderTimestamp, strconvNow())
	req.Header.Set(services.CallbackHeaderNonce, "n1")
	req.Header.Set(services.CallbackHeaderSignature, "deadbeef")

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d, body=%s", resp.Code, resp.Body.String())
	}
	if !strings.Contains(resp.Body.String(), "INVALID_SIGNATURE") {
		t.Fatalf("expected INVALID_SIGNATURE code, got body=%s", resp.Body.String())
	}

	var persisted models.BillingOrder
	if err = db.First(&persisted, order.ID).Error; err != nil {
		t.Fatalf("query order failed: %v", err)
	}
	if persisted.Status != services.OrderStatusPending {
		t.Fatalf("expected status pending, got %s", persisted.Status)
	}
}

func TestPaymentCallbackValidSignature(t *testing.T) {
	router, db, order := setupCommercialCallbackRouterTest(t)

	amount := int64(1000)
	reqBody := services.PaymentCallbackRequest{
		OrderNo:      order.OrderNo,
		PaymentTxnID: "txn-valid-signature",
		Status:       services.OrderStatusPaid,
		AmountCents:  &amount,
	}
	timestamp := strconvNow()
	nonce := "n2"
	signature := services.GeneratePaymentCallbackSignature("alipay-secret-test", "alipay", &reqBody, timestamp, nonce)

	raw, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("marshal request failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/commercial/payments/callback/alipay", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(services.CallbackHeaderTimestamp, timestamp)
	req.Header.Set(services.CallbackHeaderNonce, nonce)
	req.Header.Set(services.CallbackHeaderSignature, signature)

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", resp.Code, resp.Body.String())
	}

	var persisted models.BillingOrder
	if err = db.First(&persisted, order.ID).Error; err != nil {
		t.Fatalf("query order failed: %v", err)
	}
	if persisted.Status != services.OrderStatusPaid {
		t.Fatalf("expected status paid, got %s", persisted.Status)
	}
}

func setupCommercialCallbackRouterTest(t *testing.T) (*gin.Engine, *gorm.DB, *models.BillingOrder) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", sanitizeHandlerTestName(t.Name()))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}

	if err = db.AutoMigrate(
		&models.LicensePlan{},
		&models.License{},
		&models.BillingOrder{},
		&models.BillingOrderEvent{},
		&models.LicenseTransferLog{},
		&models.TrialIssue{},
	); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}

	plan := models.LicensePlan{
		Code:         "PLAN-" + sanitizeHandlerTestName(t.Name()),
		Name:         "Plan",
		MaxMachines:  3,
		DurationDays: 365,
		Status:       "active",
	}
	if err = db.Create(&plan).Error; err != nil {
		t.Fatalf("create plan failed: %v", err)
	}

	expAt := time.Now().Add(7 * 24 * time.Hour)
	license := models.License{
		Key:       "NP-TEST-CALLBACK-" + strings.ToUpper(sanitizeHandlerTestName(t.Name())),
		PlanID:    plan.ID,
		Customer:  "Callback Test",
		Status:    "active",
		ExpiresAt: &expAt,
		CreatedBy: 1,
	}
	if err = db.Create(&license).Error; err != nil {
		t.Fatalf("create license failed: %v", err)
	}

	verifier := services.NewPaymentCallbackVerifier(true, 300, map[string]string{
		"alipay": "alipay-secret-test",
	})
	commercialService := services.NewCommercialService(db, verifier)
	order, err := commercialService.CreateRenewOrder(&services.CreateRenewOrderRequest{
		LicenseID:      license.ID,
		PeriodDays:     30,
		AmountCents:    1000,
		Currency:       "CNY",
		PaymentChannel: "alipay",
	}, 1)
	if err != nil {
		t.Fatalf("create renew order failed: %v", err)
	}

	handler := NewCommercialHandler(commercialService)
	router := gin.New()
	router.POST("/api/v1/commercial/payments/callback/:channel", handler.PaymentCallback)
	return router, db, order
}

func sanitizeHandlerTestName(name string) string {
	replacer := strings.NewReplacer("/", "_", " ", "_", "-", "_")
	return strings.ToLower(replacer.Replace(name))
}

func strconvNow() string {
	return fmt.Sprintf("%d", time.Now().Unix())
}
