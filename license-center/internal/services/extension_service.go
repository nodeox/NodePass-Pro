package services

import (
	"encoding/json"
	"fmt"
	"strings"

	"nodepass-license-center/internal/models"

	"gorm.io/gorm"
)

// ExtensionService 扩展功能服务
type ExtensionService struct {
	db      *gorm.DB
	webhook *WebhookService
}

// NewExtensionService 创建扩展功能服务
func NewExtensionService(db *gorm.DB, webhook *WebhookService) *ExtensionService {
	return &ExtensionService{
		db:      db,
		webhook: webhook,
	}
}

// TransferLicense 转移授权码
func (s *ExtensionService) TransferLicense(licenseID uint, toCustomer, reason string, operatorID uint) error {
	var license models.LicenseKey
	if err := s.db.First(&license, licenseID).Error; err != nil {
		return err
	}

	fromCustomer := license.Customer

	return s.db.Transaction(func(tx *gorm.DB) error {
		// 更新授权码客户
		if err := tx.Model(&models.LicenseKey{}).
			Where("id = ?", licenseID).
			Update("customer", toCustomer).Error; err != nil {
			return err
		}

		// 记录转移日志
		log := &models.LicenseTransferLog{
			LicenseID:    licenseID,
			FromCustomer: fromCustomer,
			ToCustomer:   toCustomer,
			Reason:       reason,
			OperatorID:   operatorID,
		}
		if err := tx.Create(log).Error; err != nil {
			return err
		}

		// 触发 Webhook
		_ = s.webhook.TriggerEvent("license.transferred", map[string]interface{}{
			"license_id":    licenseID,
			"license_key":   license.Key,
			"from_customer": fromCustomer,
			"to_customer":   toCustomer,
			"reason":        reason,
			"operator_id":   operatorID,
		})

		return nil
	})
}

// BatchUpdateLicenses 批量更新授权码
func (s *ExtensionService) BatchUpdateLicenses(licenseIDs []uint, updates map[string]interface{}) error {
	if len(licenseIDs) == 0 {
		return fmt.Errorf("授权码 ID 列表不能为空")
	}

	return s.db.Model(&models.LicenseKey{}).
		Where("id IN ?", licenseIDs).
		Updates(updates).Error
}

// BatchRevokeLicenses 批量吊销授权码
func (s *ExtensionService) BatchRevokeLicenses(licenseIDs []uint) error {
	return s.BatchUpdateLicenses(licenseIDs, map[string]interface{}{"status": "revoked"})
}

// BatchRestoreLicenses 批量恢复授权码
func (s *ExtensionService) BatchRestoreLicenses(licenseIDs []uint) error {
	return s.BatchUpdateLicenses(licenseIDs, map[string]interface{}{"status": "active"})
}

// BatchDeleteLicenses 批量删除授权码
func (s *ExtensionService) BatchDeleteLicenses(licenseIDs []uint) error {
	if len(licenseIDs) == 0 {
		return fmt.Errorf("授权码 ID 列表不能为空")
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("license_id IN ?", licenseIDs).Delete(&models.LicenseActivation{}).Error; err != nil {
			return err
		}
		if err := tx.Where("license_key_id IN ?", licenseIDs).Delete(&models.LicenseKeyTag{}).Error; err != nil {
			return err
		}
		return tx.Where("id IN ?", licenseIDs).Delete(&models.LicenseKey{}).Error
	})
}

// ListTags 查询标签
func (s *ExtensionService) ListTags() ([]models.LicenseTag, error) {
	var tags []models.LicenseTag
	err := s.db.Order("name ASC").Find(&tags).Error
	return tags, err
}

// CreateTag 创建标签
func (s *ExtensionService) CreateTag(name, color string) (*models.LicenseTag, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("标签名称不能为空")
	}

	tag := &models.LicenseTag{
		Name:  name,
		Color: color,
	}

	if err := s.db.Create(tag).Error; err != nil {
		return nil, err
	}

	return tag, nil
}

