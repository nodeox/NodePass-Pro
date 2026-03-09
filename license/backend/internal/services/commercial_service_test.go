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

func TestHandlePaymentCallbackPaidIdempotent(t *testing.T) {
	service, db, _, _, license := setupCommercialServiceTest(t)

	order, err := service.CreateRenewOrder(&CreateRenewOrderRequest{
		LicenseID:      license.ID,
		PeriodDays:     30,
		AmountCents:    19900,
		Currency:       "cny",
		PaymentChannel: "alipay",
	}, 1)
	if err != nil {
		t.Fatalf("CreateRenewOrder failed: %v", err)
	}

	amount := int64(19900)
	paidReq := &PaymentCallbackRequest{
		OrderNo:      order.OrderNo,
		PaymentTxnID: "txn-1001",
		Status:       OrderStatusPaid,
		AmountCents:  &amount,
	}

	updated, err := service.HandlePaymentCallback("alipay", paidReq)
	if err != nil {
		t.Fatalf("HandlePaymentCallback first paid failed: %v", err)
	}
	if updated.Status != OrderStatusPaid {
		t.Fatalf("expected order status paid, got %s", updated.Status)
	}

	updated, err = service.HandlePaymentCallback("alipay", paidReq)
	if err != nil {
		t.Fatalf("HandlePaymentCallback duplicate paid failed: %v", err)
	}
	if updated.Status != OrderStatusPaid {
		t.Fatalf("expected duplicate callback keep paid, got %s", updated.Status)
	}

	var paidEventCount int64
	if err = db.Model(&models.BillingOrderEvent{}).
		Where("order_id = ? AND event_type = ?", order.ID, OrderEventPaymentPaid).
		Count(&paidEventCount).Error; err != nil {
		t.Fatalf("count paid events failed: %v", err)
	}
	if paidEventCount != 1 {
		t.Fatalf("expected 1 paid event, got %d", paidEventCount)
	}

	_, err = service.HandlePaymentCallback("alipay", &PaymentCallbackRequest{
		OrderNo:      order.OrderNo,
		PaymentTxnID: "txn-1002",
		Status:       OrderStatusPaid,
		AmountCents:  &amount,
	})
	if err == nil {
		t.Fatalf("expected different txn id callback to fail")
	}
}

func TestHandlePaymentCallbackAmountMismatch(t *testing.T) {
	service, db, _, _, license := setupCommercialServiceTest(t)

	order, err := service.CreateRenewOrder(&CreateRenewOrderRequest{
		LicenseID:      license.ID,
		PeriodDays:     30,
		AmountCents:    9900,
		Currency:       "CNY",
		PaymentChannel: "wechat",
	}, 1)
	if err != nil {
		t.Fatalf("CreateRenewOrder failed: %v", err)
	}

	actualAmount := int64(9800)
	_, err = service.HandlePaymentCallback("", &PaymentCallbackRequest{
		OrderNo:      order.OrderNo,
		PaymentTxnID: "txn-mismatch-1",
		Status:       OrderStatusPaid,
		AmountCents:  &actualAmount,
	})
	if err == nil {
		t.Fatalf("expected amount mismatch callback to fail")
	}
	if !strings.Contains(err.Error(), "回调金额与订单金额不一致") {
		t.Fatalf("unexpected error message: %v", err)
	}

	var persisted models.BillingOrder
	if err = db.First(&persisted, order.ID).Error; err != nil {
		t.Fatalf("query order failed: %v", err)
	}
	if persisted.Status != OrderStatusPending {
		t.Fatalf("expected status pending after mismatch, got %s", persisted.Status)
	}

	if persisted.PaymentChannel != "wechat" {
		t.Fatalf("expected channel fallback to order channel wechat, got %s", persisted.PaymentChannel)
	}

	var mismatchEventCount int64
	if err = db.Model(&models.BillingOrderEvent{}).
		Where("order_id = ? AND event_type = ?", order.ID, OrderEventPaymentAmountMismatch).
		Count(&mismatchEventCount).Error; err != nil {
		t.Fatalf("count mismatch events failed: %v", err)
	}
	if mismatchEventCount != 1 {
		t.Fatalf("expected 1 amount mismatch event, got %d", mismatchEventCount)
	}
}

