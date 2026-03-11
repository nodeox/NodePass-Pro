package postgres

import (
	"context"
	"errors"
	"fmt"

	"nodepass-pro/backend/internal/domain/systemconfig"
	"nodepass-pro/backend/internal/models"

	"gorm.io/gorm"
)

// SystemConfigRepository 系统配置仓储实现
type SystemConfigRepository struct {
	db *gorm.DB
}

// NewSystemConfigRepository 创建系统配置仓储
func NewSystemConfigRepository(db *gorm.DB) *SystemConfigRepository {
	return &SystemConfigRepository{db: db}
}

// FindByKey 根据键查找配置
func (r *SystemConfigRepository) FindByKey(ctx context.Context, key string) (*systemconfig.SystemConfig, error) {
	var model models.SystemConfig
	if err := r.db.WithContext(ctx).Where("key = ?", key).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, systemconfig.ErrConfigNotFound
		}
		return nil, fmt.Errorf("查找配置失败: %w", err)
	}
	return r.toDomain(&model), nil
}

// FindAll 查找所有配置
func (r *SystemConfigRepository) FindAll(ctx context.Context) ([]*systemconfig.SystemConfig, error) {
	var models []*models.SystemConfig
	if err := r.db.WithContext(ctx).Order("id ASC").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("查询配置列表失败: %w", err)
	}

	configs := make([]*systemconfig.SystemConfig, len(models))
	for i, model := range models {
		configs[i] = r.toDomain(model)
	}

	return configs, nil
}

// Upsert 创建或更新配置
func (r *SystemConfigRepository) Upsert(ctx context.Context, config *systemconfig.SystemConfig) error {
	model := r.toModel(config)

	// 尝试查找现有记录
	var existing models.SystemConfig
	err := r.db.WithContext(ctx).Where("key = ?", config.Key).First(&existing).Error

	if err == nil {
		// 更新现有记录
		model.ID = existing.ID
		if err := r.db.WithContext(ctx).Save(model).Error; err != nil {
			return fmt.Errorf("更新配置失败: %w", err)
		}
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		// 创建新记录
		if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
			return fmt.Errorf("创建配置失败: %w", err)
		}
		config.ID = model.ID
	} else {
		return fmt.Errorf("查询配置失败: %w", err)
	}

	return nil
}

// Delete 删除配置
func (r *SystemConfigRepository) Delete(ctx context.Context, key string) error {
	if err := r.db.WithContext(ctx).Where("key = ?", key).Delete(&models.SystemConfig{}).Error; err != nil {
		return fmt.Errorf("删除配置失败: %w", err)
	}
	return nil
}

// GetAllAsMap 获取所有配置作为 map
func (r *SystemConfigRepository) GetAllAsMap(ctx context.Context) (map[string]string, error) {
	configs, err := r.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string, len(configs))
	for _, config := range configs {
		result[config.Key] = config.GetValueOrDefault("")
	}

	return result, nil
}

// toModel 转换为数据库模型
func (r *SystemConfigRepository) toModel(config *systemconfig.SystemConfig) *models.SystemConfig {
	return &models.SystemConfig{
		ID:          config.ID,
		Key:         config.Key,
		Value:       config.Value,
		Description: config.Description,
		UpdatedAt:   config.UpdatedAt,
	}
}

// toDomain 转换为领域对象
func (r *SystemConfigRepository) toDomain(model *models.SystemConfig) *systemconfig.SystemConfig {
	return &systemconfig.SystemConfig{
		ID:          model.ID,
		Key:         model.Key,
		Value:       model.Value,
		Description: model.Description,
		UpdatedAt:   model.UpdatedAt,
	}
}
