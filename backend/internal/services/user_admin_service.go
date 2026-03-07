package services

import (
	"errors"
	"fmt"
	"strings"

	"nodepass-pro/backend/internal/models"

	"gorm.io/gorm"
)

// UserAdminService 用户管理服务（管理员）。
type UserAdminService struct {
	db *gorm.DB
}

// UserListFilters 用户列表过滤条件。
type UserListFilters struct {
	Role     string
	Status   string
	VIPLevel *int
	Keyword  string
	Page     int
	PageSize int
}

// UserListResult 用户分页结果。
type UserListResult struct {
	List     []models.User `json:"list"`
	Total    int64         `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"page_size"`
}

// NewUserAdminService 创建用户管理服务。
func NewUserAdminService(db *gorm.DB) *UserAdminService {
	return &UserAdminService{db: db}
}

// ListUsers 查询用户列表。
func (s *UserAdminService) ListUsers(adminUserID uint, filters UserListFilters) (*UserListResult, error) {
	if _, err := ensureAdminUser(s.db, adminUserID); err != nil {
		return nil, err
	}

	page := filters.Page
	if page <= 0 {
		page = 1
	}
	pageSize := filters.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}

	query := s.db.Model(&models.User{})

	if role := normalizeUserRole(filters.Role); role != "" {
		query = query.Where("role = ?", role)
	}
	if status := normalizeUserStatus(filters.Status); status != "" {
		query = query.Where("status = ?", status)
	}
	if filters.VIPLevel != nil {
		query = query.Where("vip_level = ?", *filters.VIPLevel)
	}
	if keyword := strings.TrimSpace(filters.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("username LIKE ? OR email LIKE ?", like, like)
	}

	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("查询用户总数失败: %w", err)
	}

	items := make([]models.User, 0, pageSize)
	if err := query.
		Order("id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&items).Error; err != nil {
		return nil, fmt.Errorf("查询用户列表失败: %w", err)
	}

	return &UserListResult{
		List:     items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// UpdateUserRole 更新用户角色。
func (s *UserAdminService) UpdateUserRole(adminUserID uint, targetUserID uint, role string) (*models.User, error) {
	if _, err := ensureAdminUser(s.db, adminUserID); err != nil {
		return nil, err
	}
	if targetUserID == 0 {
		return nil, fmt.Errorf("%w: 目标用户 ID 无效", ErrInvalidParams)
	}

	normalizedRole := normalizeUserRole(role)
	if normalizedRole == "" {
		return nil, fmt.Errorf("%w: role 仅支持 admin/user", ErrInvalidParams)
	}

	user, err := s.getUserByID(targetUserID)
	if err != nil {
		return nil, err
	}

	if strings.EqualFold(strings.TrimSpace(user.Role), "admin") && normalizedRole == "user" {
		var adminCount int64
		if err := s.db.Model(&models.User{}).Where("role = ?", "admin").Count(&adminCount).Error; err != nil {
			return nil, fmt.Errorf("查询管理员数量失败: %w", err)
		}
		if adminCount <= 1 {
			return nil, fmt.Errorf("%w: 至少保留一个管理员账户", ErrConflict)
		}
	}

	if err := s.db.Model(&models.User{}).
		Where("id = ?", targetUserID).
		Update("role", normalizedRole).Error; err != nil {
		return nil, fmt.Errorf("更新用户角色失败: %w", err)
	}

	return s.getUserByID(targetUserID)
}

// UpdateUserStatus 更新用户状态。
func (s *UserAdminService) UpdateUserStatus(adminUserID uint, targetUserID uint, status string) (*models.User, error) {
	if _, err := ensureAdminUser(s.db, adminUserID); err != nil {
		return nil, err
	}
	if targetUserID == 0 {
		return nil, fmt.Errorf("%w: 目标用户 ID 无效", ErrInvalidParams)
	}

	normalizedStatus := normalizeUserStatus(status)
	if normalizedStatus == "" {
		return nil, fmt.Errorf("%w: status 仅支持 normal/paused/banned/overlimit", ErrInvalidParams)
	}

	if _, err := s.getUserByID(targetUserID); err != nil {
		return nil, err
	}

	if err := s.db.Model(&models.User{}).
		Where("id = ?", targetUserID).
		Update("status", normalizedStatus).Error; err != nil {
		return nil, fmt.Errorf("更新用户状态失败: %w", err)
	}

	return s.getUserByID(targetUserID)
}

func (s *UserAdminService) getUserByID(userID uint) (*models.User, error) {
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: 用户不存在", ErrNotFound)
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	return &user, nil
}

func normalizeUserRole(role string) string {
	role = strings.ToLower(strings.TrimSpace(role))
	switch role {
	case "admin", "user":
		return role
	default:
		return ""
	}
}

func normalizeUserStatus(status string) string {
	status = strings.ToLower(strings.TrimSpace(status))
	switch status {
	case "normal", "paused", "banned", "overlimit":
		return status
	default:
		return ""
	}
}
