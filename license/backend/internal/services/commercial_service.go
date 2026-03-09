package services

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"nodepass-license-unified/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	OrderActionRenew    = "renew"
	OrderActionUpgrade  = "upgrade"
	OrderActionTransfer = "transfer"

	OrderStatusPending  = "pending"
	OrderStatusPaid     = "paid"
	OrderStatusFailed   = "failed"
	OrderStatusCanceled = "canceled"

	OrderEventCreated               = "order_created"
	OrderEventPaymentPaid           = "payment_paid"
	OrderEventPaymentFailed         = "payment_failed"
	OrderEventPaymentCanceled       = "payment_canceled"
	OrderEventPaymentAmountMismatch = "payment_amount_mismatch"
)

// CommercialService 商业化服务。
type CommercialService struct {
	db               *gorm.DB
	callbackVerifier *PaymentCallbackVerifier
}

// NewCommercialService 创建商业化服务。
func NewCommercialService(db *gorm.DB, callbackVerifier ...*PaymentCallbackVerifier) *CommercialService {
	service := &CommercialService{db: db}
	if len(callbackVerifier) > 0 {
		service.callbackVerifier = callbackVerifier[0]
	}
	return service
}

// IssueTrialRequest 发放试用请求。
type IssueTrialRequest struct {
	PlanID       uint   `json:"plan_id" binding:"required"`
	Customer     string `json:"customer" binding:"required"`
	TrialDays    int    `json:"trial_days"`
	MaxMachines  *int   `json:"max_machines"`
	MetadataJSON string `json:"metadata_json"`
	Note         string `json:"note"`
}

// IssueTrialResult 发放试用结果。
type IssueTrialResult struct {
	License   models.License    `json:"license"`
	TrialInfo models.TrialIssue `json:"trial_info"`
}

// CreateRenewOrderRequest 创建续费订单请求。
type CreateRenewOrderRequest struct {
	LicenseID      uint   `json:"license_id" binding:"required"`
	PeriodDays     int    `json:"period_days"`
	AmountCents    int64  `json:"amount_cents"`
	Currency       string `json:"currency"`
	PaymentChannel string `json:"payment_channel"`
}

// CreateUpgradeOrderRequest 创建升级订单请求。
type CreateUpgradeOrderRequest struct {
	LicenseID      uint   `json:"license_id" binding:"required"`
	ToPlanID       uint   `json:"to_plan_id" binding:"required"`
	PeriodDays     int    `json:"period_days"`
	AmountCents    int64  `json:"amount_cents"`
	Currency       string `json:"currency"`
	PaymentChannel string `json:"payment_channel"`
}

// CreateTransferOrderRequest 创建转移订单请求。
type CreateTransferOrderRequest struct {
	LicenseID      uint   `json:"license_id" binding:"required"`
	ToCustomer     string `json:"to_customer" binding:"required"`
	Reason         string `json:"reason"`
	AmountCents    int64  `json:"amount_cents"`
	Currency       string `json:"currency"`
	PaymentChannel string `json:"payment_channel"`
}

// PaymentCallbackRequest 支付回调请求。
type PaymentCallbackRequest struct {
	OrderNo      string          `json:"order_no" binding:"required"`
	PaymentTxnID string          `json:"payment_txn_id"`
	Status       string          `json:"status" binding:"required"`
	AmountCents  *int64          `json:"amount_cents"`
	RawPayload   json.RawMessage `json:"raw_payload"`
}

// MarkOrderPaidRequest 手动确认支付请求。
type MarkOrderPaidRequest struct {
	PaymentTxnID string `json:"payment_txn_id"`
	Channel      string `json:"channel"`
}

// OrderFilter 订单过滤条件。
type OrderFilter struct {
	Status    string
	Action    string
	Customer  string
	LicenseID uint
	Page      int
	PageSize  int
}

// OrderListResult 订单列表分页。
type OrderListResult struct {
	Items    []models.BillingOrder `json:"items"`
	Total    int64                 `json:"total"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"page_size"`
}

// OrderDetail 订单详情。
type OrderDetail struct {
	Order  models.BillingOrder        `json:"order"`
	Events []models.BillingOrderEvent `json:"events"`
}

