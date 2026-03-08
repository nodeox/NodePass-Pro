package services

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"nodepass-pro/backend/internal/cache"
	"nodepass-pro/backend/internal/models"
	"nodepass-pro/backend/internal/utils"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// SystemService 系统级配置与统计服务。
type SystemService struct {
	db *gorm.DB
}

const (
	systemConfigCacheKey = "system:config:all"
	systemStatsCacheKey  = "system:stats:summary"
)

// SystemStats 系统统计数据。
type SystemStats struct {
	TotalUsers   int64 `json:"total_users"`
	OnlineNodes  int64 `json:"online_nodes"`
	RunningRules int64 `json:"running_rules"`
	TodayTraffic int64 `json:"today_traffic"`
}

// SystemConfigEntry 系统配置更新项。
type SystemConfigEntry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// NewSystemService 创建系统服务实例。
func NewSystemService(db *gorm.DB) *SystemService {
	return &SystemService{db: db}
}

// GetConfig 从 system_config 表加载所有配置。
func (s *SystemService) GetConfig() (map[string]string, error) {
	ctx := context.Background()
	if cache.Enabled() {
		cached := make(map[string]string)
		hit, err := cache.GetJSON(ctx, systemConfigCacheKey, &cached)
		if err != nil {
			zap.L().Warn("读取系统配置缓存失败", zap.Error(err))
		} else if hit {
			return cached, nil
		}
	}

	items := make([]models.SystemConfig, 0)
	if err := s.db.Model(&models.SystemConfig{}).
		Order("id ASC").
		Find(&items).Error; err != nil {
		return nil, fmt.Errorf("查询系统配置失败: %w", err)
	}

	result := make(map[string]string, len(items))
	for _, item := range items {
		if item.Value == nil {
			result[item.Key] = ""
			continue
		}
		result[item.Key] = *item.Value
	}

	if cache.Enabled() {
		if err := cache.SetJSON(ctx, systemConfigCacheKey, result, cache.DefaultTTL()); err != nil {
			zap.L().Warn("写入系统配置缓存失败", zap.Error(err))
		}
	}

	return result, nil
}

// UpdateConfig 更新系统配置。
func (s *SystemService) UpdateConfig(key string, value string) error {
	return s.UpdateConfigs([]SystemConfigEntry{{Key: key, Value: value}})
}

// UpdateConfigs 批量更新系统配置。
func (s *SystemService) UpdateConfigs(entries []SystemConfigEntry) error {
	if len(entries) == 0 {
		return fmt.Errorf("%w: 配置项不能为空", ErrInvalidParams)
	}

	normalizedMap := make(map[string]string, len(entries))
	for _, entry := range entries {
		key := strings.TrimSpace(entry.Key)
		if key == "" {
			return fmt.Errorf("%w: key 不能为空", ErrInvalidParams)
		}
		validator, ok := systemConfigValidators[key]
		if !ok {
			return fmt.Errorf("%w: 不支持的配置项 %s", ErrInvalidParams, key)
		}
		value, err := validator(entry.Value)
		if err != nil {
			return err
		}
		normalizedMap[key] = value
	}

	effective, err := s.loadConfigMap()
	if err != nil {
		return err
	}
	for key, value := range normalizedMap {
		effective[key] = value
	}
	if validateErr := validateSMTPEnableState(effective); validateErr != nil {
		return validateErr
	}

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		for key, value := range normalizedMap {
			var existing models.SystemConfig
			queryErr := tx.Where("key = ?", key).First(&existing).Error
			if queryErr == nil {
				if updateErr := tx.Model(&models.SystemConfig{}).
					Where("id = ?", existing.ID).
					Updates(map[string]interface{}{"value": value}).Error; updateErr != nil {
					return fmt.Errorf("更新系统配置失败: %w", updateErr)
				}
				continue
			}
			if queryErr != nil && !errors.Is(queryErr, gorm.ErrRecordNotFound) {
				return fmt.Errorf("查询系统配置失败: %w", queryErr)
			}

			item := &models.SystemConfig{
				Key:   key,
				Value: &value,
			}
			if createErr := tx.Create(item).Error; createErr != nil {
				return fmt.Errorf("创建系统配置失败: %w", createErr)
			}
		}
		return nil
	}); err != nil {
		return err
	}

	s.invalidateSystemConfigCache()
	return nil
}

