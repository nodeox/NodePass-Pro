package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"nodepass-pro/backend/internal/domain/healthcheck"
	"nodepass-pro/backend/internal/models"

	"gorm.io/gorm"
)

// HealthCheckRepository 健康检查仓储实现
type HealthCheckRepository struct {
	db *gorm.DB
}

// NewHealthCheckRepository 创建健康检查仓储
func NewHealthCheckRepository(db *gorm.DB) *HealthCheckRepository {
	return &HealthCheckRepository{db: db}
}

// CreateHealthCheck 创建健康检查配置
func (r *HealthCheckRepository) CreateHealthCheck(ctx context.Context, check *healthcheck.HealthCheck) error {
	model := r.healthCheckToModel(check)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("创建健康检查配置失败: %w", err)
	}
	check.ID = model.ID
	return nil
}

// FindHealthCheckByID 根据 ID 查找健康检查配置
func (r *HealthCheckRepository) FindHealthCheckByID(ctx context.Context, id uint) (*healthcheck.HealthCheck, error) {
	var model models.NodeHealthCheck
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, healthcheck.ErrHealthCheckNotFound
		}
		return nil, fmt.Errorf("查找健康检查配置失败: %w", err)
	}
	return r.healthCheckToDomain(&model), nil
}

// FindHealthCheckByNodeInstance 根据节点实例 ID 查找健康检查配置
func (r *HealthCheckRepository) FindHealthCheckByNodeInstance(ctx context.Context, nodeInstanceID uint) (*healthcheck.HealthCheck, error) {
	var model models.NodeHealthCheck
	if err := r.db.WithContext(ctx).Where("node_instance_id = ?", nodeInstanceID).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, healthcheck.ErrHealthCheckNotFound
		}
		return nil, fmt.Errorf("查找健康检查配置失败: %w", err)
	}
	return r.healthCheckToDomain(&model), nil
}

// UpdateHealthCheck 更新健康检查配置
func (r *HealthCheckRepository) UpdateHealthCheck(ctx context.Context, check *healthcheck.HealthCheck) error {
	model := r.healthCheckToModel(check)
	if err := r.db.WithContext(ctx).Save(model).Error; err != nil {
		return fmt.Errorf("更新健康检查配置失败: %w", err)
	}
	return nil
}

// DeleteHealthCheck 删除健康检查配置
func (r *HealthCheckRepository) DeleteHealthCheck(ctx context.Context, id uint) error {
	if err := r.db.WithContext(ctx).Delete(&models.NodeHealthCheck{}, id).Error; err != nil {
		return fmt.Errorf("删除健康检查配置失败: %w", err)
	}
	return nil
}

// ListEnabledHealthChecks 列出所有启用的健康检查配置
func (r *HealthCheckRepository) ListEnabledHealthChecks(ctx context.Context) ([]*healthcheck.HealthCheck, error) {
	var models []*models.NodeHealthCheck
	if err := r.db.WithContext(ctx).Where("enabled = ?", true).Find(&models).Error; err != nil {
		return nil, fmt.Errorf("查询启用的健康检查配置失败: %w", err)
	}

	checks := make([]*healthcheck.HealthCheck, len(models))
	for i, model := range models {
		checks[i] = r.healthCheckToDomain(model)
	}

	return checks, nil
}

// CreateHealthRecord 创建健康检查记录
func (r *HealthCheckRepository) CreateHealthRecord(ctx context.Context, record *healthcheck.HealthRecord) error {
	model := r.healthRecordToModel(record)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("创建健康检查记录失败: %w", err)
	}
	record.ID = model.ID
	return nil
}

// FindHealthRecordsByNodeInstance 根据节点实例 ID 查找健康检查记录
func (r *HealthCheckRepository) FindHealthRecordsByNodeInstance(ctx context.Context, nodeInstanceID uint, limit int) ([]*healthcheck.HealthRecord, error) {
	var models []*models.NodeHealthRecord
	if err := r.db.WithContext(ctx).
		Where("node_instance_id = ?", nodeInstanceID).
		Order("checked_at DESC").
		Limit(limit).
		Find(&models).Error; err != nil {
		return nil, fmt.Errorf("查询健康检查记录失败: %w", err)
	}

	records := make([]*healthcheck.HealthRecord, len(models))
	for i, model := range models {
		records[i] = r.healthRecordToDomain(model)
	}

	return records, nil
}

