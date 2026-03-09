package services

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"nodepass-license-unified/internal/models"

	semver "github.com/Masterminds/semver/v3"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// UnifiedService 统一授权+版本服务。
type UnifiedService struct {
	db                 *gorm.DB
	versionSyncMu      sync.Mutex
	versionSyncRunning map[string]bool
	verifyActivationMu sync.Mutex
	verifyActivation   map[string]*sync.Mutex
}

// NewUnifiedService 创建统一服务。
func NewUnifiedService(db *gorm.DB) *UnifiedService {
	return &UnifiedService{
		db:                 db,
		versionSyncRunning: make(map[string]bool),
		verifyActivation:   make(map[string]*sync.Mutex),
	}
}

// CreatePlanRequest 创建套餐请求。
type CreatePlanRequest struct {
	Code         string `json:"code" binding:"required"`
	Name         string `json:"name" binding:"required"`
	Description  string `json:"description"`
	MaxMachines  int    `json:"max_machines" binding:"min=1"`
	DurationDays int    `json:"duration_days" binding:"min=1"`
	Status       string `json:"status"`
}

// ClonePlanRequest 克隆套餐请求。
type ClonePlanRequest struct {
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Status      string  `json:"status"`
}

// GenerateLicenseRequest 批量生成授权请求。
type GenerateLicenseRequest struct {
	PlanID       uint   `json:"plan_id" binding:"required"`
	Customer     string `json:"customer" binding:"required"`
	Count        int    `json:"count"`
	ExpireDays   int    `json:"expire_days"`
	MaxMachines  *int   `json:"max_machines"`
	MetadataJSON string `json:"metadata_json"`
	Note         string `json:"note"`
}

// LicenseFilter 授权查询过滤。
type LicenseFilter struct {
	Status     string
	Customer   string
	PlanID     uint
	ExpireFrom *time.Time
	ExpireTo   *time.Time
	SortBy     string
	SortOrder  string
	Page       int
	PageSize   int
}

// LicenseListResult 授权分页结果。
type LicenseListResult struct {
	Items    []models.License `json:"items"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
}

// UpdateLicenseRequest 更新授权请求。
type UpdateLicenseRequest struct {
	Key          *string    `json:"key"`
	PlanID       *uint      `json:"plan_id"`
	Customer     *string    `json:"customer"`
	Status       *string    `json:"status"`
	ExpiresAt    *time.Time `json:"expires_at"`
	ClearExpires bool       `json:"clear_expires_at"`
	MaxMachines  *int       `json:"max_machines"`
	ClearMax     bool       `json:"clear_max_machines"`
	MetadataJSON *string    `json:"metadata_json"`
	Note         *string    `json:"note"`
}

// CreateReleaseRequest 创建产品发布记录。
type CreateReleaseRequest struct {
	Product      string     `json:"product" binding:"required"`
	Version      string     `json:"version" binding:"required"`
	Channel      string     `json:"channel"`
	IsMandatory  bool       `json:"is_mandatory"`
	ReleaseNotes string     `json:"release_notes"`
	PublishedAt  *time.Time `json:"published_at"`
	IsActive     *bool      `json:"is_active"`
}

// UpdateReleaseRequest 更新产品发布记录请求。
type UpdateReleaseRequest struct {
	Product      *string    `json:"product"`
	Version      *string    `json:"version"`
	Channel      *string    `json:"channel"`
	IsMandatory  *bool      `json:"is_mandatory"`
	ReleaseNotes *string    `json:"release_notes"`
	PublishedAt  *time.Time `json:"published_at"`
	IsActive     *bool      `json:"is_active"`
}

// ReleasePackageInfo 发布安装包元数据。
type ReleasePackageInfo struct {
	FileName   string
	FilePath   string
	FileSize   int64
	FileSHA256 string
}

// CreateVersionPolicyRequest 创建版本策略请求。
type CreateVersionPolicyRequest struct {
	Product             string `json:"product" binding:"required"`
	Channel             string `json:"channel"`
	MinSupportedVersion string `json:"min_supported_version" binding:"required"`
	RecommendedVersion  string `json:"recommended_version"`
	Message             string `json:"message"`
	IsActive            *bool  `json:"is_active"`
}

// UpdateVersionPolicyRequest 更新版本策略请求。
type UpdateVersionPolicyRequest struct {
	Product             string `json:"product" binding:"required"`
	Channel             string `json:"channel"`
	MinSupportedVersion string `json:"min_supported_version" binding:"required"`
	RecommendedVersion  string `json:"recommended_version"`
	Message             string `json:"message"`
	IsActive            *bool  `json:"is_active"`
}

// VerifyRequest 统一校验请求。
type VerifyRequest struct {
	LicenseKey    string `json:"license_key" binding:"required"`
	MachineID     string `json:"machine_id" binding:"required"`
	Hostname      string `json:"hostname"`
	Product       string `json:"product" binding:"required"`
	ClientVersion string `json:"client_version" binding:"required"`
	Channel       string `json:"channel"`
}

// LicenseCheckResult 授权校验结果。
type LicenseCheckResult struct {
	Valid           bool       `json:"valid"`
	Status          string     `json:"status"`
	Message         string     `json:"message"`
	LicenseID       uint       `json:"license_id,omitempty"`
	PlanCode        string     `json:"plan_code,omitempty"`
	Customer        string     `json:"customer,omitempty"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
	MaxMachines     int        `json:"max_machines,omitempty"`
	CurrentMachines int64      `json:"current_machines,omitempty"`
}

// VersionCheckResult 版本校验结果。
type VersionCheckResult struct {
	Compatible          bool   `json:"compatible"`
	Status              string `json:"status"`
	Message             string `json:"message"`
	Product             string `json:"product"`
	Channel             string `json:"channel"`
	CurrentVersion      string `json:"current_version"`
	LatestVersion       string `json:"latest_version,omitempty"`
	MinSupportedVersion string `json:"min_supported_version,omitempty"`
	RecommendedVersion  string `json:"recommended_version,omitempty"`
	Mandatory           bool   `json:"mandatory"`
}

// VerifyResult 统一校验结果。
type VerifyResult struct {
	Verified   bool               `json:"verified"`
	Status     string             `json:"status"`
	License    LicenseCheckResult `json:"license"`
	Version    VersionCheckResult `json:"version"`
	ServerTime time.Time          `json:"server_time"`
}

// VerifyLogFilter 日志查询过滤。
type VerifyLogFilter struct {
	LicenseKey string
	Status     string
	Product    string
	Page       int
	PageSize   int
}

// VerifyLogListResult 日志分页结果。
type VerifyLogListResult struct {
	Items    []models.VerifyLog `json:"items"`
	Total    int64              `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"page_size"`
}

