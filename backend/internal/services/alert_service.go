package services

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"nodepass-pro/backend/internal/models"

	"gorm.io/gorm"
)

// AlertService 告警服务
type AlertService struct {
	db *gorm.DB
}

// NewAlertService 创建告警服务
func NewAlertService(db *gorm.DB) *AlertService {
	return &AlertService{db: db}
}

// CreateAlert 创建告警
func (s *AlertService) CreateAlert(alert *models.Alert) error {
	// 生成指纹用于去重
	if alert.Fingerprint == "" {
		alert.Fingerprint = s.generateFingerprint(alert)
	}

	// 检查是否已存在相同指纹的告警
	var existing models.Alert
	err := s.db.Where("fingerprint = ? AND status IN ?", alert.Fingerprint,
		[]models.AlertStatus{models.AlertStatusPending, models.AlertStatusFiring}).
		First(&existing).Error

	if err == nil {
		// 已存在，更新现有告警
		now := time.Now()
		existing.LastFiredAt = now
		existing.Value = alert.Value
		existing.Message = alert.Message
		existing.Status = models.AlertStatusFiring

		return s.db.Save(&existing).Error
	}

	if err != gorm.ErrRecordNotFound {
		return fmt.Errorf("查询现有告警失败: %w", err)
	}

	// 不存在，创建新告警
	now := time.Now()
	alert.FirstFiredAt = now
	alert.LastFiredAt = now
	alert.Status = models.AlertStatusFiring

	return s.db.Create(alert).Error
}

// ResolveAlert 解决告警
func (s *AlertService) ResolveAlert(id uint, resolvedBy uint, notes string) error {
	now := time.Now()
	return s.db.Model(&models.Alert{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":      models.AlertStatusResolved,
		"resolved_at": &now,
		"resolved_by": resolvedBy,
		"notes":       notes,
	}).Error
}

// AcknowledgeAlert 确认告警
func (s *AlertService) AcknowledgeAlert(id uint, acknowledgedBy uint) error {
	now := time.Now()
	return s.db.Model(&models.Alert{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":           models.AlertStatusAcknowledged,
		"acknowledged_at":  &now,
		"acknowledged_by":  acknowledgedBy,
	}).Error
}

// SilenceAlert 静默告警
func (s *AlertService) SilenceAlert(id uint, duration time.Duration) error {
	silencedUntil := time.Now().Add(duration)
	return s.db.Model(&models.Alert{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":         models.AlertStatusSilenced,
		"silenced_until": &silencedUntil,
	}).Error
}

// GetAlert 获取告警详情
func (s *AlertService) GetAlert(id uint) (*models.Alert, error) {
	var alert models.Alert
	if err := s.db.First(&alert, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("%w: 告警不存在", ErrNotFound)
		}
		return nil, fmt.Errorf("查询告警失败: %w", err)
	}
	return &alert, nil
}

// ListAlerts 列出告警
func (s *AlertService) ListAlerts(status []models.AlertStatus, level []models.AlertLevel,
	resourceType string, page, pageSize int) ([]*models.Alert, int64, error) {

	query := s.db.Model(&models.Alert{})

	if len(status) > 0 {
		query = query.Where("status IN ?", status)
	}
	if len(level) > 0 {
		query = query.Where("level IN ?", level)
	}
	if resourceType != "" {
		query = query.Where("resource_type = ?", resourceType)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计告警数量失败: %w", err)
	}

	var alerts []*models.Alert
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&alerts).Error; err != nil {
		return nil, 0, fmt.Errorf("查询告警列表失败: %w", err)
	}

	return alerts, total, nil
}

// GetFiringAlerts 获取正在触发的告警
func (s *AlertService) GetFiringAlerts() ([]*models.Alert, error) {
	var alerts []*models.Alert
	if err := s.db.Where("status = ?", models.AlertStatusFiring).
		Order("level DESC, created_at DESC").
		Find(&alerts).Error; err != nil {
		return nil, fmt.Errorf("查询触发中的告警失败: %w", err)
	}
	return alerts, nil
}