func TestMarkOrderPaidApplyRenew(t *testing.T) {
	service, db, _, _, license := setupCommercialServiceTest(t)

	startExpiry := time.Now().Add(15 * 24 * time.Hour).Round(time.Second)
	if err := db.Model(&models.License{}).Where("id = ?", license.ID).Update("expires_at", startExpiry).Error; err != nil {
		t.Fatalf("preset expires_at failed: %v", err)
	}

	order, err := service.CreateRenewOrder(&CreateRenewOrderRequest{
		LicenseID:      license.ID,
		PeriodDays:     10,
		AmountCents:    12000,
		Currency:       "CNY",
		PaymentChannel: "manual",
	}, 1)
	if err != nil {
		t.Fatalf("CreateRenewOrder failed: %v", err)
	}

	_, err = service.MarkOrderPaid(order.ID, &MarkOrderPaidRequest{
		PaymentTxnID: "manual-renew-1",
		Channel:      "manual",
	}, 1)
	if err != nil {
		t.Fatalf("MarkOrderPaid failed: %v", err)
	}

	var renewed models.License
	if err = db.First(&renewed, license.ID).Error; err != nil {
		t.Fatalf("query license failed: %v", err)
	}
	if renewed.ExpiresAt == nil {
		t.Fatalf("expected renewed license has expires_at")
	}

	expected := startExpiry.Add(10 * 24 * time.Hour)
	diff := renewed.ExpiresAt.Sub(expected)
	if diff < 0 {
		diff = -diff
	}
	if diff > 2*time.Second {
		t.Fatalf("renewed expiry mismatch, expected around %v, got %v", expected, renewed.ExpiresAt)
	}
}

func TestMarkOrderPaidApplyUpgrade(t *testing.T) {
	service, db, _, targetPlan, license := setupCommercialServiceTest(t)

	order, err := service.CreateUpgradeOrder(&CreateUpgradeOrderRequest{
		LicenseID:      license.ID,
		ToPlanID:       targetPlan.ID,
		PeriodDays:     0,
		AmountCents:    5000,
		Currency:       "CNY",
		PaymentChannel: "manual",
	}, 1)
	if err != nil {
		t.Fatalf("CreateUpgradeOrder failed: %v", err)
	}

	_, err = service.MarkOrderPaid(order.ID, &MarkOrderPaidRequest{
		PaymentTxnID: "manual-upgrade-1",
		Channel:      "manual",
	}, 1)
	if err != nil {
		t.Fatalf("MarkOrderPaid failed: %v", err)
	}

	var upgraded models.License
	if err = db.First(&upgraded, license.ID).Error; err != nil {
		t.Fatalf("query upgraded license failed: %v", err)
	}
	if upgraded.PlanID != targetPlan.ID {
		t.Fatalf("expected upgraded plan id %d, got %d", targetPlan.ID, upgraded.PlanID)
	}
}

func TestMarkOrderPaidApplyTransfer(t *testing.T) {
	service, db, _, _, license := setupCommercialServiceTest(t)

	order, err := service.CreateTransferOrder(&CreateTransferOrderRequest{
		LicenseID:      license.ID,
		ToCustomer:     "Beta LLC",
		Reason:         "客户主体变更",
		AmountCents:    1000,
		Currency:       "CNY",
		PaymentChannel: "manual",
	}, 9)
	if err != nil {
		t.Fatalf("CreateTransferOrder failed: %v", err)
	}

	_, err = service.MarkOrderPaid(order.ID, &MarkOrderPaidRequest{
		PaymentTxnID: "manual-transfer-1",
		Channel:      "manual",
	}, 9)
	if err != nil {
		t.Fatalf("MarkOrderPaid failed: %v", err)
	}

	var transferred models.License
	if err = db.First(&transferred, license.ID).Error; err != nil {
		t.Fatalf("query transferred license failed: %v", err)
	}
	if transferred.Customer != "Beta LLC" {
		t.Fatalf("expected transferred customer Beta LLC, got %s", transferred.Customer)
	}

	var logs []models.LicenseTransferLog
	if err = db.Where("license_id = ?", license.ID).Find(&logs).Error; err != nil {
		t.Fatalf("query transfer logs failed: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 transfer log, got %d", len(logs))
	}
	if logs[0].OrderID == nil || *logs[0].OrderID != order.ID {
		t.Fatalf("expected transfer log order_id = %d, got %+v", order.ID, logs[0].OrderID)
	}
	if logs[0].OperatorID != 9 {
		t.Fatalf("expected transfer log operator_id 9, got %d", logs[0].OperatorID)
	}
}

