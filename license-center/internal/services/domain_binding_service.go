package services

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"nodepass-license-center/internal/models"

	"gorm.io/gorm"
)

// DomainBindingService 域名绑定服务
type DomainBindingService struct {
	db      *gorm.DB
	webhook *WebhookService
	alert   *AlertService
}

// NewDomainBindingService 创建域名绑定服务
func NewDomainBindingService(db *gorm.DB, webhook *WebhookService, alert *AlertService) *DomainBindingService {
	return &DomainBindingService{
		db:      db,
		webhook: webhook,
		alert:   alert,
	}
}

// DomainBindingConfig 域名绑定配置
type DomainBindingConfig struct {
	Enabled                   bool     `json:"enabled"`
	AutoBindOnFirstVerify     bool     `json:"auto_bind_on_first_verify"`
	AllowDomainChange         bool     `json:"allow_domain_change"`
	DomainChangeCooldown      int      `json:"domain_change_cooldown"` // 秒
	RequireDomainVerification bool     `json:"require_domain_verification"`
	AllowTestDomains          bool     `json:"allow_test_domains"`
	TestDomains               []string `json:"test_domains"`
}

// DefaultDomainBindingConfig 默认配置
var DefaultDomainBindingConfig = DomainBindingConfig{
	Enabled:                   true,
	AutoBindOnFirstVerify:     true,
	AllowDomainChange:         true,
	DomainChangeCooldown:      2592000, // 30天
	RequireDomainVerification: false,
	AllowTestDomains:          true,
	TestDomains: []string{
		"localhost",
		"127.0.0.1",
		"::1",
		"*.test",
		"*.local",
	},
}

// VerifyDomain 验证域名
func (s *DomainBindingService) VerifyDomain(license *models.LicenseKey, requestDomain, ip string, config DomainBindingConfig) error {
	if !config.Enabled {
		return nil
	}

	// 提取域名
	domain := extractDomain(requestDomain)
	if domain == "" {
		return fmt.Errorf("无效的域名")
	}

	// 检查是否是测试域名
	if config.AllowTestDomains && isTestDomain(domain, config.TestDomains) {
		// 测试域名不进行绑定，但记录使用
		s.recordTestDomainUsage(license.ID, domain, ip)
		return nil
	}

	// 如果授权码未绑定域名
	if license.BoundDomain == "" {
		if config.AutoBindOnFirstVerify && !license.DomainLocked {
			// 首次自动绑定
			return s.bindDomain(license, domain, ip, 0, "首次自动绑定")
		}
		return fmt.Errorf("授权码未绑定域名")
	}

	// 检查域名是否匹配
	if !s.matchDomain(license, domain) {
		// 域名不匹配，记录告警
		s.alert.CreateAlert(
			"domain_mismatch",
			"warning",
			fmt.Sprintf("域名不匹配: %s", license.Key),
			fmt.Sprintf("授权码已绑定到 %s，但尝试从 %s 访问", license.BoundDomain, domain),
			&license.ID,
			map[string]interface{}{
				"bound_domain":   license.BoundDomain,
				"request_domain": domain,
				"ip_address":     ip,
			},
		)

		return fmt.Errorf("域名不匹配，授权码已绑定到: %s", license.BoundDomain)
	}

	// 更新域名 IP 绑定记录
	s.updateDomainIPBinding(domain, ip)

	return nil
}

