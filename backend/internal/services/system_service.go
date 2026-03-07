package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"nodepass-pro/backend/internal/cache"
	"nodepass-pro/backend/internal/models"

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
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("%w: key 不能为空", ErrInvalidParams)
	}

	trimmedValue := strings.TrimSpace(value)
	var existing models.SystemConfig
	err := s.db.Where("key = ?", key).First(&existing).Error
	if err == nil {
		if updateErr := s.db.Model(&models.SystemConfig{}).
			Where("id = ?", existing.ID).
			Updates(map[string]interface{}{"value": trimmedValue}).Error; updateErr != nil {
			return updateErr
		}

		if cache.Enabled() {
			if cacheErr := cache.Delete(context.Background(), systemConfigCacheKey); cacheErr != nil {
				zap.L().Warn("清理系统配置缓存失败", zap.Error(cacheErr))
			}
		}
		return nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("查询系统配置失败: %w", err)
	}

	item := &models.SystemConfig{
		Key:   key,
		Value: &trimmedValue,
	}
	if createErr := s.db.Create(item).Error; createErr != nil {
		return fmt.Errorf("创建系统配置失败: %w", createErr)
	}

	if cache.Enabled() {
		if err := cache.Delete(context.Background(), systemConfigCacheKey); err != nil {
			zap.L().Warn("清理系统配置缓存失败", zap.Error(err))
		}
	}
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