// GetAlertStats 获取告警统计
func (s *AlertService) GetAlertStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 按状态统计
	var statusStats []struct {
		Status models.AlertStatus
		Count  int64
	}
	if err := s.db.Model(&models.Alert{}).
		Select("status, COUNT(*) as count").
		Group("status").
		Scan(&statusStats).Error; err != nil {
		return nil, fmt.Errorf("统计告警状态失败: %w", err)
	}
	stats["by_status"] = statusStats

	// 按级别统计
	var levelStats []struct {
		Level models.AlertLevel
		Count int64
	}
	if err := s.db.Model(&models.Alert{}).
		Where("status IN ?", []models.AlertStatus{models.AlertStatusPending, models.AlertStatusFiring}).
		Select("level, COUNT(*) as count").
		Group("level").
		Scan(&levelStats).Error; err != nil {
		return nil, fmt.Errorf("统计告警级别失败: %w", err)
	}
	stats["by_level"] = levelStats

	// 按类型统计
	var typeStats []struct {
		Type  models.AlertType
		Count int64
	}
	if err := s.db.Model(&models.Alert{}).
		Where("status IN ?", []models.AlertStatus{models.AlertStatusPending, models.AlertStatusFiring}).
		Select("type, COUNT(*) as count").
		Group("type").
		Scan(&typeStats).Error; err != nil {
		return nil, fmt.Errorf("统计告警类型失败: %w", err)
	}
	stats["by_type"] = typeStats

	return stats, nil
}

// CleanupResolvedAlerts 清理已解决的告警
func (s *AlertService) CleanupResolvedAlerts(days int) error {
	cutoff := time.Now().AddDate(0, 0, -days)
	result := s.db.Where("status = ? AND resolved_at < ?", models.AlertStatusResolved, cutoff).
		Delete(&models.Alert{})

	if result.Error != nil {
		return fmt.Errorf("清理已解决告警失败: %w", result.Error)
	}

	return nil
}

// generateFingerprint 生成告警指纹
func (s *AlertService) generateFingerprint(alert *models.Alert) string {
	data := fmt.Sprintf("%s:%s:%d:%s",
		alert.Type,
		alert.ResourceType,
		alert.ResourceID,
		alert.Title,
	)
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash)
}

// MarkNotificationSent 标记通知已发送
func (s *AlertService) MarkNotificationSent(id uint) error {
	now := time.Now()
	return s.db.Model(&models.Alert{}).Where("id = ?", id).Updates(map[string]interface{}{
		"notification_sent":  true,
		"notification_count": gorm.Expr("notification_count + 1"),
		"last_notified_at":   &now,
	}).Error
}

// AlertRuleService 告警规则服务
type AlertRuleService struct {
	db *gorm.DB
}

// NewAlertRuleService 创建告警规则服务
func NewAlertRuleService(db *gorm.DB) *AlertRuleService {
	return &AlertRuleService{db: db}
}

// CreateAlertRule 创建告警规则
func (s *AlertRuleService) CreateAlertRule(rule *models.AlertRule) error {
	return s.db.Create(rule).Error
}

