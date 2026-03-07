package services

import (
	"fmt"
	"strings"
	"time"

	"nodepass-license-center/internal/models"
	"nodepass-license-center/internal/utils"

	"gorm.io/gorm"
)

// LicenseService 授权服务。
type LicenseService struct {
	db *gorm.DB
}

// NewLicenseService 创建授权服务。
func NewLicenseService(db *gorm.DB) *LicenseService {
	return &LicenseService{db: db}
}

// VerifyVersionInfo 版本信息。
type VerifyVersionInfo struct {
	Panel      string `json:"panel"`
	Backend    string `json:"backend"`
	Frontend   string `json:"frontend"`
	Nodeclient string `json:"nodeclient"`
}

// VerifyRequest 授权校验请求。
type VerifyRequest struct {
	LicenseKey  string            `json:"license_key"`
	MachineID   string            `json:"machine_id"`
	MachineName string            `json:"machine_name"`
	Action      string            `json:"action"`
	Versions    VerifyVersionInfo `json:"versions"`
	Branch      string            `json:"branch"`
	Commit      string            `json:"commit"`
}

// VersionPolicy 授权版本策略。
type VersionPolicy struct {
	MinPanelVersion      string `json:"min_panel_version"`
	MaxPanelVersion      string `json:"max_panel_version"`
	MinBackendVersion    string `json:"min_backend_version"`
	MaxBackendVersion    string `json:"max_backend_version"`
	MinFrontendVersion   string `json:"min_frontend_version"`
	MaxFrontendVersion   string `json:"max_frontend_version"`
	MinNodeclientVersion string `json:"min_nodeclient_version"`
	MaxNodeclientVersion string `json:"max_nodeclient_version"`
}

// VerifyResult 授权校验结果。
type VerifyResult struct {
	Valid         bool          `json:"valid"`
	Message       string        `json:"message"`
	LicenseID     uint          `json:"license_id,omitempty"`
	Plan          string        `json:"plan,omitempty"`
	Customer      string        `json:"customer,omitempty"`
	ExpiresAt     *time.Time    `json:"expires_at,omitempty"`
	VersionPolicy VersionPolicy `json:"version_policy,omitempty"`
}

// GenerateLicenseRequest 批量生成授权码请求。
type GenerateLicenseRequest struct {
	PlanID      uint       `json:"plan_id"`
	Customer    string     `json:"customer"`
	Count       int        `json:"count"`
	ExpiresAt   *time.Time `json:"expires_at"`
	MaxMachines *int       `json:"max_machines"`
	Note        string     `json:"note"`
	Prefix      string     `json:"prefix"`
	CreatedBy   uint       `json:"created_by"`
}

// LicenseFilter 授权码查询过滤器。
type LicenseFilter struct {
	Status   string `json:"status" form:"status"`
	Customer string `json:"customer" form:"customer"`
	PlanID   uint   `json:"plan_id" form:"plan_id"`
	Page     int    `json:"page" form:"page"`
	PageSize int    `json:"page_size" form:"page_size"`
}

