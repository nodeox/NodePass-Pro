package services

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"nodepass-license-center/internal/models"

	"gorm.io/gorm"
)

// WebhookService Webhook 服务
type WebhookService struct {
	db *gorm.DB
}

// NewWebhookService 创建 Webhook 服务
func NewWebhookService(db *gorm.DB) *WebhookService {
	return &WebhookService{db: db}
}

// WebhookEvent Webhook 事件
type WebhookEvent struct {
	Event     string                 `json:"event"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// TriggerEvent 触发事件
func (s *WebhookService) TriggerEvent(event string, data map[string]interface{}) error {
	var configs []models.WebhookConfig
	if err := s.db.Where("is_enabled = ?", true).Find(&configs).Error; err != nil {
		return err
	}

	webhookEvent := WebhookEvent{
		Event:     event,
		Timestamp: time.Now().UTC(),
		Data:      data,
	}

	for _, config := range configs {
		var events []string
		if err := json.Unmarshal([]byte(config.Events), &events); err != nil {
			continue
		}

		shouldTrigger := false
		for _, e := range events {
			if e == event || e == "*" {
				shouldTrigger = true
				break
			}
		}

		if shouldTrigger {
			go s.sendWebhook(config, webhookEvent)
		}
	}

	return nil
}

// sendWebhook 发送 Webhook
func (s *WebhookService) sendWebhook(config models.WebhookConfig, event WebhookEvent) {
	payload, err := json.Marshal(event)
	if err != nil {
		s.logWebhook(config.ID, event.Event, string(payload), "", 0, false, err.Error())
		return
	}

	req, err := http.NewRequestWithContext(context.Background(), "POST", config.URL, bytes.NewBuffer(payload))
	if err != nil {
		s.logWebhook(config.ID, event.Event, string(payload), "", 0, false, err.Error())
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "NodePass-License-Center/1.0")

	if config.Secret != "" {
		signature := s.calculateWebhookSignature(config.Secret, payload)
		req.Header.Set("X-Webhook-Signature", signature)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		s.logWebhook(config.ID, event.Event, string(payload), "", 0, false, err.Error())
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	success := resp.StatusCode >= 200 && resp.StatusCode < 300
	errorMsg := ""
	if !success {
		errorMsg = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	s.logWebhook(config.ID, event.Event, string(payload), string(respBody), resp.StatusCode, success, errorMsg)
}

// calculateWebhookSignature 计算 Webhook 签名
func (s *WebhookService) calculateWebhookSignature(secret string, payload []byte) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}

// logWebhook 记录 Webhook 日志
func (s *WebhookService) logWebhook(webhookID uint, event, payload, response string, statusCode int, success bool, errorMsg string) {
	log := &models.WebhookLog{
		WebhookID:    webhookID,
		Event:        event,
		Payload:      payload,
		Response:     response,
		StatusCode:   statusCode,
		Success:      success,
		ErrorMessage: errorMsg,
	}
	_ = s.db.Create(log).Error
}

// ListWebhooks 查询 Webhook 配置
func (s *WebhookService) ListWebhooks() ([]models.WebhookConfig, error) {
	var configs []models.WebhookConfig
	err := s.db.Order("id DESC").Find(&configs).Error
	return configs, err
}

// CreateWebhook 创建 Webhook
func (s *WebhookService) CreateWebhook(name, url, secret string, events []string, isEnabled bool) (*models.WebhookConfig, error) {
	eventsJSON, err := json.Marshal(events)
	if err != nil {
		return nil, err
	}

	config := &models.WebhookConfig{
		Name:      name,
		URL:       url,
		Secret:    secret,
		Events:    string(eventsJSON),
		IsEnabled: isEnabled,
	}

	if err := s.db.Create(config).Error; err != nil {
		return nil, err
	}

	return config, nil
}

// UpdateWebhook 更新 Webhook
func (s *WebhookService) UpdateWebhook(id uint, updates map[string]interface{}) error {
	return s.db.Model(&models.WebhookConfig{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteWebhook 删除 Webhook
func (s *WebhookService) DeleteWebhook(id uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("webhook_id = ?", id).Delete(&models.WebhookLog{}).Error; err != nil {
			return err
		}
		return tx.Delete(&models.WebhookConfig{}, id).Error
	})
}

// ListWebhookLogs 查询 Webhook 日志
func (s *WebhookService) ListWebhookLogs(webhookID uint, page, pageSize int) (*PaginatedResult[models.WebhookLog], error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	query := s.db.Model(&models.WebhookLog{})
	if webhookID > 0 {
		query = query.Where("webhook_id = ?", webhookID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	items := make([]models.WebhookLog, 0)
	offset := (page - 1) * pageSize
	if err := query.Order("id DESC").Offset(offset).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, err
	}

	return &PaginatedResult[models.WebhookLog]{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}
