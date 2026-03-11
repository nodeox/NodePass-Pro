package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"nodepass-pro/backend/internal/domain/benefitcode"
	"nodepass-pro/backend/internal/models"

	"gorm.io/gorm"
)

// BenefitCodeRepository 权益码仓储实现
type BenefitCodeRepository struct {
	db *gorm.DB
}

// NewBenefitCodeRepository 创建权益码仓储
func NewBenefitCodeRepository(db *gorm.DB) *BenefitCodeRepository {
	return &BenefitCodeRepository{
		db: db,
	}
}

// Create 创建权益码
func (r *BenefitCodeRepository) Create(ctx context.Context, code *benefitcode.BenefitCode) error {
	model := r.toModel(code)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		if isDuplicateKeyError(err) {
			return benefitcode.ErrBenefitCodeAlreadyExists
		}
		return fmt.Errorf("创建权益码失败: %w", err)
	}
	code.ID = model.ID
	return nil
}

// BatchCreate 批量创建权益码
func (r *BenefitCodeRepository) BatchCreate(ctx context.Context, codes []*benefitcode.BenefitCode) error {
	if len(codes) == 0 {
		return nil
	}

	models := make([]*models.BenefitCode, len(codes))
	for i, code := range codes {
		models[i] = r.toModel(code)
	}

	if err := r.db.WithContext(ctx).CreateInBatches(models, 100).Error; err != nil {
		if isDuplicateKeyError(err) {
			return benefitcode.ErrBenefitCodeAlreadyExists
		}
		return fmt.Errorf("批量创建权益码失败: %w", err)
	}

	// 更新 ID
	for i, model := range models {
		codes[i].ID = model.ID
	}

	return nil
}

// FindByID 根据 ID 查找
func (r *BenefitCodeRepository) FindByID(ctx context.Context, id uint) (*benefitcode.BenefitCode, error) {
	var model models.BenefitCode
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, benefitcode.ErrBenefitCodeNotFound
		}
		return nil, fmt.Errorf("查找权益码失败: %w", err)
	}
	return r.toDomain(&model), nil
}

// FindByCode 根据 Code 查找
func (r *BenefitCodeRepository) FindByCode(ctx context.Context, code string) (*benefitcode.BenefitCode, error) {
	var model models.BenefitCode
	if err := r.db.WithContext(ctx).Where("code = ?", code).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, benefitcode.ErrBenefitCodeNotFound
		}
		return nil, fmt.Errorf("查找权益码失败: %w", err)
	}
	return r.toDomain(&model), nil
}

// Update 更新权益码
func (r *BenefitCodeRepository) Update(ctx context.Context, code *benefitcode.BenefitCode) error {
	model := r.toModel(code)
	if err := r.db.WithContext(ctx).Save(model).Error; err != nil {
		return fmt.Errorf("更新权益码失败: %w", err)
	}
	return nil
}

// Delete 删除权益码
func (r *BenefitCodeRepository) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&models.BenefitCode{}, id)
	if result.Error != nil {
		return fmt.Errorf("删除权益码失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return benefitcode.ErrBenefitCodeNotFound
	}
	return nil
}

// BatchDelete 批量删除权益码
func (r *BenefitCodeRepository) BatchDelete(ctx context.Context, ids []uint) (int64, error) {
	result := r.db.WithContext(ctx).Where("id IN ?", ids).Delete(&models.BenefitCode{})
	if result.Error != nil {
		return 0, fmt.Errorf("批量删除权益码失败: %w", result.Error)
	}
	return result.RowsAffected, nil
}

// List 列表查询
func (r *BenefitCodeRepository) List(ctx context.Context, filter benefitcode.ListFilter) ([]*benefitcode.BenefitCode, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.BenefitCode{})

	// 应用过滤条件
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.VIPLevel != nil {
		query = query.Where("vip_level = ?", *filter.VIPLevel)
	}
	if filter.UsedBy != nil {
		query = query.Where("used_by = ?", *filter.UsedBy)
	}

	// 统计总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计权益码总数失败: %w", err)
	}

	// 分页查询
	var models []*models.BenefitCode
	offset := (filter.Page - 1) * filter.PageSize
	if err := query.Order("id DESC").Offset(offset).Limit(filter.PageSize).Find(&models).Error; err != nil {
		return nil, 0, fmt.Errorf("查询权益码列表失败: %w", err)
	}

	// 转换为领域对象
	codes := make([]*benefitcode.BenefitCode, len(models))
	for i, model := range models {
		codes[i] = r.toDomain(model)
	}

	return codes, total, nil
}

// CountByStatus 按状态统计
func (r *BenefitCodeRepository) CountByStatus(ctx context.Context, status benefitcode.BenefitCodeStatus) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.BenefitCode{}).Where("status = ?", status).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("统计权益码失败: %w", err)
	}
	return count, nil
}

// FindExpiredCodes 查找过期的权益码
func (r *BenefitCodeRepository) FindExpiredCodes(ctx context.Context, limit int) ([]*benefitcode.BenefitCode, error) {
	var models []*models.BenefitCode
	now := time.Now()
	if err := r.db.WithContext(ctx).
		Where("expires_at IS NOT NULL AND expires_at < ? AND status = ?", now, benefitcode.BenefitCodeStatusUnused).
		Limit(limit).
		Find(&models).Error; err != nil {
		return nil, fmt.Errorf("查找过期权益码失败: %w", err)
	}

	codes := make([]*benefitcode.BenefitCode, len(models))
	for i, model := range models {
		codes[i] = r.toDomain(model)
	}

	return codes, nil
}

// toModel 转换为数据库模型
func (r *BenefitCodeRepository) toModel(code *benefitcode.BenefitCode) *models.BenefitCode {
	return &models.BenefitCode{
		ID:           code.ID,
		Code:         code.Code,
		VipLevel:     code.VIPLevel,
		DurationDays: code.DurationDays,
		Status:       string(code.Status),
		IsEnabled:    code.IsEnabled,
		UsedBy:       code.UsedBy,
		UsedAt:       code.UsedAt,
		ExpiresAt:    code.ExpiresAt,
		CreatedAt:    code.CreatedAt,
	}
}

// toDomain 转换为领域对象
func (r *BenefitCodeRepository) toDomain(model *models.BenefitCode) *benefitcode.BenefitCode {
	return &benefitcode.BenefitCode{
		ID:           model.ID,
		Code:         model.Code,
		VIPLevel:     model.VipLevel,
		DurationDays: model.DurationDays,
		Status:       benefitcode.BenefitCodeStatus(model.Status),
		IsEnabled:    model.IsEnabled,
		UsedBy:       model.UsedBy,
		UsedAt:       model.UsedAt,
		ExpiresAt:    model.ExpiresAt,
		CreatedAt:    model.CreatedAt,
		UpdatedAt:    time.Now(),
	}
}

// isDuplicateKeyError 检查是否为重复键错误
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := strings.ToLower(err.Error())
	return strings.Contains(errMsg, "duplicate") ||
		strings.Contains(errMsg, "unique constraint") ||
		strings.Contains(errMsg, "unique index")
}
