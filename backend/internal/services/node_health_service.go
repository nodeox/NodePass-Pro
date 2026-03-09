package services

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"nodepass-pro/backend/internal/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// NodeHealthService 节点健康检查服务
type NodeHealthService struct {
	db *gorm.DB
}

// NewNodeHealthService 创建节点健康检查服务
func NewNodeHealthService(db *gorm.DB) *NodeHealthService {
	return &NodeHealthService{db: db}
}

// CreateHealthCheck 创建健康检查配置
func (s *NodeHealthService) CreateHealthCheck(check *models.NodeHealthCheck) error {
	if check == nil {
		return fmt.Errorf("健康检查配置不能为空")
	}

	// 验证节点实例是否存在
	var instance models.NodeInstance
	if err := s.db.First(&instance, check.NodeInstanceID).Error; err != nil {
		return fmt.Errorf("节点实例不存在: %w", err)
	}

	// 设置默认值
	if check.Interval <= 0 {
		check.Interval = 30
	}
	if check.Timeout <= 0 {
		check.Timeout = 5
	}
	if check.Retries <= 0 {
		check.Retries = 3
	}
	if check.SuccessThreshold <= 0 {
		check.SuccessThreshold = 2
	}
	if check.FailureThreshold <= 0 {
		check.FailureThreshold = 3
	}

	return s.db.Create(check).Error
}

// GetHealthCheck 获取健康检查配置
func (s *NodeHealthService) GetHealthCheck(nodeInstanceID uint) (*models.NodeHealthCheck, error) {
	var check models.NodeHealthCheck
	if err := s.db.Where("node_instance_id = ?", nodeInstanceID).First(&check).Error; err != nil {
		return nil, err
	}
	return &check, nil
}

// UpdateHealthCheck 更新健康检查配置
func (s *NodeHealthService) UpdateHealthCheck(nodeInstanceID uint, updates map[string]interface{}) error {
	return s.db.Model(&models.NodeHealthCheck{}).
		Where("node_instance_id = ?", nodeInstanceID).
		Updates(updates).Error
}

// DeleteHealthCheck 删除健康检查配置
func (s *NodeHealthService) DeleteHealthCheck(nodeInstanceID uint) error {
	return s.db.Where("node_instance_id = ?", nodeInstanceID).
		Delete(&models.NodeHealthCheck{}).Error
}

// PerformHealthCheck 执行健康检查
func (s *NodeHealthService) PerformHealthCheck(nodeInstanceID uint) (*models.NodeHealthRecord, error) {
	// 获取节点实例
	var instance models.NodeInstance
	if err := s.db.First(&instance, nodeInstanceID).Error; err != nil {
		return nil, fmt.Errorf("节点实例不存在: %w", err)
	}

	// 获取健康检查配置
	check, err := s.GetHealthCheck(nodeInstanceID)
	if err != nil {
		// 如果没有配置，使用默认 TCP 检查
		check = &models.NodeHealthCheck{
			Type:    models.HealthCheckTypeTCP,
			Timeout: 5,
		}
	}

	// 执行检查
	var record models.NodeHealthRecord
	record.NodeInstanceID = nodeInstanceID
	record.CheckType = check.Type
	record.CheckedAt = time.Now()

	switch check.Type {
	case models.HealthCheckTypeTCP:
		record = s.performTCPCheck(&instance, check)
	case models.HealthCheckTypeHTTP:
		record = s.performHTTPCheck(&instance, check)
	case models.HealthCheckTypeICMP:
		record = s.performICMPCheck(&instance, check)
	default:
		record.Status = models.HealthCheckStatusUnknown
		errMsg := "不支持的健康检查类型"
		record.ErrorMessage = &errMsg
	}

	// 保存检查记录
	if err := s.db.Create(&record).Error; err != nil {
		zap.L().Error("保存健康检查记录失败",
			zap.Uint("node_instance_id", nodeInstanceID),
			zap.Error(err))
	}

	// 更新节点状态
	s.updateNodeStatus(&instance, &record)

	// 更新质量评分
	s.updateQualityScore(nodeInstanceID)

	return &record, nil
}