// DashboardStats 控制台统计。
type DashboardStats struct {
	TotalLicenses        int64   `json:"total_licenses"`
	ActiveLicenses       int64   `json:"active_licenses"`
	ExpiringSoon30Days   int64   `json:"expiring_soon_30_days"`
	TotalActivations     int64   `json:"total_activations"`
	VerifyRequests24h    int64   `json:"verify_requests_24h"`
	VerifySuccessRate24h float64 `json:"verify_success_rate_24h"`
}

// ListPlans 查询套餐。
func (s *UnifiedService) ListPlans() ([]models.LicensePlan, error) {
	var plans []models.LicensePlan
	if err := s.db.Order("id desc").Find(&plans).Error; err != nil {
		return nil, err
	}
	if len(plans) == 0 {
		return plans, nil
	}

	planIDs := make([]uint, 0, len(plans))
	for _, plan := range plans {
		planIDs = append(planIDs, plan.ID)
	}

	type countRow struct {
		PlanID uint
		Count  int64
	}

	licenseCounts := make(map[uint]int64, len(plans))
	var licenseRows []countRow
	if err := s.db.Model(&models.License{}).
		Select("plan_id, COUNT(*) as count").
		Where("plan_id IN ?", planIDs).
		Group("plan_id").
		Scan(&licenseRows).Error; err != nil {
		return nil, err
	}
	for _, row := range licenseRows {
		licenseCounts[row.PlanID] = row.Count
	}

	activeCounts := make(map[uint]int64, len(plans))
	var activeRows []countRow
	if err := s.db.Model(&models.License{}).
		Select("plan_id, COUNT(*) as count").
		Where("plan_id IN ? AND status = ?", planIDs, "active").
		Group("plan_id").
		Scan(&activeRows).Error; err != nil {
		return nil, err
	}
	for _, row := range activeRows {
		activeCounts[row.PlanID] = row.Count
	}

	bindingCounts := make(map[uint]int64, len(plans))
	var bindingRows []countRow
	if err := s.db.Table("license_activations AS la").
		Select("l.plan_id AS plan_id, COUNT(*) AS count").
		Joins("JOIN licenses AS l ON l.id = la.license_id").
		Where("l.plan_id IN ?", planIDs).
		Group("l.plan_id").
		Scan(&bindingRows).Error; err != nil {
		return nil, err
	}
	for _, row := range bindingRows {
		bindingCounts[row.PlanID] = row.Count
	}

	for idx := range plans {
		planID := plans[idx].ID
		plans[idx].LicenseCount = licenseCounts[planID]
		plans[idx].ActiveCount = activeCounts[planID]
		plans[idx].BindingCount = bindingCounts[planID]
	}

	return plans, nil
}

// CreatePlan 创建套餐。
func (s *UnifiedService) CreatePlan(req *CreatePlanRequest) (*models.LicensePlan, error) {
	code := strings.TrimSpace(req.Code)
	if code == "" {
		return nil, errors.New("code 不能为空")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, errors.New("name 不能为空")
	}
	if req.MaxMachines <= 0 {
		return nil, errors.New("max_machines 必须大于 0")
	}
	if req.DurationDays <= 0 {
		return nil, errors.New("duration_days 必须大于 0")
	}

	status, err := normalizePlanStatus(req.Status)
	if err != nil {
		return nil, err
	}

	plan := &models.LicensePlan{
		Code:         code,
		Name:         name,
		Description:  strings.TrimSpace(req.Description),
		MaxMachines:  req.MaxMachines,
		DurationDays: req.DurationDays,
		Status:       status,
	}

	if err := s.db.Create(plan).Error; err != nil {
		return nil, err
	}
	return plan, nil
}

// UpdatePlan 更新套餐。
func (s *UnifiedService) UpdatePlan(id uint, req *CreatePlanRequest) (*models.LicensePlan, error) {
	var plan models.LicensePlan
	if err := s.db.First(&plan, id).Error; err != nil {
		return nil, err
	}

	code := strings.TrimSpace(req.Code)
	if code == "" {
		return nil, errors.New("code 不能为空")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, errors.New("name 不能为空")
	}
	if req.MaxMachines <= 0 {
		return nil, errors.New("max_machines 必须大于 0")
	}
	if req.DurationDays <= 0 {
		return nil, errors.New("duration_days 必须大于 0")
	}

	status, err := normalizePlanStatus(req.Status)
	if err != nil {
		return nil, err
	}

	updates := map[string]interface{}{
		"code":          code,
		"name":          name,
		"description":   strings.TrimSpace(req.Description),
		"max_machines":  req.MaxMachines,
		"duration_days": req.DurationDays,
		"status":        status,
	}
	if err := s.db.Model(&plan).Updates(updates).Error; err != nil {
		return nil, err
	}

	if err := s.db.First(&plan, id).Error; err != nil {
		return nil, err
	}
	return &plan, nil
}

// ClonePlan 克隆套餐。
func (s *UnifiedService) ClonePlan(id uint, req *ClonePlanRequest) (*models.LicensePlan, error) {
	var source models.LicensePlan
	if err := s.db.First(&source, id).Error; err != nil {
		return nil, err
	}

	payload := &ClonePlanRequest{}
	if req != nil {
		payload = req
	}

	code := strings.TrimSpace(payload.Code)
	if code == "" {
		code = buildClonePlanCode(source.Code, time.Now().UTC())
	}
	name := strings.TrimSpace(payload.Name)
	if name == "" {
		name = strings.TrimSpace(source.Name) + " 副本"
	}
	if name == "副本" {
		name = source.Name + " 副本"
	}

	description := strings.TrimSpace(source.Description)
	if payload.Description != nil {
		description = strings.TrimSpace(*payload.Description)
	}

	status, err := normalizePlanStatus(source.Status)
	if err != nil {
		status = "active"
	}
	if strings.TrimSpace(payload.Status) != "" {
		status, err = normalizePlanStatus(payload.Status)
		if err != nil {
			return nil, err
		}
	}

	plan := &models.LicensePlan{
		Code:         code,
		Name:         name,
		Description:  description,
		MaxMachines:  source.MaxMachines,
		DurationDays: source.DurationDays,
		Status:       status,
	}
	if err := s.db.Create(plan).Error; err != nil {
		return nil, err
	}
	return plan, nil
}

// DeletePlan 删除套餐。
func (s *UnifiedService) DeletePlan(id uint) error {
	return s.DeletePlanWithForce(id, false)
}

