package services

import (
	"context"
	"fmt"
	"time"

	"nodepass-pro/backend/internal/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// NodeAutomationService 节点自动化管理服务
type NodeAutomationService struct {
	db                 *gorm.DB
	healthService      *NodeHealthService
	performanceService *NodePerformanceService
}

// NewNodeAutomationService 创建节点自动化管理服务
func NewNodeAutomationService(db *gorm.DB) *NodeAutomationService {
	return &NodeAutomationService{
		db:                 db,
		healthService:      NewNodeHealthService(db),
		performanceService: NewNodePerformanceService(db),
	}
}

// CreatePolicy 创建自动化策略
func (s *NodeAutomationService) CreatePolicy(policy *models.NodeAutomationPolicy) error {
	if policy == nil {
		return fmt.Errorf("自动化策略不能为空")
	}

	// 设置默认值
	if policy.MinNodes <= 0 {
		policy.MinNodes = 1
	}
	if policy.MaxNodes <= 0 {
		policy.MaxNodes = 10
	}
	if policy.ScaleUpThreshold <= 0 {
		policy.ScaleUpThreshold = 80
	}
	if policy.ScaleDownThreshold <= 0 {
		policy.ScaleDownThreshold = 30
	}
	if policy.ScaleCooldown <= 0 {
		policy.ScaleCooldown = 300
	}
	if policy.FailoverTimeout <= 0 {
		policy.FailoverTimeout = 180
	}
	if policy.RecoveryCheckInterval <= 0 {
		policy.RecoveryCheckInterval = 60
	}

	return s.db.Create(policy).Error
}

// GetPolicy 获取自动化策略
func (s *NodeAutomationService) GetPolicy(nodeGroupID uint) (*models.NodeAutomationPolicy, error) {
	var policy models.NodeAutomationPolicy
	if err := s.db.Where("node_group_id = ?", nodeGroupID).First(&policy).Error; err != nil {
		return nil, err
	}
	return &policy, nil
}

// UpdatePolicy 更新自动化策略
func (s *NodeAutomationService) UpdatePolicy(nodeGroupID uint, updates map[string]interface{}) error {
	return s.db.Model(&models.NodeAutomationPolicy{}).
		Where("node_group_id = ?", nodeGroupID).
		Updates(updates).Error
}

// DeletePolicy 删除自动化策略
func (s *NodeAutomationService) DeletePolicy(nodeGroupID uint) error {
	return s.db.Where("node_group_id = ?", nodeGroupID).
		Delete(&models.NodeAutomationPolicy{}).Error
}

// CheckAndExecuteAutomation 检查并执行自动化操作
func (s *NodeAutomationService) CheckAndExecuteAutomation(ctx context.Context) error {
	// 获取所有启用的自动化策略
	var policies []models.NodeAutomationPolicy
	if err := s.db.Where("enabled = ?", true).Find(&policies).Error; err != nil {
		return err
	}

	zap.L().Info("开始检查自动化策略", zap.Int("count", len(policies)))

	for _, policy := range policies {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// 检查自动故障处理
			if policy.AutoFailoverEnabled {
				if err := s.checkAndExecuteFailover(policy.NodeGroupID); err != nil {
					zap.L().Error("自动故障处理失败",
						zap.Uint("node_group_id", policy.NodeGroupID),
						zap.Error(err))
				}
			}

			// 检查自动恢复
			if policy.AutoRecoveryEnabled {
				if err := s.checkAndExecuteRecovery(policy.NodeGroupID); err != nil {
					zap.L().Error("自动恢复失败",
						zap.Uint("node_group_id", policy.NodeGroupID),
						zap.Error(err))
				}
			}

			// 检查自动扩缩容
			if policy.AutoScalingEnabled {
				if err := s.checkAndExecuteScaling(&policy); err != nil {
					zap.L().Error("自动扩缩容失败",
						zap.Uint("node_group_id", policy.NodeGroupID),
						zap.Error(err))
				}
			}
		}
	}

	return nil
}