// PaginatedResult 分页返回结构。
type PaginatedResult[T any] struct {
	Items    []T   `json:"items"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}

// CreatePlanRequest 创建套餐请求。
type CreatePlanRequest struct {
	Name                 string `json:"name"`
	Code                 string `json:"code"`
	Description          string `json:"description"`
	IsEnabled            bool   `json:"is_enabled"`
	MaxMachines          int    `json:"max_machines"`
	DurationDays         int    `json:"duration_days"`
	MinPanelVersion      string `json:"min_panel_version"`
	MaxPanelVersion      string `json:"max_panel_version"`
	MinBackendVersion    string `json:"min_backend_version"`
	MaxBackendVersion    string `json:"max_backend_version"`
	MinFrontendVersion   string `json:"min_frontend_version"`
	MaxFrontendVersion   string `json:"max_frontend_version"`
	MinNodeclientVersion string `json:"min_nodeclient_version"`
	MaxNodeclientVersion string `json:"max_nodeclient_version"`
}

// Verify 验证授权。
func (s *LicenseService) Verify(req *VerifyRequest, ip, ua string) (*VerifyResult, error) {
	if req == nil {
		return nil, fmt.Errorf("请求不能为空")
	}
	req.LicenseKey = strings.TrimSpace(req.LicenseKey)
	req.MachineID = strings.TrimSpace(req.MachineID)
	req.Action = strings.TrimSpace(req.Action)
	if req.Action == "" {
		req.Action = "install"
	}
	if req.LicenseKey == "" || req.MachineID == "" {
		_ = s.logVerify(nil, req, ip, ua, "failed", "license_key/machine_id 不能为空")
		return &VerifyResult{Valid: false, Message: "license_key/machine_id 不能为空"}, nil
	}

	var license models.LicenseKey
	if err := s.db.Preload("Plan").Where("key = ?", req.LicenseKey).First(&license).Error; err != nil {
		_ = s.logVerify(nil, req, ip, ua, "failed", "授权码不存在")
		return &VerifyResult{Valid: false, Message: "授权码不存在"}, nil
	}

	if license.Status != "active" {
		_ = s.logVerify(&license, req, ip, ua, "failed", "授权码状态不可用")
		return &VerifyResult{Valid: false, Message: "授权码状态不可用"}, nil
	}
	if license.ExpiresAt != nil && license.ExpiresAt.Before(time.Now().UTC()) {
		_ = s.logVerify(&license, req, ip, ua, "failed", "授权码已过期")
		return &VerifyResult{Valid: false, Message: "授权码已过期"}, nil
	}
	if !license.Plan.IsEnabled {
		_ = s.logVerify(&license, req, ip, ua, "failed", "套餐已禁用")
		return &VerifyResult{Valid: false, Message: "套餐已禁用"}, nil
	}

	policy := planToVersionPolicy(license.Plan)
	if err := checkAllVersionRange(req.Versions, policy); err != nil {
		_ = s.logVerify(&license, req, ip, ua, "failed", err.Error())
		return &VerifyResult{Valid: false, Message: err.Error(), VersionPolicy: policy}, nil
	}

	if err := s.ensureMachineBinding(&license, req, ip); err != nil {
		_ = s.logVerify(&license, req, ip, ua, "failed", err.Error())
		return &VerifyResult{Valid: false, Message: err.Error(), VersionPolicy: policy}, nil
	}

	_ = s.logVerify(&license, req, ip, ua, "success", "ok")
	return &VerifyResult{
		Valid:         true,
		Message:       "ok",
		LicenseID:     license.ID,
		Plan:          license.Plan.Code,
		Customer:      license.Customer,
		ExpiresAt:     license.ExpiresAt,
		VersionPolicy: policy,
	}, nil
}

func (s *LicenseService) ensureMachineBinding(license *models.LicenseKey, req *VerifyRequest, ip string) error {
	if license == nil {
		return fmt.Errorf("授权码不存在")
	}

	var activation models.LicenseActivation
	err := s.db.Where("license_id = ? AND machine_id = ?", license.ID, req.MachineID).First(&activation).Error
	now := time.Now().UTC()
	if err == nil {
		updates := map[string]interface{}{
			"machine_name":     req.MachineName,
			"ip_address":       ip,
			"last_verified_at": now,
			"verify_count":     gorm.Expr("verify_count + 1"),
			"is_active":        true,
		}
		return s.db.Model(&models.LicenseActivation{}).Where("id = ?", activation.ID).Updates(updates).Error
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return fmt.Errorf("查询机器绑定失败")
	}

	var activeCount int64
	if err := s.db.Model(&models.LicenseActivation{}).
		Where("license_id = ? AND is_active = ?", license.ID, true).
		Count(&activeCount).Error; err != nil {
		return fmt.Errorf("查询机器数量失败")
	}

	maxMachines := license.Plan.MaxMachines
	if license.MaxMachines != nil && *license.MaxMachines > 0 {
		maxMachines = *license.MaxMachines
	}
	if maxMachines > 0 && int(activeCount) >= maxMachines {
		return fmt.Errorf("超出机器绑定上限")
	}

	newActivation := &models.LicenseActivation{
		LicenseID:       license.ID,
		MachineID:       req.MachineID,
		MachineName:     req.MachineName,
		IPAddress:       ip,
		FirstVerifiedAt: now,
		LastVerifiedAt:  now,
		VerifyCount:     1,
		IsActive:        true,
	}
	return s.db.Create(newActivation).Error
}

func checkAllVersionRange(v VerifyVersionInfo, policy VersionPolicy) error {
	if err := utils.CheckVersionRange(v.Panel, policy.MinPanelVersion, policy.MaxPanelVersion); err != nil {
		return fmt.Errorf("panel %w", err)
	}
	if err := utils.CheckVersionRange(v.Backend, policy.MinBackendVersion, policy.MaxBackendVersion); err != nil {
		return fmt.Errorf("backend %w", err)
	}
	if err := utils.CheckVersionRange(v.Frontend, policy.MinFrontendVersion, policy.MaxFrontendVersion); err != nil {
		return fmt.Errorf("frontend %w", err)
	}
	if err := utils.CheckVersionRange(v.Nodeclient, policy.MinNodeclientVersion, policy.MaxNodeclientVersion); err != nil {
		return fmt.Errorf("nodeclient %w", err)
	}
	return nil
}

func planToVersionPolicy(plan models.LicensePlan) VersionPolicy {
	return VersionPolicy{
		MinPanelVersion:      plan.MinPanelVersion,
		MaxPanelVersion:      plan.MaxPanelVersion,
		MinBackendVersion:    plan.MinBackendVersion,
		MaxBackendVersion:    plan.MaxBackendVersion,
		MinFrontendVersion:   plan.MinFrontendVersion,
		MaxFrontendVersion:   plan.MaxFrontendVersion,
		MinNodeclientVersion: plan.MinNodeclientVersion,
		MaxNodeclientVersion: plan.MaxNodeclientVersion,
	}
}

func (s *LicenseService) logVerify(license *models.LicenseKey, req *VerifyRequest, ip, ua, result, reason string) error {
	logRecord := &models.VerifyLog{
		LicenseKey:        strings.TrimSpace(req.LicenseKey),
		MachineID:         strings.TrimSpace(req.MachineID),
		Action:            strings.TrimSpace(req.Action),
		Result:            result,
		Reason:            reason,
		PanelVersion:      strings.TrimSpace(req.Versions.Panel),
		BackendVersion:    strings.TrimSpace(req.Versions.Backend),
		FrontendVersion:   strings.TrimSpace(req.Versions.Frontend),
		NodeclientVersion: strings.TrimSpace(req.Versions.Nodeclient),
		Branch:            strings.TrimSpace(req.Branch),
		Commit:            strings.TrimSpace(req.Commit),
		IPAddress:         strings.TrimSpace(ip),
		UserAgent:         strings.TrimSpace(ua),
	}
	if license != nil {
		id := license.ID
		logRecord.LicenseID = &id
	}
	return s.db.Create(logRecord).Error
}

// ListPlans 查询授权套餐。
func (s *LicenseService) ListPlans() ([]models.LicensePlan, error) {
	plans := make([]models.LicensePlan, 0)
	err := s.db.Order("id ASC").Find(&plans).Error
	return plans, err
}

// CreatePlan 创建套餐。
func (s *LicenseService) CreatePlan(req *CreatePlanRequest) (*models.LicensePlan, error) {
	if req == nil {
		return nil, fmt.Errorf("请求不能为空")
	}
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Code) == "" {
		return nil, fmt.Errorf("name/code 不能为空")
	}
	if req.MaxMachines <= 0 {
		req.MaxMachines = 1
	}
	if req.DurationDays <= 0 {
		req.DurationDays = 365
	}

	plan := &models.LicensePlan{
		Name:                 strings.TrimSpace(req.Name),
		Code:                 strings.TrimSpace(req.Code),
		Description:          req.Description,
		IsEnabled:            req.IsEnabled,
		MaxMachines:          req.MaxMachines,
		DurationDays:         req.DurationDays,
		MinPanelVersion:      req.MinPanelVersion,
		MaxPanelVersion:      req.MaxPanelVersion,
		MinBackendVersion:    req.MinBackendVersion,
		MaxBackendVersion:    req.MaxBackendVersion,
		MinFrontendVersion:   req.MinFrontendVersion,
		MaxFrontendVersion:   req.MaxFrontendVersion,
		MinNodeclientVersion: req.MinNodeclientVersion,
		MaxNodeclientVersion: req.MaxNodeclientVersion,
	}
	if err := s.db.Create(plan).Error; err != nil {
		return nil, err
	}
	return plan, nil
}

// UpdatePlan 更新套餐。
func (s *LicenseService) UpdatePlan(id uint, req *CreatePlanRequest) (*models.LicensePlan, error) {
	if id == 0 || req == nil {
		return nil, fmt.Errorf("参数无效")
	}
	var plan models.LicensePlan
	if err := s.db.First(&plan, id).Error; err != nil {
		return nil, err
	}

	updates := map[string]interface{}{
		"name":                   strings.TrimSpace(req.Name),
		"code":                   strings.TrimSpace(req.Code),
		"description":            req.Description,
		"is_enabled":             req.IsEnabled,
		"max_machines":           req.MaxMachines,
		"duration_days":          req.DurationDays,
		"min_panel_version":      req.MinPanelVersion,
		"max_panel_version":      req.MaxPanelVersion,
		"min_backend_version":    req.MinBackendVersion,
		"max_backend_version":    req.MaxBackendVersion,
		"min_frontend_version":   req.MinFrontendVersion,
		"max_frontend_version":   req.MaxFrontendVersion,
		"min_nodeclient_version": req.MinNodeclientVersion,
		"max_nodeclient_version": req.MaxNodeclientVersion,
	}
	if err := s.db.Model(&plan).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.First(&plan, id).Error; err != nil {
		return nil, err
	}
	return &plan, nil
}

// DeletePlan 删除套餐。
func (s *LicenseService) DeletePlan(id uint) error {
	if id == 0 {
		return fmt.Errorf("id 无效")
	}
	var count int64
	if err := s.db.Model(&models.LicenseKey{}).Where("plan_id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("套餐下存在授权码，无法删除")
	}
	return s.db.Delete(&models.LicensePlan{}, id).Error
}

// GenerateLicenses 批量生成授权码。
func (s *LicenseService) GenerateLicenses(req *GenerateLicenseRequest) ([]models.LicenseKey, error) {
	if req == nil {
		return nil, fmt.Errorf("请求不能为空")
	}
	if req.PlanID == 0 {
		return nil, fmt.Errorf("plan_id 不能为空")
	}
	if req.Count <= 0 {
		req.Count = 1
	}
	if req.Count > 200 {
		return nil, fmt.Errorf("单次生成数量不能超过 200")
	}
	if strings.TrimSpace(req.Customer) == "" {
		return nil, fmt.Errorf("customer 不能为空")
	}

	var plan models.LicensePlan
	if err := s.db.First(&plan, req.PlanID).Error; err != nil {
		return nil, fmt.Errorf("套餐不存在")
	}

	items := make([]models.LicenseKey, 0, req.Count)
	for i := 0; i < req.Count; i++ {
		license := models.LicenseKey{
			Key:         utils.GenerateLicenseKey(req.Prefix),
			PlanID:      req.PlanID,
			Customer:    strings.TrimSpace(req.Customer),
			Status:      "active",
			ExpiresAt:   req.ExpiresAt,
			MaxMachines: req.MaxMachines,
			Note:        req.Note,
			CreatedBy:   req.CreatedBy,
		}
		if err := s.db.Create(&license).Error; err != nil {
			return nil, err
		}
		items = append(items, license)
	}
	return items, nil
}

// ListLicenses 查询授权码。
func (s *LicenseService) ListLicenses(filter LicenseFilter) (*PaginatedResult[models.LicenseKey], error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.PageSize > 200 {
		filter.PageSize = 200
	}

	query := s.db.Model(&models.LicenseKey{}).Preload("Plan")
	if strings.TrimSpace(filter.Status) != "" {
		query = query.Where("status = ?", strings.TrimSpace(filter.Status))
	}
	if strings.TrimSpace(filter.Customer) != "" {
		query = query.Where("customer LIKE ?", "%"+strings.TrimSpace(filter.Customer)+"%")
	}
	if filter.PlanID > 0 {
		query = query.Where("plan_id = ?", filter.PlanID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	items := make([]models.LicenseKey, 0)
	offset := (filter.Page - 1) * filter.PageSize
	if err := query.Order("id DESC").Offset(offset).Limit(filter.PageSize).Find(&items).Error; err != nil {
		return nil, err
	}

	return &PaginatedResult[models.LicenseKey]{
		Items:    items,
		Total:    total,
		Page:     filter.Page,
		PageSize: filter.PageSize,
	}, nil
}

// GetLicense 获取授权码详情。
func (s *LicenseService) GetLicense(id uint) (*models.LicenseKey, error) {
	var license models.LicenseKey
	if err := s.db.Preload("Plan").First(&license, id).Error; err != nil {
		return nil, err
	}
	return &license, nil
}

// UpdateLicense 更新授权码信息。
func (s *LicenseService) UpdateLicense(id uint, payload map[string]interface{}) (*models.LicenseKey, error) {
	if id == 0 {
		return nil, fmt.Errorf("id 无效")
	}
	if err := s.db.Model(&models.LicenseKey{}).Where("id = ?", id).Updates(payload).Error; err != nil {
		return nil, err
	}
	return s.GetLicense(id)
}

// RevokeLicense 吊销授权码。
func (s *LicenseService) RevokeLicense(id uint) error {
	if id == 0 {
		return fmt.Errorf("id 无效")
	}
	return s.db.Model(&models.LicenseKey{}).Where("id = ?", id).Update("status", "revoked").Error
}

// RestoreLicense 恢复授权码。
func (s *LicenseService) RestoreLicense(id uint) error {
	if id == 0 {
		return fmt.Errorf("id 无效")
	}
	return s.db.Model(&models.LicenseKey{}).Where("id = ?", id).Update("status", "active").Error
}

// DeleteLicense 删除授权码。
func (s *LicenseService) DeleteLicense(id uint) error {
	if id == 0 {
		return fmt.Errorf("id 无效")
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("license_id = ?", id).Delete(&models.LicenseActivation{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&models.LicenseKey{}, id).Error; err != nil {
			return err
		}
		return nil
	})
}

// ListActivations 查询授权码绑定机器。
func (s *LicenseService) ListActivations(licenseID uint) ([]models.LicenseActivation, error) {
	items := make([]models.LicenseActivation, 0)
	err := s.db.Where("license_id = ?", licenseID).Order("id DESC").Find(&items).Error
	return items, err
}

// UnbindActivation 解绑机器。
func (s *LicenseService) UnbindActivation(licenseID, activationID uint) error {
	if licenseID == 0 || activationID == 0 {
		return fmt.Errorf("参数无效")
	}
	return s.db.Model(&models.LicenseActivation{}).
		Where("id = ? AND license_id = ?", activationID, licenseID).
		Update("is_active", false).Error
}

// ListVerifyLogs 查询验证日志。
func (s *LicenseService) ListVerifyLogs(page, pageSize int) (*PaginatedResult[models.VerifyLog], error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}

	query := s.db.Model(&models.VerifyLog{})
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	items := make([]models.VerifyLog, 0)
	offset := (page - 1) * pageSize
	if err := query.Order("id DESC").Offset(offset).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, err
	}

	return &PaginatedResult[models.VerifyLog]{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// Stats 查询统计数据。
func (s *LicenseService) Stats() (map[string]interface{}, error) {
	stats := map[string]interface{}{}

	var licenseTotal int64
	if err := s.db.Model(&models.LicenseKey{}).Count(&licenseTotal).Error; err != nil {
		return nil, err
	}
	var activeTotal int64
	if err := s.db.Model(&models.LicenseKey{}).Where("status = ?", "active").Count(&activeTotal).Error; err != nil {
		return nil, err
	}
	var activationTotal int64
	if err := s.db.Model(&models.LicenseActivation{}).Where("is_active = ?", true).Count(&activationTotal).Error; err != nil {
		return nil, err
	}
	var verifyToday int64
	start := time.Now().UTC().Truncate(24 * time.Hour)
	if err := s.db.Model(&models.VerifyLog{}).Where("created_at >= ?", start).Count(&verifyToday).Error; err != nil {
		return nil, err
	}

	stats["license_total"] = licenseTotal
	stats["license_active"] = activeTotal
	stats["activation_total"] = activationTotal
	stats["verify_today"] = verifyToday
	return stats, nil
}