// DeletePlanWithForce 删除套餐（force=true 时会删除该套餐下所有授权）。
func (s *UnifiedService) DeletePlanWithForce(id uint, force bool) error {
	if id == 0 {
		return errors.New("id 无效")
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		var plan models.LicensePlan
		if err := tx.First(&plan, id).Error; err != nil {
			return err
		}

		var licenseIDs []uint
		if err := tx.Model(&models.License{}).Where("plan_id = ?", id).Pluck("id", &licenseIDs).Error; err != nil {
			return err
		}
		if len(licenseIDs) > 0 && !force {
			return errors.New("该套餐仍有关联授权，不能删除")
		}
		if len(licenseIDs) > 0 {
			if err := cleanupLicenseReferencesByIDsTx(tx, licenseIDs); err != nil {
				return err
			}
			if err := tx.Where("id IN ?", licenseIDs).Delete(&models.License{}).Error; err != nil {
				return err
			}
		}

		result := tx.Delete(&models.LicensePlan{}, id)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

// GenerateLicenses 批量生成授权码。
func (s *UnifiedService) GenerateLicenses(req *GenerateLicenseRequest, createdBy uint) ([]models.License, error) {
	count := req.Count
	if count <= 0 {
		count = 1
	}
	if count > 500 {
		return nil, errors.New("单次最多生成 500 个授权码")
	}

	var plan models.LicensePlan
	if err := s.db.First(&plan, req.PlanID).Error; err != nil {
		return nil, errors.New("套餐不存在")
	}

	licenses := make([]models.License, 0, count)
	for i := 0; i < count; i++ {
		key, err := generateLicenseKey()
		if err != nil {
			return nil, err
		}

		license := models.License{
			Key:          key,
			PlanID:       plan.ID,
			Customer:     strings.TrimSpace(req.Customer),
			Status:       "active",
			MaxMachines:  req.MaxMachines,
			MetadataJSON: strings.TrimSpace(req.MetadataJSON),
			Note:         strings.TrimSpace(req.Note),
			CreatedBy:    createdBy,
		}

		durationDays := req.ExpireDays
		if durationDays <= 0 {
			durationDays = plan.DurationDays
		}
		if durationDays > 0 {
			exp := time.Now().Add(time.Duration(durationDays) * 24 * time.Hour)
			license.ExpiresAt = &exp
		}

		if err := s.db.Create(&license).Error; err != nil {
			return nil, err
		}

		if err := s.db.Preload("Plan").First(&license, license.ID).Error; err != nil {
			return nil, err
		}
		licenses = append(licenses, license)
	}

	return licenses, nil
}

// ListLicenses 查询授权码。
func (s *UnifiedService) ListLicenses(filter LicenseFilter) (*LicenseListResult, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.PageSize > 200 {
		filter.PageSize = 200
	}

	query := s.db.Model(&models.License{}).Preload("Plan")
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Customer != "" {
		query = query.Where("customer LIKE ?", "%"+filter.Customer+"%")
	}
	if filter.PlanID > 0 {
		query = query.Where("plan_id = ?", filter.PlanID)
	}
	if filter.ExpireFrom != nil {
		query = query.Where("expires_at IS NOT NULL AND expires_at >= ?", *filter.ExpireFrom)
	}
	if filter.ExpireTo != nil {
		query = query.Where("expires_at IS NOT NULL AND expires_at <= ?", *filter.ExpireTo)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	orderBy := buildLicenseListOrder(filter.SortBy, filter.SortOrder)

	var items []models.License
	err := query.Order(orderBy).Offset((filter.Page - 1) * filter.PageSize).Limit(filter.PageSize).Find(&items).Error
	if err != nil {
		return nil, err
	}

	return &LicenseListResult{
		Items:    items,
		Total:    total,
		Page:     filter.Page,
		PageSize: filter.PageSize,
	}, nil
}

var licenseListSortColumnMap = map[string]string{
	"created_at": "created_at",
	"expires_at": "expires_at",
	"status":     "status",
}

func buildLicenseListOrder(sortBy, sortOrder string) string {
	column, ok := licenseListSortColumnMap[strings.ToLower(strings.TrimSpace(sortBy))]
	if !ok {
		return "id desc"
	}

	order := "desc"
	if strings.EqualFold(strings.TrimSpace(sortOrder), "asc") {
		order = "asc"
	}
	return fmt.Sprintf("%s %s, id desc", column, order)
}

// GetLicense 获取授权详情。
func (s *UnifiedService) GetLicense(id uint) (*models.License, error) {
	var item models.License
	if err := s.db.Preload("Plan").Preload("Activations").First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

// UpdateLicense 更新授权。
func (s *UnifiedService) UpdateLicense(id uint, req *UpdateLicenseRequest) (*models.License, error) {
	var item models.License
	if err := s.db.First(&item, id).Error; err != nil {
		return nil, err
	}

	updates := map[string]interface{}{}
	if req.Key != nil {
		key := strings.TrimSpace(*req.Key)
		if key == "" {
			return nil, errors.New("key 不能为空")
		}
		updates["key"] = key
	}
	if req.PlanID != nil {
		if *req.PlanID == 0 {
			return nil, errors.New("plan_id 无效")
		}
		var count int64
		if err := s.db.Model(&models.LicensePlan{}).Where("id = ?", *req.PlanID).Count(&count).Error; err != nil {
			return nil, err
		}
		if count == 0 {
			return nil, errors.New("套餐不存在")
		}
		updates["plan_id"] = *req.PlanID
	}
	if req.Customer != nil {
		customer := strings.TrimSpace(*req.Customer)
		if customer == "" {
			return nil, errors.New("customer 不能为空")
		}
		updates["customer"] = customer
	}
	if req.Status != nil {
		status := strings.TrimSpace(strings.ToLower(*req.Status))
		if !isAllowedLicenseStatus(status) {
			return nil, errors.New("status 仅支持 active/revoked/expired")
		}
		updates["status"] = status
	}
	if req.ClearExpires {
		updates["expires_at"] = nil
	}
	if req.ExpiresAt != nil && !req.ClearExpires {
		updates["expires_at"] = req.ExpiresAt
	}
	if req.ClearMax {
		updates["max_machines"] = nil
	}
	if req.MaxMachines != nil && !req.ClearMax {
		if *req.MaxMachines <= 0 {
			return nil, errors.New("max_machines 必须大于 0")
		}
		updates["max_machines"] = req.MaxMachines
	}
	if req.MetadataJSON != nil {
		metadataJSON := strings.TrimSpace(*req.MetadataJSON)
		if metadataJSON != "" && !json.Valid([]byte(metadataJSON)) {
			return nil, errors.New("metadata_json 必须为合法 JSON")
		}
		updates["metadata_json"] = metadataJSON
	}
	if req.Note != nil {
		updates["note"] = strings.TrimSpace(*req.Note)
	}

	if len(updates) == 0 {
		return nil, errors.New("没有可更新字段")
	}

	if err := s.db.Model(&item).Updates(updates).Error; err != nil {
		return nil, err
	}

	return s.GetLicense(id)
}

// RevokeLicense 吊销授权。
func (s *UnifiedService) RevokeLicense(id uint) error {
	return s.db.Model(&models.License{}).Where("id = ?", id).Update("status", "revoked").Error
}

// RestoreLicense 恢复授权。
func (s *UnifiedService) RestoreLicense(id uint) error {
	return s.db.Model(&models.License{}).Where("id = ?", id).Update("status", "active").Error
}

// DeleteLicense 删除授权（物理删除）。
func (s *UnifiedService) DeleteLicense(id uint) error {
	if id == 0 {
		return errors.New("id 无效")
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		var item models.License
		if err := tx.First(&item, id).Error; err != nil {
			return err
		}

		if err := cleanupLicenseReferencesByIDsTx(tx, []uint{id}); err != nil {
			return err
		}

		result := tx.Delete(&models.License{}, id)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

// BatchDeleteLicenses 批量删除授权（物理删除）。
func (s *UnifiedService) BatchDeleteLicenses(licenseIDs []uint) (int64, error) {
	ids := normalizeUniqueUintIDs(licenseIDs)
	if len(ids) == 0 {
		return 0, errors.New("license_ids 不能为空")
	}

	var deletedCount int64
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var existingIDs []uint
		if err := tx.Model(&models.License{}).Where("id IN ?", ids).Pluck("id", &existingIDs).Error; err != nil {
			return err
		}
		if len(existingIDs) == 0 {
			return gorm.ErrRecordNotFound
		}

		if err := cleanupLicenseReferencesByIDsTx(tx, existingIDs); err != nil {
			return err
		}

		deleteTx := tx.Where("id IN ?", existingIDs).Delete(&models.License{})
		if deleteTx.Error != nil {
			return deleteTx.Error
		}
		deletedCount = deleteTx.RowsAffected
		return nil
	})
	if err != nil {
		return 0, err
	}
	return deletedCount, nil
}

func cleanupLicenseReferencesByIDsTx(tx *gorm.DB, licenseIDs []uint) error {
	if len(licenseIDs) == 0 {
		return nil
	}
	if err := tx.Model(&models.VerifyLog{}).Where("license_id IN ?", licenseIDs).Update("license_id", nil).Error; err != nil {
		return err
	}
	if err := tx.Where("license_id IN ?", licenseIDs).Delete(&models.LicenseActivation{}).Error; err != nil {
		return err
	}
	return nil
}

func normalizeUniqueUintIDs(raw []uint) []uint {
	seen := make(map[uint]struct{}, len(raw))
	ids := make([]uint, 0, len(raw))
	for _, id := range raw {
		if id == 0 {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids
}

// ListActivations 查询授权绑定。
func (s *UnifiedService) ListActivations(licenseID uint) ([]models.LicenseActivation, error) {
	var items []models.LicenseActivation
	err := s.db.Where("license_id = ?", licenseID).Order("last_seen_at desc").Find(&items).Error
	return items, err
}

// UnbindActivationByID 解绑单个设备绑定记录。
func (s *UnifiedService) UnbindActivationByID(licenseID uint, activationID uint) error {
	if licenseID == 0 || activationID == 0 {
		return errors.New("参数无效")
	}

	exists, err := s.licenseExists(licenseID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("授权不存在")
	}

	tx := s.db.Where("id = ? AND license_id = ?", activationID, licenseID).Delete(&models.LicenseActivation{})
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return errors.New("绑定记录不存在")
	}
	return nil
}

// ClearActivations 清空授权下所有设备绑定记录。
func (s *UnifiedService) ClearActivations(licenseID uint) (int64, error) {
	return s.ClearActivationsWithOperator(licenseID, 0)
}

// ClearActivationsWithOperator 清空授权下所有设备绑定记录（含审计日志）。
func (s *UnifiedService) ClearActivationsWithOperator(licenseID uint, operatorID uint) (int64, error) {
	if licenseID == 0 {
		return 0, errors.New("参数无效")
	}

	var rowsAffected int64
	err := s.db.Transaction(func(tx *gorm.DB) error {
		exists, existsErr := s.licenseExistsByTx(tx, licenseID)
		if existsErr != nil {
			return existsErr
		}
		if !exists {
			return errors.New("授权不存在")
		}

		deleteTx := tx.Where("license_id = ?", licenseID).Delete(&models.LicenseActivation{})
		if deleteTx.Error != nil {
			return deleteTx.Error
		}
		rowsAffected = deleteTx.RowsAffected

		return createAdminAuditLog(tx, operatorID, AuditActionLicenseClearActivations, "license", map[string]interface{}{
			"license_id":    licenseID,
			"cleared_count": rowsAffected,
		})
	})
	if err != nil {
		return 0, err
	}
	return rowsAffected, nil
}

// ListReleases 查询发布记录。
func (s *UnifiedService) ListReleases(product, channel string) ([]models.ProductRelease, error) {
	query := s.db.Model(&models.ProductRelease{})
	if product != "" {
		query = query.Where("product = ?", strings.TrimSpace(product))
	}
	if channel != "" {
		query = query.Where("channel = ?", strings.TrimSpace(channel))
	}

	var items []models.ProductRelease
	err := query.Order("published_at desc").Order("id desc").Find(&items).Error
	return items, err
}

// ListDeletedReleases 查询回收站中的发布记录。
func (s *UnifiedService) ListDeletedReleases(product, channel string) ([]models.ProductRelease, error) {
	query := s.db.Unscoped().Model(&models.ProductRelease{}).Where("deleted_at IS NOT NULL")
	if product != "" {
		query = query.Where("product = ?", strings.TrimSpace(product))
	}
	if channel != "" {
		query = query.Where("channel = ?", strings.TrimSpace(channel))
	}

	var items []models.ProductRelease
	err := query.Order("deleted_at desc").Order("id desc").Find(&items).Error
	return items, err
}

// CreateRelease 创建发布记录。
func (s *UnifiedService) CreateRelease(req *CreateReleaseRequest) (*models.ProductRelease, error) {
	return s.createRelease(req, nil)
}

// CreateReleaseWithPackage 创建带安装包的发布记录。
func (s *UnifiedService) CreateReleaseWithPackage(req *CreateReleaseRequest, pkg *ReleasePackageInfo) (*models.ProductRelease, error) {
	if pkg == nil {
		return nil, errors.New("package info 不能为空")
	}
	if strings.TrimSpace(pkg.FileName) == "" || strings.TrimSpace(pkg.FilePath) == "" {
		return nil, errors.New("安装包信息不完整")
	}
	if pkg.FileSize <= 0 {
		return nil, errors.New("安装包大小无效")
	}
	return s.createRelease(req, pkg)
}

func (s *UnifiedService) createRelease(req *CreateReleaseRequest, pkg *ReleasePackageInfo) (*models.ProductRelease, error) {
	channel := strings.TrimSpace(req.Channel)
	if channel == "" {
		channel = "stable"
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	release := &models.ProductRelease{
		Product:      strings.TrimSpace(req.Product),
		Version:      strings.TrimSpace(req.Version),
		Channel:      channel,
		IsMandatory:  req.IsMandatory,
		ReleaseNotes: strings.TrimSpace(req.ReleaseNotes),
		PublishedAt:  req.PublishedAt,
		IsActive:     isActive,
	}
	if pkg != nil {
		release.FileName = strings.TrimSpace(pkg.FileName)
		release.FilePath = strings.TrimSpace(pkg.FilePath)
		release.FileSize = pkg.FileSize
		release.FileSHA256 = strings.TrimSpace(strings.ToLower(pkg.FileSHA256))
	}

	if release.PublishedAt == nil {
		now := time.Now()
		release.PublishedAt = &now
	}

	if err := s.db.Create(release).Error; err != nil {
		return nil, err
	}
	return release, nil
}

// UpdateRelease 更新发布记录。
func (s *UnifiedService) UpdateRelease(id uint, req *UpdateReleaseRequest) (*models.ProductRelease, error) {
	return s.UpdateReleaseWithOperator(id, req, 0)
}

// UpdateReleaseWithOperator 更新发布记录（含审计日志）。
func (s *UnifiedService) UpdateReleaseWithOperator(id uint, req *UpdateReleaseRequest, operatorID uint) (*models.ProductRelease, error) {
	if req == nil {
		return nil, errors.New("参数无效")
	}

	updates, err := buildReleaseUpdates(req)
	if err != nil {
		return nil, err
	}
	if len(updates) == 0 {
		return nil, errors.New("没有可更新字段")
	}

	var release models.ProductRelease
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err = tx.First(&release, id).Error; err != nil {
			return err
		}
		if err = tx.Model(&release).Updates(updates).Error; err != nil {
			return err
		}
		if err = createAdminAuditLog(tx, operatorID, AuditActionReleaseUpdate, "release", map[string]interface{}{
			"release_id": id,
			"updates":    updates,
		}); err != nil {
			return err
		}
		return tx.First(&release, id).Error
	})
	if err != nil {
		return nil, err
	}
	return &release, nil
}

// ReplaceReleasePackage 替换发布安装包，返回替换前文件路径。
func (s *UnifiedService) ReplaceReleasePackage(id uint, pkg *ReleasePackageInfo) (*models.ProductRelease, string, error) {
	return s.ReplaceReleasePackageWithOperator(id, pkg, 0)
}

// ReplaceReleasePackageWithOperator 替换发布安装包（含审计日志），返回替换前文件路径。
func (s *UnifiedService) ReplaceReleasePackageWithOperator(id uint, pkg *ReleasePackageInfo, operatorID uint) (*models.ProductRelease, string, error) {
	if pkg == nil {
		return nil, "", errors.New("package info 不能为空")
	}
	if strings.TrimSpace(pkg.FileName) == "" || strings.TrimSpace(pkg.FilePath) == "" {
		return nil, "", errors.New("安装包信息不完整")
	}
	if pkg.FileSize <= 0 {
		return nil, "", errors.New("安装包大小无效")
	}

	var release models.ProductRelease
	oldPath := strings.TrimSpace(release.FilePath)
	oldFileName := strings.TrimSpace(release.FileName)
	oldFileSize := release.FileSize

	updates := map[string]interface{}{
		"file_name":   strings.TrimSpace(pkg.FileName),
		"file_path":   strings.TrimSpace(pkg.FilePath),
		"file_size":   pkg.FileSize,
		"file_sha256": strings.TrimSpace(strings.ToLower(pkg.FileSHA256)),
	}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&release, id).Error; err != nil {
			return err
		}
		oldPath = strings.TrimSpace(release.FilePath)
		oldFileName = strings.TrimSpace(release.FileName)
		oldFileSize = release.FileSize
		if err := tx.Model(&release).Updates(updates).Error; err != nil {
			return err
		}
		if err := createAdminAuditLog(tx, operatorID, AuditActionReleaseReplacePackage, "release", map[string]interface{}{
			"release_id":      id,
			"old_file_name":   oldFileName,
			"old_file_size":   oldFileSize,
			"new_file_name":   updates["file_name"],
			"new_file_size":   updates["file_size"],
			"new_file_sha256": updates["file_sha256"],
		}); err != nil {
			return err
		}
		return tx.First(&release, id).Error
	})
	if err != nil {
		return nil, "", err
	}
	return &release, oldPath, nil
}

// DeleteRelease 删除发布记录并返回安装包路径。
func (s *UnifiedService) DeleteRelease(id uint) (string, error) {
	return s.DeleteReleaseWithOperator(id, 0)
}

// DeleteReleaseWithOperator 删除发布记录并返回安装包路径（含审计日志）。
func (s *UnifiedService) DeleteReleaseWithOperator(id uint, operatorID uint) (string, error) {
	var oldPath string
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var release models.ProductRelease
		if err := tx.First(&release, id).Error; err != nil {
			return err
		}
		oldPath = strings.TrimSpace(release.FilePath)

		if err := tx.Delete(&models.ProductRelease{}, id).Error; err != nil {
			return err
		}
		return createAdminAuditLog(tx, operatorID, AuditActionReleaseDelete, "release", map[string]interface{}{
			"release_id": id,
			"product":    release.Product,
			"version":    release.Version,
			"channel":    release.Channel,
			"file_name":  release.FileName,
			"file_size":  release.FileSize,
		})
	})
	if err != nil {
		return "", err
	}
	return oldPath, nil
}

