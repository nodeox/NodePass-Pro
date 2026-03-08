package services

import (
	"context"
	"fmt"
	"time"

	"nodepass-license-center/internal/cache"
	"nodepass-license-center/internal/models"

	"gorm.io/gorm"
)

// AlertService 告警服务
type AlertService struct {
	db      *gorm.DB
	cache   *cache.RedisCache
	webhook *WebhookService
}

// NewAlertService 创建告警服务
func NewAlertService(db *gorm.DB, cache *cache.RedisCache, webhook *WebhookService) *AlertService {
	return &AlertService{
		db:      db,
		cache:   cache,
		webhook: webhook,
	}
}

// CreateAlert 创建告警
func (s *AlertService) CreateAlert(alertType, level, title, message string, licenseID *uint, metadata map[string]interface{}) error {
	alert := &models.Alert{
		Type:      alertType,
		Level:     level,
		Title:     title,
		Message:   message,
		LicenseID: licenseID,
		IsRead:    false,
		IsSent:    false,
	}

	if err := s.db.Create(alert).Error; err != nil {
		return err
	}

	// 触发 Webhook
	data := map[string]interface{}{
		"alert_id":   alert.ID,
		"type":       alertType,
		"level":      level,
		"title":      title,
		"message":    message,
		"license_id": licenseID,
		"metadata":   metadata,
	}
	_ = s.webhook.TriggerEvent("alert.created", data)

	return nil
}

// CheckExpiringLicenses 检查即将过期的授权码
func (s *AlertService) CheckExpiringLicenses(days int) error {
	now := time.Now().UTC()
	threshold := now.AddDate(0, 0, days)

	var licenses []models.LicenseKey
	if err := s.db.Preload("Plan").
		Where("status = ? AND expires_at IS NOT NULL AND expires_at > ? AND expires_at <= ?", "active", now, threshold).
		Find(&licenses).Error; err != nil {
		return err
	}

	for _, license := range licenses {
		// 检查是否已经创建过告警
		var count int64
		if err := s.db.Model(&models.Alert{}).
			Where("type = ? AND license_id = ? AND created_at > ?", "license_expiring", license.ID, now.AddDate(0, 0, -days)).
			Count(&count).Error; err != nil {
			continue
		}

		if count > 0 {
			continue
		}

		daysLeft := int(license.ExpiresAt.Sub(now).Hours() / 24)
		title := fmt.Sprintf("授权码即将过期: %s", license.Key)
		message := fmt.Sprintf("客户 %s 的授权码将在 %d 天后过期", license.Customer, daysLeft)

		_ = s.CreateAlert("license_expiring", "warning", title, message, &license.ID, map[string]interface{}{
			"customer":   license.Customer,
			"expires_at": license.ExpiresAt,
			"days_left":  daysLeft,
		})
	}

	return nil
}

// CheckQuotaExceeded 检查配额超限
func (s *AlertService) CheckQuotaExceeded() error {
	var licenses []models.LicenseKey
	if err := s.db.Preload("Plan").Where("status = ?", "active").Find(&licenses).Error; err != nil {
		return err
	}

	for _, license := range licenses {
		var activeCount int64
		if err := s.db.Model(&models.LicenseActivation{}).
			Where("license_id = ? AND is_active = ?", license.ID, true).
			Count(&activeCount).Error; err != nil {
			continue
		}

		maxMachines := license.Plan.MaxMachines
		if license.MaxMachines != nil && *license.MaxMachines > 0 {
			maxMachines = *license.MaxMachines
		}

		if maxMachines > 0 && int(activeCount) >= maxMachines {
			// 检查是否已经创建过告警
			var count int64
			now := time.Now().UTC()
			if err := s.db.Model(&models.Alert{}).
				Where("type = ? AND license_id = ? AND created_at > ?", "quota_exceeded", license.ID, now.Add(-24*time.Hour)).
				Count(&count).Error; err != nil {
				continue
			}

			if count > 0 {
				continue
			}

			title := fmt.Sprintf("授权码配额已满: %s", license.Key)
			message := fmt.Sprintf("客户 %s 的授权码已达到机器绑定上限 (%d/%d)", license.Customer, activeCount, maxMachines)

			_ = s.CreateAlert("quota_exceeded", "warning", title, message, &license.ID, map[string]interface{}{
				"customer":      license.Customer,
				"active_count":  activeCount,
				"max_machines":  maxMachines,
			})
		}
	}

	return nil
}

