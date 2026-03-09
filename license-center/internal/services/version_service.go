package services

import (
	"errors"
	"fmt"
	"strings"

	"nodepass-license-center/internal/models"

	"gorm.io/gorm"
)

// VersionService 版本管理服务
type VersionService struct {
	db *gorm.DB
}

// NewVersionService 创建版本管理服务
func NewVersionService(db *gorm.DB) *VersionService {
	return &VersionService{db: db}
}

// SystemVersionInfo 系统版本信息
type SystemVersionInfo struct {
	Backend       *models.ComponentVersion `json:"backend"`
	Frontend      *models.ComponentVersion `json:"frontend"`
	NodeClient    *models.ComponentVersion `json:"node_client"`
	LicenseCenter *models.ComponentVersion `json:"license_center"`
	Compatibility *CompatibilityInfo       `json:"compatibility"`
}

// CompatibilityInfo 兼容性信息
type CompatibilityInfo struct {
	IsCompatible bool     `json:"is_compatible"`
	Warnings     []string `json:"warnings"`
	Errors       []string `json:"errors"`
}

// GetSystemVersionInfo 获取系统版本信息
func (s *VersionService) GetSystemVersionInfo() (*SystemVersionInfo, error) {
	info := &SystemVersionInfo{}

	// 获取各组件版本
	backend, _ := s.GetComponentVersion(models.ComponentTypeBackend)
	frontend, _ := s.GetComponentVersion(models.ComponentTypeFrontend)
	nodeClient, _ := s.GetComponentVersion(models.ComponentTypeNodeClient)
	licenseCenter, _ := s.GetComponentVersion(models.ComponentTypeLicenseCenter)

	info.Backend = backend
	info.Frontend = frontend
	info.NodeClient = nodeClient
	info.LicenseCenter = licenseCenter

	// 检查版本兼容性
	if backend != nil {
		compatibility, err := s.CheckCompatibility(backend.Version)
		if err == nil {
			info.Compatibility = compatibility
		}
	}

	return info, nil
}

// GetComponentVersion 获取组件版本
func (s *VersionService) GetComponentVersion(component models.ComponentType) (*models.ComponentVersion, error) {
	var ver models.ComponentVersion
	if err := s.db.Where("component = ? AND is_active = ?", component, true).
		Order("created_at DESC").
		First(&ver).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &ver, nil
}

// UpdateComponentVersion 更新组件版本
func (s *VersionService) UpdateComponentVersion(ver *models.ComponentVersion) error {
	// 将旧版本设为非活跃
	if err := s.db.Model(&models.ComponentVersion{}).
		Where("component = ? AND is_active = ?", ver.Component, true).
		Update("is_active", false).Error; err != nil {
		return err
	}

	// 创建新版本记录
	ver.IsActive = true
	return s.db.Create(ver).Error
}

// ListComponentVersions 列出组件版本历史
func (s *VersionService) ListComponentVersions(component models.ComponentType, limit int) ([]models.ComponentVersion, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	var versions []models.ComponentVersion
	if err := s.db.Where("component = ?", component).
		Order("created_at DESC").
		Limit(limit).
		Find(&versions).Error; err != nil {
		return nil, err
	}
	return versions, nil
}

// CheckCompatibility 检查版本兼容性
func (s *VersionService) CheckCompatibility(backendVersion string) (*CompatibilityInfo, error) {
	info := &CompatibilityInfo{
		IsCompatible: true,
		Warnings:     []string{},
		Errors:       []string{},
	}

	// 查询兼容性配置
	var compat models.VersionCompatibility
	if err := s.db.Where("backend_version = ? AND is_active = ?", backendVersion, true).
		First(&compat).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			info.Warnings = append(info.Warnings, "未找到版本兼容性配置")
			return info, nil
		}
		return nil, err
	}

	// 检查前端版本
	frontend, _ := s.GetComponentVersion(models.ComponentTypeFrontend)
	if frontend != nil {
		if !isVersionCompatible(frontend.Version, compat.MinFrontendVersion) {
			info.IsCompatible = false
			info.Errors = append(info.Errors, fmt.Sprintf(
				"前端版本 %s 低于最低要求版本 %s",
				frontend.Version, compat.MinFrontendVersion,
			))
		}
	} else {
		info.Warnings = append(info.Warnings, "未检测到前端版本")
	}

	// 检查节点客户端版本
	nodeClient, _ := s.GetComponentVersion(models.ComponentTypeNodeClient)
	if nodeClient != nil {
		if !isVersionCompatible(nodeClient.Version, compat.MinNodeClientVersion) {
			info.Warnings = append(info.Warnings, fmt.Sprintf(
				"节点客户端版本 %s 低于推荐版本 %s",
				nodeClient.Version, compat.MinNodeClientVersion,
			))
		}
	}

	// 检查授权中心版本
	licenseCenter, _ := s.GetComponentVersion(models.ComponentTypeLicenseCenter)
	if licenseCenter != nil {
		if !isVersionCompatible(licenseCenter.Version, compat.MinLicenseCenterVersion) {
			info.Warnings = append(info.Warnings, fmt.Sprintf(
				"授权中心版本 %s 低于推荐版本 %s",
				licenseCenter.Version, compat.MinLicenseCenterVersion,
			))
		}
	}

	return info, nil
}

// CreateCompatibilityConfig 创建兼容性配置
func (s *VersionService) CreateCompatibilityConfig(config *models.VersionCompatibility) error {
	// 将旧配置设为非活跃
	if err := s.db.Model(&models.VersionCompatibility{}).
		Where("backend_version = ? AND is_active = ?", config.BackendVersion, true).
		Update("is_active", false).Error; err != nil {
		return err
	}

	// 创建新配置
	config.IsActive = true
	return s.db.Create(config).Error
}

// ListCompatibilityConfigs 列出兼容性配置
func (s *VersionService) ListCompatibilityConfigs() ([]models.VersionCompatibility, error) {
	var configs []models.VersionCompatibility
	if err := s.db.Where("is_active = ?", true).
		Order("created_at DESC").
		Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

// isVersionCompatible 检查版本是否兼容（简单的版本比较）
func isVersionCompatible(current, minimum string) bool {
	current = strings.TrimPrefix(current, "v")
	minimum = strings.TrimPrefix(minimum, "v")

	currentParts := strings.Split(current, ".")
	minimumParts := strings.Split(minimum, ".")

	for i := 0; i < len(minimumParts) && i < len(currentParts); i++ {
		if currentParts[i] < minimumParts[i] {
			return false
		}
		if currentParts[i] > minimumParts[i] {
			return true
		}
	}

	return len(currentParts) >= len(minimumParts)
}
