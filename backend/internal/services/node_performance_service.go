package services

import (
	"context"
	"fmt"
	"time"

	"nodepass-pro/backend/internal/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// NodePerformanceService 节点性能监控服务
type NodePerformanceService struct {
	db *gorm.DB
}

// NewNodePerformanceService 创建节点性能监控服务
func NewNodePerformanceService(db *gorm.DB) *NodePerformanceService {
	return &NodePerformanceService{db: db}
}

// RecordMetric 记录性能指标
func (s *NodePerformanceService) RecordMetric(metric *models.NodePerformanceMetric) error {
	if metric == nil {
		return fmt.Errorf("性能指标不能为空")
	}

	if metric.CollectedAt.IsZero() {
		metric.CollectedAt = time.Now()
	}

	return s.db.Create(metric).Error
}

// GetLatestMetric 获取最新的性能指标
func (s *NodePerformanceService) GetLatestMetric(nodeInstanceID uint) (*models.NodePerformanceMetric, error) {
	var metric models.NodePerformanceMetric
	if err := s.db.Where("node_instance_id = ?", nodeInstanceID).
		Order("collected_at DESC").
		First(&metric).Error; err != nil {
		return nil, err
	}
	return &metric, nil
}

// GetMetrics 获取性能指标历史
func (s *NodePerformanceService) GetMetrics(nodeInstanceID uint, startTime, endTime time.Time, limit int) ([]models.NodePerformanceMetric, error) {
	if limit <= 0 {
		limit = 100
	}

	var metrics []models.NodePerformanceMetric
	query := s.db.Where("node_instance_id = ?", nodeInstanceID)

	if !startTime.IsZero() {
		query = query.Where("collected_at >= ?", startTime)
	}
	if !endTime.IsZero() {
		query = query.Where("collected_at <= ?", endTime)
	}

	if err := query.Order("collected_at DESC").Limit(limit).Find(&metrics).Error; err != nil {
		return nil, err
	}

	return metrics, nil
}

// GetMetricsStats 获取性能统计
func (s *NodePerformanceService) GetMetricsStats(nodeInstanceID uint, duration time.Duration) (map[string]interface{}, error) {
	startTime := time.Now().Add(-duration)

	var metrics []models.NodePerformanceMetric
	if err := s.db.Where("node_instance_id = ? AND collected_at >= ?", nodeInstanceID, startTime).
		Find(&metrics).Error; err != nil {
		return nil, err
	}

	if len(metrics) == 0 {
		return map[string]interface{}{
			"sample_count": 0,
		}, nil
	}

	var (
		totalCPU, totalMemory, totalDisk float64
		maxCPU, maxMemory, maxDisk       float64
		totalNetworkIn, totalNetworkOut  int64
		totalConnections                 int
		maxConnections                   int
	)

	for _, m := range metrics {
		totalCPU += m.CPUUsage
		totalMemory += m.MemoryUsage
		totalDisk += m.DiskUsage
		totalNetworkIn += m.NetworkIn
		totalNetworkOut += m.NetworkOut
		totalConnections += m.Connections

		if m.CPUUsage > maxCPU {
			maxCPU = m.CPUUsage
		}
		if m.MemoryUsage > maxMemory {
			maxMemory = m.MemoryUsage
		}
		if m.DiskUsage > maxDisk {
			maxDisk = m.DiskUsage
		}
		if m.Connections > maxConnections {
			maxConnections = m.Connections
		}
	}

	count := len(metrics)
	return map[string]interface{}{
		"sample_count":      count,
		"avg_cpu_usage":     totalCPU / float64(count),
		"max_cpu_usage":     maxCPU,
		"avg_memory_usage":  totalMemory / float64(count),
		"max_memory_usage":  maxMemory,
		"avg_disk_usage":    totalDisk / float64(count),
		"max_disk_usage":    maxDisk,
		"total_network_in":  totalNetworkIn,
		"total_network_out": totalNetworkOut,
		"avg_connections":   totalConnections / count,
		"max_connections":   maxConnections,
		"duration":          duration.String(),
	}, nil
}

// CreateAlert 创建性能告警配置
func (s *NodePerformanceService) CreateAlert(alert *models.NodePerformanceAlert) error {
	if alert == nil {
		return fmt.Errorf("告警配置不能为空")
	}

	// 设置默认值
	if alert.CPUThreshold <= 0 {
		alert.CPUThreshold = 80
	}
	if alert.MemoryThreshold <= 0 {
		alert.MemoryThreshold = 80
	}
	if alert.DiskThreshold <= 0 {
		alert.DiskThreshold = 90
	}
	if alert.AlertCooldown <= 0 {
		alert.AlertCooldown = 300
	}

	return s.db.Create(alert).Error
}

// GetAlert 获取性能告警配置
func (s *NodePerformanceService) GetAlert(nodeInstanceID uint) (*models.NodePerformanceAlert, error) {
	var alert models.NodePerformanceAlert
	if err := s.db.Where("node_instance_id = ?", nodeInstanceID).First(&alert).Error; err != nil {
		return nil, err
	}
	return &alert, nil
}

// UpdateAlert 更新性能告警配置
func (s *NodePerformanceService) UpdateAlert(nodeInstanceID uint, updates map[string]interface{}) error {
	return s.db.Model(&models.NodePerformanceAlert{}).
		Where("node_instance_id = ?", nodeInstanceID).
		Updates(updates).Error
}

// DeleteAlert 删除性能告警配置
func (s *NodePerformanceService) DeleteAlert(nodeInstanceID uint) error {
	return s.db.Where("node_instance_id = ?", nodeInstanceID).
		Delete(&models.NodePerformanceAlert{}).Error
}

// CheckAndTriggerAlerts 检查并触发告警
func (s *NodePerformanceService) CheckAndTriggerAlerts(metric *models.NodePerformanceMetric) error {
	if metric == nil {
		return fmt.Errorf("性能指标不能为空")
	}

	// 获取告警配置
	alert, err := s.GetAlert(metric.NodeInstanceID)
	if err != nil {
		// 没有配置告警，跳过
		return nil
	}

	if !alert.Enabled {
		return nil
	}

	// 检查是否在冷却期内
	var lastAlert models.NodePerformanceAlertRecord
	if err := s.db.Where("node_instance_id = ? AND resolved = ?", metric.NodeInstanceID, false).
		Order("created_at DESC").
		First(&lastAlert).Error; err == nil {
		cooldownEnd := lastAlert.CreatedAt.Add(time.Duration(alert.AlertCooldown) * time.Second)
		if time.Now().Before(cooldownEnd) {
			return nil // 在冷却期内，不触发新告警
		}
	}

	// 检查各项指标
	alerts := []models.NodePerformanceAlertRecord{}

	// CPU 告警
	if metric.CPUUsage > alert.CPUThreshold {
		alerts = append(alerts, models.NodePerformanceAlertRecord{
			NodeInstanceID: metric.NodeInstanceID,
			AlertType:      "cpu",
			Threshold:      alert.CPUThreshold,
			ActualValue:    metric.CPUUsage,
			Message:        fmt.Sprintf("CPU 使用率 %.2f%% 超过阈值 %.2f%%", metric.CPUUsage, alert.CPUThreshold),
			CreatedAt:      time.Now(),
		})
	}

	// 内存告警
	if metric.MemoryUsage > alert.MemoryThreshold {
		alerts = append(alerts, models.NodePerformanceAlertRecord{
			NodeInstanceID: metric.NodeInstanceID,
			AlertType:      "memory",
			Threshold:      alert.MemoryThreshold,
			ActualValue:    metric.MemoryUsage,
			Message:        fmt.Sprintf("内存使用率 %.2f%% 超过阈值 %.2f%%", metric.MemoryUsage, alert.MemoryThreshold),
			CreatedAt:      time.Now(),
		})
	}

	// 磁盘告警
	if metric.DiskUsage > alert.DiskThreshold {
		alerts = append(alerts, models.NodePerformanceAlertRecord{
			NodeInstanceID: metric.NodeInstanceID,
			AlertType:      "disk",
			Threshold:      alert.DiskThreshold,
			ActualValue:    metric.DiskUsage,
			Message:        fmt.Sprintf("磁盘使用率 %.2f%% 超过阈值 %.2f%%", metric.DiskUsage, alert.DiskThreshold),
			CreatedAt:      time.Now(),
		})
	}

	// 延迟告警
	if alert.LatencyThreshold != nil && metric.Latency != nil && *metric.Latency > *alert.LatencyThreshold {
		alerts = append(alerts, models.NodePerformanceAlertRecord{
			NodeInstanceID: metric.NodeInstanceID,
			AlertType:      "latency",
			Threshold:      float64(*alert.LatencyThreshold),
			ActualValue:    float64(*metric.Latency),
			Message:        fmt.Sprintf("延迟 %d ms 超过阈值 %d ms", *metric.Latency, *alert.LatencyThreshold),
			CreatedAt:      time.Now(),
		})
	}

	// 丢包率告警
	if alert.PacketLossThreshold != nil && metric.PacketLoss != nil && *metric.PacketLoss > *alert.PacketLossThreshold {
		alerts = append(alerts, models.NodePerformanceAlertRecord{
			NodeInstanceID: metric.NodeInstanceID,
			AlertType:      "packet_loss",
			Threshold:      *alert.PacketLossThreshold,
			ActualValue:    *metric.PacketLoss,
			Message:        fmt.Sprintf("丢包率 %.2f%% 超过阈值 %.2f%%", *metric.PacketLoss, *alert.PacketLossThreshold),
			CreatedAt:      time.Now(),
		})
	}

	// 批量创建告警记录
	if len(alerts) > 0 {
		if err := s.db.Create(&alerts).Error; err != nil {
			return err
		}

		zap.L().Warn("节点性能告警触发",
			zap.Uint("node_instance_id", metric.NodeInstanceID),
			zap.Int("alert_count", len(alerts)))
	}

	return nil
}

// GetAlertRecords 获取告警记录
func (s *NodePerformanceService) GetAlertRecords(nodeInstanceID uint, resolved *bool, limit int) ([]models.NodePerformanceAlertRecord, error) {
	if limit <= 0 {
		limit = 100
	}

	query := s.db.Where("node_instance_id = ?", nodeInstanceID)
	if resolved != nil {
		query = query.Where("resolved = ?", *resolved)
	}

	var records []models.NodePerformanceAlertRecord
	if err := query.Order("created_at DESC").Limit(limit).Find(&records).Error; err != nil {
		return nil, err
	}

	return records, nil
}

// ResolveAlert 解决告警
func (s *NodePerformanceService) ResolveAlert(alertID uint) error {
	now := time.Now()
	return s.db.Model(&models.NodePerformanceAlertRecord{}).
		Where("id = ?", alertID).
		Updates(map[string]interface{}{
			"resolved":    true,
			"resolved_at": &now,
		}).Error
}

// AggregateMetrics 聚合性能指标（按小时/天）
func (s *NodePerformanceService) AggregateMetrics(ctx context.Context, period string) error {
	var periodStart time.Time
	var periodEnd time.Time

	now := time.Now()
	switch period {
	case "hourly":
		// 聚合上一个小时的数据
		periodStart = now.Add(-1 * time.Hour).Truncate(time.Hour)
		periodEnd = periodStart.Add(time.Hour)
	case "daily":
		// 聚合昨天的数据
		periodStart = now.AddDate(0, 0, -1).Truncate(24 * time.Hour)
		periodEnd = periodStart.Add(24 * time.Hour)
	default:
		return fmt.Errorf("不支持的聚合周期: %s", period)
	}

	// 获取所有节点实例
	var instances []models.NodeInstance
	if err := s.db.Find(&instances).Error; err != nil {
		return err
	}

	for _, instance := range instances {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := s.aggregateNodeMetrics(instance.ID, period, periodStart, periodEnd); err != nil {
				zap.L().Error("聚合节点性能指标失败",
					zap.Uint("node_instance_id", instance.ID),
					zap.String("period", period),
					zap.Error(err))
			}
		}
	}

	return nil
}

