package services

import (
	"fmt"
	"strings"
	"time"

	"nodepass-license-center/internal/models"
	"nodepass-license-center/internal/utils"

	"gorm.io/gorm"
)

// LicenseTemplateService 授权码模板服务
type LicenseTemplateService struct {
	db *gorm.DB
}

// NewLicenseTemplateService 创建模板服务
func NewLicenseTemplateService(db *gorm.DB) *LicenseTemplateService {
	return &LicenseTemplateService{db: db}
}

// CreateTemplateRequest 创建模板请求
type CreateTemplateRequest struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	PlanID       uint   `json:"plan_id"`
	DurationDays *int   `json:"duration_days"`
	MaxMachines  *int   `json:"max_machines"`
	MaxDomains   *int   `json:"max_domains"`
	Prefix       string `json:"prefix"`
	Note         string `json:"note"`
}

// GenerateFromTemplateRequest 从模板生成授权码请求
type GenerateFromTemplateRequest struct {
	TemplateID uint       `json:"template_id"`
	Customer   string     `json:"customer"`
	Count      int        `json:"count"`
	ExpiresAt  *time.Time `json:"expires_at"`
	CreatedBy  uint       `json:"created_by"`
}

// ListTemplates 查询模板列表
func (s *LicenseTemplateService) ListTemplates() ([]models.LicenseTemplate, error) {
	templates := make([]models.LicenseTemplate, 0)
	err := s.db.Preload("Plan").Order("id DESC").Find(&templates).Error
	return templates, err
}

// GetTemplate 获取模板详情
func (s *LicenseTemplateService) GetTemplate(id uint) (*models.LicenseTemplate, error) {
	var template models.LicenseTemplate
	if err := s.db.Preload("Plan").First(&template, id).Error; err != nil {
		return nil, err
	}
	return &template, nil
}

// CreateTemplate 创建模板
func (s *LicenseTemplateService) CreateTemplate(req *CreateTemplateRequest, createdBy uint) (*models.LicenseTemplate, error) {
	if req == nil {
		return nil, fmt.Errorf("请求不能为空")
	}
	if strings.TrimSpace(req.Name) == "" {
		return nil, fmt.Errorf("模板名称不能为空")
	}
	if req.PlanID == 0 {
		return nil, fmt.Errorf("套餐ID不能为空")
	}

	// 检查套餐是否存在
	var plan models.LicensePlan
	if err := s.db.First(&plan, req.PlanID).Error; err != nil {
		return nil, fmt.Errorf("套餐不存在")
	}

	template := &models.LicenseTemplate{
		Name:         strings.TrimSpace(req.Name),
		Description:  req.Description,
		PlanID:       req.PlanID,
		DurationDays: req.DurationDays,
		MaxMachines:  req.MaxMachines,
		MaxDomains:   req.MaxDomains,
		Prefix:       strings.TrimSpace(req.Prefix),
		Note:         req.Note,
		IsEnabled:    true,
		CreatedBy:    createdBy,
	}

	if err := s.db.Create(template).Error; err != nil {
		return nil, err
	}
	return template, nil
}

// UpdateTemplate 更新模板
func (s *LicenseTemplateService) UpdateTemplate(id uint, req *CreateTemplateRequest) (*models.LicenseTemplate, error) {
	if id == 0 || req == nil {
		return nil, fmt.Errorf("参数无效")
	}

	var template models.LicenseTemplate
	if err := s.db.First(&template, id).Error; err != nil {
		return nil, err
	}

	updates := map[string]interface{}{
		"name":          strings.TrimSpace(req.Name),
		"description":   req.Description,
		"plan_id":       req.PlanID,
		"duration_days": req.DurationDays,
		"max_machines":  req.MaxMachines,
		"max_domains":   req.MaxDomains,
		"prefix":        strings.TrimSpace(req.Prefix),
		"note":          req.Note,
	}

	if err := s.db.Model(&template).Updates(updates).Error; err != nil {
		return nil, err
	}

	return s.GetTemplate(id)
}

// DeleteTemplate 删除模板
func (s *LicenseTemplateService) DeleteTemplate(id uint) error {
	if id == 0 {
		return fmt.Errorf("id 无效")
	}
	return s.db.Delete(&models.LicenseTemplate{}, id).Error
}

// GenerateFromTemplate 从模板生成授权码
func (s *LicenseTemplateService) GenerateFromTemplate(req *GenerateFromTemplateRequest) ([]models.LicenseKey, error) {
	if req == nil {
		return nil, fmt.Errorf("请求不能为空")
	}
	if req.TemplateID == 0 {
		return nil, fmt.Errorf("模板ID不能为空")
	}
	if strings.TrimSpace(req.Customer) == "" {
		return nil, fmt.Errorf("客户名称不能为空")
	}
	if req.Count <= 0 {
		req.Count = 1
	}
	if req.Count > 200 {
		return nil, fmt.Errorf("单次生成数量不能超过 200")
	}

	// 获取模板
	template, err := s.GetTemplate(req.TemplateID)
	if err != nil {
		return nil, fmt.Errorf("模板不存在")
	}
	if !template.IsEnabled {
		return nil, fmt.Errorf("模板已禁用")
	}

	// 生成授权码
	items := make([]models.LicenseKey, 0, req.Count)
	now := time.Now().UTC()

	for i := 0; i < req.Count; i++ {
		// 计算过期时间
		var expiresAt *time.Time
		if req.ExpiresAt != nil {
			expiresAt = req.ExpiresAt
		} else if template.DurationDays != nil && *template.DurationDays > 0 {
			calculated := now.AddDate(0, 0, *template.DurationDays)
			expiresAt = &calculated
		} else if template.Plan.DurationDays > 0 {
			calculated := now.AddDate(0, 0, template.Plan.DurationDays)
			expiresAt = &calculated
		}

		if expiresAt != nil && !expiresAt.After(now) {
			return nil, fmt.Errorf("过期时间必须晚于当前时间")
		}

		// 确定最大机器数
		var maxMachines *int
		if template.MaxMachines != nil {
			maxMachines = template.MaxMachines
		}

		// 确定最大域名数
		maxDomains := 1
		if template.MaxDomains != nil {
			maxDomains = *template.MaxDomains
		}

		license := models.LicenseKey{
			Key:         utils.GenerateLicenseKey(template.Prefix),
			PlanID:      template.PlanID,
			Customer:    strings.TrimSpace(req.Customer),
			Status:      "active",
			ExpiresAt:   expiresAt,
			MaxMachines: maxMachines,
			MaxDomains:  maxDomains,
			Note:        template.Note,
			CreatedBy:   req.CreatedBy,
		}

		if err := s.db.Create(&license).Error; err != nil {
			return nil, err
		}
		items = append(items, license)
	}

	// 更新模板使用次数
	_ = s.db.Model(&models.LicenseTemplate{}).
		Where("id = ?", req.TemplateID).
		Update("usage_count", gorm.Expr("usage_count + ?", req.Count)).Error

	return items, nil
}

// ToggleTemplate 启用/禁用模板
func (s *LicenseTemplateService) ToggleTemplate(id uint, enabled bool) error {
	if id == 0 {
		return fmt.Errorf("id 无效")
	}
	return s.db.Model(&models.LicenseTemplate{}).
		Where("id = ?", id).
		Update("is_enabled", enabled).Error
}
