package postgres

import (
	"context"
	"errors"
	"fmt"

	"nodepass-pro/backend/internal/domain/alert"
	"nodepass-pro/backend/internal/models"

	"gorm.io/gorm"
)

// AlertRepository 告警仓储实现
type AlertRepository struct {
	db *gorm.DB
}

// NewAlertRepository 创建告警仓储
func NewAlertRepository(db *gorm.DB) *AlertRepository {
	return &AlertRepository{db: db}
}

// Create 创建告警
func (r *AlertRepository) Create(ctx context.Context, a *alert.Alert) error {
	model := r.toModel(a)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("创建告警失败: %w", err)
	}
	a.ID = model.ID
	return nil
}

// FindByID 根据 ID 查找
func (r *AlertRepository) FindByID(ctx context.Context, id uint) (*alert.Alert, error) {
	var model models.Alert
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, alert.ErrAlertNotFound
		}
		return nil, fmt.Errorf("查找告警失败: %w", err)
	}
	return r.toDomain(&model), nil
}

// FindByFingerprint 根据指纹查找
func (r *AlertRepository) FindByFingerprint(ctx context.Context, fingerprint string) (*alert.Alert, error) {
	var model models.Alert
	if err := r.db.WithContext(ctx).Where("fingerprint = ?", fingerprint).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("查找告警失败: %w", err)
	}
	return r.toDomain(&model), nil
}

// Update 更新告警
func (r *AlertRepository) Update(ctx context.Context, a *alert.Alert) error {
	model := r.toModel(a)
	if err := r.db.WithContext(ctx).Save(model).Error; err != nil {
		return fmt.Errorf("更新告警失败: %w", err)
	}
	return nil
}

// List 列表查询
func (r *AlertRepository) List(ctx context.Context, filter alert.ListFilter) ([]*alert.Alert, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.Alert{})

	// 应用过滤条件
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Level != "" {
		query = query.Where("level = ?", filter.Level)
	}
	if filter.Type != "" {
		query = query.Where("type = ?", filter.Type)
	}
	if filter.ResourceType != "" {
		query = query.Where("resource_type = ?", filter.ResourceType)
	}
	if filter.ResourceID != nil {
		query = query.Where("resource_id = ?", *filter.ResourceID)
	}
	if filter.StartTime != nil {
		query = query.Where("created_at >= ?", *filter.StartTime)
	}
	if filter.EndTime != nil {
		query = query.Where("created_at <= ?", *filter.EndTime)
	}

	// 统计总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计告警总数失败: %w", err)
	}

	// 分页查询
	var models []*models.Alert
	offset := (filter.Page - 1) * filter.PageSize
	if err := query.Order("id DESC").Offset(offset).Limit(filter.PageSize).Find(&models).Error; err != nil {
		return nil, 0, fmt.Errorf("查询告警列表失败: %w", err)
	}

	// 转换为领域对象
	alerts := make([]*alert.Alert, len(models))
	for i, model := range models {
		alerts[i] = r.toDomain(model)
	}

	return alerts, total, nil
}

// CountByStatus 按状态统计
func (r *AlertRepository) CountByStatus(ctx context.Context, status alert.AlertStatus) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.Alert{}).Where("status = ?", status).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("统计告警失败: %w", err)
	}
	return count, nil
}

// CountByLevel 按级别统计
func (r *AlertRepository) CountByLevel(ctx context.Context, level alert.AlertLevel) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.Alert{}).Where("level = ?", level).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("统计告警失败: %w", err)
	}
	return count, nil
}

// FindFiringAlerts 查找正在触发的告警
func (r *AlertRepository) FindFiringAlerts(ctx context.Context) ([]*alert.Alert, error) {
	var models []*models.Alert
	if err := r.db.WithContext(ctx).Where("status = ?", alert.AlertStatusFiring).Find(&models).Error; err != nil {
		return nil, fmt.Errorf("查找触发中的告警失败: %w", err)
	}

	alerts := make([]*alert.Alert, len(models))
	for i, model := range models {
		alerts[i] = r.toDomain(model)
	}

	return alerts, nil
}

// toModel 转换为数据库模型
func (r *AlertRepository) toModel(a *alert.Alert) *models.Alert {
	return &models.Alert{
		ID:             a.ID,
		Type:           models.AlertType(a.Type),
		Level:          models.AlertLevel(a.Level),
		Status:         models.AlertStatus(a.Status),
		Title:          a.Title,
		Message:        a.Message,
		Fingerprint:    a.Fingerprint,
		ResourceType:   a.ResourceType,
		ResourceID:     a.ResourceID,
		ResourceName:   a.ResourceName,
		Value:          a.Value,
		Threshold:      a.Threshold,
		FirstFiredAt:   a.FirstFiredAt,
		LastFiredAt:    a.LastFiredAt,
		ResolvedAt:     a.ResolvedAt,
		AcknowledgedAt: a.AcknowledgedAt,
		SilencedUntil:  a.SilencedUntil,
		AcknowledgedBy: a.AcknowledgedBy,
		ResolvedBy:     a.ResolvedBy,
		Notes:          a.Notes,
		CreatedAt:      a.CreatedAt,
		UpdatedAt:      a.UpdatedAt,
	}
}

// toDomain 转换为领域对象
func (r *AlertRepository) toDomain(model *models.Alert) *alert.Alert {
	return &alert.Alert{
		ID:             model.ID,
		Type:           string(model.Type),
		Level:          alert.AlertLevel(model.Level),
		Status:         alert.AlertStatus(model.Status),
		Title:          model.Title,
		Message:        model.Message,
		Fingerprint:    model.Fingerprint,
		ResourceType:   model.ResourceType,
		ResourceID:     model.ResourceID,
		ResourceName:   model.ResourceName,
		Value:          model.Value,
		Threshold:      model.Threshold,
		FirstFiredAt:   model.FirstFiredAt,
		LastFiredAt:    model.LastFiredAt,
		ResolvedAt:     model.ResolvedAt,
		AcknowledgedAt: model.AcknowledgedAt,
		SilencedUntil:  model.SilencedUntil,
		AcknowledgedBy: model.AcknowledgedBy,
		ResolvedBy:     model.ResolvedBy,
		Notes:          model.Notes,
		CreatedAt:      model.CreatedAt,
		UpdatedAt:      model.UpdatedAt,
	}
}
