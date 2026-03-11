package vip

import (
	"context"
	"errors"
	"fmt"
	"time"

	"nodepass-pro/backend/internal/domain/vip"
	"nodepass-pro/backend/internal/models"

	"gorm.io/gorm"
)

// VIPRepository VIP 仓储实现
type VIPRepository struct {
	db *gorm.DB
}

// NewVIPRepository 创建 VIP 仓储
func NewVIPRepository(db *gorm.DB) vip.Repository {
	return &VIPRepository{db: db}
}

// FindLevelByID 通过 ID 查找 VIP 等级
func (r *VIPRepository) FindLevelByID(ctx context.Context, id uint) (*vip.VIPLevel, error) {
	var level models.VIPLevel
	if err := r.db.WithContext(ctx).First(&level, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, vip.ErrLevelNotFound
		}
		return nil, err
	}
	return r.toDomainLevel(&level), nil
}

// FindLevelByLevel 通过等级数字查找 VIP 等级
func (r *VIPRepository) FindLevelByLevel(ctx context.Context, level int) (*vip.VIPLevel, error) {
	var vipLevel models.VIPLevel
	if err := r.db.WithContext(ctx).Where("level = ?", level).First(&vipLevel).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, vip.ErrLevelNotFound
		}
		return nil, err
	}
	return r.toDomainLevel(&vipLevel), nil
}

// FindByLevel 通过等级数字查找 VIP 等级（别名方法）
func (r *VIPRepository) FindByLevel(ctx context.Context, level int) (*vip.VIPLevel, error) {
	return r.FindLevelByLevel(ctx, level)
}

// ListLevels 列出所有 VIP 等级
func (r *VIPRepository) ListLevels(ctx context.Context) ([]*vip.VIPLevel, error) {
	var levels []models.VIPLevel
	if err := r.db.WithContext(ctx).Order("level ASC").Find(&levels).Error; err != nil {
		return nil, err
	}

	result := make([]*vip.VIPLevel, len(levels))
	for i, level := range levels {
		result[i] = r.toDomainLevel(&level)
	}
	return result, nil
}

// CreateLevel 创建 VIP 等级
func (r *VIPRepository) CreateLevel(ctx context.Context, level *vip.VIPLevel) error {
	modelLevel := r.toModelLevel(level)
	if err := r.db.WithContext(ctx).Create(modelLevel).Error; err != nil {
		return err
	}
	level.ID = modelLevel.ID
	level.CreatedAt = modelLevel.CreatedAt
	level.UpdatedAt = modelLevel.UpdatedAt
	return nil
}

// UpdateLevel 更新 VIP 等级
func (r *VIPRepository) UpdateLevel(ctx context.Context, level *vip.VIPLevel) error {
	modelLevel := r.toModelLevel(level)
	return r.db.WithContext(ctx).Save(modelLevel).Error
}

// DeleteLevel 删除 VIP 等级
func (r *VIPRepository) DeleteLevel(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.VIPLevel{}, id).Error
}

// CheckLevelExists 检查 VIP 等级是否存在
func (r *VIPRepository) CheckLevelExists(ctx context.Context, level int) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.VIPLevel{}).Where("level = ?", level).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetUserVIP 获取用户 VIP 信息
func (r *VIPRepository) GetUserVIP(ctx context.Context, userID uint) (*vip.UserVIP, error) {
	var user models.User
	if err := r.db.WithContext(ctx).First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, vip.ErrUserNotFound
		}
		return nil, err
	}

	// 查找 VIP 等级详情
	var levelDetail *vip.VIPLevel
	var vipLevel models.VIPLevel
	if err := r.db.WithContext(ctx).Where("level = ?", user.VipLevel).First(&vipLevel).Error; err == nil {
		levelDetail = r.toDomainLevel(&vipLevel)
	}

	return &vip.UserVIP{
		UserID:       user.ID,
		VIPLevel:     user.VipLevel,
		VIPExpiresAt: user.VipExpiresAt,
		LevelDetail:  levelDetail,
	}, nil
}

