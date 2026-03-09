package services

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"nodepass-license-center/internal/models"

	"gorm.io/gorm"
)

// LicenseEnhancedService 授权码增强服务
type LicenseEnhancedService struct {
	db *gorm.DB
}

// NewLicenseEnhancedService 创建增强服务
func NewLicenseEnhancedService(db *gorm.DB) *LicenseEnhancedService {
	return &LicenseEnhancedService{db: db}
}

// BatchUpdateRequest 批量更新请求
type BatchUpdateRequest struct {
	LicenseIDs  []uint                 `json:"license_ids"`
	Updates     map[string]interface{} `json:"updates"`
}

// BatchTransferRequest 批量转移客户请求
type BatchTransferRequest struct {
	LicenseIDs  []uint `json:"license_ids"`
	NewCustomer string `json:"new_customer"`
}

// AdvancedSearchRequest 高级搜索请求
type AdvancedSearchRequest struct {
	Status       []string   `json:"status"`
	Customer     string     `json:"customer"`
	PlanIDs      []uint     `json:"plan_ids"`
	GroupIDs     []uint     `json:"group_ids"`
	ExpiresFrom  *time.Time `json:"expires_from"`
	ExpiresTo    *time.Time `json:"expires_to"`
	CreatedFrom  *time.Time `json:"created_from"`
	CreatedTo    *time.Time `json:"created_to"`
	KeyPattern   string     `json:"key_pattern"`
	Note         string     `json:"note"`
	HasActivations bool     `json:"has_activations"`
	Page         int        `json:"page"`
	PageSize     int        `json:"page_size"`
}

// SavedSearchRequest 保存搜索请求
type SavedSearchRequest struct {
	Name        string                `json:"name"`
	Description string                `json:"description"`
	Filter      AdvancedSearchRequest `json:"filter"`
	IsPublic    bool                  `json:"is_public"`
}

// LicenseStatistics 授权码统计
type LicenseStatistics struct {
	TotalCount       int                    `json:"total_count"`
	ActiveCount      int                    `json:"active_count"`
	ExpiredCount     int                    `json:"expired_count"`
	RevokedCount     int                    `json:"revoked_count"`
	ExpiringIn7Days  int                    `json:"expiring_in_7_days"`
	ExpiringIn30Days int                    `json:"expiring_in_30_days"`
	CustomerCount    int                    `json:"customer_count"`
	ByPlan           map[string]int         `json:"by_plan"`
	ByStatus         map[string]int         `json:"by_status"`
	TrendData        []map[string]interface{} `json:"trend_data"`
}

// BatchUpdate 批量更新授权码
func (s *LicenseEnhancedService) BatchUpdate(req *BatchUpdateRequest) error {
	if req == nil || len(req.LicenseIDs) == 0 {
		return fmt.Errorf("授权码ID列表不能为空")
	}
	if len(req.Updates) == 0 {
		return fmt.Errorf("更新内容不能为空")
	}

	// 只允许更新特定字段
	allowed := map[string]bool{
		"status": true, "expires_at": true, "max_machines": true,
		"note": true, "max_domains": true,
	}
	for k := range req.Updates {
		if !allowed[k] {
			return fmt.Errorf("不允许更新字段: %s", k)
		}
	}

	return s.db.Model(&models.LicenseKey{}).
		Where("id IN ?", req.LicenseIDs).
		Updates(req.Updates).Error
}

// BatchTransfer 批量转移客户
func (s *LicenseEnhancedService) BatchTransfer(req *BatchTransferRequest) error {
	if req == nil || len(req.LicenseIDs) == 0 {
		return fmt.Errorf("授权码ID列表不能为空")
	}
	if strings.TrimSpace(req.NewCustomer) == "" {
		return fmt.Errorf("新客户名称不能为空")
	}

	return s.db.Model(&models.LicenseKey{}).
		Where("id IN ?", req.LicenseIDs).
		Update("customer", strings.TrimSpace(req.NewCustomer)).Error
}

// BatchRevoke 批量吊销
func (s *LicenseEnhancedService) BatchRevoke(licenseIDs []uint) error {
	if len(licenseIDs) == 0 {
		return fmt.Errorf("授权码ID列表不能为空")
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		// 更新授权码状态
		if err := tx.Model(&models.LicenseKey{}).
			Where("id IN ?", licenseIDs).
			Update("status", "revoked").Error; err != nil {
			return err
		}

		// 解绑所有机器
		return tx.Model(&models.LicenseActivation{}).
			Where("license_id IN ? AND is_active = ?", licenseIDs, true).
			Update("is_active", false).Error
	})
}

// BatchRestore 批量恢复
func (s *LicenseEnhancedService) BatchRestore(licenseIDs []uint) error {
	if len(licenseIDs) == 0 {
		return fmt.Errorf("授权码ID列表不能为空")
	}

	return s.db.Model(&models.LicenseKey{}).
		Where("id IN ? AND status = ?", licenseIDs, "revoked").
		Update("status", "active").Error
}