// bindDomain 绑定域名
func (s *DomainBindingService) bindDomain(license *models.LicenseKey, domain, ip string, operatorID uint, reason string) error {
	now := time.Now().UTC()

	// 记录绑定历史
	binding := &models.LicenseDomainBinding{
		LicenseID:  license.ID,
		OldDomain:  license.BoundDomain,
		NewDomain:  domain,
		Reason:     reason,
		OperatorID: operatorID,
		IPAddress:  ip,
	}
	if err := s.db.Create(binding).Error; err != nil {
		return err
	}

	// 更新授权码
	updates := map[string]interface{}{
		"bound_domain":    domain,
		"domain_bound_at": now,
	}
	if err := s.db.Model(&models.LicenseKey{}).Where("id = ?", license.ID).Updates(updates).Error; err != nil {
		return err
	}

	// 触发 Webhook
	s.webhook.TriggerEvent("license.domain_bound", map[string]interface{}{
		"license_id":  license.ID,
		"license_key": license.Key,
		"old_domain":  license.BoundDomain,
		"new_domain":  domain,
		"reason":      reason,
		"operator_id": operatorID,
	})

	return nil
}

// ChangeDomain 更换域名
func (s *DomainBindingService) ChangeDomain(licenseID uint, newDomain, reason string, operatorID uint, config DomainBindingConfig) error {
	if !config.AllowDomainChange {
		return fmt.Errorf("不允许更换域名")
	}

	var license models.LicenseKey
	if err := s.db.First(&license, licenseID).Error; err != nil {
		return err
	}

	// 检查冷却期
	if license.DomainBoundAt != nil && config.DomainChangeCooldown > 0 {
		cooldownEnd := license.DomainBoundAt.Add(time.Duration(config.DomainChangeCooldown) * time.Second)
		if time.Now().Before(cooldownEnd) {
			remainingTime := time.Until(cooldownEnd)
			return fmt.Errorf("域名更换冷却期未结束，还需等待 %s", remainingTime.Round(time.Hour))
		}
	}

	// 验证新域名
	newDomain = extractDomain(newDomain)
	if newDomain == "" {
		return fmt.Errorf("无效的域名")
	}

	// 执行绑定
	return s.bindDomain(&license, newDomain, "", operatorID, reason)
}

// UnbindDomain 解绑域名
func (s *DomainBindingService) UnbindDomain(licenseID uint, reason string, operatorID uint) error {
	var license models.LicenseKey
	if err := s.db.First(&license, licenseID).Error; err != nil {
		return err
	}

	// 记录解绑历史
	binding := &models.LicenseDomainBinding{
		LicenseID:  license.ID,
		OldDomain:  license.BoundDomain,
		NewDomain:  "",
		Reason:     reason,
		OperatorID: operatorID,
	}
	if err := s.db.Create(binding).Error; err != nil {
		return err
	}

	// 更新授权码
	updates := map[string]interface{}{
		"bound_domain":    "",
		"domain_bound_at": nil,
		"domain_locked":   false,
	}
	if err := s.db.Model(&models.LicenseKey{}).Where("id = ?", licenseID).Updates(updates).Error; err != nil {
		return err
	}

	// 触发 Webhook
	s.webhook.TriggerEvent("license.domain_unbound", map[string]interface{}{
		"license_id":  license.ID,
		"license_key": license.Key,
		"old_domain":  license.BoundDomain,
		"reason":      reason,
		"operator_id": operatorID,
	})

	return nil
}

// LockDomain 锁定域名（预设域名，防止自动绑定）
func (s *DomainBindingService) LockDomain(licenseID uint, domain string, operatorID uint) error {
	domain = extractDomain(domain)
	if domain == "" {
		return fmt.Errorf("无效的域名")
	}

	updates := map[string]interface{}{
		"bound_domain":  domain,
		"domain_locked": true,
	}
	return s.db.Model(&models.LicenseKey{}).Where("id = ?", licenseID).Updates(updates).Error
}