// VerifyPaymentCallbackSignature 验证支付回调签名。
func (s *CommercialService) VerifyPaymentCallbackSignature(channel string, req *PaymentCallbackRequest, signature, timestamp, nonce string) error {
	if s == nil || s.callbackVerifier == nil {
		return nil
	}
	return s.callbackVerifier.Verify(channel, req, signature, timestamp, nonce)
}

// IssueTrial 发放试用授权。
func (s *CommercialService) IssueTrial(req *IssueTrialRequest, adminID uint) (*IssueTrialResult, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("service unavailable")
	}

	trialDays := req.TrialDays
	if trialDays <= 0 {
		trialDays = 14
	}
	if trialDays > 365 {
		return nil, errors.New("trial_days 不能超过 365")
	}

	var plan models.LicensePlan
	if err := s.db.Where("id = ?", req.PlanID).First(&plan).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("套餐不存在")
		}
		return nil, err
	}

	licenseKey, err := generateOrderLicenseKey()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	expiresAt := now.Add(time.Duration(trialDays) * 24 * time.Hour)
	trialNote := strings.TrimSpace(req.Note)
	if trialNote == "" {
		trialNote = fmt.Sprintf("试用授权 %d 天", trialDays)
	}

	license := models.License{
		Key:          licenseKey,
		PlanID:       plan.ID,
		Customer:     strings.TrimSpace(req.Customer),
		Status:       "active",
		ExpiresAt:    &expiresAt,
		MaxMachines:  req.MaxMachines,
		MetadataJSON: strings.TrimSpace(req.MetadataJSON),
		Note:         trialNote,
		CreatedBy:    adminID,
	}

	trial := models.TrialIssue{
		Customer:  strings.TrimSpace(req.Customer),
		TrialDays: trialDays,
		IssuedBy:  adminID,
		ExpiresAt: &expiresAt,
	}

	if err = s.db.Transaction(func(tx *gorm.DB) error {
		if errTx := tx.Create(&license).Error; errTx != nil {
			return errTx
		}
		trial.LicenseID = license.ID
		if errTx := tx.Create(&trial).Error; errTx != nil {
			return errTx
		}
		return nil
	}); err != nil {
		return nil, err
	}

	if err = s.db.Preload("Plan").First(&license, license.ID).Error; err != nil {
		return nil, err
	}

	return &IssueTrialResult{
		License:   license,
		TrialInfo: trial,
	}, nil
}

// CreateRenewOrder 创建续费订单。
func (s *CommercialService) CreateRenewOrder(req *CreateRenewOrderRequest, adminID uint) (*models.BillingOrder, error) {
	if req.AmountCents < 0 {
		return nil, errors.New("amount_cents 不能为负数")
	}

	periodDays := req.PeriodDays
	if periodDays <= 0 {
		periodDays = 30
	}

	var license models.License
	if err := s.db.First(&license, req.LicenseID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("授权不存在")
		}
		return nil, err
	}

	fromPlanID := license.PlanID
	order, err := s.createOrder(&models.BillingOrder{
		LicenseID:      license.ID,
		Action:         OrderActionRenew,
		FromPlanID:     &fromPlanID,
		PeriodDays:     periodDays,
		AmountCents:    req.AmountCents,
		Currency:       normalizeCurrency(req.Currency),
		Status:         OrderStatusPending,
		PaymentChannel: normalizePaymentChannel(req.PaymentChannel),
		CreatedBy:      adminID,
	}, map[string]interface{}{
		"action":      OrderActionRenew,
		"license_id":  license.ID,
		"period_days": periodDays,
	})
	if err != nil {
		return nil, err
	}
	return order, nil
}

