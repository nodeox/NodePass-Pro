package services

import (
	"fmt"
	"strings"

	"nodepass-license-center/internal/models"

	"gorm.io/gorm"
)

// LicenseGroupService 授权码分组服务
type LicenseGroupService struct {
	db *gorm.DB
}

// NewLicenseGroupService 创建分组服务
func NewLicenseGroupService(db *gorm.DB) *LicenseGroupService {
	return &LicenseGroupService{db: db}
}

// CreateGroupRequest 创建分组请求
type CreateGroupRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Color       string `json:"color"`
	Icon        string `json:"icon"`
	SortOrder   int    `json:"sort_order"`
}

// GroupWithStats 分组统计信息
type GroupWithStats struct {
	models.LicenseGroup
	ActiveCount  int `json:"active_count"`
	ExpiredCount int `json:"expired_count"`
	RevokedCount int `json:"revoked_count"`
}

// ListGroups 查询分组列表
func (s *LicenseGroupService) ListGroups() ([]GroupWithStats, error) {
	groups := make([]models.LicenseGroup, 0)
	if err := s.db.Order("sort_order ASC, id DESC").Find(&groups).Error; err != nil {
		return nil, err
	}

	result := make([]GroupWithStats, 0, len(groups))
	for _, group := range groups {
		stats, _ := s.GetGroupStats(group.ID)
		result = append(result, GroupWithStats{
			LicenseGroup: group,
			ActiveCount:  stats["active_count"].(int),
			ExpiredCount: stats["expired_count"].(int),
			RevokedCount: stats["revoked_count"].(int),
		})
	}

	return result, nil
}

// GetGroup 获取分组详情
func (s *LicenseGroupService) GetGroup(id uint) (*models.LicenseGroup, error) {
	var group models.LicenseGroup
	if err := s.db.First(&group, id).Error; err != nil {
		return nil, err
	}
	return &group, nil
}

// CreateGroup 创建分组
func (s *LicenseGroupService) CreateGroup(req *CreateGroupRequest, createdBy uint) (*models.LicenseGroup, error) {
	if req == nil {
		return nil, fmt.Errorf("请求不能为空")
	}
	if strings.TrimSpace(req.Name) == "" {
		return nil, fmt.Errorf("分组名称不能为空")
	}

	groupType := strings.TrimSpace(req.Type)
	if groupType == "" {
		groupType = "custom"
	}
	if groupType != "project" && groupType != "customer" && groupType != "custom" {
		return nil, fmt.Errorf("分组类型无效")
	}

	group := &models.LicenseGroup{
		Name:        strings.TrimSpace(req.Name),
		Description: req.Description,
		Type:        groupType,
		Color:       strings.TrimSpace(req.Color),
		Icon:        strings.TrimSpace(req.Icon),
		SortOrder:   req.SortOrder,
		IsEnabled:   true,
		CreatedBy:   createdBy,
	}

	if err := s.db.Create(group).Error; err != nil {
		return nil, err
	}
	return group, nil
}

// UpdateGroup 更新分组
func (s *LicenseGroupService) UpdateGroup(id uint, req *CreateGroupRequest) (*models.LicenseGroup, error) {
	if id == 0 || req == nil {
		return nil, fmt.Errorf("参数无效")
	}

	var group models.LicenseGroup
	if err := s.db.First(&group, id).Error; err != nil {
		return nil, err
	}

	updates := map[string]interface{}{
		"name":        strings.TrimSpace(req.Name),
		"description": req.Description,
		"type":        strings.TrimSpace(req.Type),
		"color":       strings.TrimSpace(req.Color),
		"icon":        strings.TrimSpace(req.Icon),
		"sort_order":  req.SortOrder,
	}

	if err := s.db.Model(&group).Updates(updates).Error; err != nil {
		return nil, err
	}

	return s.GetGroup(id)
}

// DeleteGroup 删除分组
func (s *LicenseGroupService) DeleteGroup(id uint) error {
	if id == 0 {
		return fmt.Errorf("id 无效")
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		// 删除分组成员关系
		if err := tx.Where("group_id = ?", id).Delete(&models.LicenseGroupMember{}).Error; err != nil {
			return err
		}
		// 删除分组
		if err := tx.Delete(&models.LicenseGroup{}, id).Error; err != nil {
			return err
		}
		return nil
	})
}

// AddLicensesToGroup 添加授权码到分组
func (s *LicenseGroupService) AddLicensesToGroup(groupID uint, licenseIDs []uint) error {
	if groupID == 0 {
		return fmt.Errorf("分组ID无效")
	}
	if len(licenseIDs) == 0 {
		return fmt.Errorf("授权码ID列表不能为空")
	}

	// 检查分组是否存在
	var group models.LicenseGroup
	if err := s.db.First(&group, groupID).Error; err != nil {
		return fmt.Errorf("分组不存在")
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		for _, licenseID := range licenseIDs {
			// 检查是否已存在
			var count int64
			if err := tx.Model(&models.LicenseGroupMember{}).
				Where("group_id = ? AND license_id = ?", groupID, licenseID).
				Count(&count).Error; err != nil {
				return err
			}

			if count == 0 {
				member := &models.LicenseGroupMember{
					GroupID:   groupID,
					LicenseID: licenseID,
				}
				if err := tx.Create(member).Error; err != nil {
					return err
				}
			}
		}

		// 更新分组授权码数量
		var memberCount int64
		if err := tx.Model(&models.LicenseGroupMember{}).
			Where("group_id = ?", groupID).
			Count(&memberCount).Error; err != nil {
			return err
		}

		return tx.Model(&models.LicenseGroup{}).
			Where("id = ?", groupID).
			Update("license_count", memberCount).Error
	})
}