// UpgradeUserVIP 升级用户 VIP
func (r *VIPRepository) UpgradeUserVIP(ctx context.Context, userID uint, level int, expiresAt *time.Time) error {
	// 查找 VIP 等级配置
	vipLevel, err := r.FindLevelByLevel(ctx, level)
	if err != nil {
		return err
	}

	// 更新用户 VIP 信息和配额
	updates := map[string]interface{}{
		"vip_level":                   level,
		"vip_expires_at":              expiresAt,
		"traffic_quota":               vipLevel.TrafficQuota,
		"max_rules":                   vipLevel.MaxRules,
		"max_bandwidth":               vipLevel.MaxBandwidth,
		"max_self_hosted_entry_nodes": vipLevel.MaxSelfHostedEntryNodes,
		"max_self_hosted_exit_nodes":  vipLevel.MaxSelfHostedExitNodes,
	}

	return r.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", userID).Updates(updates).Error
}

// CheckExpiredUsers 检查过期的 VIP 用户
func (r *VIPRepository) CheckExpiredUsers(ctx context.Context) ([]uint, error) {
	var userIDs []uint
	now := time.Now()

	if err := r.db.WithContext(ctx).
		Model(&models.User{}).
		Where("vip_level > 0").
		Where("vip_expires_at IS NOT NULL").
		Where("vip_expires_at < ?", now).
		Pluck("id", &userIDs).Error; err != nil {
		return nil, err
	}

	return userIDs, nil
}

// DegradeExpiredUsers 降级过期的 VIP 用户
func (r *VIPRepository) DegradeExpiredUsers(ctx context.Context, userIDs []uint) (int64, error) {
	if len(userIDs) == 0 {
		return 0, nil
	}

	// 查找免费等级配置
	freeLevel, err := r.FindLevelByLevel(ctx, 0)
	if err != nil {
		return 0, fmt.Errorf("查找免费等级失败: %w", err)
	}

	// 批量降级用户
	updates := map[string]interface{}{
		"vip_level":                   0,
		"vip_expires_at":              nil,
		"traffic_quota":               freeLevel.TrafficQuota,
		"max_rules":                   freeLevel.MaxRules,
		"max_bandwidth":               freeLevel.MaxBandwidth,
		"max_self_hosted_entry_nodes": freeLevel.MaxSelfHostedEntryNodes,
		"max_self_hosted_exit_nodes":  freeLevel.MaxSelfHostedExitNodes,
	}

	result := r.db.WithContext(ctx).Model(&models.User{}).Where("id IN ?", userIDs).Updates(updates)
	return result.RowsAffected, result.Error
}

// toDomainLevel 转换为领域 VIP 等级
func (r *VIPRepository) toDomainLevel(level *models.VIPLevel) *vip.VIPLevel {
	return &vip.VIPLevel{
		ID:                      level.ID,
		Level:                   level.Level,
		Name:                    level.Name,
		Description:             level.Description,
		TrafficQuota:            level.TrafficQuota,
		MaxRules:                level.MaxRules,
		MaxBandwidth:            level.MaxBandwidth,
		MaxSelfHostedEntryNodes: level.MaxSelfHostedEntryNodes,
		MaxSelfHostedExitNodes:  level.MaxSelfHostedExitNodes,
		AccessibleNodeLevel:     level.AccessibleNodeLevel,
		TrafficMultiplier:       level.TrafficMultiplier,
		CustomFeatures:          level.CustomFeatures,
		Price:                   level.Price,
		DurationDays:            level.DurationDays,
		CreatedAt:               level.CreatedAt,
		UpdatedAt:               level.UpdatedAt,
	}
}

// toModelLevel 转换为模型 VIP 等级
func (r *VIPRepository) toModelLevel(level *vip.VIPLevel) *models.VIPLevel {
	return &models.VIPLevel{
		ID:                      level.ID,
		Level:                   level.Level,
		Name:                    level.Name,
		Description:             level.Description,
		TrafficQuota:            level.TrafficQuota,
		MaxRules:                level.MaxRules,
		MaxBandwidth:            level.MaxBandwidth,
		MaxSelfHostedEntryNodes: level.MaxSelfHostedEntryNodes,
		MaxSelfHostedExitNodes:  level.MaxSelfHostedExitNodes,
		AccessibleNodeLevel:     level.AccessibleNodeLevel,
		TrafficMultiplier:       level.TrafficMultiplier,
		CustomFeatures:          level.CustomFeatures,
		Price:                   level.Price,
		DurationDays:            level.DurationDays,
		CreatedAt:               level.CreatedAt,
		UpdatedAt:               level.UpdatedAt,
	}
}