// RestoreRelease 从回收站恢复发布记录。
func (s *UnifiedService) RestoreRelease(id uint) (*models.ProductRelease, error) {
	return s.RestoreReleaseWithOperator(id, 0)
}

// RestoreReleaseWithOperator 从回收站恢复发布记录（含审计日志）。
func (s *UnifiedService) RestoreReleaseWithOperator(id uint, operatorID uint) (*models.ProductRelease, error) {
	var release models.ProductRelease
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Where("id = ? AND deleted_at IS NOT NULL", id).First(&release).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Model(&models.ProductRelease{}).Where("id = ?", id).Update("deleted_at", nil).Error; err != nil {
			return err
		}
		if err := createAdminAuditLog(tx, operatorID, AuditActionReleaseRestore, "release", map[string]interface{}{
			"release_id": id,
			"product":    release.Product,
			"version":    release.Version,
			"channel":    release.Channel,
		}); err != nil {
			return err
		}
		return tx.First(&release, id).Error
	})
	if err != nil {
		return nil, err
	}
	return &release, nil
}

// PurgeRelease 从回收站永久删除发布记录。
func (s *UnifiedService) PurgeRelease(id uint) (string, error) {
	return s.PurgeReleaseWithOperator(id, 0)
}

// PurgeReleaseWithOperator 从回收站永久删除发布记录（含审计日志），返回安装包路径。
func (s *UnifiedService) PurgeReleaseWithOperator(id uint, operatorID uint) (string, error) {
	var oldPath string
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var release models.ProductRelease
		if err := tx.Unscoped().Where("id = ? AND deleted_at IS NOT NULL", id).First(&release).Error; err != nil {
			return err
		}

		oldPath = strings.TrimSpace(release.FilePath)
		if err := tx.Unscoped().Delete(&models.ProductRelease{}, id).Error; err != nil {
			return err
		}

		return createAdminAuditLog(tx, operatorID, AuditActionReleasePurge, "release", map[string]interface{}{
			"release_id": id,
			"product":    release.Product,
			"version":    release.Version,
			"channel":    release.Channel,
			"file_name":  release.FileName,
			"file_size":  release.FileSize,
		})
	})
	if err != nil {
		return "", err
	}
	return oldPath, nil
}