// RemoveLicensesFromGroup 从分组移除授权码
func (s *LicenseGroupService) RemoveLicensesFromGroup(groupID uint, licenseIDs []uint) error {
	if groupID == 0 {
		return fmt.Errorf("分组ID无效")
	}
	if len(licenseIDs) == 0 {
		return fmt.Errorf("授权码ID列表不能为空")
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		// 删除成员关系
		if err := tx.Where("group_id = ? AND license_id IN ?", groupID, licenseIDs).
			Delete(&models.LicenseGroupMember{}).Error; err != nil {
			return err
		}

		// 更新分组授权码数量
		var memberCount int64
		if err := tx.Model(&models.LicenseGroupMember{}).
			Where("group_id = ?", groupID).
			Count(&memberCount).Error; err != nil {
			return err
		}

		return tx.Model(&models.LicenseGroup{}).
			Where("id = ?", groupID).
			Update("license_count", memberCount).Error
	})
}

// GetGroupLicenses 获取分组内的授权码
func (s *LicenseGroupService) GetGroupLicenses(groupID uint, page, pageSize int) (*PaginatedResult[models.LicenseKey], error) {
	if groupID == 0 {
		return nil, fmt.Errorf("分组ID无效")
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}

	// 获取分组内的授权码ID列表
	var members []models.LicenseGroupMember
	if err := s.db.Where("group_id = ?", groupID).Find(&members).Error; err != nil {
		return nil, err
	}

	licenseIDs := make([]uint, 0, len(members))
	for _, m := range members {
		licenseIDs = append(licenseIDs, m.LicenseID)
	}

	if len(licenseIDs) == 0 {
		return &PaginatedResult[models.LicenseKey]{
			Items:    []models.LicenseKey{},
			Total:    0,
			Page:     page,
			PageSize: pageSize,
		}, nil
	}

	// 查询授权码
	query := s.db.Model(&models.LicenseKey{}).Preload("Plan").Where("id IN ?", licenseIDs)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	items := make([]models.LicenseKey, 0)
	offset := (page - 1) * pageSize
	if err := query.Order("id DESC").Offset(offset).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, err
	}

	return &PaginatedResult[models.LicenseKey]{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// GetGroupStats 获取分组统计信息
func (s *LicenseGroupService) GetGroupStats(groupID uint) (map[string]interface{}, error) {
	if groupID == 0 {
		return nil, fmt.Errorf("分组ID无效")
	}

	// 获取分组内的授权码ID列表
	var members []models.LicenseGroupMember
	if err := s.db.Where("group_id = ?", groupID).Find(&members).Error; err != nil {
		return nil, err
	}

	licenseIDs := make([]uint, 0, len(members))
	for _, m := range members {
		licenseIDs = append(licenseIDs, m.LicenseID)
	}

	stats := map[string]interface{}{
		"total_count":   len(licenseIDs),
		"active_count":  0,
		"expired_count": 0,
		"revoked_count": 0,
	}

	if len(licenseIDs) == 0 {
		return stats, nil
	}

	// 统计各状态数量
	var activeCount, expiredCount, revokedCount int64
	s.db.Model(&models.LicenseKey{}).Where("id IN ? AND status = ?", licenseIDs, "active").Count(&activeCount)
	s.db.Model(&models.LicenseKey{}).Where("id IN ? AND status = ?", licenseIDs, "expired").Count(&expiredCount)
	s.db.Model(&models.LicenseKey{}).Where("id IN ? AND status = ?", licenseIDs, "revoked").Count(&revokedCount)

	stats["active_count"] = int(activeCount)
	stats["expired_count"] = int(expiredCount)
	stats["revoked_count"] = int(revokedCount)

	return stats, nil
}

// GetLicenseGroups 获取授权码所属的分组列表
func (s *LicenseGroupService) GetLicenseGroups(licenseID uint) ([]models.LicenseGroup, error) {
	if licenseID == 0 {
		return nil, fmt.Errorf("授权码ID无效")
	}

	var members []models.LicenseGroupMember
	if err := s.db.Where("license_id = ?", licenseID).Find(&members).Error; err != nil {
		return nil, err
	}

	if len(members) == 0 {
		return []models.LicenseGroup{}, nil
	}

	groupIDs := make([]uint, 0, len(members))
	for _, m := range members {
		groupIDs = append(groupIDs, m.GroupID)
	}

	groups := make([]models.LicenseGroup, 0)
	if err := s.db.Where("id IN ?", groupIDs).Order("sort_order ASC").Find(&groups).Error; err != nil {
		return nil, err
	}

	return groups, nil
}