// matchDomain 匹配域名
func (s *DomainBindingService) matchDomain(license *models.LicenseKey, requestDomain string) bool {
	requestDomain = strings.ToLower(strings.TrimSpace(requestDomain))
	boundDomain := strings.ToLower(strings.TrimSpace(license.BoundDomain))

	// 精确匹配
	if boundDomain == requestDomain {
		return true
	}

	// 支持通配符 *.example.com
	if wildcardMatchDomain(boundDomain, requestDomain) {
		return true
	}

	// 检查允许的域名列表（多域名支持）
	if license.AllowedDomains != "" {
		var allowedDomains []string
		if err := json.Unmarshal([]byte(license.AllowedDomains), &allowedDomains); err == nil {
			for _, allowed := range allowedDomains {
				allowed = strings.ToLower(strings.TrimSpace(allowed))
				if allowed == requestDomain {
					return true
				}
				if wildcardMatchDomain(allowed, requestDomain) {
					return true
				}
			}
		}
	}

	return false
}

// updateDomainIPBinding 更新域名 IP 绑定
func (s *DomainBindingService) updateDomainIPBinding(domain, ip string) {
	var binding models.DomainIPBinding
	now := time.Now().UTC()

	err := s.db.Where("domain = ?", domain).First(&binding).Error
	if err == gorm.ErrRecordNotFound {
		// 创建新记录
		binding = models.DomainIPBinding{
			Domain:    domain,
			IPAddress: ip,
			FirstSeen: now,
			LastSeen:  now,
			HitCount:  1,
		}
		s.db.Create(&binding)
		return
	}

	// 检查 IP 是否变更
	if binding.IPAddress != ip {
		// IP 变更，创建告警
		s.alert.CreateAlert(
			"domain_ip_changed",
			"info",
			fmt.Sprintf("域名 IP 变更: %s", domain),
			fmt.Sprintf("域名 %s 的 IP 从 %s 变更为 %s", domain, binding.IPAddress, ip),
			nil,
			map[string]interface{}{
				"domain": domain,
				"old_ip": binding.IPAddress,
				"new_ip": ip,
			},
		)
	}

	// 更新记录
	s.db.Model(&binding).Updates(map[string]interface{}{
		"ip_address": ip,
		"last_seen":  now,
		"hit_count":  gorm.Expr("hit_count + 1"),
	})
}

// recordTestDomainUsage 记录测试域名使用
func (s *DomainBindingService) recordTestDomainUsage(licenseID uint, domain, ip string) {
	// 可以记录到日志或单独的表
	// 这里简单记录到 domain_ip_bindings
	s.updateDomainIPBinding(domain, ip)
}

// GetBindingHistory 获取绑定历史
func (s *DomainBindingService) GetBindingHistory(licenseID uint) ([]models.LicenseDomainBinding, error) {
	var history []models.LicenseDomainBinding
	err := s.db.Where("license_id = ?", licenseID).Order("id DESC").Find(&history).Error
	return history, err
}

// extractDomain 提取域名
func extractDomain(input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}

	// 如果是完整 URL，解析出域名
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		u, err := url.Parse(input)
		if err != nil {
			return ""
		}
		return u.Hostname()
	}

	// 移除端口号
	if idx := strings.Index(input, ":"); idx > 0 {
		input = input[:idx]
	}

	// 移除路径
	if idx := strings.Index(input, "/"); idx > 0 {
		input = input[:idx]
	}

	return strings.ToLower(input)
}

// isTestDomain 检查是否是测试域名
func isTestDomain(domain string, testDomains []string) bool {
	domain = strings.ToLower(strings.TrimSpace(domain))
	for _, td := range testDomains {
		td = strings.ToLower(strings.TrimSpace(td))
		if td == domain {
			return true
		}
		// 支持通配符
		if wildcardMatchDomain(td, domain) {
			return true
		}
	}
	return false
}

func wildcardMatchDomain(pattern, domain string) bool {
	if !strings.HasPrefix(pattern, "*.") {
		return false
	}
	base := strings.TrimPrefix(pattern, "*.")
	if base == "" || domain == "" {
		return false
	}
	// *.example.com 仅匹配 example.com 的子域名，不匹配自身或 badexample.com。
	if domain == base {
		return false
	}
	return strings.HasSuffix(domain, "."+base)
}
