package postgres

import (
	"context"
	"errors"
	"time"
	
	"nodepass-pro/backend/internal/domain/user"
	"nodepass-pro/backend/internal/models"
	
	"gorm.io/gorm"
)

// UserRepository PostgreSQL 用户仓储实现
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository 创建用户仓储
func NewUserRepository(db *gorm.DB) user.Repository {
	return &UserRepository{db: db}
}

// Create 创建用户
func (r *UserRepository) Create(ctx context.Context, u *user.User) error {
	model := toUserModel(u)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return user.ErrEmailExists
		}
		return err
	}
	u.ID = model.ID
	u.CreatedAt = model.CreatedAt
	u.UpdatedAt = model.UpdatedAt
	return nil
}

// FindByID 根据 ID 查找用户
func (r *UserRepository) FindByID(ctx context.Context, id uint) (*user.User, error) {
	var model models.User
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, user.ErrUserNotFound
		}
		return nil, err
	}
	return toUserEntity(&model), nil
}

// FindByEmail 根据邮箱查找用户
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*user.User, error) {
	var model models.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, user.ErrUserNotFound
		}
		return nil, err
	}
	return toUserEntity(&model), nil
}

// FindByUsername 根据用户名查找用户
func (r *UserRepository) FindByUsername(ctx context.Context, username string) (*user.User, error) {
	var model models.User
	if err := r.db.WithContext(ctx).Where("username = ?", username).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, user.ErrUserNotFound
		}
		return nil, err
	}
	return toUserEntity(&model), nil
}

// Update 更新用户
func (r *UserRepository) Update(ctx context.Context, u *user.User) error {
	model := toUserModel(u)
	return r.db.WithContext(ctx).Save(model).Error
}

// Delete 删除用户
func (r *UserRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.User{}, id).Error
}

// FindByIDs 批量查找用户
func (r *UserRepository) FindByIDs(ctx context.Context, ids []uint) ([]*user.User, error) {
	var userModels []models.User
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&userModels).Error; err != nil {
		return nil, err
	}
	
	users := make([]*user.User, len(userModels))
	for i, m := range userModels {
		users[i] = toUserEntity(&m)
	}
	return users, nil
}

// List 列表查询
func (r *UserRepository) List(ctx context.Context, filter user.ListFilter) ([]*user.User, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.User{})
	
	// 过滤条件
	if filter.Role != "" {
		query = query.Where("role = ?", filter.Role)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Keyword != "" {
		keyword := "%" + filter.Keyword + "%"
		query = query.Where("username LIKE ? OR email LIKE ?", keyword, keyword)
	}
	
	// 总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	
	// 分页
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	offset := (filter.Page - 1) * filter.PageSize
	
	var userModels []models.User
	if err := query.Offset(offset).Limit(filter.PageSize).Order("id DESC").Find(&userModels).Error; err != nil {
		return nil, 0, err
	}
	
	users := make([]*user.User, len(userModels))
	for i, m := range userModels {
		users[i] = toUserEntity(&m)
	}
	
	return users, total, nil
}

// FindActiveUsers 查找活跃用户
func (r *UserRepository) FindActiveUsers(ctx context.Context, limit int) ([]*user.User, error) {
	var userModels []models.User
	if err := r.db.WithContext(ctx).
		Where("status = ?", "normal").
		Order("last_login_at DESC").
		Limit(limit).
		Find(&userModels).Error; err != nil {
		return nil, err
	}
	
	users := make([]*user.User, len(userModels))
	for i, m := range userModels {
		users[i] = toUserEntity(&m)
	}
	return users, nil
}

// CountByRole 按角色统计
func (r *UserRepository) CountByRole(ctx context.Context, role string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.User{}).Where("role = ?", role).Count(&count).Error
	return count, err
}

// UpdateLastLogin 更新最后登录时间
func (r *UserRepository) UpdateLastLogin(ctx context.Context, userID uint, loginTime time.Time) error {
	return r.db.WithContext(ctx).Model(&models.User{}).
		Where("id = ?", userID).
		Update("last_login_at", loginTime).Error
}

// toUserEntity 模型转实体
func toUserEntity(m *models.User) *user.User {
	return &user.User{
		ID:                      m.ID,
		Username:                m.Username,
		Email:                   m.Email,
		PasswordHash:            m.PasswordHash,
		Role:                    m.Role,
		Status:                  m.Status,
		VipLevel:                m.VipLevel,
		VipExpiresAt:            m.VipExpiresAt,
		TrafficQuota:            m.TrafficQuota,
		TrafficUsed:             m.TrafficUsed,
		MaxRules:                m.MaxRules,
		MaxBandwidth:            m.MaxBandwidth,
		MaxSelfHostedEntryNodes: m.MaxSelfHostedEntryNodes,
		MaxSelfHostedExitNodes:  m.MaxSelfHostedExitNodes,
		TelegramID:              m.TelegramID,
		TelegramUsername:        m.TelegramUsername,
		CreatedAt:               m.CreatedAt,
		UpdatedAt:               m.UpdatedAt,
		LastLoginAt:             m.LastLoginAt,
	}
}

// toUserModel 实体转模型
func toUserModel(u *user.User) *models.User {
	return &models.User{
		ID:                      u.ID,
		Username:                u.Username,
		Email:                   u.Email,
		PasswordHash:            u.PasswordHash,
		Role:                    u.Role,
		Status:                  u.Status,
		VipLevel:                u.VipLevel,
		VipExpiresAt:            u.VipExpiresAt,
		TrafficQuota:            u.TrafficQuota,
		TrafficUsed:             u.TrafficUsed,
		MaxRules:                u.MaxRules,
		MaxBandwidth:            u.MaxBandwidth,
		MaxSelfHostedEntryNodes: u.MaxSelfHostedEntryNodes,
		MaxSelfHostedExitNodes:  u.MaxSelfHostedExitNodes,
		TelegramID:              u.TelegramID,
		TelegramUsername:        u.TelegramUsername,
		CreatedAt:               u.CreatedAt,
		UpdatedAt:               u.UpdatedAt,
		LastLoginAt:             u.LastLoginAt,
	}
}