// FindHealthRecordsByTimeRange 根据时间范围查找健康检查记录
func (r *HealthCheckRepository) FindHealthRecordsByTimeRange(ctx context.Context, nodeInstanceID uint, startTime time.Time) ([]*healthcheck.HealthRecord, error) {
	var models []*models.NodeHealthRecord
	if err := r.db.WithContext(ctx).
		Where("node_instance_id = ? AND checked_at >= ?", nodeInstanceID, startTime).
		Order("checked_at DESC").
		Find(&models).Error; err != nil {
		return nil, fmt.Errorf("查询健康检查记录失败: %w", err)
	}

	records := make([]*healthcheck.HealthRecord, len(models))
	for i, model := range models {
		records[i] = r.healthRecordToDomain(model)
	}

	return records, nil
}

// DeleteOldHealthRecords 删除旧的健康检查记录
func (r *HealthCheckRepository) DeleteOldHealthRecords(ctx context.Context, cutoffTime time.Time) (int64, error) {
	result := r.db.WithContext(ctx).Where("checked_at < ?", cutoffTime).Delete(&models.NodeHealthRecord{})
	if result.Error != nil {
		return 0, fmt.Errorf("删除旧健康检查记录失败: %w", result.Error)
	}
	return result.RowsAffected, nil
}

// CreateOrUpdateQualityScore 创建或更新质量评分
func (r *HealthCheckRepository) CreateOrUpdateQualityScore(ctx context.Context, score *healthcheck.QualityScore) error {
	model := r.qualityScoreToModel(score)

	// 尝试更新
	result := r.db.WithContext(ctx).
		Model(&models.NodeQualityScore{}).
		Where("node_instance_id = ?", score.NodeInstanceID).
		Updates(model)

	if result.Error != nil {
		return fmt.Errorf("更新质量评分失败: %w", result.Error)
	}

	// 如果没有更新到记录，则创建
	if result.RowsAffected == 0 {
		if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
			return fmt.Errorf("创建质量评分失败: %w", err)
		}
		score.ID = model.ID
	}

	return nil
}

// FindQualityScoreByNodeInstance 根据节点实例 ID 查找质量评分
func (r *HealthCheckRepository) FindQualityScoreByNodeInstance(ctx context.Context, nodeInstanceID uint) (*healthcheck.QualityScore, error) {
	var model models.NodeQualityScore
	if err := r.db.WithContext(ctx).Where("node_instance_id = ?", nodeInstanceID).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, healthcheck.ErrHealthCheckNotFound
		}
		return nil, fmt.Errorf("查找质量评分失败: %w", err)
	}
	return r.qualityScoreToDomain(&model), nil
}

// ListQualityScoresByUser 列出用户的所有质量评分
func (r *HealthCheckRepository) ListQualityScoresByUser(ctx context.Context, userID uint) ([]*healthcheck.QualityScore, error) {
	var models []*models.NodeQualityScore
	if err := r.db.WithContext(ctx).
		Joins("JOIN node_instances ON node_instances.id = node_quality_scores.node_instance_id").
		Joins("JOIN node_groups ON node_groups.id = node_instances.node_group_id").
		Where("node_groups.user_id = ?", userID).
		Order("node_quality_scores.overall_score DESC").
		Find(&models).Error; err != nil {
		return nil, fmt.Errorf("查询质量评分失败: %w", err)
	}

	scores := make([]*healthcheck.QualityScore, len(models))
	for i, model := range models {
		scores[i] = r.qualityScoreToDomain(model)
	}

	return scores, nil
}

