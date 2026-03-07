package services

import (
	"fmt"
	"strings"
	"time"

	"nodepass-pro/backend/internal/models"

	"gorm.io/gorm"
)

// AuditService 审计日志服务。
type AuditService struct {
	db *gorm.DB
}

// AuditListFilters 审计日志查询过滤条件。
type AuditListFilters struct {
	UserID       *uint
	Action       string
	ResourceType string
	StartTime    *time.Time
	EndTime      *time.Time
	Page         int
	PageSize     int
}

// AuditListResult 审计日志分页结果。
type AuditListResult struct {
	List     []models.AuditLog `json:"list"`
	Total    int64             `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"page_size"`
}

// NewAuditService 创建审计服务实例。
func NewAuditService(db *gorm.DB) *AuditService {
	return &AuditService{db: db}
}

// List 查询审计日志。
func (s *AuditService) List(filters AuditListFilters) (*AuditListResult, error) {
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

	query := s.db.Model(&models.AuditLog{}).
		Preload("User")

	if filters.UserID != nil {
		query = query.Where("user_id = ?", *filters.UserID)
	}
	if action := strings.TrimSpace(filters.Action); action != "" {
		query = query.Where("action = ?", action)
	}
	if resourceType := strings.TrimSpace(filters.ResourceType); resourceType != "" {
		query = query.Where("resource_type = ?", resourceType)
	}
	if filters.StartTime != nil {
		query = query.Where("created_at >= ?", *filters.StartTime)
	}
	if filters.EndTime != nil {
		query = query.Where("created_at <= ?", *filters.EndTime)
	}

	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("查询审计日志总数失败: %w", err)
	}

	list := make([]models.AuditLog, 0, pageSize)
	if err := query.Order("id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&list).Error; err != nil {
		return nil, fmt.Errorf("查询审计日志失败: %w", err)
	}

	return &AuditListResult{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// Create 创建审计日志。
func (s *AuditService) Create(log *models.AuditLog) error {
	if log == nil {
		return fmt.Errorf("%w: 审计日志不能为空", ErrInvalidParams)
	}
	log.Action = strings.TrimSpace(log.Action)
	if log.Action == "" {
		return fmt.Errorf("%w: action 不能为空", ErrInvalidParams)
	}
	if err := s.db.Create(log).Error; err != nil {
		return fmt.Errorf("写入审计日志失败: %w", err)
	}
	return nil
}