// GetReleaseByID 查询单条发布记录。
func (s *UnifiedService) GetReleaseByID(id uint) (*models.ProductRelease, error) {
	var release models.ProductRelease
	if err := s.db.First(&release, id).Error; err != nil {
		return nil, err
	}
	return &release, nil
}

func buildReleaseUpdates(req *UpdateReleaseRequest) (map[string]interface{}, error) {
	updates := make(map[string]interface{})
	if req.Product != nil {
		value := strings.TrimSpace(*req.Product)
		if value == "" {
			return nil, errors.New("product 不能为空")
		}
		updates["product"] = value
	}
	if req.Version != nil {
		value := strings.TrimSpace(*req.Version)
		if value == "" {
			return nil, errors.New("version 不能为空")
		}
		updates["version"] = value
	}
	if req.Channel != nil {
		value := strings.TrimSpace(*req.Channel)
		if value == "" {
			return nil, errors.New("channel 不能为空")
		}
		updates["channel"] = value
	}
	if req.IsMandatory != nil {
		updates["is_mandatory"] = *req.IsMandatory
	}
	if req.ReleaseNotes != nil {
		updates["release_notes"] = strings.TrimSpace(*req.ReleaseNotes)
	}
	if req.PublishedAt != nil {
		updates["published_at"] = req.PublishedAt
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	return updates, nil
}

// ListVersionPolicies 查询版本策略。
func (s *UnifiedService) ListVersionPolicies(product, channel string) ([]models.VersionPolicy, error) {
	query := s.db.Model(&models.VersionPolicy{})
	if product != "" {
		query = query.Where("product = ?", strings.TrimSpace(product))
	}
	if channel != "" {
		query = query.Where("channel = ?", strings.TrimSpace(channel))
	}

	var items []models.VersionPolicy
	err := query.Order("id desc").Find(&items).Error
	return items, err
}

// CreateVersionPolicy 创建版本策略。
func (s *UnifiedService) CreateVersionPolicy(req *CreateVersionPolicyRequest) (*models.VersionPolicy, error) {
	if req == nil {
		return nil, errors.New("参数无效")
	}
	product := strings.TrimSpace(req.Product)
	if product == "" {
		return nil, errors.New("product 不能为空")
	}
	channel := strings.TrimSpace(req.Channel)
	if channel == "" {
		channel = "stable"
	}
	minSupportedVersion := strings.TrimSpace(req.MinSupportedVersion)
	if minSupportedVersion == "" {
		return nil, errors.New("min_supported_version 不能为空")
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	var exists models.VersionPolicy
	err := s.db.Where("product = ? AND channel = ?", product, channel).First(&exists).Error
	if err == nil {
		return nil, errors.New("该产品和渠道的版本策略已存在，请编辑现有策略")
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	policy := &models.VersionPolicy{
		Product:             product,
		Channel:             channel,
		MinSupportedVersion: minSupportedVersion,
		RecommendedVersion:  strings.TrimSpace(req.RecommendedVersion),
		Message:             strings.TrimSpace(req.Message),
		IsActive:            isActive,
	}
	if err = s.db.Create(policy).Error; err != nil {
		return nil, err
	}
	return policy, nil
}

// UpdateVersionPolicy 更新版本策略。
func (s *UnifiedService) UpdateVersionPolicy(id uint, req *UpdateVersionPolicyRequest) (*models.VersionPolicy, error) {
	if id == 0 {
		return nil, errors.New("id 无效")
	}
	if req == nil {
		return nil, errors.New("参数无效")
	}

	var policy models.VersionPolicy
	if err := s.db.First(&policy, id).Error; err != nil {
		return nil, err
	}

	product := strings.TrimSpace(req.Product)
	if product == "" {
		return nil, errors.New("product 不能为空")
	}
	channel := strings.TrimSpace(req.Channel)
	if channel == "" {
		channel = "stable"
	}
	minSupportedVersion := strings.TrimSpace(req.MinSupportedVersion)
	if minSupportedVersion == "" {
		return nil, errors.New("min_supported_version 不能为空")
	}

	var duplicate models.VersionPolicy
	err := s.db.Where("product = ? AND channel = ? AND id <> ?", product, channel, id).First(&duplicate).Error
	if err == nil {
		return nil, errors.New("该产品和渠道的版本策略已存在，请修改现有策略")
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	policy.Product = product
	policy.Channel = channel
	policy.MinSupportedVersion = minSupportedVersion
	policy.RecommendedVersion = strings.TrimSpace(req.RecommendedVersion)
	policy.Message = strings.TrimSpace(req.Message)
	if req.IsActive != nil {
		policy.IsActive = *req.IsActive
	}

	if err = s.db.Save(&policy).Error; err != nil {
		return nil, err
	}
	return &policy, nil
}

// DeleteVersionPolicy 删除版本策略。
func (s *UnifiedService) DeleteVersionPolicy(id uint) error {
	if id == 0 {
		return errors.New("id 无效")
	}
	result := s.db.Delete(&models.VersionPolicy{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// Verify 统一校验：一次返回授权状态+版本状态。
func (s *UnifiedService) Verify(req *VerifyRequest, clientIP, userAgent string) (*VerifyResult, error) {
	licenseResult := LicenseCheckResult{
		Valid:   false,
		Status:  "invalid_license",
		Message: "授权码无效",
	}

	versionResult := s.evaluateVersion(req.Product, req.Channel, req.ClientVersion)
	result := &VerifyResult{
		Verified:   false,
		Status:     "invalid_license",
		License:    licenseResult,
		Version:    versionResult,
		ServerTime: time.Now(),
	}

	verificationState, err := s.verifyLicenseAndTouchActivation(req, clientIP, result.ServerTime)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.writeVerifyLog(nil, req, result, clientIP, userAgent, "license_not_found")
			return result, nil
		}
		return nil, err
	}

	license := verificationState.License

	licenseResult = LicenseCheckResult{
		Valid:           true,
		Status:          license.Status,
		Message:         "授权有效",
		LicenseID:       license.ID,
		PlanCode:        license.Plan.Code,
		Customer:        license.Customer,
		ExpiresAt:       license.ExpiresAt,
		MaxMachines:     verificationState.MaxMachines,
		CurrentMachines: verificationState.MachineCount,
	}

	if license.Status != "active" {
		licenseResult.Valid = false
		licenseResult.Status = license.Status
		licenseResult.Message = "授权状态不可用"
		result.License = licenseResult
		result.Version = versionResult
		result.Status = license.Status
		s.writeVerifyLog(&license.ID, req, result, clientIP, userAgent, "license_inactive")
		return result, nil
	}

	if verificationState.LimitExceeded {
		licenseResult.Valid = false
		licenseResult.Status = "machine_limit_exceeded"
		licenseResult.Message = "绑定设备数已达上限"
		result.License = licenseResult
		result.Status = "machine_limit_exceeded"
		s.writeVerifyLog(&license.ID, req, result, clientIP, userAgent, "machine_limit_exceeded")
		return result, nil
	}

	result.License = licenseResult
	result.Version = versionResult

	if !licenseResult.Valid {
		result.Verified = false
		result.Status = licenseResult.Status
		s.writeVerifyLog(&license.ID, req, result, clientIP, userAgent, result.Status)
		return result, nil
	}

	if !versionResult.Compatible {
		result.Verified = false
		result.Status = "upgrade_required"
		s.writeVerifyLog(&license.ID, req, result, clientIP, userAgent, "version_incompatible")
		return result, nil
	}

	result.Verified = true
	result.Status = "ok"
	result.License.Message = "授权有效"
	s.writeVerifyLog(&license.ID, req, result, clientIP, userAgent, "ok")
	return result, nil
}

type verifyLicenseState struct {
	License       models.License
	MachineCount  int64
	MaxMachines   int
	LimitExceeded bool
}

func (s *UnifiedService) verifyLicenseAndTouchActivation(req *VerifyRequest, clientIP string, now time.Time) (*verifyLicenseState, error) {
	licenseKey := strings.TrimSpace(req.LicenseKey)
	machineID := strings.TrimSpace(req.MachineID)
	hostname := strings.TrimSpace(req.Hostname)

	unlock := s.lockVerifyActivation(licenseKey)
	defer unlock()

	const maxLockRetry = 3
	for attempt := 0; attempt <= maxLockRetry; attempt++ {
		state := &verifyLicenseState{}
		err := s.db.Transaction(func(tx *gorm.DB) error {
			var license models.License
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Preload("Plan").
				Where("key = ?", licenseKey).
				First(&license).Error; err != nil {
				return err
			}

			if license.ExpiresAt != nil && now.After(*license.ExpiresAt) && license.Status != "expired" {
				if err := tx.Model(&license).Update("status", "expired").Error; err != nil {
					return err
				}
				license.Status = "expired"
			}

			maxMachines := license.Plan.MaxMachines
			if license.MaxMachines != nil && *license.MaxMachines > 0 {
				maxMachines = *license.MaxMachines
			}

			var machineCount int64
			if err := tx.Model(&models.LicenseActivation{}).Where("license_id = ?", license.ID).Count(&machineCount).Error; err != nil {
				return err
			}

			if license.Status == "active" {
				var activation models.LicenseActivation
				activationErr := tx.Where("license_id = ? AND machine_id = ?", license.ID, machineID).First(&activation).Error
				if activationErr != nil {
					if !errors.Is(activationErr, gorm.ErrRecordNotFound) {
						return activationErr
					}
					if maxMachines > 0 && machineCount >= int64(maxMachines) {
						state.LimitExceeded = true
					} else {
						activation = models.LicenseActivation{
							LicenseID:  license.ID,
							MachineID:  machineID,
							Hostname:   hostname,
							IPAddress:  clientIP,
							LastSeenAt: now,
						}
						if err := tx.Create(&activation).Error; err != nil {
							return err
						}
						machineCount++
					}
				} else {
					updateMap := map[string]interface{}{
						"last_seen_at": now,
						"ip_address":   clientIP,
					}
					if hostname != "" {
						updateMap["hostname"] = hostname
					}
					if err := tx.Model(&activation).Updates(updateMap).Error; err != nil {
						return err
					}
				}
			}

			state.License = license
			state.MachineCount = machineCount
			state.MaxMachines = maxMachines
			return nil
		})
		if err == nil {
			return state, nil
		}
		if !isSQLiteLockError(err) || attempt == maxLockRetry {
			return nil, err
		}
		time.Sleep(time.Duration(attempt+1) * 10 * time.Millisecond)
	}

	return nil, errors.New("授权校验失败")
}

func (s *UnifiedService) lockVerifyActivation(licenseKey string) func() {
	key := strings.TrimSpace(licenseKey)
	if key == "" {
		key = "__empty__"
	}

	s.verifyActivationMu.Lock()
	locker, exists := s.verifyActivation[key]
	if !exists {
		locker = &sync.Mutex{}
		s.verifyActivation[key] = locker
	}
	s.verifyActivationMu.Unlock()

	locker.Lock()
	return func() {
		locker.Unlock()
	}
}

func isSQLiteLockError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "database table is locked") || strings.Contains(msg, "database is locked")
}

func (s *UnifiedService) evaluateVersion(product, channel, currentVersion string) VersionCheckResult {
	trimmedProduct := strings.TrimSpace(product)
	trimmedChannel := strings.TrimSpace(channel)
	if trimmedChannel == "" {
		trimmedChannel = "stable"
	}

	res := VersionCheckResult{
		Compatible:     true,
		Status:         "unknown",
		Message:        "未匹配到版本策略",
		Product:        trimmedProduct,
		Channel:        trimmedChannel,
		CurrentVersion: strings.TrimSpace(currentVersion),
	}

	var latest models.ProductRelease
	latestErr := s.db.Where("product = ? AND channel = ? AND is_active = ?", trimmedProduct, trimmedChannel, true).
		Order("published_at desc").Order("id desc").First(&latest).Error
	if latestErr != nil {
		if !errors.Is(latestErr, gorm.ErrRecordNotFound) {
			res.Compatible = false
			res.Status = "version_check_failed"
			res.Message = "版本检查失败"
			return res
		}
		fallbackErr := s.db.Where("product = ? AND is_active = ?", trimmedProduct, true).
			Order("published_at desc").Order("id desc").First(&latest).Error
		if fallbackErr == nil {
			res.LatestVersion = latest.Version
			res.Mandatory = latest.IsMandatory
		}
	} else {
		res.LatestVersion = latest.Version
		res.Mandatory = latest.IsMandatory
	}

	var policy models.VersionPolicy
	policyErr := s.db.Where("product = ? AND channel = ? AND is_active = ?", trimmedProduct, trimmedChannel, true).
		Order("id desc").First(&policy).Error
	if policyErr != nil {
		if !errors.Is(policyErr, gorm.ErrRecordNotFound) {
			res.Compatible = false
			res.Status = "version_check_failed"
			res.Message = "版本策略查询失败"
			return res
		}
		if trimmedChannel != "stable" {
			fallbackPolicyErr := s.db.Where("product = ? AND channel = ? AND is_active = ?", trimmedProduct, "stable", true).
				Order("id desc").First(&policy).Error
			if fallbackPolicyErr == nil {
				res.MinSupportedVersion = policy.MinSupportedVersion
				res.RecommendedVersion = policy.RecommendedVersion
				if policy.Message != "" {
					res.Message = policy.Message
				}
			}
		}
	} else {
		res.MinSupportedVersion = policy.MinSupportedVersion
		res.RecommendedVersion = policy.RecommendedVersion
		if policy.Message != "" {
			res.Message = policy.Message
		}
	}

	curr, err := parseVersion(res.CurrentVersion)
	if err != nil {
		res.Compatible = false
		res.Status = "invalid_version"
		res.Message = "客户端版本格式无效"
		return res
	}

	if res.MinSupportedVersion != "" {
		minV, minErr := parseVersion(res.MinSupportedVersion)
		if minErr == nil && curr.LessThan(minV) {
			res.Compatible = false
			res.Status = "required_upgrade"
			if policy.Message != "" {
				res.Message = policy.Message
			} else {
				res.Message = fmt.Sprintf("当前版本过低，最低支持版本 %s", res.MinSupportedVersion)
			}
			return res
		}
	}

	if res.LatestVersion != "" {
		latestV, latestErr := parseVersion(res.LatestVersion)
		if latestErr == nil {
			if curr.LessThan(latestV) {
				if res.Mandatory {
					res.Compatible = false
					res.Status = "required_upgrade"
					if res.Message == "" {
						res.Message = fmt.Sprintf("必须升级到 %s", res.LatestVersion)
					}
				} else {
					res.Compatible = true
					res.Status = "upgrade_available"
					if res.Message == "" {
						res.Message = fmt.Sprintf("有新版本可用 %s", res.LatestVersion)
					}
				}
				return res
			}
		}
	}

	res.Status = "up_to_date"
	if res.Message == "" || res.Message == "未匹配到版本策略" {
		res.Message = "版本正常"
	}
	return res
}

func (s *UnifiedService) writeVerifyLog(licenseID *uint, req *VerifyRequest, result *VerifyResult, clientIP, userAgent, reason string) {
	log := models.VerifyLog{
		LicenseID:     licenseID,
		LicenseKey:    strings.TrimSpace(req.LicenseKey),
		MachineID:     strings.TrimSpace(req.MachineID),
		Product:       strings.TrimSpace(req.Product),
		ClientVersion: strings.TrimSpace(req.ClientVersion),
		Verified:      result.Verified,
		Status:        result.Status,
		Reason:        reason,
		ClientIP:      clientIP,
		UserAgent:     userAgent,
		CreatedAt:     time.Now(),
	}
	_ = s.db.Create(&log).Error
}

// ListVerifyLogs 查询校验日志。
func (s *UnifiedService) ListVerifyLogs(filter VerifyLogFilter) (*VerifyLogListResult, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.PageSize > 200 {
		filter.PageSize = 200
	}

	query := s.db.Model(&models.VerifyLog{})
	if filter.LicenseKey != "" {
		query = query.Where("license_key = ?", strings.TrimSpace(filter.LicenseKey))
	}
	if filter.Status != "" {
		query = query.Where("status = ?", strings.TrimSpace(filter.Status))
	}
	if filter.Product != "" {
		query = query.Where("product = ?", strings.TrimSpace(filter.Product))
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	var items []models.VerifyLog
	if err := query.Order("id desc").Offset((filter.Page - 1) * filter.PageSize).Limit(filter.PageSize).Find(&items).Error; err != nil {
		return nil, err
	}

	return &VerifyLogListResult{
		Items:    items,
		Total:    total,
		Page:     filter.Page,
		PageSize: filter.PageSize,
	}, nil
}

// GetDashboardStats 查询控制台统计。
func (s *UnifiedService) GetDashboardStats() (*DashboardStats, error) {
	stats := &DashboardStats{}
	now := time.Now()
	soon := now.Add(30 * 24 * time.Hour)
	windowStart := now.Add(-24 * time.Hour)

	if err := s.db.Model(&models.License{}).Count(&stats.TotalLicenses).Error; err != nil {
		return nil, err
	}
	if err := s.db.Model(&models.License{}).Where("status = ?", "active").Count(&stats.ActiveLicenses).Error; err != nil {
		return nil, err
	}
	if err := s.db.Model(&models.License{}).
		Where("status = ? AND expires_at IS NOT NULL AND expires_at BETWEEN ? AND ?", "active", now, soon).
		Count(&stats.ExpiringSoon30Days).Error; err != nil {
		return nil, err
	}
	if err := s.db.Model(&models.LicenseActivation{}).Count(&stats.TotalActivations).Error; err != nil {
		return nil, err
	}
	if err := s.db.Model(&models.VerifyLog{}).Where("created_at >= ?", windowStart).Count(&stats.VerifyRequests24h).Error; err != nil {
		return nil, err
	}
	if stats.VerifyRequests24h > 0 {
		var okCount int64
		if err := s.db.Model(&models.VerifyLog{}).Where("created_at >= ? AND verified = ?", windowStart, true).Count(&okCount).Error; err != nil {
			return nil, err
		}
		stats.VerifySuccessRate24h = float64(okCount) / float64(stats.VerifyRequests24h) * 100
	}
	return stats, nil
}

func parseVersion(value string) (*semver.Version, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, errors.New("empty version")
	}
	if !strings.HasPrefix(trimmed, "v") {
		trimmed = "v" + trimmed
	}
	return semver.NewVersion(trimmed)
}

func generateLicenseKey() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	encoded := strings.ToUpper(hex.EncodeToString(buf))
	if len(encoded) < 24 {
		return "", errors.New("生成授权码失败")
	}
	return fmt.Sprintf("NP-%s-%s-%s", encoded[:8], encoded[8:16], encoded[16:24]), nil
}

func (s *UnifiedService) licenseExists(licenseID uint) (bool, error) {
	return s.licenseExistsByTx(s.db, licenseID)
}

func (s *UnifiedService) licenseExistsByTx(tx *gorm.DB, licenseID uint) (bool, error) {
	var count int64
	if err := tx.Model(&models.License{}).Where("id = ?", licenseID).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func isAllowedLicenseStatus(status string) bool {
	switch strings.TrimSpace(strings.ToLower(status)) {
	case "active", "revoked", "expired":
		return true
	default:
		return false
	}
}

func normalizePlanStatus(raw string) (string, error) {
	status := strings.TrimSpace(strings.ToLower(raw))
	if status == "" {
		status = "active"
	}
	switch status {
	case "active", "disabled":
		return status, nil
	default:
		return "", errors.New("plan status 仅支持 active/disabled")
	}
}

func buildClonePlanCode(base string, now time.Time) string {
	const maxLen = 64
	trimmed := strings.TrimSpace(strings.ToUpper(base))
	if trimmed == "" {
		trimmed = "PLAN"
	}

	suffix := fmt.Sprintf("COPY-%d", now.UnixNano())
	maxPrefixLen := maxLen - len(suffix) - 1
	if maxPrefixLen < 1 {
		maxPrefixLen = 1
	}
	if len(trimmed) > maxPrefixLen {
		trimmed = trimmed[:maxPrefixLen]
	}

	return fmt.Sprintf("%s-%s", trimmed, suffix)
}