// GetStats 返回系统统计信息。
func (s *SystemService) GetStats() (*SystemStats, error) {
	ctx := context.Background()
	if cache.Enabled() {
		cached := &SystemStats{}
		hit, err := cache.GetJSON(ctx, systemStatsCacheKey, cached)
		if err != nil {
			zap.L().Warn("读取系统统计缓存失败", zap.Error(err))
		} else if hit {
			return cached, nil
		}
	}

	var stats SystemStats

	if err := s.db.Model(&models.User{}).Count(&stats.TotalUsers).Error; err != nil {
		return nil, fmt.Errorf("查询用户总数失败: %w", err)
	}
	if err := s.db.Model(&models.Node{}).Where("status = ?", "online").Count(&stats.OnlineNodes).Error; err != nil {
		return nil, fmt.Errorf("查询在线节点数失败: %w", err)
	}
	if err := s.db.Model(&models.Rule{}).Where("status = ?", "running").Count(&stats.RunningRules).Error; err != nil {
		return nil, fmt.Errorf("查询运行规则数失败: %w", err)
	}

	now := time.Now()
	location := now.Location()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location)
	dayEnd := dayStart.Add(24 * time.Hour)

	type sumResult struct {
		Total int64 `gorm:"column:total"`
	}
	sum := sumResult{}
	if err := s.db.Model(&models.TrafficRecord{}).
		Select("COALESCE(SUM(calculated_traffic), 0) AS total").
		Where("hour >= ? AND hour < ?", dayStart, dayEnd).
		Scan(&sum).Error; err != nil {
		return nil, fmt.Errorf("查询今日流量失败: %w", err)
	}
	stats.TodayTraffic = sum.Total

	if cache.Enabled() {
		if err := cache.SetJSON(ctx, systemStatsCacheKey, &stats, 30*time.Second); err != nil {
			zap.L().Warn("写入系统统计缓存失败", zap.Error(err))
		}
	}

	return &stats, nil
}

func (s *SystemService) loadConfigMap() (map[string]string, error) {
	items := make([]models.SystemConfig, 0)
	if err := s.db.Model(&models.SystemConfig{}).Find(&items).Error; err != nil {
		return nil, fmt.Errorf("查询系统配置失败: %w", err)
	}

	result := make(map[string]string, len(items))
	for _, item := range items {
		if item.Value == nil {
			result[item.Key] = ""
			continue
		}
		result[item.Key] = *item.Value
	}
	return result, nil
}

func (s *SystemService) invalidateSystemConfigCache() {
	if !cache.Enabled() {
		return
	}
	if err := cache.Delete(context.Background(), systemConfigCacheKey); err != nil {
		zap.L().Warn("清理系统配置缓存失败", zap.Error(err))
	}
}

var systemConfigValidators = map[string]func(string) (string, error){
	"site_name":                      validateSiteName,
	"register_enabled":               normalizeBooleanConfig,
	"default_vip_level":              normalizeNonNegativeInteger,
	"telegram_bot_token":             normalizeOptionalText(1024),
	"telegram_bot_username":          normalizeTelegramUsername,
	"heartbeat_timeout_seconds":      normalizeIntegerRange(30, 86400),
	"traffic_stats_interval_seconds": normalizeIntegerRange(10, 86400),
	"heartbeat_interval":             normalizeIntegerRange(5, 86400),
	"config_check_interval":          normalizeIntegerRange(5, 86400),
	"traffic_report_interval":        normalizeIntegerRange(5, 86400),
	"smtp_enabled":                   normalizeBooleanConfig,
	"smtp_host":                      normalizeOptionalText(255),
	"smtp_port":                      normalizeIntegerRange(1, 65535),
	"smtp_username":                  normalizeOptionalText(255),
	"smtp_password":                  normalizeOptionalText(1024),
	"smtp_from_email":                normalizeOptionalEmail,
	"smtp_from_name":                 normalizeOptionalText(255),
	"smtp_reply_to":                  normalizeOptionalEmail,
	"smtp_encryption":                normalizeSMTPEncryption,
	"smtp_skip_verify":               normalizeBooleanConfig,
}