// checkAndExecuteFailover 检查并执行故障转移
func (s *NodeAutomationService) checkAndExecuteFailover(nodeGroupID uint) error {
	// 获取节点组的所有实例
	var instances []models.NodeInstance
	if err := s.db.Where("node_group_id = ? AND is_enabled = ?", nodeGroupID, true).
		Find(&instances).Error; err != nil {
		return err
	}

	for _, instance := range instances {
		// 检查节点是否故障
		if instance.Status == models.NodeInstanceStatusOffline {
			// 检查是否已经隔离
			var isolation models.NodeIsolation
			err := s.db.Where("node_instance_id = ? AND is_active = ?", instance.ID, true).
				First(&isolation).Error

			if err == gorm.ErrRecordNotFound {
				// 隔离故障节点
				if err := s.IsolateNode(instance.ID, "节点离线，自动隔离", "auto"); err != nil {
					zap.L().Error("隔离节点失败",
						zap.Uint("node_instance_id", instance.ID),
						zap.Error(err))
					continue
				}

				// 记录自动化操作
				s.recordAction(nodeGroupID, &instance.ID, "failover",
					fmt.Sprintf("节点 %s 离线，已自动隔离", instance.Name),
					"success", nil)

				zap.L().Info("节点已自动隔离",
					zap.Uint("node_instance_id", instance.ID),
					zap.String("name", instance.Name))
			}
		}
	}

	return nil
}

// checkAndExecuteRecovery 检查并执行自动恢复
func (s *NodeAutomationService) checkAndExecuteRecovery(nodeGroupID uint) error {
	// 获取活跃的隔离记录
	var isolations []models.NodeIsolation
	if err := s.db.Joins("JOIN node_instances ON node_instances.id = node_isolations.node_instance_id").
		Where("node_instances.node_group_id = ? AND node_isolations.is_active = ? AND node_isolations.isolated_by = ?",
			nodeGroupID, true, "auto").
		Find(&isolations).Error; err != nil {
		return err
	}

	for _, isolation := range isolations {
		// 检查节点是否恢复
		var instance models.NodeInstance
		if err := s.db.First(&instance, isolation.NodeInstanceID).Error; err != nil {
			continue
		}

		if instance.Status == models.NodeInstanceStatusOnline {
			// 恢复节点
			if err := s.RecoverNode(isolation.NodeInstanceID); err != nil {
				zap.L().Error("恢复节点失败",
					zap.Uint("node_instance_id", isolation.NodeInstanceID),
					zap.Error(err))
				continue
			}

			// 记录自动化操作
			s.recordAction(nodeGroupID, &isolation.NodeInstanceID, "recovery",
				fmt.Sprintf("节点 %s 已恢复在线，自动解除隔离", instance.Name),
				"success", nil)

			zap.L().Info("节点已自动恢复",
				zap.Uint("node_instance_id", isolation.NodeInstanceID),
				zap.String("name", instance.Name))
		}
	}

	return nil
}