// BatchDelete 批量删除
func (s *LicenseEnhancedService) BatchDelete(licenseIDs []uint) error {
	if len(licenseIDs) == 0 {
		return fmt.Errorf("授权码ID列表不能为空")
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		// 删除激活记录
		if err := tx.Where("license_id IN ?", licenseIDs).
			Delete(&models.LicenseActivation{}).Error; err != nil {
			return err
		}

		// 删除分组关系
		if err := tx.Where("license_id IN ?", licenseIDs).
			Delete(&models.LicenseGroupMember{}).Error; err != nil {
			return err
		}

		// 删除授权码
		return tx.Where("id IN ?", licenseIDs).Delete(&models.LicenseKey{}).Error
	})
}

// AdvancedSearch 高级搜索
func (s *LicenseEnhancedService) AdvancedSearch(req *AdvancedSearchRequest) (*PaginatedResult[models.LicenseKey], error) {
	if req == nil {
		req = &AdvancedSearchRequest{}
	}
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	if req.PageSize > 200 {
		req.PageSize = 200
	}

	query := s.db.Model(&models.LicenseKey{}).Preload("Plan")

	// 状态过滤
	if len(req.Status) > 0 {
		query = query.Where("status IN ?", req.Status)
	}

	// 客户过滤
	if strings.TrimSpace(req.Customer) != "" {
		query = query.Where("customer LIKE ?", "%"+strings.TrimSpace(req.Customer)+"%")
	}

	// 套餐过滤
	if len(req.PlanIDs) > 0 {
		query = query.Where("plan_id IN ?", req.PlanIDs)
	}

	// 分组过滤
	if len(req.GroupIDs) > 0 {
		var members []models.LicenseGroupMember
		s.db.Where("group_id IN ?", req.GroupIDs).Find(&members)
		licenseIDs := make([]uint, 0, len(members))
		for _, m := range members {
			licenseIDs = append(licenseIDs, m.LicenseID)
		}
		if len(licenseIDs) > 0 {
			query = query.Where("id IN ?", licenseIDs)
		} else {
			// 没有匹配的授权码
			return &PaginatedResult[models.LicenseKey]{
				Items:    []models.LicenseKey{},
				Total:    0,
				Page:     req.Page,
				PageSize: req.PageSize,
			}, nil
		}
	}

	// 过期时间范围
	if req.ExpiresFrom != nil {
		query = query.Where("expires_at >= ?", req.ExpiresFrom)
	}
	if req.ExpiresTo != nil {
		query = query.Where("expires_at <= ?", req.ExpiresTo)
	}

	// 创建时间范围
	if req.CreatedFrom != nil {
		query = query.Where("created_at >= ?", req.CreatedFrom)
	}
	if req.CreatedTo != nil {
		query = query.Where("created_at <= ?", req.CreatedTo)
	}

	// 授权码模式匹配
	if strings.TrimSpace(req.KeyPattern) != "" {
		query = query.Where("key LIKE ?", "%"+strings.TrimSpace(req.KeyPattern)+"%")
	}

	// 备注搜索
	if strings.TrimSpace(req.Note) != "" {
		query = query.Where("note LIKE ?", "%"+strings.TrimSpace(req.Note)+"%")
	}

	// 是否有激活记录
	if req.HasActivations {
		var activatedLicenseIDs []uint
		s.db.Model(&models.LicenseActivation{}).
			Where("is_active = ?", true).
			Distinct("license_id").
			Pluck("license_id", &activatedLicenseIDs)
		if len(activatedLicenseIDs) > 0 {
			query = query.Where("id IN ?", activatedLicenseIDs)
		}
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	items := make([]models.LicenseKey, 0)
	offset := (req.Page - 1) * req.PageSize
	if err := query.Order("id DESC").Offset(offset).Limit(req.PageSize).Find(&items).Error; err != nil {
		return nil, err
	}

	return &PaginatedResult[models.LicenseKey]{
		Items:    items,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// SaveSearch 保存搜索条件
func (s *LicenseEnhancedService) SaveSearch(req *SavedSearchRequest, createdBy uint) (*models.SavedSearch, error) {
	if req == nil {
		return nil, fmt.Errorf("请求不能为空")
	}
	if strings.TrimSpace(req.Name) == "" {
		return nil, fmt.Errorf("搜索名称不能为空")
	}

	filterJSON, err := json.Marshal(req.Filter)
	if err != nil {
		return nil, fmt.Errorf("序列化搜索条件失败")
	}

	savedSearch := &models.SavedSearch{
		Name:        strings.TrimSpace(req.Name),
		Description: req.Description,
		FilterJSON:  string(filterJSON),
		IsPublic:    req.IsPublic,
		CreatedBy:   createdBy,
	}

	if err := s.db.Create(savedSearch).Error; err != nil {
		return nil, err
	}
	return savedSearch, nil
}

// ListSavedSearches 查询保存的搜索
func (s *LicenseEnhancedService) ListSavedSearches(userID uint) ([]models.SavedSearch, error) {
	searches := make([]models.SavedSearch, 0)
	err := s.db.Where("created_by = ? OR is_public = ?", userID, true).
		Order("id DESC").
		Find(&searches).Error
	return searches, err
}

// GetSavedSearch 获取保存的搜索
func (s *LicenseEnhancedService) GetSavedSearch(id uint) (*models.SavedSearch, error) {
	var search models.SavedSearch
	if err := s.db.First(&search, id).Error; err != nil {
		return nil, err
	}
	return &search, nil
}

// DeleteSavedSearch 删除保存的搜索
func (s *LicenseEnhancedService) DeleteSavedSearch(id uint, userID uint) error {
	return s.db.Where("id = ? AND created_by = ?", id, userID).
		Delete(&models.SavedSearch{}).Error
}

// GetStatistics 获取统计信息
func (s *LicenseEnhancedService) GetStatistics() (*LicenseStatistics, error) {
	stats := &LicenseStatistics{
		ByPlan:   make(map[string]int),
		ByStatus: make(map[string]int),
	}

	// 总数
	var total int64
	s.db.Model(&models.LicenseKey{}).Count(&total)
	stats.TotalCount = int(total)

	// 按状态统计
	var statusCounts []struct {
		Status string
		Count  int
	}
	s.db.Model(&models.LicenseKey{}).
		Select("status, COUNT(*) as count").
		Group("status").
		Scan(&statusCounts)

	for _, sc := range statusCounts {
		stats.ByStatus[sc.Status] = sc.Count
		switch sc.Status {
		case "active":
			stats.ActiveCount = sc.Count
		case "expired":
			stats.ExpiredCount = sc.Count
		case "revoked":
			stats.RevokedCount = sc.Count
		}
	}

	// 按套餐统计
	var planCounts []struct {
		PlanCode string
		Count    int
	}
	s.db.Model(&models.LicenseKey{}).
		Select("license_plans.code as plan_code, COUNT(*) as count").
		Joins("LEFT JOIN license_plans ON license_keys.plan_id = license_plans.id").
		Group("license_plans.code").
		Scan(&planCounts)

	for _, pc := range planCounts {
		stats.ByPlan[pc.PlanCode] = pc.Count
	}

	// 即将过期统计
	now := time.Now().UTC()
	in7Days := now.AddDate(0, 0, 7)
	in30Days := now.AddDate(0, 0, 30)

	var expiring7, expiring30 int64
	s.db.Model(&models.LicenseKey{}).
		Where("status = ? AND expires_at IS NOT NULL AND expires_at > ? AND expires_at <= ?", "active", now, in7Days).
		Count(&expiring7)
	s.db.Model(&models.LicenseKey{}).
		Where("status = ? AND expires_at IS NOT NULL AND expires_at > ? AND expires_at <= ?", "active", now, in30Days).
		Count(&expiring30)

	stats.ExpiringIn7Days = int(expiring7)
	stats.ExpiringIn30Days = int(expiring30)

	// 客户数量
	var customerCount int64
	s.db.Model(&models.LicenseKey{}).
		Distinct("customer").
		Count(&customerCount)
	stats.CustomerCount = int(customerCount)

	// 趋势数据（最近7天）
	stats.TrendData = make([]map[string]interface{}, 0)
	for i := 6; i >= 0; i-- {
		date := now.AddDate(0, 0, -i).Truncate(24 * time.Hour)
		nextDate := date.AddDate(0, 0, 1)

		var count int64
		s.db.Model(&models.LicenseKey{}).
			Where("created_at >= ? AND created_at < ?", date, nextDate).
			Count(&count)

		stats.TrendData = append(stats.TrendData, map[string]interface{}{
			"date":  date.Format("2006-01-02"),
			"count": int(count),
		})
	}

	return stats, nil
}

// GetExpiringLicenses 获取即将过期的授权码
func (s *LicenseEnhancedService) GetExpiringLicenses(days int) ([]models.LicenseKey, error) {
	if days <= 0 {
		days = 7
	}

	now := time.Now().UTC()
	targetDate := now.AddDate(0, 0, days)

	licenses := make([]models.LicenseKey, 0)
	err := s.db.Preload("Plan").
		Where("status = ? AND expires_at IS NOT NULL AND expires_at > ? AND expires_at <= ?", "active", now, targetDate).
		Order("expires_at ASC").
		Find(&licenses).Error

	return licenses, err
}