// CreateUpgradeOrder 创建升级订单。
func (s *CommercialService) CreateUpgradeOrder(req *CreateUpgradeOrderRequest, adminID uint) (*models.BillingOrder, error) {
	if req.AmountCents < 0 {
		return nil, errors.New("amount_cents 不能为负数")
	}

	var license models.License
	if err := s.db.First(&license, req.LicenseID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("授权不存在")
		}
		return nil, err
	}
	if license.PlanID == req.ToPlanID {
		return nil, errors.New("目标套餐与当前套餐一致")
	}

	var targetPlan models.LicensePlan
	if err := s.db.Where("id = ?", req.ToPlanID).First(&targetPlan).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("目标套餐不存在")
		}
		return nil, err
	}

	periodDays := req.PeriodDays
	if periodDays < 0 {
		return nil, errors.New("period_days 不能为负数")
	}

	fromPlanID := license.PlanID
	toPlanID := req.ToPlanID
	order, err := s.createOrder(&models.BillingOrder{
		LicenseID:      license.ID,
		Action:         OrderActionUpgrade,
		FromPlanID:     &fromPlanID,
		ToPlanID:       &toPlanID,
		PeriodDays:     periodDays,
		AmountCents:    req.AmountCents,
		Currency:       normalizeCurrency(req.Currency),
		Status:         OrderStatusPending,
		PaymentChannel: normalizePaymentChannel(req.PaymentChannel),
		CreatedBy:      adminID,
	}, map[string]interface{}{
		"action":       OrderActionUpgrade,
		"license_id":   license.ID,
		"from_plan_id": fromPlanID,
		"to_plan_id":   toPlanID,
	})
	if err != nil {
		return nil, err
	}
	return order, nil
}

// CreateTransferOrder 创建转移订单。
func (s *CommercialService) CreateTransferOrder(req *CreateTransferOrderRequest, adminID uint) (*models.BillingOrder, error) {
	if req.AmountCents < 0 {
		return nil, errors.New("amount_cents 不能为负数")
	}

	toCustomer := strings.TrimSpace(req.ToCustomer)
	if toCustomer == "" {
		return nil, errors.New("to_customer 不能为空")
	}

	var license models.License
	if err := s.db.First(&license, req.LicenseID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("授权不存在")
		}
		return nil, err
	}
	if strings.EqualFold(strings.TrimSpace(license.Customer), toCustomer) {
		return nil, errors.New("目标客户与当前客户一致")
	}

	fromPlanID := license.PlanID
	order, err := s.createOrder(&models.BillingOrder{
		LicenseID:      license.ID,
		Action:         OrderActionTransfer,
		FromPlanID:     &fromPlanID,
		TargetCustomer: toCustomer,
		AmountCents:    req.AmountCents,
		Currency:       normalizeCurrency(req.Currency),
		Status:         OrderStatusPending,
		PaymentChannel: normalizePaymentChannel(req.PaymentChannel),
		CreatedBy:      adminID,
	}, map[string]interface{}{
		"action":        OrderActionTransfer,
		"license_id":    license.ID,
		"from_customer": license.Customer,
		"to_customer":   toCustomer,
		"reason":        strings.TrimSpace(req.Reason),
	})
	if err != nil {
		return nil, err
	}
	return order, nil
}

