package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"nodepass-panel/backend/internal/cache"
	"nodepass-panel/backend/internal/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

var allowedAnnouncementTypes = map[string]struct{}{
	"info":    {},
	"warning": {},
	"error":   {},
	"success": {},
}

// AnnouncementService 公告管理服务。
type AnnouncementService struct {
	db *gorm.DB
}

const (
	announcementCacheEnabledKey = "announcement:list:enabled"
	announcementCacheAllKey     = "announcement:list:all"
)

// AnnouncementCreateRequest 创建公告请求。
type AnnouncementCreateRequest struct {
	Title     string     `json:"title" binding:"required"`
	Content   string     `json:"content" binding:"required"`
	Type      string     `json:"type"`
	IsEnabled *bool      `json:"is_enabled"`
	StartTime *time.Time `json:"start_time"`
	EndTime   *time.Time `json:"end_time"`
}

// AnnouncementUpdateRequest 更新公告请求。
type AnnouncementUpdateRequest struct {
	Title     *string    `json:"title"`
	Content   *string    `json:"content"`
	Type      *string    `json:"type"`
	IsEnabled *bool      `json:"is_enabled"`
	StartTime *time.Time `json:"start_time"`
	EndTime   *time.Time `json:"end_time"`
}

// NewAnnouncementService 创建公告服务实例。
func NewAnnouncementService(db *gorm.DB) *AnnouncementService {
	return &AnnouncementService{db: db}
}

// List 返回公告列表。
func (s *AnnouncementService) List(onlyEnabled bool) ([]models.Announcement, error) {
	cacheKey := announcementCacheAllKey
	if onlyEnabled {
		cacheKey = announcementCacheEnabledKey
	}

	ctx := context.Background()
	if cache.Enabled() {
		cached := make([]models.Announcement, 0)
		hit, err := cache.GetJSON(ctx, cacheKey, &cached)
		if err != nil {
			zap.L().Warn("读取公告缓存失败", zap.Error(err), zap.String("cache_key", cacheKey))
		} else if hit {
			return cached, nil
		}
	}

	query := s.db.Model(&models.Announcement{})
	if onlyEnabled {
		now := time.Now()
		query = query.
			Where("is_enabled = ?", true).
			Where("(start_time IS NULL OR start_time <= ?)", now).
			Where("(end_time IS NULL OR end_time >= ?)", now)
	}

	result := make([]models.Announcement, 0)
	if err := query.Order("id DESC").Find(&result).Error; err != nil {
		return nil, fmt.Errorf("查询公告列表失败: %w", err)
	}

	if cache.Enabled() {
		if err := cache.SetJSON(ctx, cacheKey, result, cache.DefaultTTL()); err != nil {
			zap.L().Warn("写入公告缓存失败", zap.Error(err), zap.String("cache_key", cacheKey))
		}
	}

	return result, nil
}

// Create 创建公告（管理员）。
func (s *AnnouncementService) Create(adminUserID uint, req *AnnouncementCreateRequest) (*models.Announcement, error) {
	if _, err := ensureAdminUser(s.db, adminUserID); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, fmt.Errorf("%w: 请求体不能为空", ErrInvalidParams)
	}

	title := strings.TrimSpace(req.Title)
	content := strings.TrimSpace(req.Content)
	if title == "" || content == "" {
		return nil, fmt.Errorf("%w: title 和 content 不能为空", ErrInvalidParams)
	}

	announcementType := strings.ToLower(strings.TrimSpace(req.Type))
	if announcementType == "" {
		announcementType = "info"
	}
	if _, ok := allowedAnnouncementTypes[announcementType]; !ok {
		return nil, fmt.Errorf("%w: type 不合法", ErrInvalidParams)
	}
	if req.StartTime != nil && req.EndTime != nil && req.EndTime.Before(*req.StartTime) {
		return nil, fmt.Errorf("%w: end_time 不能早于 start_time", ErrInvalidParams)
	}

	isEnabled := true
	if req.IsEnabled != nil {
		isEnabled = *req.IsEnabled
	}

	item := &models.Announcement{
		Title:     title,
		Content:   content,
		Type:      announcementType,
		IsEnabled: isEnabled,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
	}
	if err := s.db.Create(item).Error; err != nil {
		return nil, fmt.Errorf("创建公告失败: %w", err)
	}
	s.invalidateListCache()
	return item, nil
}

// Update 更新公告（管理员）。
func (s *AnnouncementService) Update(adminUserID uint, id uint, req *AnnouncementUpdateRequest) (*models.Announcement, error) {
	if _, err := ensureAdminUser(s.db, adminUserID); err != nil {
		return nil, err
	}
	if id == 0 || req == nil {
		return nil, fmt.Errorf("%w: 参数无效", ErrInvalidParams)
	}

	item, err := s.getByID(id)
	if err != nil {
		return nil, err
	}

	updates := map[string]interface{}{}
	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title == "" {
			return nil, fmt.Errorf("%w: title 不能为空", ErrInvalidParams)
		}
		updates["title"] = title
	}
	if req.Content != nil {
		content := strings.TrimSpace(*req.Content)
		if content == "" {
			return nil, fmt.Errorf("%w: content 不能为空", ErrInvalidParams)
		}
		updates["content"] = content
	}
	if req.Type != nil {
		announcementType := strings.ToLower(strings.TrimSpace(*req.Type))
		if _, ok := allowedAnnouncementTypes[announcementType]; !ok {
			return nil, fmt.Errorf("%w: type 不合法", ErrInvalidParams)
		}
		updates["type"] = announcementType
	}
	if req.IsEnabled != nil {
		updates["is_enabled"] = *req.IsEnabled
	}
	if req.StartTime != nil {
		updates["start_time"] = req.StartTime
	}
	if req.EndTime != nil {
		updates["end_time"] = req.EndTime
	}

	startTime := item.StartTime
	if req.StartTime != nil {
		startTime = req.StartTime
	}
	endTime := item.EndTime
	if req.EndTime != nil {
		endTime = req.EndTime
	}
	if startTime != nil && endTime != nil && endTime.Before(*startTime) {
		return nil, fmt.Errorf("%w: end_time 不能早于 start_time", ErrInvalidParams)
	}

	if len(updates) > 0 {
		if err = s.db.Model(&models.Announcement{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			return nil, fmt.Errorf("更新公告失败: %w", err)
		}
	}

	s.invalidateListCache()
	return s.getByID(id)
}

// Delete 删除公告（管理员）。
func (s *AnnouncementService) Delete(adminUserID uint, id uint) error {
	if _, err := ensureAdminUser(s.db, adminUserID); err != nil {
		return err
	}
	if id == 0 {
		return fmt.Errorf("%w: 公告 ID 无效", ErrInvalidParams)
	}

	item, err := s.getByID(id)
	if err != nil {
		return err
	}
	if err = s.db.Delete(&models.Announcement{}, item.ID).Error; err != nil {
		return fmt.Errorf("删除公告失败: %w", err)
	}
	s.invalidateListCache()
	return nil
}

func (s *AnnouncementService) getByID(id uint) (*models.Announcement, error) {
	var item models.Announcement
	if err := s.db.First(&item, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: 公告不存在", ErrNotFound)
		}
		return nil, fmt.Errorf("查询公告失败: %w", err)
	}
	return &item, nil
}

func (s *AnnouncementService) invalidateListCache() {
	if !cache.Enabled() {
		return
	}
	if err := cache.Delete(context.Background(), announcementCacheEnabledKey, announcementCacheAllKey); err != nil {
		zap.L().Warn("清理公告缓存失败", zap.Error(err))
	}
}