// UpdateTag 更新标签
func (s *ExtensionService) UpdateTag(id uint, name, color string) error {
	updates := map[string]interface{}{}
	if name != "" {
		updates["name"] = strings.TrimSpace(name)
	}
	if color != "" {
		updates["color"] = color
	}

	return s.db.Model(&models.LicenseTag{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteTag 删除标签
func (s *ExtensionService) DeleteTag(id uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("tag_id = ?", id).Delete(&models.LicenseKeyTag{}).Error; err != nil {
			return err
		}
		return tx.Delete(&models.LicenseTag{}, id).Error
	})
}

// AddTagsToLicense 为授权码添加标签
func (s *ExtensionService) AddTagsToLicense(licenseID uint, tagIDs []uint) error {
	if len(tagIDs) == 0 {
		return nil
	}

	// 检查授权码是否存在
	var license models.LicenseKey
	if err := s.db.First(&license, licenseID).Error; err != nil {
		return err
	}

	// 检查标签是否存在
	var count int64
	if err := s.db.Model(&models.LicenseTag{}).Where("id IN ?", tagIDs).Count(&count).Error; err != nil {
		return err
	}
	if int(count) != len(tagIDs) {
		return fmt.Errorf("部分标签不存在")
	}

	// 添加标签
	for _, tagID := range tagIDs {
		// 检查是否已存在
		var existing models.LicenseKeyTag
		err := s.db.Where("license_key_id = ? AND tag_id = ?", licenseID, tagID).First(&existing).Error
		if err == nil {
			continue // 已存在，跳过
		}

		keyTag := &models.LicenseKeyTag{
			LicenseKeyID: licenseID,
			TagID:        tagID,
		}
		if err := s.db.Create(keyTag).Error; err != nil {
			return err
		}
	}

	return nil
}

// RemoveTagsFromLicense 从授权码移除标签
func (s *ExtensionService) RemoveTagsFromLicense(licenseID uint, tagIDs []uint) error {
	if len(tagIDs) == 0 {
		return nil
	}

	return s.db.Where("license_key_id = ? AND tag_id IN ?", licenseID, tagIDs).
		Delete(&models.LicenseKeyTag{}).Error
}

// GetLicenseTags 获取授权码的标签
func (s *ExtensionService) GetLicenseTags(licenseID uint) ([]models.LicenseTag, error) {
	var tags []models.LicenseTag
	err := s.db.Raw(`
		SELECT lt.* FROM license_tags lt
		INNER JOIN license_key_tags lkt ON lt.id = lkt.tag_id
		WHERE lkt.license_key_id = ?
		ORDER BY lt.name ASC
	`, licenseID).Scan(&tags).Error
	return tags, err
}

// UpdateLicenseMetadata 更新授权码自定义字段
func (s *ExtensionService) UpdateLicenseMetadata(licenseID uint, metadata map[string]interface{}) error {
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	return s.db.Model(&models.LicenseKey{}).
		Where("id = ?", licenseID).
		Update("metadata_json", string(metadataJSON)).Error
}

// GetLicenseMetadata 获取授权码自定义字段
func (s *ExtensionService) GetLicenseMetadata(licenseID uint) (map[string]interface{}, error) {
	var license models.LicenseKey
	if err := s.db.First(&license, licenseID).Error; err != nil {
		return nil, err
	}

	if license.MetadataJSON == "" {
		return map[string]interface{}{}, nil
	}

	var metadata map[string]interface{}
	if err := json.Unmarshal([]byte(license.MetadataJSON), &metadata); err != nil {
		return nil, err
	}

	return metadata, nil
}

// GetTransferLogs 获取转移日志
func (s *ExtensionService) GetTransferLogs(licenseID uint) ([]models.LicenseTransferLog, error) {
	var logs []models.LicenseTransferLog
	query := s.db.Order("id DESC")
	if licenseID > 0 {
		query = query.Where("license_id = ?", licenseID)
	}
	err := query.Find(&logs).Error
	return logs, err
}