// ListOrders 查询订单列表。
func (s *CommercialService) ListOrders(filter OrderFilter) (*OrderListResult, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.PageSize > 200 {
		filter.PageSize = 200
	}

	query := s.db.Model(&models.BillingOrder{})
	if filter.Status != "" {
		query = query.Where("status = ?", strings.TrimSpace(filter.Status))
	}
	if filter.Action != "" {
		query = query.Where("action = ?", strings.TrimSpace(filter.Action))
	}
	if filter.LicenseID > 0 {
		query = query.Where("license_id = ?", filter.LicenseID)
	}
	if strings.TrimSpace(filter.Customer) != "" {
		query = query.Joins("JOIN licenses ON licenses.id = billing_orders.license_id").
			Where("licenses.customer LIKE ?", "%"+strings.TrimSpace(filter.Customer)+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	var items []models.BillingOrder
	if err := query.Order("id desc").Offset((filter.Page - 1) * filter.PageSize).Limit(filter.PageSize).Find(&items).Error; err != nil {
		return nil, err
	}

	return &OrderListResult{
		Items:    items,
		Total:    total,
		Page:     filter.Page,
		PageSize: filter.PageSize,
	}, nil
}

// GetOrder 获取订单详情。
func (s *CommercialService) GetOrder(id uint) (*OrderDetail, error) {
	var order models.BillingOrder
	if err := s.db.First(&order, id).Error; err != nil {
		return nil, err
	}

	var events []models.BillingOrderEvent
	if err := s.db.Where("order_id = ?", id).Order("id asc").Find(&events).Error; err != nil {
		return nil, err
	}

	return &OrderDetail{Order: order, Events: events}, nil
}

// MarkOrderPaid 手动确认支付。
func (s *CommercialService) MarkOrderPaid(orderID uint, req *MarkOrderPaidRequest, adminID uint) (*models.BillingOrder, error) {
	channel := normalizePaymentChannel(req.Channel)
	txnID := strings.TrimSpace(req.PaymentTxnID)
	if txnID == "" {
		txnID = fmt.Sprintf("manual-%d", time.Now().UnixNano())
	}

	var resultOrder models.BillingOrder
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var order models.BillingOrder
		if errTx := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", orderID).First(&order).Error; errTx != nil {
			return errTx
		}

		updated, errTx := s.transitionOrderToPaid(tx, &order, channel, txnID, map[string]interface{}{
			"operator_id": adminID,
			"source":      "manual_mark_paid",
		})
		if errTx != nil {
			return errTx
		}
		resultOrder = *updated
		return nil
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("订单不存在")
		}
		return nil, err
	}

	return &resultOrder, nil
}

// HandlePaymentCallback 处理支付回调（幂等）。
func (s *CommercialService) HandlePaymentCallback(channel string, req *PaymentCallbackRequest) (*models.BillingOrder, error) {
	callbackStatus := normalizeCallbackStatus(req.Status)
	if callbackStatus == "" {
		return nil, errors.New("status 仅支持 paid/failed/canceled")
	}

	txnID := strings.TrimSpace(req.PaymentTxnID)
	requestedChannel := strings.TrimSpace(strings.ToLower(channel))

	payload := map[string]interface{}{
		"channel":        requestedChannel,
		"status":         callbackStatus,
		"payment_txn_id": txnID,
	}
	if len(req.RawPayload) > 0 {
		payload["raw_payload"] = json.RawMessage(req.RawPayload)
	}
	if req.AmountCents != nil {
		payload["amount_cents"] = *req.AmountCents
	}

	var resultOrder models.BillingOrder
	var callbackErr error
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var order models.BillingOrder
		if errTx := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("order_no = ?", strings.TrimSpace(req.OrderNo)).
			First(&order).Error; errTx != nil {
			return errTx
		}

		normalizedChannel := normalizePaymentChannel(requestedChannel)
		if requestedChannel == "" {
			normalizedChannel = normalizePaymentChannel(order.PaymentChannel)
		}
		payload["channel"] = normalizedChannel

		switch callbackStatus {
		case OrderStatusPaid:
			if req.AmountCents != nil && *req.AmountCents != order.AmountCents {
				payload["expected_amount_cents"] = order.AmountCents
				if errTx := createOrderEvent(tx, order.ID, OrderEventPaymentAmountMismatch, payload); errTx != nil {
					return errTx
				}
				callbackErr = fmt.Errorf("回调金额与订单金额不一致，expected=%d, actual=%d", order.AmountCents, *req.AmountCents)
				return nil
			}

			updated, errTx := s.transitionOrderToPaid(tx, &order, normalizedChannel, txnID, payload)
			if errTx != nil {
				return errTx
			}
			resultOrder = *updated
			return nil
		case OrderStatusFailed, OrderStatusCanceled:
			updated, errTx := s.transitionOrderToClosed(tx, &order, callbackStatus, normalizedChannel, txnID, payload)
			if errTx != nil {
				return errTx
			}
			resultOrder = *updated
			return nil
		default:
			return errors.New("status 仅支持 paid/failed/canceled")
		}
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("订单不存在")
		}
		return nil, err
	}
	if callbackErr != nil {
		return nil, callbackErr
	}
	return &resultOrder, nil
}