// checkAndExecuteScaling 检查并执行自动扩缩容
func (s *NodeAutomationService) checkAndExecuteScaling(policy *models.NodeAutomationPolicy) error {
	// 获取节点组的在线节点数
	var onlineCount int64
	if err := s.db.Model(&models.NodeInstance{}).
		Where("node_group_id = ? AND status = ? AND is_enabled = ?",
			policy.NodeGroupID, models.NodeInstanceStatusOnline, true).
		Count(&onlineCount).Error; err != nil {
		return err
	}

	// 检查是否在冷却期
	var lastAction models.NodeAutomationAction
	if err := s.db.Where("node_group_id = ? AND (action_type = ? OR action_type = ?)",
		policy.NodeGroupID, "scale_up", "scale_down").
		Order("executed_at DESC").
		First(&lastAction).Error; err == nil {
		cooldownEnd := lastAction.ExecutedAt.Add(time.Duration(policy.ScaleCooldown) * time.Second)
		if time.Now().Before(cooldownEnd) {
			return nil // 在冷却期内
		}
	}

	// 获取节点组的平均负载
	var instances []models.NodeInstance
	if err := s.db.Where("node_group_id = ? AND status = ?",
		policy.NodeGroupID, models.NodeInstanceStatusOnline).
		Find(&instances).Error; err != nil {
		return err
	}

	if len(instances) == 0 {
		return nil
	}

	var totalCPU, totalMemory float64
	var metricCount int

	for _, instance := range instances {
		metric, err := s.performanceService.GetLatestMetric(instance.ID)
		if err != nil {
			continue
		}
		totalCPU += metric.CPUUsage
		totalMemory += metric.MemoryUsage
		metricCount++
	}

	if metricCount == 0 {
		return nil
	}

	avgCPU := totalCPU / float64(metricCount)
	avgMemory := totalMemory / float64(metricCount)
	avgLoad := (avgCPU + avgMemory) / 2

	// 判断是否需要扩容
	if avgLoad > policy.ScaleUpThreshold && int(onlineCount) < policy.MaxNodes {
		reason := fmt.Sprintf("平均负载 %.2f%% 超过扩容阈值 %.2f%%", avgLoad, policy.ScaleUpThreshold)
		s.recordAction(policy.NodeGroupID, nil, "scale_up", reason, "success", nil)
		zap.L().Info("触发自动扩容",
			zap.Uint("node_group_id", policy.NodeGroupID),
			zap.Float64("avg_load", avgLoad))
		return nil
	}

	// 判断是否需要缩容
	if avgLoad < policy.ScaleDownThreshold && int(onlineCount) > policy.MinNodes {
		reason := fmt.Sprintf("平均负载 %.2f%% 低于缩容阈值 %.2f%%", avgLoad, policy.ScaleDownThreshold)
		s.recordAction(policy.NodeGroupID, nil, "scale_down", reason, "success", nil)
		zap.L().Info("触发自动缩容",
			zap.Uint("node_group_id", policy.NodeGroupID),
			zap.Float64("avg_load", avgLoad))
		return nil
	}

	return nil
}

// IsolateNode 隔离节点
func (s *NodeAutomationService) IsolateNode(nodeInstanceID uint, reason, isolatedBy string) error {
	isolation := models.NodeIsolation{
		NodeInstanceID: nodeInstanceID,
		Reason:         reason,
		IsolatedBy:     isolatedBy,
		IsolatedAt:     time.Now(),
		IsActive:       true,
	}

	if err := s.db.Create(&isolation).Error; err != nil {
		return err
	}

	// 禁用节点实例
	return s.db.Model(&models.NodeInstance{}).
		Where("id = ?", nodeInstanceID).
		Update("is_enabled", false).Error
}

// RecoverNode 恢复节点
func (s *NodeAutomationService) RecoverNode(nodeInstanceID uint) error {
	now := time.Now()

	// 更新隔离记录
	if err := s.db.Model(&models.NodeIsolation{}).
		Where("node_instance_id = ? AND is_active = ?", nodeInstanceID, true).
		Updates(map[string]interface{}{
			"is_active":    false,
			"recovered_at": &now,
		}).Error; err != nil {
		return err
	}

	// 启用节点实例
	return s.db.Model(&models.NodeInstance{}).
		Where("id = ?", nodeInstanceID).
		Update("is_enabled", true).Error
}

