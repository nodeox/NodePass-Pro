package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"nodepass-pro/backend/internal/domain/auditlog"
	"nodepass-pro/backend/internal/models"

	"gorm.io/gorm"
)

// AuditLogRepository 审计日志仓储实现
type AuditLogRepository struct {
	db *gorm.DB
}

// NewAuditLogRepository 创建审计日志仓储
func NewAuditLogRepository(db *gorm.DB) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

// Create 创建审计日志
func (r *AuditLogRepository) Create(ctx context.Context, log *auditlog.AuditLog) error {
	model := r.toModel(log)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("创建审计日志失败: %w", err)
	}
	log.ID = model.ID
	return nil
}

// BatchCreate 批量创建审计日志
func (r *AuditLogRepository) BatchCreate(ctx context.Context, logs []*auditlog.AuditLog) error {
	if len(logs) == 0 {
		return nil
	}

	models := make([]*models.AuditLog, len(logs))
	for i, log := range logs {
		models[i] = r.toModel(log)
	}

	if err := r.db.WithContext(ctx).CreateInBatches(models, 100).Error; err != nil {
		return fmt.Errorf("批量创建审计日志失败: %w", err)
	}

	// 更新 ID
	for i, model := range models {
		logs[i].ID = model.ID
	}

	return nil
}

// FindByID 根据 ID 查找
func (r *AuditLogRepository) FindByID(ctx context.Context, id uint) (*auditlog.AuditLog, error) {
	var model models.AuditLog
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, auditlog.ErrAuditLogNotFound
		}
		return nil, fmt.Errorf("查找审计日志失败: %w", err)
	}
	return r.toDomain(&model), nil
}

// List 列表查询
func (r *AuditLogRepository) List(ctx context.Context, filter auditlog.ListFilter) ([]*auditlog.AuditLog, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.AuditLog{})

	// 应用过滤条件
	if filter.UserID != nil {
		query = query.Where("user_id = ?", *filter.UserID)
	}
	if filter.Action != "" {
		query = query.Where("action = ?", filter.Action)
	}
	if filter.ResourceType != "" {
		query = query.Where("resource_type = ?", filter.ResourceType)
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
		return nil, 0, fmt.Errorf("统计审计日志总数失败: %w", err)
	}

	// 分页查询
	var models []*models.AuditLog
	offset := (filter.Page - 1) * filter.PageSize
	if err := query.Order("id DESC").Offset(offset).Limit(filter.PageSize).Find(&models).Error; err != nil {
		return nil, 0, fmt.Errorf("查询审计日志列表失败: %w", err)
	}

	// 转换为领域对象
	logs := make([]*auditlog.AuditLog, len(models))
	for i, model := range models {
		logs[i] = r.toDomain(model)
	}

	return logs, total, nil
}

// CountByAction 按操作统计
func (r *AuditLogRepository) CountByAction(ctx context.Context, action string, startTime, endTime time.Time) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&models.AuditLog{}).
		Where("action = ?", action).
		Where("created_at >= ? AND created_at <= ?", startTime, endTime)

	if err := query.Count(&count).Error; err != nil {
		return 0, fmt.Errorf("统计操作失败: %w", err)
	}
	return count, nil
}

// CountByUser 按用户统计
func (r *AuditLogRepository) CountByUser(ctx context.Context, userID uint, startTime, endTime time.Time) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&models.AuditLog{}).
		Where("user_id = ?", userID).
		Where("created_at >= ? AND created_at <= ?", startTime, endTime)

	if err := query.Count(&count).Error; err != nil {
		return 0, fmt.Errorf("统计用户操作失败: %w", err)
	}
	return count, nil
}

// DeleteOldLogs 删除旧日志
func (r *AuditLogRepository) DeleteOldLogs(ctx context.Context, before time.Time) (int64, error) {
	result := r.db.WithContext(ctx).Where("created_at < ?", before).Delete(&models.AuditLog{})
	if result.Error != nil {
		return 0, fmt.Errorf("删除旧日志失败: %w", result.Error)
	}
	return result.RowsAffected, nil
}

// toModel 转换为数据库模型
func (r *AuditLogRepository) toModel(log *auditlog.AuditLog) *models.AuditLog {
	var resourceType *string
	if log.ResourceType != "" {
		resourceType = &log.ResourceType
	}

	var details *string
	if log.Details != "" {
		details = &log.Details
	}

	var ipAddress *string
	if log.IPAddress != "" {
		ipAddress = &log.IPAddress
	}

	var userAgent *string
	if log.UserAgent != "" {
		userAgent = &log.UserAgent
	}

	return &models.AuditLog{
		ID:           log.ID,
		UserID:       log.UserID,
		Action:       log.Action,
		ResourceType: resourceType,
		ResourceID:   log.ResourceID,
		Details:      details,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		CreatedAt:    log.CreatedAt,
	}
}

// toDomain 转换为领域对象
func (r *AuditLogRepository) toDomain(model *models.AuditLog) *auditlog.AuditLog {
	resourceType := ""
	if model.ResourceType != nil {
		resourceType = *model.ResourceType
	}

	details := ""
	if model.Details != nil {
		details = *model.Details
	}

	ipAddress := ""
	if model.IPAddress != nil {
		ipAddress = *model.IPAddress
	}

	userAgent := ""
	if model.UserAgent != nil {
		userAgent = *model.UserAgent
	}

	return &auditlog.AuditLog{
		ID:           model.ID,
		UserID:       model.UserID,
		Action:       model.Action,
		ResourceType: resourceType,
		ResourceID:   model.ResourceID,
		Details:      details,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		CreatedAt:    model.CreatedAt,
	}
}