func (s *CommercialService) transitionOrderToPaid(tx *gorm.DB, order *models.BillingOrder, channel string, txnID string, payload map[string]interface{}) (*models.BillingOrder, error) {
	if order.Status == OrderStatusPaid {
		// 幂等回调：已支付直接返回
		if txnID != "" && order.PaymentTxnID != nil && *order.PaymentTxnID != txnID {
			return nil, errors.New("订单已支付且交易号不一致")
		}
		return order, nil
	}
	if order.Status != OrderStatusPending {
		return nil, fmt.Errorf("当前订单状态不允许支付确认: %s", order.Status)
	}

	now := time.Now()
	updates := map[string]interface{}{
		"status":          OrderStatusPaid,
		"paid_at":         &now,
		"payment_channel": channel,
	}
	if txnID != "" {
		txnIDCopy := txnID
		updates["payment_txn_id"] = &txnIDCopy
		order.PaymentTxnID = &txnIDCopy
	}
	if err := tx.Model(&models.BillingOrder{}).Where("id = ?", order.ID).Updates(updates).Error; err != nil {
		return nil, err
	}

	order.Status = OrderStatusPaid
	order.PaidAt = &now
	order.PaymentChannel = channel

	if err := s.applyPaidOrder(tx, order); err != nil {
		return nil, err
	}
	if err := createOrderEvent(tx, order.ID, OrderEventPaymentPaid, payload); err != nil {
		return nil, err
	}
	return order, nil
}

func (s *CommercialService) transitionOrderToClosed(tx *gorm.DB, order *models.BillingOrder, targetStatus, channel, txnID string, payload map[string]interface{}) (*models.BillingOrder, error) {
	if order.Status == OrderStatusPaid {
		return nil, errors.New("订单已支付，不能回退状态")
	}
	if order.Status == targetStatus {
		// 幂等重复回调
		return order, nil
	}
	if order.Status != OrderStatusPending {
		return nil, fmt.Errorf("当前订单状态不允许变更为 %s: %s", targetStatus, order.Status)
	}

	updates := map[string]interface{}{
		"status":          targetStatus,
		"payment_channel": channel,
	}
	if txnID != "" {
		txnIDCopy := txnID
		updates["payment_txn_id"] = &txnIDCopy
		order.PaymentTxnID = &txnIDCopy
	}
	if err := tx.Model(&models.BillingOrder{}).Where("id = ?", order.ID).Updates(updates).Error; err != nil {
		return nil, err
	}
	order.Status = targetStatus
	order.PaymentChannel = channel

	eventType := OrderEventPaymentFailed
	if targetStatus == OrderStatusCanceled {
		eventType = OrderEventPaymentCanceled
	}
	if err := createOrderEvent(tx, order.ID, eventType, payload); err != nil {
		return nil, err
	}
	return order, nil
}

func (s *CommercialService) applyPaidOrder(tx *gorm.DB, order *models.BillingOrder) error {
	switch order.Action {
	case OrderActionRenew:
		return s.applyRenew(tx, order)
	case OrderActionUpgrade:
		return s.applyUpgrade(tx, order)
	case OrderActionTransfer:
		return s.applyTransfer(tx, order)
	default:
		return fmt.Errorf("不支持的订单动作: %s", order.Action)
	}
}

func (s *CommercialService) applyRenew(tx *gorm.DB, order *models.BillingOrder) error {
	var license models.License
	if err := tx.Where("id = ?", order.LicenseID).First(&license).Error; err != nil {
		return err
	}

	periodDays := order.PeriodDays
	if periodDays <= 0 {
		periodDays = 30
	}

	base := time.Now()
	if license.ExpiresAt != nil && license.ExpiresAt.After(base) {
		base = *license.ExpiresAt
	}
	newExpiresAt := base.Add(time.Duration(periodDays) * 24 * time.Hour)

	updateMap := map[string]interface{}{
		"expires_at": newExpiresAt,
		"status":     "active",
	}
	return tx.Model(&models.License{}).Where("id = ?", license.ID).Updates(updateMap).Error
}