// healthCheckToModel 转换为数据库模型
func (r *HealthCheckRepository) healthCheckToModel(check *healthcheck.HealthCheck) *models.NodeHealthCheck {
	return &models.NodeHealthCheck{
		ID:               check.ID,
		NodeInstanceID:   check.NodeInstanceID,
		Type:             models.HealthCheckType(check.Type),
		Enabled:          check.Enabled,
		Interval:         check.Interval,
		Timeout:          check.Timeout,
		Retries:          check.Retries,
		SuccessThreshold: check.SuccessThreshold,
		FailureThreshold: check.FailureThreshold,
		HTTPPath:         check.HTTPPath,
		HTTPMethod:       check.HTTPMethod,
		ExpectedStatus:   check.ExpectedStatus,
		CreatedAt:        check.CreatedAt,
		UpdatedAt:        check.UpdatedAt,
	}
}

// healthCheckToDomain 转换为领域对象
func (r *HealthCheckRepository) healthCheckToDomain(model *models.NodeHealthCheck) *healthcheck.HealthCheck {
	return &healthcheck.HealthCheck{
		ID:               model.ID,
		NodeInstanceID:   model.NodeInstanceID,
		Type:             healthcheck.CheckType(model.Type),
		Enabled:          model.Enabled,
		Interval:         model.Interval,
		Timeout:          model.Timeout,
		Retries:          model.Retries,
		SuccessThreshold: model.SuccessThreshold,
		FailureThreshold: model.FailureThreshold,
		HTTPPath:         model.HTTPPath,
		HTTPMethod:       model.HTTPMethod,
		ExpectedStatus:   model.ExpectedStatus,
		CreatedAt:        model.CreatedAt,
		UpdatedAt:        model.UpdatedAt,
	}
}

// healthRecordToModel 转换为数据库模型
func (r *HealthCheckRepository) healthRecordToModel(record *healthcheck.HealthRecord) *models.NodeHealthRecord {
	return &models.NodeHealthRecord{
		ID:             record.ID,
		NodeInstanceID: record.NodeInstanceID,
		CheckType:      models.HealthCheckType(record.CheckType),
		Status:         models.HealthCheckStatus(record.Status),
		Latency:        record.Latency,
		ErrorMessage:   record.ErrorMessage,
		CheckedAt:      record.CheckedAt,
	}
}

// healthRecordToDomain 转换为领域对象
func (r *HealthCheckRepository) healthRecordToDomain(model *models.NodeHealthRecord) *healthcheck.HealthRecord {
	return &healthcheck.HealthRecord{
		ID:             model.ID,
		NodeInstanceID: model.NodeInstanceID,
		CheckType:      healthcheck.CheckType(model.CheckType),
		Status:         healthcheck.CheckStatus(model.Status),
		Latency:        model.Latency,
		ErrorMessage:   model.ErrorMessage,
		CheckedAt:      model.CheckedAt,
	}
}

// qualityScoreToModel 转换为数据库模型
func (r *HealthCheckRepository) qualityScoreToModel(score *healthcheck.QualityScore) *models.NodeQualityScore {
	return &models.NodeQualityScore{
		ID:             score.ID,
		NodeInstanceID: score.NodeInstanceID,
		LatencyScore:   score.LatencyScore,
		StabilityScore: score.StabilityScore,
		LoadScore:      score.LoadScore,
		OverallScore:   score.OverallScore,
		AvgLatency:     score.AvgLatency,
		Uptime:         score.Uptime,
		SuccessRate:    score.SuccessRate,
		LastCheckedAt:  score.LastCheckedAt,
		UpdatedAt:      score.UpdatedAt,
	}
}

// qualityScoreToDomain 转换为领域对象
func (r *HealthCheckRepository) qualityScoreToDomain(model *models.NodeQualityScore) *healthcheck.QualityScore {
	return &healthcheck.QualityScore{
		ID:             model.ID,
		NodeInstanceID: model.NodeInstanceID,
		LatencyScore:   model.LatencyScore,
		StabilityScore: model.StabilityScore,
		LoadScore:      model.LoadScore,
		OverallScore:   model.OverallScore,
		AvgLatency:     model.AvgLatency,
		Uptime:         model.Uptime,
		SuccessRate:    model.SuccessRate,
		LastCheckedAt:  model.LastCheckedAt,
		UpdatedAt:      model.UpdatedAt,
	}
}