// ListAlerts 查询告警
func (s *AlertService) ListAlerts(isRead *bool, level string, page, pageSize int) (*PaginatedResult[models.Alert], error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	query := s.db.Model(&models.Alert{})
	if isRead != nil {
		query = query.Where("is_read = ?", *isRead)
	}
	if level != "" {
		query = query.Where("level = ?", level)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	items := make([]models.Alert, 0)
	offset := (page - 1) * pageSize
	if err := query.Order("id DESC").Offset(offset).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, err
	}

	return &PaginatedResult[models.Alert]{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// MarkAlertRead 标记告警已读
func (s *AlertService) MarkAlertRead(id uint) error {
	return s.db.Model(&models.Alert{}).Where("id = ?", id).Update("is_read", true).Error
}

// MarkAllAlertsRead 标记所有告警已读
func (s *AlertService) MarkAllAlertsRead() error {
	return s.db.Model(&models.Alert{}).Where("is_read = ?", false).Update("is_read", true).Error
}

// DeleteAlert 删除告警
func (s *AlertService) DeleteAlert(id uint) error {
	return s.db.Delete(&models.Alert{}, id).Error
}

// GetAlertStats 获取告警统计
func (s *AlertService) GetAlertStats() (map[string]interface{}, error) {
	stats := map[string]interface{}{}

	var unreadCount int64
	if err := s.db.Model(&models.Alert{}).Where("is_read = ?", false).Count(&unreadCount).Error; err != nil {
		return nil, err
	}

	var criticalCount int64
	if err := s.db.Model(&models.Alert{}).Where("level = ? AND is_read = ?", "critical", false).Count(&criticalCount).Error; err != nil {
		return nil, err
	}

	var warningCount int64
	if err := s.db.Model(&models.Alert{}).Where("level = ? AND is_read = ?", "warning", false).Count(&warningCount).Error; err != nil {
		return nil, err
	}

	stats["unread_count"] = unreadCount
	stats["critical_count"] = criticalCount
	stats["warning_count"] = warningCount

	return stats, nil
}

// MonitoringService 监控服务
type MonitoringService struct {
	db    *gorm.DB
	cache *cache.RedisCache
}

// NewMonitoringService 创建监控服务
func NewMonitoringService(db *gorm.DB, cache *cache.RedisCache) *MonitoringService {
	return &MonitoringService{
		db:    db,
		cache: cache,
	}
}

// GetDashboardStats 获取仪表盘统计
func (s *MonitoringService) GetDashboardStats(ctx context.Context) (map[string]interface{}, error) {
	stats := map[string]interface{}{}

	// 尝试从缓存获取
	if s.cache != nil {
		if err := s.cache.Get(ctx, "dashboard:stats", &stats); err == nil {
			return stats, nil
		}
	}

	// 授权码统计
	var licenseTotal, licenseActive, licenseExpired, licenseRevoked int64
	s.db.Model(&models.LicenseKey{}).Count(&licenseTotal)
	s.db.Model(&models.LicenseKey{}).Where("status = ?", "active").Count(&licenseActive)
	s.db.Model(&models.LicenseKey{}).Where("status = ?", "expired").Count(&licenseExpired)
	s.db.Model(&models.LicenseKey{}).Where("status = ?", "revoked").Count(&licenseRevoked)

	// 机器绑定统计
	var activationTotal int64
	s.db.Model(&models.LicenseActivation{}).Where("is_active = ?", true).Count(&activationTotal)

	// 验证统计
	now := time.Now().UTC()
	startOfDay := now.Truncate(24 * time.Hour)
	startOfWeek := now.AddDate(0, 0, -7).Truncate(24 * time.Hour)
	startOfMonth := now.AddDate(0, -1, 0).Truncate(24 * time.Hour)

	var verifyToday, verifyWeek, verifyMonth int64
	s.db.Model(&models.VerifyLog{}).Where("created_at >= ?", startOfDay).Count(&verifyToday)
	s.db.Model(&models.VerifyLog{}).Where("created_at >= ?", startOfWeek).Count(&verifyWeek)
	s.db.Model(&models.VerifyLog{}).Where("created_at >= ?", startOfMonth).Count(&verifyMonth)

	var verifySuccessToday, verifyFailedToday int64
	s.db.Model(&models.VerifyLog{}).Where("created_at >= ? AND result = ?", startOfDay, "success").Count(&verifySuccessToday)
	s.db.Model(&models.VerifyLog{}).Where("created_at >= ? AND result = ?", startOfDay, "failed").Count(&verifyFailedToday)

	// 即将过期的授权码
	expiringThreshold := now.AddDate(0, 0, 30)
	var expiringCount int64
	s.db.Model(&models.LicenseKey{}).
		Where("status = ? AND expires_at IS NOT NULL AND expires_at > ? AND expires_at <= ?", "active", now, expiringThreshold).
		Count(&expiringCount)

	stats["license_total"] = licenseTotal
	stats["license_active"] = licenseActive
	stats["license_expired"] = licenseExpired
	stats["license_revoked"] = licenseRevoked
	stats["activation_total"] = activationTotal
	stats["verify_today"] = verifyToday
	stats["verify_week"] = verifyWeek
	stats["verify_month"] = verifyMonth
	stats["verify_success_today"] = verifySuccessToday
	stats["verify_failed_today"] = verifyFailedToday
	stats["expiring_count"] = expiringCount

	// 缓存 5 分钟
	if s.cache != nil {
		_ = s.cache.Set(ctx, "dashboard:stats", stats, 5*time.Minute)
	}

	return stats, nil
}

// GetVerifyTrend 获取验证趋势
func (s *MonitoringService) GetVerifyTrend(days int) ([]map[string]interface{}, error) {
	if days <= 0 {
		days = 7
	}
	if days > 90 {
		days = 90
	}

	now := time.Now().UTC()
	startDate := now.AddDate(0, 0, -days).Truncate(24 * time.Hour)

	type DailyStats struct {
		Date    string
		Total   int64
		Success int64
		Failed  int64
	}

	results := make([]map[string]interface{}, 0)
	for i := 0; i < days; i++ {
		date := startDate.AddDate(0, 0, i)
		nextDate := date.AddDate(0, 0, 1)

		var total, success, failed int64
		s.db.Model(&models.VerifyLog{}).
			Where("created_at >= ? AND created_at < ?", date, nextDate).
			Count(&total)
		s.db.Model(&models.VerifyLog{}).
			Where("created_at >= ? AND created_at < ? AND result = ?", date, nextDate, "success").
			Count(&success)
		s.db.Model(&models.VerifyLog{}).
			Where("created_at >= ? AND created_at < ? AND result = ?", date, nextDate, "failed").
			Count(&failed)

		results = append(results, map[string]interface{}{
			"date":    date.Format("2006-01-02"),
			"total":   total,
			"success": success,
			"failed":  failed,
		})
	}

	return results, nil
}

// GetTopCustomers 获取 Top 客户
func (s *MonitoringService) GetTopCustomers(limit int) ([]map[string]interface{}, error) {
	if limit <= 0 {
		limit = 10
	}

	type CustomerStats struct {
		Customer        string
		LicenseCount    int64
		ActivationCount int64
	}

	var customers []CustomerStats
	if err := s.db.Raw(`
		SELECT
			lk.customer,
			COUNT(DISTINCT lk.id) as license_count,
			COUNT(DISTINCT la.id) as activation_count
		FROM license_keys lk
		LEFT JOIN license_activations la ON lk.id = la.license_id AND la.is_active = true
		WHERE lk.status = 'active'
		GROUP BY lk.customer
		ORDER BY license_count DESC, activation_count DESC
		LIMIT ?
	`, limit).Scan(&customers).Error; err != nil {
		return nil, err
	}

	results := make([]map[string]interface{}, len(customers))
	for i, c := range customers {
		results[i] = map[string]interface{}{
			"customer":         c.Customer,
			"license_count":    c.LicenseCount,
			"activation_count": c.ActivationCount,
		}
	}

	return results, nil
}