// recordAction 记录自动化操作
func (s *NodeAutomationService) recordAction(nodeGroupID uint, nodeInstanceID *uint, actionType, reason, status string, errorMsg *string) {
	action := models.NodeAutomationAction{
		NodeGroupID:    nodeGroupID,
		NodeInstanceID: nodeInstanceID,
		ActionType:     actionType,
		Reason:         reason,
		Status:         status,
		ErrorMessage:   errorMsg,
		ExecutedAt:     time.Now(),
	}

	if status != "pending" {
		now := time.Now()
		action.CompletedAt = &now
	}

	if err := s.db.Create(&action).Error; err != nil {
		zap.L().Error("记录自动化操作失败", zap.Error(err))
	}
}

// GetActions 获取自动化操作记录
func (s *NodeAutomationService) GetActions(nodeGroupID uint, limit int) ([]models.NodeAutomationAction, error) {
	if limit <= 0 {
		limit = 100
	}

	var actions []models.NodeAutomationAction
	if err := s.db.Where("node_group_id = ?", nodeGroupID).
		Order("executed_at DESC").
		Limit(limit).
		Find(&actions).Error; err != nil {
		return nil, err
	}

	return actions, nil
}

// GetIsolations 获取隔离记录
func (s *NodeAutomationService) GetIsolations(nodeGroupID uint, activeOnly bool) ([]models.NodeIsolation, error) {
	query := s.db.Joins("JOIN node_instances ON node_instances.id = node_isolations.node_instance_id").
		Where("node_instances.node_group_id = ?", nodeGroupID)

	if activeOnly {
		query = query.Where("node_isolations.is_active = ?", true)
	}

	var isolations []models.NodeIsolation
	if err := query.Order("node_isolations.isolated_at DESC").Find(&isolations).Error; err != nil {
		return nil, err
	}

	return isolations, nil
}

// GenerateOptimizationSuggestions 生成优化建议
func (s *NodeAutomationService) GenerateOptimizationSuggestions(ctx context.Context) error {
	// 获取所有节点组
	var groups []models.NodeGroup
	if err := s.db.Find(&groups).Error; err != nil {
		return err
	}

	for _, group := range groups {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := s.generateGroupSuggestions(group.ID); err != nil {
				zap.L().Error("生成优化建议失败",
					zap.Uint("node_group_id", group.ID),
					zap.Error(err))
			}
		}
	}

	return nil
}

// generateGroupSuggestions 为节点组生成优化建议
func (s *NodeAutomationService) generateGroupSuggestions(nodeGroupID uint) error {
	// 获取节点组的所有实例
	var instances []models.NodeInstance
	if err := s.db.Where("node_group_id = ?", nodeGroupID).Find(&instances).Error; err != nil {
		return err
	}

	suggestions := []models.NodeOptimizationSuggestion{}

	// 检查性能问题
	for _, instance := range instances {
		metric, err := s.performanceService.GetLatestMetric(instance.ID)
		if err != nil {
			continue
		}

		// CPU 使用率过高
		if metric.CPUUsage > 90 {
			suggestions = append(suggestions, models.NodeOptimizationSuggestion{
				NodeGroupID:    &nodeGroupID,
				NodeInstanceID: &instance.ID,
				Category:       "performance",
				Priority:       "high",
				Title:          "CPU 使用率过高",
				Description:    fmt.Sprintf("节点 %s 的 CPU 使用率为 %.2f%%，建议优化或扩容", instance.Name, metric.CPUUsage),
				Impact:         "可能导致服务响应缓慢",
				Action:         "考虑增加节点或优化应用程序",
				Status:         "pending",
				CreatedAt:      time.Now(),
			})
		}

		// 内存使用率过高
		if metric.MemoryUsage > 90 {
			suggestions = append(suggestions, models.NodeOptimizationSuggestion{
				NodeGroupID:    &nodeGroupID,
				NodeInstanceID: &instance.ID,
				Category:       "performance",
				Priority:       "high",
				Title:          "内存使用率过高",
				Description:    fmt.Sprintf("节点 %s 的内存使用率为 %.2f%%，建议优化或扩容", instance.Name, metric.MemoryUsage),
				Impact:         "可能导致 OOM 错误",
				Action:         "考虑增加内存或优化内存使用",
				Status:         "pending",
				CreatedAt:      time.Now(),
			})
		}

		// 磁盘使用率过高
		if metric.DiskUsage > 85 {
			suggestions = append(suggestions, models.NodeOptimizationSuggestion{
				NodeGroupID:    &nodeGroupID,
				NodeInstanceID: &instance.ID,
				Category:       "reliability",
				Priority:       "medium",
				Title:          "磁盘使用率过高",
				Description:    fmt.Sprintf("节点 %s 的磁盘使用率为 %.2f%%，建议清理或扩容", instance.Name, metric.DiskUsage),
				Impact:         "可能导致写入失败",
				Action:         "清理日志文件或增加磁盘空间",
				Status:         "pending",
				CreatedAt:      time.Now(),
			})
		}
	}

	// 批量创建建议
	if len(suggestions) > 0 {
		if err := s.db.Create(&suggestions).Error; err != nil {
			return err
		}
		zap.L().Info("生成优化建议",
			zap.Uint("node_group_id", nodeGroupID),
			zap.Int("count", len(suggestions)))
	}

	return nil
}

