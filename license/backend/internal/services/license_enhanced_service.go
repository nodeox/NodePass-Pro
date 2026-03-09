package services

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"nodepass-license-unified/internal/models"

	"gorm.io/gorm"
)

// LicenseEnhancedService 授权码增强服务
type LicenseEnhancedService struct {
	db             *gorm.DB
	unifiedService *UnifiedService
}

// NewLicenseEnhancedService 创建增强服务
func NewLicenseEnhancedService(db *gorm.DB, unifiedService *UnifiedService) *LicenseEnhancedService {
	return &LicenseEnhancedService{
		db:             db,
		unifiedService: unifiedService,
	}
}

// ExportRequest 导出请求
type ExportRequest struct {
	LicenseIDs []uint   `json:"license_ids"`
	Status     []string `json:"status"`
	Customer   string   `json:"customer"`
	PlanIDs    []uint   `json:"plan_ids"`
}

// RenewalRequest 续期请求
type RenewalRequest struct {
	LicenseIDs []uint `json:"license_ids" binding:"required"`
	Days       int    `json:"days"`
	Months     int    `json:"months"`
	Years      int    `json:"years"`
}

// CloneRequest 克隆请求
type CloneRequest struct {
	SourceLicenseID uint   `json:"source_license_id" binding:"required"`
	Count           int    `json:"count" binding:"required,min=1,max=100"`
	Customer        string `json:"customer"`
}

// BatchUpdateRequest 批量更新请求
type BatchUpdateRequest struct {
	LicenseIDs []uint                 `json:"license_ids" binding:"required"`
	Updates    map[string]interface{} `json:"updates" binding:"required"`
}