// performTCPCheck 执行 TCP 健康检查
func (s *NodeHealthService) performTCPCheck(instance *models.NodeInstance, check *models.NodeHealthCheck) models.NodeHealthRecord {
	record := models.NodeHealthRecord{
		NodeInstanceID: instance.ID,
		CheckType:      models.HealthCheckTypeTCP,
		CheckedAt:      time.Now(),
	}

	if instance.Host == nil || instance.Port == nil {
		record.Status = models.HealthCheckStatusUnhealthy
		errMsg := "节点地址或端口未配置"
		record.ErrorMessage = &errMsg
		return record
	}

	address := fmt.Sprintf("%s:%d", *instance.Host, *instance.Port)
	timeout := time.Duration(check.Timeout) * time.Second

	start := time.Now()
	conn, err := net.DialTimeout("tcp", address, timeout)
	latency := int(time.Since(start).Milliseconds())
	record.Latency = &latency

	if err != nil {
		record.Status = models.HealthCheckStatusUnhealthy
		errMsg := err.Error()
		record.ErrorMessage = &errMsg
		return record
	}
	defer conn.Close()

	record.Status = models.HealthCheckStatusHealthy
	return record
}

// performHTTPCheck 执行 HTTP 健康检查
func (s *NodeHealthService) performHTTPCheck(instance *models.NodeInstance, check *models.NodeHealthCheck) models.NodeHealthRecord {
	record := models.NodeHealthRecord{
		NodeInstanceID: instance.ID,
		CheckType:      models.HealthCheckTypeHTTP,
		CheckedAt:      time.Now(),
	}

	if instance.Host == nil || instance.Port == nil {
		record.Status = models.HealthCheckStatusUnhealthy
		errMsg := "节点地址或端口未配置"
		record.ErrorMessage = &errMsg
		return record
	}

	path := "/"
	if check.HTTPPath != nil {
		path = *check.HTTPPath
	}

	method := "GET"
	if check.HTTPMethod != nil {
		method = *check.HTTPMethod
	}

	url := fmt.Sprintf("http://%s:%d%s", *instance.Host, *instance.Port, path)
	timeout := time.Duration(check.Timeout) * time.Second

	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		record.Status = models.HealthCheckStatusUnhealthy
		errMsg := err.Error()
		record.ErrorMessage = &errMsg
		return record
	}

	start := time.Now()
	resp, err := client.Do(req)
	latency := int(time.Since(start).Milliseconds())
	record.Latency = &latency

	if err != nil {
		record.Status = models.HealthCheckStatusUnhealthy
		errMsg := err.Error()
		record.ErrorMessage = &errMsg
		return record
	}
	defer resp.Body.Close()

	// 检查状态码
	expectedStatus := 200
	if check.ExpectedStatus != nil {
		expectedStatus = *check.ExpectedStatus
	}

	if resp.StatusCode == expectedStatus {
		record.Status = models.HealthCheckStatusHealthy
	} else {
		record.Status = models.HealthCheckStatusUnhealthy
		errMsg := fmt.Sprintf("HTTP 状态码不匹配: 期望 %d, 实际 %d", expectedStatus, resp.StatusCode)
		record.ErrorMessage = &errMsg
	}

	return record
}

