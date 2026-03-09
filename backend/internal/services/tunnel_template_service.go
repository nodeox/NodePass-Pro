package services

import (
	"errors"
	"fmt"
	"strings"

	"nodepass-pro/backend/internal/models"

	"gorm.io/gorm"
)

// TunnelTemplateService 隧道模板服务。
type TunnelTemplateService struct {
	db *gorm.DB
}

// NewTunnelTemplateService 创建隧道模板服务。
func NewTunnelTemplateService(db *gorm.DB) *TunnelTemplateService {
	return &TunnelTemplateService{db: db}
}

// CreateTemplateRequest 创建模板请求。
type CreateTemplateRequest struct {
	Name        string                          `json:"name" binding:"required"`
	Description *string                         `json:"description"`
	Protocol    string                          `json:"protocol" binding:"required"`
	Config      *models.TunnelTemplateConfig    `json:"config" binding:"required"`
	IsPublic    bool                            `json:"is_public"`
}

// UpdateTemplateRequest 更新模板请求。
type UpdateTemplateRequest struct {
	Name        *string                         `json:"name"`
	Description *string                         `json:"description"`
	Config      *models.TunnelTemplateConfig    `json:"config"`
	IsPublic    *bool                           `json:"is_public"`
}

// ListTemplateParams 模板列表参数。
type ListTemplateParams struct {
	Protocol *string `json:"protocol"`
	IsPublic *bool   `json:"is_public"`
	Page     int     `json:"page"`
	PageSize int     `json:"page_size"`
}

// Create 创建模板。
func (s *TunnelTemplateService) Create(userID uint, req *CreateTemplateRequest) (*models.TunnelTemplate, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("tunnel template service 未初始化")
	}
	if req == nil {
		return nil, fmt.Errorf("%w: 请求体不能为空", ErrInvalidParams)
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("%w: name 不能为空", ErrInvalidParams)
	}
	if len(name) > 100 {
		return nil, fmt.Errorf("%w: name 长度不能超过 100", ErrInvalidParams)
	}

	protocol, err := tunnelNormalizeProtocol(req.Protocol)
	if err != nil {
		return nil, err
	}

	if req.Config == nil {
		return nil, fmt.Errorf("%w: config 不能为空", ErrInvalidParams)
	}

	template := &models.TunnelTemplate{
		UserID:   userID,
		Name:     name,
		Description: req.Description,
		Protocol: protocol,
		IsPublic: req.IsPublic,
	}

	if err = template.SetConfig(req.Config); err != nil {
		return nil, fmt.Errorf("设置模板配置失败: %w", err)
	}

	if err = s.db.Create(template).Error; err != nil {
		return nil, fmt.Errorf("创建模板失败: %w", err)
	}

	return s.Get(userID, template.ID)
}

// List 列出模板。
func (s *TunnelTemplateService) List(userID uint, params *ListTemplateParams) ([]models.TunnelTemplate, int64, error) {
	if s == nil || s.db == nil {
		return nil, 0, fmt.Errorf("tunnel template service 未初始化")
	}

	page := 1
	pageSize := 20
	if params != nil {
		if params.Page > 0 {
			page = params.Page
		}
		if params.PageSize > 0 {
			pageSize = params.PageSize
		}
	}
	if pageSize > 200 {
		pageSize = 200
	}

	query := s.db.Model(&models.TunnelTemplate{})

	// 用户只能看到自己的模板和公开模板
	if userID > 0 {
		query = query.Where("user_id = ? OR is_public = ?", userID, true)
	}

	if params != nil {
		if params.Protocol != nil {
			protocol := strings.TrimSpace(*params.Protocol)
			if protocol != "" {
				query = query.Where("protocol = ?", protocol)
			}
		}
		if params.IsPublic != nil {
			query = query.Where("is_public = ?", *params.IsPublic)
		}
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询模板总数失败: %w", err)
	}

	list := make([]models.TunnelTemplate, 0, pageSize)
	if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, fmt.Errorf("查询模板列表失败: %w", err)
	}

	return list, total, nil
}

// Get 获取模板详情。
func (s *TunnelTemplateService) Get(userID uint, id uint) (*models.TunnelTemplate, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("tunnel template service 未初始化")
	}
	if id == 0 {
		return nil, fmt.Errorf("%w: 模板 ID 无效", ErrInvalidParams)
	}

	var template models.TunnelTemplate
	query := s.db.Model(&models.TunnelTemplate{})

	// 用户只能访问自己的模板或公开模板
	if userID > 0 {
		query = query.Where("(user_id = ? OR is_public = ?)", userID, true)
	}

	if err := query.First(&template, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: 模板不存在", ErrNotFound)
		}
		return nil, fmt.Errorf("查询模板失败: %w", err)
	}

	return &template, nil
}

// Update 更新模板。
func (s *TunnelTemplateService) Update(userID uint, id uint, req *UpdateTemplateRequest) (*models.TunnelTemplate, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: 请求体不能为空", ErrInvalidParams)
	}

	template, err := s.Get(userID, id)
	if err != nil {
		return nil, err
	}

	// 只能更新自己的模板
	if template.UserID != userID {
		return nil, fmt.Errorf("%w: 无权修改此模板", ErrForbidden)
	}

	updates := make(map[string]interface{})

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, fmt.Errorf("%w: name 不能为空", ErrInvalidParams)
		}
		if len(name) > 100 {
			return nil, fmt.Errorf("%w: name 长度不能超过 100", ErrInvalidParams)
		}
		updates["name"] = name
	}

	if req.Description != nil {
		updates["description"] = *req.Description
	}

	if req.IsPublic != nil {
		updates["is_public"] = *req.IsPublic
	}

	if req.Config != nil {
		if err = template.SetConfig(req.Config); err != nil {
			return nil, fmt.Errorf("设置模板配置失败: %w", err)
		}
		updates["config_json"] = template.ConfigJSON
	}

	if len(updates) > 0 {
		if err = s.db.Model(&models.TunnelTemplate{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			return nil, fmt.Errorf("更新模板失败: %w", err)
		}
	}

	return s.Get(userID, id)
}

// Delete 删除模板。
func (s *TunnelTemplateService) Delete(userID uint, id uint) error {
	template, err := s.Get(userID, id)
	if err != nil {
		return err
	}

	// 只能删除自己的模板
	if template.UserID != userID {
		return fmt.Errorf("%w: 无权删除此模板", ErrForbidden)
	}

	if err = s.db.Delete(&models.TunnelTemplate{}, id).Error; err != nil {
		return fmt.Errorf("删除模板失败: %w", err)
	}
	return nil
}

// IncrementUsage 增加模板使用次数。
func (s *TunnelTemplateService) IncrementUsage(id uint) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("tunnel template service 未初始化")
	}

	if err := s.db.Model(&models.TunnelTemplate{}).Where("id = ?", id).
		Update("usage_count", gorm.Expr("usage_count + 1")).Error; err != nil {
		return fmt.Errorf("更新模板使用次数失败: %w", err)
	}
	return nil
}