// aggregateNodeMetrics 聚合单个节点的性能指标
func (s *NodePerformanceService) aggregateNodeMetrics(nodeInstanceID uint, period string, periodStart, periodEnd time.Time) error {
	var metrics []models.NodePerformanceMetric
	if err := s.db.Where("node_instance_id = ? AND collected_at >= ? AND collected_at < ?",
		nodeInstanceID, periodStart, periodEnd).
		Find(&metrics).Error; err != nil {
		return err
	}

	if len(metrics) == 0 {
		return nil
	}

	var (
		totalCPU, totalMemory, totalDisk float64
		maxCPU, maxMemory, maxDisk       float64
		totalNetworkIn, totalNetworkOut  int64
		totalConnections                 int
		maxConnections                   int
	)

	for _, m := range metrics {
		totalCPU += m.CPUUsage
		totalMemory += m.MemoryUsage
		totalDisk += m.DiskUsage
		totalNetworkIn += m.NetworkIn
		totalNetworkOut += m.NetworkOut
		totalConnections += m.Connections

		if m.CPUUsage > maxCPU {
			maxCPU = m.CPUUsage
		}
		if m.MemoryUsage > maxMemory {
			maxMemory = m.MemoryUsage
		}
		if m.DiskUsage > maxDisk {
			maxDisk = m.DiskUsage
		}
		if m.Connections > maxConnections {
			maxConnections = m.Connections
		}
	}

	count := len(metrics)
	summary := models.NodePerformanceSummary{
		NodeInstanceID:  nodeInstanceID,
		Period:          period,
		PeriodStart:     periodStart,
		AvgCPUUsage:     totalCPU / float64(count),
		MaxCPUUsage:     maxCPU,
		AvgMemoryUsage:  totalMemory / float64(count),
		MaxMemoryUsage:  maxMemory,
		AvgDiskUsage:    totalDisk / float64(count),
		MaxDiskUsage:    maxDisk,
		TotalNetworkIn:  totalNetworkIn,
		TotalNetworkOut: totalNetworkOut,
		AvgConnections:  totalConnections / count,
		MaxConnections:  maxConnections,
		SampleCount:     count,
	}

	return s.db.Create(&summary).Error
}