// performICMPCheck 执行 ICMP (Ping) 健康检查
func (s *NodeHealthService) performICMPCheck(instance *models.NodeInstance, check *models.NodeHealthCheck) models.NodeHealthRecord {
	record := models.NodeHealthRecord{
		NodeInstanceID: instance.ID,
		CheckType:      models.HealthCheckTypeICMP,
		CheckedAt:      time.Now(),
	}

	if instance.Host == nil {
		record.Status = models.HealthCheckStatusUnhealthy
		errMsg := "节点地址未配置"
		record.ErrorMessage = &errMsg
		return record
	}

	// 简化实现：使用 TCP 连接测试代替 ICMP
	// 真实的 ICMP 需要 root 权限，这里用 TCP 80 端口测试
	address := fmt.Sprintf("%s:80", *instance.Host)
	timeout := time.Duration(check.Timeout) * time.Second

	start := time.Now()
	conn, err := net.DialTimeout("tcp", address, timeout)
	latency := int(time.Since(start).Milliseconds())
	record.Latency = &latency

	if err != nil {
		record.Status = models.HealthCheckStatusUnhealthy
		errMsg := err.Error()
		record.ErrorMessage = &errMsg
		return record
	}
	defer conn.Close()

	record.Status = models.HealthCheckStatusHealthy
	return record
}

// updateNodeStatus 根据健康检查结果更新节点状态
func (s *NodeHealthService) updateNodeStatus(instance *models.NodeInstance, record *models.NodeHealthRecord) {
	var newStatus models.NodeInstanceStatus

	if record.Status == models.HealthCheckStatusHealthy {
		newStatus = models.NodeInstanceStatusOnline
	} else {
		newStatus = models.NodeInstanceStatusOffline
	}

	if instance.Status != newStatus {
		if err := s.db.Model(&models.NodeInstance{}).
			Where("id = ?", instance.ID).
			Update("status", newStatus).Error; err != nil {
			zap.L().Error("更新节点状态失败",
				zap.Uint("node_instance_id", instance.ID),
				zap.Error(err))
		}
	}
}

// updateQualityScore 更新节点质量评分
func (s *NodeHealthService) updateQualityScore(nodeInstanceID uint) {
	// 获取最近 100 条健康检查记录
	var records []models.NodeHealthRecord
	if err := s.db.Where("node_instance_id = ?", nodeInstanceID).
		Order("checked_at DESC").
		Limit(100).
		Find(&records).Error; err != nil {
		zap.L().Error("查询健康检查记录失败",
			zap.Uint("node_instance_id", nodeInstanceID),
			zap.Error(err))
		return
	}

	if len(records) == 0 {
		return
	}

	// 计算评分
	var totalLatency int64
	var healthyCount int
	var latencyCount int

	for _, record := range records {
		if record.Status == models.HealthCheckStatusHealthy {
			healthyCount++
		}
		if record.Latency != nil {
			totalLatency += int64(*record.Latency)
			latencyCount++
		}
	}

	// 计算平均延迟
	var avgLatency int
	if latencyCount > 0 {
		avgLatency = int(totalLatency / int64(latencyCount))
	}

	// 计算成功率
	successRate := float64(healthyCount) / float64(len(records)) * 100

	// 计算延迟评分 (延迟越低分数越高)
	latencyScore := 100.0
	if avgLatency > 0 {
		// 0-50ms: 100分, 50-100ms: 90分, 100-200ms: 70分, 200-500ms: 40分, >500ms: 10分
		switch {
		case avgLatency <= 50:
			latencyScore = 100
		case avgLatency <= 100:
			latencyScore = 90
		case avgLatency <= 200:
			latencyScore = 70
		case avgLatency <= 500:
			latencyScore = 40
		default:
			latencyScore = 10
		}
	}

	// 计算稳定性评分 (成功率)
	stabilityScore := successRate

	// 计算负载评分 (暂时使用固定值，后续可以根据实际负载计算)
	loadScore := 80.0

	// 更新或创建质量评分
	score := models.NodeQualityScore{
		NodeInstanceID: nodeInstanceID,
		LatencyScore:   latencyScore,
		StabilityScore: stabilityScore,
		LoadScore:      loadScore,
		AvgLatency:     &avgLatency,
		Uptime:         successRate,
		SuccessRate:    successRate,
		LastCheckedAt:  &records[0].CheckedAt,
	}
	score.CalculateOverallScore()

	// 尝试更新，如果不存在则创建
	result := s.db.Model(&models.NodeQualityScore{}).
		Where("node_instance_id = ?", nodeInstanceID).
		Updates(&score)

	if result.RowsAffected == 0 {
		if err := s.db.Create(&score).Error; err != nil {
			zap.L().Error("创建质量评分失败",
				zap.Uint("node_instance_id", nodeInstanceID),
				zap.Error(err))
		}
	}
}