func TestHandlePaymentCallbackRejectPaidAfterFailed(t *testing.T) {
	service, db, _, _, license := setupCommercialServiceTest(t)

	order, err := service.CreateRenewOrder(&CreateRenewOrderRequest{
		LicenseID:      license.ID,
		PeriodDays:     30,
		AmountCents:    19900,
		Currency:       "CNY",
		PaymentChannel: "alipay",
	}, 1)
	if err != nil {
		t.Fatalf("CreateRenewOrder failed: %v", err)
	}

	_, err = service.HandlePaymentCallback("alipay", &PaymentCallbackRequest{
		OrderNo:      order.OrderNo,
		PaymentTxnID: "txn-failed-1",
		Status:       OrderStatusFailed,
	})
	if err != nil {
		t.Fatalf("mark failed by callback should succeed: %v", err)
	}

	paidAmount := int64(19900)
	_, err = service.HandlePaymentCallback("alipay", &PaymentCallbackRequest{
		OrderNo:      order.OrderNo,
		PaymentTxnID: "txn-paid-after-failed",
		Status:       OrderStatusPaid,
		AmountCents:  &paidAmount,
	})
	if err == nil {
		t.Fatalf("expected paid after failed to be rejected")
	}
	if !strings.Contains(err.Error(), "当前订单状态不允许支付确认") {
		t.Fatalf("unexpected error: %v", err)
	}

	var persisted models.BillingOrder
	if err = db.First(&persisted, order.ID).Error; err != nil {
		t.Fatalf("query order failed: %v", err)
	}
	if persisted.Status != OrderStatusFailed {
		t.Fatalf("expected status keep failed, got %s", persisted.Status)
	}
}

func TestHandlePaymentCallbackRejectRollbackBetweenClosedStatuses(t *testing.T) {
	service, db, _, _, license := setupCommercialServiceTest(t)

	order, err := service.CreateRenewOrder(&CreateRenewOrderRequest{
		LicenseID:      license.ID,
		PeriodDays:     30,
		AmountCents:    19900,
		Currency:       "CNY",
		PaymentChannel: "wechat",
	}, 1)
	if err != nil {
		t.Fatalf("CreateRenewOrder failed: %v", err)
	}

	_, err = service.HandlePaymentCallback("wechat", &PaymentCallbackRequest{
		OrderNo:      order.OrderNo,
		PaymentTxnID: "txn-canceled-1",
		Status:       OrderStatusCanceled,
	})
	if err != nil {
		t.Fatalf("mark canceled by callback should succeed: %v", err)
	}

	_, err = service.HandlePaymentCallback("wechat", &PaymentCallbackRequest{
		OrderNo:      order.OrderNo,
		PaymentTxnID: "txn-failed-after-canceled",
		Status:       OrderStatusFailed,
	})
	if err == nil {
		t.Fatalf("expected failed after canceled to be rejected")
	}
	if !strings.Contains(err.Error(), "当前订单状态不允许变更为 failed") {
		t.Fatalf("unexpected error: %v", err)
	}

	var persisted models.BillingOrder
	if err = db.First(&persisted, order.ID).Error; err != nil {
		t.Fatalf("query order failed: %v", err)
	}
	if persisted.Status != OrderStatusCanceled {
		t.Fatalf("expected status keep canceled, got %s", persisted.Status)
	}
}

func setupCommercialServiceTest(t *testing.T) (*CommercialService, *gorm.DB, models.LicensePlan, models.LicensePlan, models.License) {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", sanitizeTestName(t.Name()))
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

	basePlan := models.LicensePlan{
		Code:         "BASIC-" + sanitizeTestName(t.Name()),
		Name:         "Basic Plan",
		Description:  "for test",
		MaxMachines:  3,
		DurationDays: 365,
		Status:       "active",
	}
	if err = db.Create(&basePlan).Error; err != nil {
		t.Fatalf("create base plan failed: %v", err)
	}

	targetPlan := models.LicensePlan{
		Code:         "PRO-" + sanitizeTestName(t.Name()),
		Name:         "Pro Plan",
		Description:  "for test",
		MaxMachines:  10,
		DurationDays: 365,
		Status:       "active",
	}
	if err = db.Create(&targetPlan).Error; err != nil {
		t.Fatalf("create target plan failed: %v", err)
	}

	exp := time.Now().Add(30 * 24 * time.Hour)
	license := models.License{
		Key:       "NP-TEST-" + strings.ToUpper(sanitizeTestName(t.Name())),
		PlanID:    basePlan.ID,
		Customer:  "Acme Inc",
		Status:    "active",
		ExpiresAt: &exp,
		CreatedBy: 1,
	}
	if err = db.Create(&license).Error; err != nil {
		t.Fatalf("create license failed: %v", err)
	}

	return NewCommercialService(db), db, basePlan, targetPlan, license
}

func sanitizeTestName(name string) string {
	replacer := strings.NewReplacer("/", "_", " ", "_", "-", "_")
	return strings.ToLower(replacer.Replace(name))
}