// ExportToCSV 导出授权码到 CSV
func (s *LicenseEnhancedService) ExportToCSV(req *ExportRequest) (string, error) {
	if req == nil {
		return "", fmt.Errorf("请求不能为空")
	}

	query := s.db.Model(&models.License{}).Preload("Plan")

	// 应用过滤条件
	if len(req.LicenseIDs) > 0 {
		query = query.Where("id IN ?", req.LicenseIDs)
	}
	if len(req.Status) > 0 {
		query = query.Where("status IN ?", req.Status)
	}
	if strings.TrimSpace(req.Customer) != "" {
		customer := strings.TrimSpace(req.Customer)
		query = query.Where("customer LIKE ?", "%"+customer+"%")
	}
	if len(req.PlanIDs) > 0 {
		query = query.Where("plan_id IN ?", req.PlanIDs)
	}

	var licenses []models.License
	if err := query.Find(&licenses).Error; err != nil {
		return "", err
	}

	// 创建临时文件
	file, err := os.CreateTemp("", "licenses_export_*.csv")
	if err != nil {
		return "", err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 写入表头
	headers := []string{"ID", "授权码", "套餐", "客户", "状态", "过期时间", "最大机器数", "创建时间", "备注"}
	if err := writer.Write(headers); err != nil {
		return "", err
	}

	// 写入数据
	for _, license := range licenses {
		expiresAt := ""
		if license.ExpiresAt != nil {
			expiresAt = license.ExpiresAt.Format("2006-01-02 15:04:05")
		}

		maxMachines := strconv.Itoa(license.Plan.MaxMachines)
		if license.MaxMachines != nil {
			maxMachines = strconv.Itoa(*license.MaxMachines)
		}

		row := []string{
			strconv.Itoa(int(license.ID)),
			license.Key,
			license.Plan.Name,
			license.Customer,
			license.Status,
			expiresAt,
			maxMachines,
			license.CreatedAt.Format("2006-01-02 15:04:05"),
			license.Note,
		}

		if err := writer.Write(row); err != nil {
			return "", err
		}
	}

	return file.Name(), nil
}

// RenewLicenses 批量续期
func (s *LicenseEnhancedService) RenewLicenses(req *RenewalRequest) error {
	if req == nil || len(req.LicenseIDs) == 0 {
		return fmt.Errorf("授权码ID列表不能为空")
	}

	if req.Days == 0 && req.Months == 0 && req.Years == 0 {
		return fmt.Errorf("续期时长不能为空")
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		var licenses []models.License
		if err := tx.Where("id IN ?", req.LicenseIDs).Find(&licenses).Error; err != nil {
			return err
		}

		for _, license := range licenses {
			var newExpiresAt time.Time
			if license.ExpiresAt != nil {
				newExpiresAt = license.ExpiresAt.UTC().AddDate(req.Years, req.Months, req.Days)
			} else {
				newExpiresAt = time.Now().UTC().AddDate(req.Years, req.Months, req.Days)
			}

			if err := tx.Model(&models.License{}).
				Where("id = ?", license.ID).
				Updates(map[string]interface{}{
					"expires_at": newExpiresAt,
					"status":     "active",
				}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// CloneLicense 克隆授权码
func (s *LicenseEnhancedService) CloneLicense(req *CloneRequest, createdBy uint) ([]models.License, error) {
	if req == nil {
		return nil, fmt.Errorf("请求不能为空")
	}

	// 获取源授权码
	var source models.License
	if err := s.db.Preload("Plan").First(&source, req.SourceLicenseID).Error; err != nil {
		return nil, fmt.Errorf("源授权码不存在")
	}

	customer := req.Customer
	if customer == "" {
		customer = source.Customer
	}

	items := make([]models.License, 0, req.Count)

	err := s.db.Transaction(func(tx *gorm.DB) error {
		for i := 0; i < req.Count; i++ {
			key, err := generateLicenseKey()
			if err != nil {
				return fmt.Errorf("生成授权码失败: %w", err)
			}

			license := models.License{
				Key:         key,
				PlanID:      source.PlanID,
				Customer:    customer,
				Status:      "active",
				ExpiresAt:   source.ExpiresAt,
				MaxMachines: source.MaxMachines,
				Note:        fmt.Sprintf("克隆自 %s", source.Key),
				CreatedBy:   createdBy,
			}

			if err := tx.Create(&license).Error; err != nil {
				return err
			}
			items = append(items, license)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return items, nil
}

// BatchUpdate 批量更新
func (s *LicenseEnhancedService) BatchUpdate(req *BatchUpdateRequest) error {
	return s.BatchUpdateWithOperator(req, 0)
}

// BatchUpdateWithOperator 批量更新（含审计日志）。
func (s *LicenseEnhancedService) BatchUpdateWithOperator(req *BatchUpdateRequest, operatorID uint) error {
	if req == nil || len(req.LicenseIDs) == 0 {
		return fmt.Errorf("授权码ID列表不能为空")
	}
	if len(req.Updates) == 0 {
		return fmt.Errorf("更新内容不能为空")
	}

	updates, err := normalizeBatchLicenseUpdates(req.Updates)
	if err != nil {
		return err
	}

	if planIDRaw, ok := updates["plan_id"]; ok {
		planID := planIDRaw.(uint)
		var count int64
		if err = s.db.Model(&models.LicensePlan{}).Where("id = ?", planID).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			return fmt.Errorf("套餐不存在")
		}
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		updateTx := tx.Model(&models.License{}).
			Where("id IN ?", req.LicenseIDs).
			Updates(updates)
		if updateTx.Error != nil {
			return updateTx.Error
		}

		return createAdminAuditLog(tx, operatorID, AuditActionLicenseBatchUpdate, "license", map[string]interface{}{
			"license_ids":     req.LicenseIDs,
			"updates":         updates,
			"rows_affected":   updateTx.RowsAffected,
			"requested_count": len(req.LicenseIDs),
		})
	})
}

func normalizeBatchLicenseUpdates(raw map[string]interface{}) (map[string]interface{}, error) {
	updates := make(map[string]interface{}, len(raw))
	for key, value := range raw {
		switch key {
		case "plan_id":
			planID, err := parseUintFieldValue(value, "plan_id")
			if err != nil {
				return nil, err
			}
			updates["plan_id"] = planID
		case "status":
			status, ok := value.(string)
			if !ok {
				return nil, fmt.Errorf("status 必须是字符串")
			}
			normalized := strings.ToLower(strings.TrimSpace(status))
			if normalized != "active" && normalized != "revoked" && normalized != "expired" {
				return nil, fmt.Errorf("status 仅支持 active/revoked/expired")
			}
			updates["status"] = normalized
		case "expires_at":
			switch v := value.(type) {
			case nil:
				updates["expires_at"] = nil
			case string:
				trimmed := strings.TrimSpace(v)
				if trimmed == "" {
					updates["expires_at"] = nil
					continue
				}
				parsed, err := time.Parse(time.RFC3339, trimmed)
				if err != nil {
					return nil, fmt.Errorf("expires_at 时间格式无效")
				}
				updates["expires_at"] = parsed
			default:
				return nil, fmt.Errorf("expires_at 必须是 RFC3339 字符串或 null")
			}
		case "max_machines":
			maxMachines, err := parseIntFieldValue(value, "max_machines")
			if err != nil {
				return nil, err
			}
			if maxMachines <= 0 {
				return nil, fmt.Errorf("max_machines 必须大于 0")
			}
			updates["max_machines"] = maxMachines
		case "note":
			text, ok := value.(string)
			if !ok {
				return nil, fmt.Errorf("%s 必须是字符串", key)
			}
			updates[key] = strings.TrimSpace(text)
		case "metadata_json":
			text, ok := value.(string)
			if !ok {
				return nil, fmt.Errorf("metadata_json 必须是字符串")
			}
			trimmed := strings.TrimSpace(text)
			if trimmed != "" && !json.Valid([]byte(trimmed)) {
				return nil, fmt.Errorf("metadata_json 必须为合法 JSON")
			}
			updates[key] = trimmed
		default:
			return nil, fmt.Errorf("不允许更新字段: %s", key)
		}
	}
	if len(updates) == 0 {
		return nil, fmt.Errorf("更新内容不能为空")
	}
	return updates, nil
}

func parseUintFieldValue(value interface{}, field string) (uint, error) {
	switch v := value.(type) {
	case uint:
		if v == 0 {
			return 0, fmt.Errorf("%s 必须大于 0", field)
		}
		return v, nil
	case uint64:
		if v == 0 {
			return 0, fmt.Errorf("%s 必须大于 0", field)
		}
		return uint(v), nil
	case int:
		if v <= 0 {
			return 0, fmt.Errorf("%s 必须大于 0", field)
		}
		return uint(v), nil
	case int64:
		if v <= 0 {
			return 0, fmt.Errorf("%s 必须大于 0", field)
		}
		return uint(v), nil
	case float64:
		if v <= 0 || math.Trunc(v) != v {
			return 0, fmt.Errorf("%s 必须为正整数", field)
		}
		return uint(v), nil
	case string:
		trimmed := strings.TrimSpace(v)
		parsed, err := strconv.ParseUint(trimmed, 10, 64)
		if err != nil || parsed == 0 {
			return 0, fmt.Errorf("%s 必须为正整数", field)
		}
		return uint(parsed), nil
	default:
		return 0, fmt.Errorf("%s 必须为正整数", field)
	}
}

func parseIntFieldValue(value interface{}, field string) (int, error) {
	switch v := value.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case uint:
		return int(v), nil
	case uint64:
		return int(v), nil
	case float64:
		if math.Trunc(v) != v {
			return 0, fmt.Errorf("%s 必须为整数", field)
		}
		return int(v), nil
	case string:
		trimmed := strings.TrimSpace(v)
		parsed, err := strconv.Atoi(trimmed)
		if err != nil {
			return 0, fmt.Errorf("%s 必须为整数", field)
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("%s 必须为整数", field)
	}
}

// BatchRevoke 批量吊销
func (s *LicenseEnhancedService) BatchRevoke(licenseIDs []uint) error {
	return s.BatchRevokeWithOperator(licenseIDs, 0)
}

// BatchRevokeWithOperator 批量吊销（含审计日志）。
func (s *LicenseEnhancedService) BatchRevokeWithOperator(licenseIDs []uint, operatorID uint) error {
	if len(licenseIDs) == 0 {
		return fmt.Errorf("授权码ID列表不能为空")
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		updateTx := tx.Model(&models.License{}).
			Where("id IN ?", licenseIDs).
			Update("status", "revoked")
		if updateTx.Error != nil {
			return updateTx.Error
		}

		return createAdminAuditLog(tx, operatorID, AuditActionLicenseBatchRevoke, "license", map[string]interface{}{
			"license_ids":     licenseIDs,
			"rows_affected":   updateTx.RowsAffected,
			"requested_count": len(licenseIDs),
		})
	})
}

// BatchRestore 批量恢复
func (s *LicenseEnhancedService) BatchRestore(licenseIDs []uint) error {
	return s.BatchRestoreWithOperator(licenseIDs, 0)
}

// BatchRestoreWithOperator 批量恢复（含审计日志）。
func (s *LicenseEnhancedService) BatchRestoreWithOperator(licenseIDs []uint, operatorID uint) error {
	if len(licenseIDs) == 0 {
		return fmt.Errorf("授权码ID列表不能为空")
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		updateTx := tx.Model(&models.License{}).
			Where("id IN ? AND status = ?", licenseIDs, "revoked").
			Update("status", "active")
		if updateTx.Error != nil {
			return updateTx.Error
		}

		return createAdminAuditLog(tx, operatorID, AuditActionLicenseBatchRestore, "license", map[string]interface{}{
			"license_ids":     licenseIDs,
			"rows_affected":   updateTx.RowsAffected,
			"requested_count": len(licenseIDs),
		})
	})
}

// GetUsageReport 获取使用报告
func (s *LicenseEnhancedService) GetUsageReport(licenseID uint) (map[string]interface{}, error) {
	var license models.License
	if err := s.db.Preload("Plan").Preload("Activations").First(&license, licenseID).Error; err != nil {
		return nil, fmt.Errorf("授权码不存在")
	}

	report := make(map[string]interface{})
	report["license_key"] = license.Key
	report["customer"] = license.Customer
	report["status"] = license.Status
	report["plan"] = license.Plan.Name

	// 过期信息
	if license.ExpiresAt != nil {
		report["expires_at"] = license.ExpiresAt.Format("2006-01-02 15:04:05")
		daysLeft := int(time.Until(*license.ExpiresAt).Hours() / 24)
		report["days_left"] = daysLeft
		report["is_expired"] = daysLeft < 0
	}

	// 机器绑定统计
	activeCount := len(license.Activations)

	maxMachines := license.Plan.MaxMachines
	if license.MaxMachines != nil {
		maxMachines = *license.MaxMachines
	}

	report["max_machines"] = maxMachines
	report["active_machines"] = activeCount
	if maxMachines > 0 {
		report["usage_rate"] = float64(activeCount) / float64(maxMachines) * 100
	}

	// 激活详情
	activationDetails := make([]map[string]interface{}, 0)
	for _, a := range license.Activations {
		activationDetails = append(activationDetails, map[string]interface{}{
			"machine_id":   a.MachineID,
			"hostname":     a.Hostname,
			"ip_address":   a.IPAddress,
			"last_seen_at": a.LastSeenAt.Format("2006-01-02 15:04:05"),
			"created_at":   a.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	report["activations"] = activationDetails

	return report, nil
}

// GetCustomerReport 获取客户报告
func (s *LicenseEnhancedService) GetCustomerReport(customer string) (map[string]interface{}, error) {
	if strings.TrimSpace(customer) == "" {
		return nil, fmt.Errorf("客户名称不能为空")
	}

	customer = strings.TrimSpace(customer)
	report := make(map[string]interface{})
	report["customer"] = customer

	var licenses []models.License
	s.db.Preload("Plan").Where("customer = ?", customer).Find(&licenses)

	report["total_licenses"] = len(licenses)

	activeCount := 0
	expiredCount := 0
	revokedCount := 0
	totalMachines := 0

	for _, license := range licenses {
		switch license.Status {
		case "active":
			activeCount++
		case "expired":
			expiredCount++
		case "revoked":
			revokedCount++
		}

		var count int64
		s.db.Model(&models.LicenseActivation{}).
			Where("license_id = ?", license.ID).
			Count(&count)
		totalMachines += int(count)
	}

	report["active_licenses"] = activeCount
	report["expired_licenses"] = expiredCount
	report["revoked_licenses"] = revokedCount
	report["total_machines"] = totalMachines

	// 即将过期的授权码
	now := time.Now().UTC()
	in30Days := now.AddDate(0, 0, 30)
	expiringLicenses := make([]map[string]interface{}, 0)

	for _, license := range licenses {
		if license.Status == "active" && license.ExpiresAt != nil {
			if license.ExpiresAt.After(now) && license.ExpiresAt.Before(in30Days) {
				expiringLicenses = append(expiringLicenses, map[string]interface{}{
					"key":        license.Key,
					"plan":       license.Plan.Name,
					"expires_at": license.ExpiresAt.Format("2006-01-02"),
					"days_left":  int(time.Until(*license.ExpiresAt).Hours() / 24),
				})
			}
		}
	}
	report["expiring_soon"] = expiringLicenses

	return report, nil
}