// UpdateAlertRule 更新告警规则
func (s *AlertRuleService) UpdateAlertRule(id uint, updates map[string]interface{}) error {
	return s.db.Model(&models.AlertRule{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteAlertRule 删除告警规则
func (s *AlertRuleService) DeleteAlertRule(id uint) error {
	return s.db.Delete(&models.AlertRule{}, id).Error
}

// GetAlertRule 获取告警规则
func (s *AlertRuleService) GetAlertRule(id uint) (*models.AlertRule, error) {
	var rule models.AlertRule
	if err := s.db.First(&rule, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("%w: 告警规则不存在", ErrNotFound)
		}
		return nil, fmt.Errorf("查询告警规则失败: %w", err)
	}
	return &rule, nil
}

// ListAlertRules 列出告警规则
func (s *AlertRuleService) ListAlertRules(isEnabled *bool) ([]*models.AlertRule, error) {
	query := s.db.Model(&models.AlertRule{})

	if isEnabled != nil {
		query = query.Where("is_enabled = ?", *isEnabled)
	}

	var rules []*models.AlertRule
	if err := query.Order("created_at DESC").Find(&rules).Error; err != nil {
		return nil, fmt.Errorf("查询告警规则列表失败: %w", err)
	}

	return rules, nil
}

// GetEnabledRules 获取启用的告警规则
func (s *AlertRuleService) GetEnabledRules() ([]*models.AlertRule, error) {
	enabled := true
	return s.ListAlertRules(&enabled)
}

// NotificationChannelService 通知渠道服务
type NotificationChannelService struct {
	db *gorm.DB
}

// NewNotificationChannelService 创建通知渠道服务
func NewNotificationChannelService(db *gorm.DB) *NotificationChannelService {
	return &NotificationChannelService{db: db}
}

// CreateChannel 创建通知渠道
func (s *NotificationChannelService) CreateChannel(channel *models.NotificationChannel) error {
	return s.db.Create(channel).Error
}

// UpdateChannel 更新通知渠道
func (s *NotificationChannelService) UpdateChannel(id uint, updates map[string]interface{}) error {
	return s.db.Model(&models.NotificationChannel{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteChannel 删除通知渠道
func (s *NotificationChannelService) DeleteChannel(id uint) error {
	return s.db.Delete(&models.NotificationChannel{}, id).Error
}

// GetChannel 获取通知渠道
func (s *NotificationChannelService) GetChannel(id uint) (*models.NotificationChannel, error) {
	var channel models.NotificationChannel
	if err := s.db.First(&channel, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("%w: 通知渠道不存在", ErrNotFound)
		}
		return nil, fmt.Errorf("查询通知渠道失败: %w", err)
	}
	return &channel, nil
}

// ListChannels 列出通知渠道
func (s *NotificationChannelService) ListChannels(isEnabled *bool) ([]*models.NotificationChannel, error) {
	query := s.db.Model(&models.NotificationChannel{})

	if isEnabled != nil {
		query = query.Where("is_enabled = ?", *isEnabled)
	}

	var channels []*models.NotificationChannel
	if err := query.Order("created_at DESC").Find(&channels).Error; err != nil {
		return nil, fmt.Errorf("查询通知渠道列表失败: %w", err)
	}

	return channels, nil
}

// GetEnabledChannels 获取启用的通知渠道
func (s *NotificationChannelService) GetEnabledChannels() ([]*models.NotificationChannel, error) {
	enabled := true
	return s.ListChannels(&enabled)
}

// MarkSent 标记通知已发送
func (s *NotificationChannelService) MarkSent(id uint) error {
	now := time.Now()
	return s.db.Model(&models.NotificationChannel{}).Where("id = ?", id).Updates(map[string]interface{}{
		"sent_count":   gorm.Expr("sent_count + 1"),
		"last_sent_at": &now,
	}).Error
}

// MarkFailed 标记通知发送失败
func (s *NotificationChannelService) MarkFailed(id uint, errorMsg string) error {
	now := time.Now()
	return s.db.Model(&models.NotificationChannel{}).Where("id = ?", id).Updates(map[string]interface{}{
		"failed_count":   gorm.Expr("failed_count + 1"),
		"last_failed_at": &now,
		"last_error":     errorMsg,
	}).Error
}

// TestChannel 测试通知渠道
func (s *NotificationChannelService) TestChannel(id uint) error {
	channel, err := s.GetChannel(id)
	if err != nil {
		return err
	}

	// 解析配置
	var config map[string]interface{}
	if err := json.Unmarshal([]byte(channel.Config), &config); err != nil {
		return fmt.Errorf("解析渠道配置失败: %w", err)
	}

	// 根据类型发送测试通知
	testAlert := &models.Alert{
		Type:    models.AlertTypeSystemError,
		Level:   models.AlertLevelInfo,
		Title:   "测试通知",
		Message: "这是一条测试通知，用于验证通知渠道配置是否正确。",
	}

	return s.sendNotification(channel, testAlert)
}

// sendNotification 发送通知（简化版，实际实现需要根据渠道类型调用不同的发送逻辑）
func (s *NotificationChannelService) sendNotification(channel *models.NotificationChannel, alert *models.Alert) error {
	// TODO: 实现具体的通知发送逻辑
	// 根据 channel.Type 调用不同的发送方法
	// - email: 发送邮件
	// - telegram: 发送 Telegram 消息
	// - webhook: 发送 HTTP 请求
	// - slack: 发送 Slack 消息

	return nil
}