// GetQualityScore 获取节点质量评分
func (s *NodeHealthService) GetQualityScore(nodeInstanceID uint) (*models.NodeQualityScore, error) {
	var score models.NodeQualityScore
	if err := s.db.Where("node_instance_id = ?", nodeInstanceID).First(&score).Error; err != nil {
		return nil, err
	}
	return &score, nil
}

// GetHealthRecords 获取健康检查记录
func (s *NodeHealthService) GetHealthRecords(nodeInstanceID uint, limit int) ([]models.NodeHealthRecord, error) {
	if limit <= 0 {
		limit = 100
	}

	var records []models.NodeHealthRecord
	if err := s.db.Where("node_instance_id = ?", nodeInstanceID).
		Order("checked_at DESC").
		Limit(limit).
		Find(&records).Error; err != nil {
		return nil, err
	}

	return records, nil
}

// GetHealthStats 获取健康统计信息
func (s *NodeHealthService) GetHealthStats(nodeInstanceID uint, duration time.Duration) (map[string]interface{}, error) {
	startTime := time.Now().Add(-duration)

	var records []models.NodeHealthRecord
	if err := s.db.Where("node_instance_id = ? AND checked_at >= ?", nodeInstanceID, startTime).
		Order("checked_at DESC").
		Find(&records).Error; err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return map[string]interface{}{
			"total_checks":   0,
			"healthy_checks": 0,
			"success_rate":   0,
			"avg_latency":    0,
		}, nil
	}

	var healthyCount int
	var totalLatency int64
	var latencyCount int

	for _, record := range records {
		if record.Status == models.HealthCheckStatusHealthy {
			healthyCount++
		}
		if record.Latency != nil {
			totalLatency += int64(*record.Latency)
			latencyCount++
		}
	}

	avgLatency := 0
	if latencyCount > 0 {
		avgLatency = int(totalLatency / int64(latencyCount))
	}

	successRate := float64(healthyCount) / float64(len(records)) * 100

	return map[string]interface{}{
		"total_checks":   len(records),
		"healthy_checks": healthyCount,
		"success_rate":   successRate,
		"avg_latency":    avgLatency,
		"duration":       duration.String(),
	}, nil
}

// BatchPerformHealthCheck 批量执行健康检查
func (s *NodeHealthService) BatchPerformHealthCheck(ctx context.Context) error {
	// 获取所有启用的健康检查配置
	var checks []models.NodeHealthCheck
	if err := s.db.Where("enabled = ?", true).Find(&checks).Error; err != nil {
		return err
	}

	zap.L().Info("开始批量健康检查", zap.Int("count", len(checks)))

	for _, check := range checks {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if _, err := s.PerformHealthCheck(check.NodeInstanceID); err != nil {
				zap.L().Error("健康检查失败",
					zap.Uint("node_instance_id", check.NodeInstanceID),
					zap.Error(err))
			}
		}
	}

	return nil
}

// CleanupOldRecords 清理旧的健康检查记录
func (s *NodeHealthService) CleanupOldRecords(retentionDays int) (int64, error) {
	if retentionDays <= 0 {
		retentionDays = 30
	}

	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)
	result := s.db.Where("checked_at < ?", cutoffTime).Delete(&models.NodeHealthRecord{})

	if result.Error != nil {
		return 0, result.Error
	}

	return result.RowsAffected, nil
}

// GetDB 获取数据库实例（用于复杂查询）
func (s *NodeHealthService) GetDB() *gorm.DB {
	return s.db
}