func validateSiteName(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("%w: site_name 不能为空", ErrInvalidParams)
	}
	if len(trimmed) > 100 {
		return "", fmt.Errorf("%w: site_name 长度不能超过 100", ErrInvalidParams)
	}
	return trimmed, nil
}

func normalizeBooleanConfig(value string) (string, error) {
	boolValue, err := parseBooleanConfig(value)
	if err != nil {
		return "", err
	}
	if boolValue {
		return "true", nil
	}
	return "false", nil
}

func parseBooleanConfig(value string) (bool, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "1", "true", "yes", "on":
		return true, nil
	case "0", "false", "no", "off":
		return false, nil
	default:
		return false, fmt.Errorf("%w: 布尔值仅支持 true/false/1/0", ErrInvalidParams)
	}
}

func normalizeNonNegativeInteger(value string) (string, error) {
	return normalizeIntegerRange(0, 1_000_000)(value)
}

func normalizeIntegerRange(minValue int, maxValue int) func(string) (string, error) {
	return func(value string) (string, error) {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return "", fmt.Errorf("%w: 数值不能为空", ErrInvalidParams)
		}
		parsed, err := strconv.Atoi(trimmed)
		if err != nil {
			return "", fmt.Errorf("%w: 数值格式错误", ErrInvalidParams)
		}
		if parsed < minValue || parsed > maxValue {
			return "", fmt.Errorf("%w: 数值范围应在 %d-%d", ErrInvalidParams, minValue, maxValue)
		}
		return strconv.Itoa(parsed), nil
	}
}

func normalizeOptionalText(maxLength int) func(string) (string, error) {
	return func(value string) (string, error) {
		trimmed := strings.TrimSpace(value)
		if len(trimmed) > maxLength {
			return "", fmt.Errorf("%w: 文本长度不能超过 %d", ErrInvalidParams, maxLength)
		}
		return trimmed, nil
	}
}

func normalizeOptionalEmail(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", nil
	}
	if err := utils.ValidateEmail(trimmed); err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidParams, err)
	}
	return strings.ToLower(trimmed), nil
}

func normalizeTelegramUsername(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", nil
	}
	if strings.HasPrefix(trimmed, "@") {
		trimmed = strings.TrimPrefix(trimmed, "@")
	}
	if len(trimmed) > 64 {
		return "", fmt.Errorf("%w: telegram_bot_username 长度不能超过 64", ErrInvalidParams)
	}
	return trimmed, nil
}

func normalizeSMTPEncryption(value string) (string, error) {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	if trimmed == "" {
		return "starttls", nil
	}
	switch trimmed {
	case "none", "starttls", "ssl":
		return trimmed, nil
	default:
		return "", fmt.Errorf("%w: smtp_encryption 仅支持 none/starttls/ssl", ErrInvalidParams)
	}
}

func validateSMTPEnableState(configMap map[string]string) error {
	enabledRaw, ok := configMap["smtp_enabled"]
	if !ok {
		return nil
	}
	enabled, err := parseBooleanConfig(enabledRaw)
	if err != nil {
		return err
	}
	if !enabled {
		return nil
	}

	host := strings.TrimSpace(configMap["smtp_host"])
	fromEmail := strings.TrimSpace(configMap["smtp_from_email"])
	port := strings.TrimSpace(configMap["smtp_port"])
	if host == "" {
		return fmt.Errorf("%w: 启用 SMTP 时 smtp_host 不能为空", ErrInvalidParams)
	}
	if fromEmail == "" {
		return fmt.Errorf("%w: 启用 SMTP 时 smtp_from_email 不能为空", ErrInvalidParams)
	}
	if port == "" {
		return fmt.Errorf("%w: 启用 SMTP 时 smtp_port 不能为空", ErrInvalidParams)
	}
	return nil
}