func (s *CommercialService) applyUpgrade(tx *gorm.DB, order *models.BillingOrder) error {
	if order.ToPlanID == nil {
		return errors.New("升级订单缺少目标套餐")
	}

	var plan models.LicensePlan
	if err := tx.Where("id = ?", *order.ToPlanID).First(&plan).Error; err != nil {
		return err
	}

	var license models.License
	if err := tx.Where("id = ?", order.LicenseID).First(&license).Error; err != nil {
		return err
	}

	updateMap := map[string]interface{}{
		"plan_id": *order.ToPlanID,
		"status":  "active",
	}

	if order.PeriodDays > 0 {
		base := time.Now()
		if license.ExpiresAt != nil && license.ExpiresAt.After(base) {
			base = *license.ExpiresAt
		}
		newExpiresAt := base.Add(time.Duration(order.PeriodDays) * 24 * time.Hour)
		updateMap["expires_at"] = newExpiresAt
	}

	return tx.Model(&models.License{}).Where("id = ?", license.ID).Updates(updateMap).Error
}

func (s *CommercialService) applyTransfer(tx *gorm.DB, order *models.BillingOrder) error {
	toCustomer := strings.TrimSpace(order.TargetCustomer)
	if toCustomer == "" {
		return errors.New("转移订单缺少目标客户")
	}

	var license models.License
	if err := tx.Where("id = ?", order.LicenseID).First(&license).Error; err != nil {
		return err
	}

	fromCustomer := strings.TrimSpace(license.Customer)
	if strings.EqualFold(fromCustomer, toCustomer) {
		return nil
	}

	if err := tx.Model(&models.License{}).Where("id = ?", license.ID).Update("customer", toCustomer).Error; err != nil {
		return err
	}

	transferLog := models.LicenseTransferLog{
		LicenseID:    license.ID,
		OrderID:      &order.ID,
		FromCustomer: fromCustomer,
		ToCustomer:   toCustomer,
		Reason:       "商业化转移订单执行",
		OperatorID:   order.CreatedBy,
	}
	return tx.Create(&transferLog).Error
}

func (s *CommercialService) createOrder(order *models.BillingOrder, payload map[string]interface{}) (*models.BillingOrder, error) {
	orderNo, err := generateOrderNo()
	if err != nil {
		return nil, err
	}
	order.OrderNo = orderNo

	now := time.Now()
	expireAt := now.Add(24 * time.Hour)
	order.ExpireAt = &expireAt

	if err = s.db.Transaction(func(tx *gorm.DB) error {
		if errTx := tx.Create(order).Error; errTx != nil {
			return errTx
		}
		if errTx := createOrderEvent(tx, order.ID, OrderEventCreated, payload); errTx != nil {
			return errTx
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return order, nil
}

func createOrderEvent(tx *gorm.DB, orderID uint, eventType string, payload map[string]interface{}) error {
	raw, _ := json.Marshal(payload)
	event := models.BillingOrderEvent{
		OrderID:     orderID,
		EventType:   eventType,
		PayloadJSON: string(raw),
	}
	return tx.Create(&event).Error
}

func generateOrderNo() (string, error) {
	buf := make([]byte, 5)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	token := strings.ToUpper(hex.EncodeToString(buf))
	return fmt.Sprintf("NPORD-%s-%s", time.Now().Format("20060102150405"), token), nil
}

func generateOrderLicenseKey() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	encoded := strings.ToUpper(hex.EncodeToString(buf))
	if len(encoded) < 24 {
		return "", errors.New("生成授权码失败")
	}
	return fmt.Sprintf("NP-%s-%s-%s", encoded[:8], encoded[8:16], encoded[16:24]), nil
}

func normalizeCurrency(raw string) string {
	trimmed := strings.TrimSpace(strings.ToUpper(raw))
	if trimmed == "" {
		return "CNY"
	}
	return trimmed
}

func normalizePaymentChannel(raw string) string {
	trimmed := strings.TrimSpace(strings.ToLower(raw))
	if trimmed == "" {
		return "manual"
	}
	return trimmed
}

func normalizeCallbackStatus(raw string) string {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case OrderStatusPaid:
		return OrderStatusPaid
	case OrderStatusFailed:
		return OrderStatusFailed
	case OrderStatusCanceled:
		return OrderStatusCanceled
	default:
		return ""
	}
}
