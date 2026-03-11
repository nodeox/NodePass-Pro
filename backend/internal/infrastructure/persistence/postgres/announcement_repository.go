package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"nodepass-pro/backend/internal/domain/announcement"
	"nodepass-pro/backend/internal/models"

	"gorm.io/gorm"
)

// AnnouncementRepository 公告仓储实现
type AnnouncementRepository struct {
	db *gorm.DB
}

// NewAnnouncementRepository 创建公告仓储
func NewAnnouncementRepository(db *gorm.DB) *AnnouncementRepository {
	return &AnnouncementRepository{db: db}
}

// Create 创建公告
func (r *AnnouncementRepository) Create(ctx context.Context, ann *announcement.Announcement) error {
	model := r.toModel(ann)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("创建公告失败: %w", err)
	}
	ann.ID = model.ID
	return nil
}

// FindByID 根据 ID 查找公告
func (r *AnnouncementRepository) FindByID(ctx context.Context, id uint) (*announcement.Announcement, error) {
	var model models.Announcement
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, announcement.ErrAnnouncementNotFound
		}
		return nil, fmt.Errorf("查找公告失败: %w", err)
	}
	return r.toDomain(&model), nil
}

// Update 更新公告
func (r *AnnouncementRepository) Update(ctx context.Context, ann *announcement.Announcement) error {
	model := r.toModel(ann)
	if err := r.db.WithContext(ctx).Save(model).Error; err != nil {
		return fmt.Errorf("更新公告失败: %w", err)
	}
	return nil
}

// Delete 删除公告
func (r *AnnouncementRepository) Delete(ctx context.Context, id uint) error {
	if err := r.db.WithContext(ctx).Delete(&models.Announcement{}, id).Error; err != nil {
		return fmt.Errorf("删除公告失败: %w", err)
	}
	return nil
}

// ListAll 列出所有公告
func (r *AnnouncementRepository) ListAll(ctx context.Context) ([]*announcement.Announcement, error) {
	var models []*models.Announcement
	if err := r.db.WithContext(ctx).Order("id DESC").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("查询公告列表失败: %w", err)
	}

	announcements := make([]*announcement.Announcement, len(models))
	for i, model := range models {
		announcements[i] = r.toDomain(model)
	}

	return announcements, nil
}

// ListEnabled 列出启用的公告
func (r *AnnouncementRepository) ListEnabled(ctx context.Context) ([]*announcement.Announcement, error) {
	now := time.Now()
	var models []*models.Announcement

	if err := r.db.WithContext(ctx).
		Where("is_enabled = ?", true).
		Where("(start_time IS NULL OR start_time <= ?)", now).
		Where("(end_time IS NULL OR end_time >= ?)", now).
		Order("id DESC").
		Find(&models).Error; err != nil {
		return nil, fmt.Errorf("查询启用的公告列表失败: %w", err)
	}

	announcements := make([]*announcement.Announcement, len(models))
	for i, model := range models {
		announcements[i] = r.toDomain(model)
	}

	return announcements, nil
}

// toModel 转换为数据库模型
func (r *AnnouncementRepository) toModel(ann *announcement.Announcement) *models.Announcement {
	return &models.Announcement{
		ID:        ann.ID,
		Title:     ann.Title,
		Content:   ann.Content,
		Type:      string(ann.Type),
		IsEnabled: ann.IsEnabled,
		StartTime: ann.StartTime,
		EndTime:   ann.EndTime,
		CreatedAt: ann.CreatedAt,
		UpdatedAt: ann.UpdatedAt,
	}
}

// toDomain 转换为领域对象
func (r *AnnouncementRepository) toDomain(model *models.Announcement) *announcement.Announcement {
	return &announcement.Announcement{
		ID:        model.ID,
		Title:     model.Title,
		Content:   model.Content,
		Type:      announcement.AnnouncementType(model.Type),
		IsEnabled: model.IsEnabled,
		StartTime: model.StartTime,
		EndTime:   model.EndTime,
		CreatedAt: model.CreatedAt,
		UpdatedAt: model.UpdatedAt,
	}
}