// GetSummaries 获取性能汇总
func (s *NodePerformanceService) GetSummaries(nodeInstanceID uint, period string, limit int) ([]models.NodePerformanceSummary, error) {
	if limit <= 0 {
		limit = 30
	}

	var summaries []models.NodePerformanceSummary
	query := s.db.Where("node_instance_id = ?", nodeInstanceID)
	if period != "" {
		query = query.Where("period = ?", period)
	}

	if err := query.Order("period_start DESC").Limit(limit).Find(&summaries).Error; err != nil {
		return nil, err
	}

	return summaries, nil
}

// CleanupOldMetrics 清理旧的性能指标
func (s *NodePerformanceService) CleanupOldMetrics(retentionDays int) (int64, error) {
	if retentionDays <= 0 {
		retentionDays = 7 // 默认保留 7 天
	}

	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)
	result := s.db.Where("collected_at < ?", cutoffTime).Delete(&models.NodePerformanceMetric{})

	if result.Error != nil {
		return 0, result.Error
	}

	return result.RowsAffected, nil
}

// CleanupOldSummaries 清理旧的性能汇总
func (s *NodePerformanceService) CleanupOldSummaries(retentionDays int) (int64, error) {
	if retentionDays <= 0 {
		retentionDays = 90 // 默认保留 90 天
	}

	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)
	result := s.db.Where("period_start < ?", cutoffTime).Delete(&models.NodePerformanceSummary{})

	if result.Error != nil {
		return 0, result.Error
	}

	return result.RowsAffected, nil
}

// GetDB 获取数据库实例
func (s *NodePerformanceService) GetDB() *gorm.DB {
	return s.db
}