// GetSuggestions 获取优化建议
func (s *NodeAutomationService) GetSuggestions(nodeGroupID *uint, status string, limit int) ([]models.NodeOptimizationSuggestion, error) {
	if limit <= 0 {
		limit = 100
	}

	query := s.db.Model(&models.NodeOptimizationSuggestion{})
	if nodeGroupID != nil {
		query = query.Where("node_group_id = ?", *nodeGroupID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var suggestions []models.NodeOptimizationSuggestion
	if err := query.Order("priority DESC, created_at DESC").Limit(limit).Find(&suggestions).Error; err != nil {
		return nil, err
	}

	return suggestions, nil
}

// ApplySuggestion 应用优化建议
func (s *NodeAutomationService) ApplySuggestion(suggestionID uint) error {
	now := time.Now()
	return s.db.Model(&models.NodeOptimizationSuggestion{}).
		Where("id = ?", suggestionID).
		Updates(map[string]interface{}{
			"status":     "applied",
			"applied_at": &now,
		}).Error
}

// DismissSuggestion 忽略优化建议
func (s *NodeAutomationService) DismissSuggestion(suggestionID uint) error {
	now := time.Now()
	return s.db.Model(&models.NodeOptimizationSuggestion{}).
		Where("id = ?", suggestionID).
		Updates(map[string]interface{}{
			"status":       "dismissed",
			"dismissed_at": &now,
		}).Error
}

// GetAutomationStats 获取自动化统计
func (s *NodeAutomationService) GetAutomationStats(nodeGroupID uint, duration time.Duration) (map[string]interface{}, error) {
	startTime := time.Now().Add(-duration)

	var actions []models.NodeAutomationAction
	if err := s.db.Where("node_group_id = ? AND executed_at >= ?", nodeGroupID, startTime).
		Find(&actions).Error; err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"total_actions":   len(actions),
		"scale_up_count":  0,
		"scale_down_count": 0,
		"failover_count":  0,
		"recovery_count":  0,
		"success_count":   0,
		"failed_count":    0,
	}

	for _, action := range actions {
		switch action.ActionType {
		case "scale_up":
			stats["scale_up_count"] = stats["scale_up_count"].(int) + 1
		case "scale_down":
			stats["scale_down_count"] = stats["scale_down_count"].(int) + 1
		case "failover":
			stats["failover_count"] = stats["failover_count"].(int) + 1
		case "recovery":
			stats["recovery_count"] = stats["recovery_count"].(int) + 1
		}

		if action.Status == "success" {
			stats["success_count"] = stats["success_count"].(int) + 1
		} else if action.Status == "failed" {
			stats["failed_count"] = stats["failed_count"].(int) + 1
		}
	}

	return stats, nil
}
