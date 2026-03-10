package license

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"nodepass-pro/backend/internal/config"
	"nodepass-pro/backend/internal/version"

	"go.uber.org/zap"
)

const defaultVerifyInterval = 300 * time.Second

const (
	licenseVerifyScheme = "https://"
	licenseVerifyHost1  = "key."
	licenseVerifyHost2  = "hahaha."
	licenseVerifyHost3  = "ooo"
	licenseVerifyPath   = "/api/v1/verify"
)

// fixedLicenseVerifyURL 返回旧版授权中心地址（兼容）。
func fixedLicenseVerifyURL() string {
	return licenseVerifyScheme + licenseVerifyHost1 + licenseVerifyHost2 + licenseVerifyHost3 + licenseVerifyPath
}

func resolveVerifyURL(override string) string {
	if strings.TrimSpace(override) != "" {
		return strings.TrimSpace(override)
	}
	return fixedLicenseVerifyURL()
}

// Status 运行时授权状态。
type Status struct {
	Enabled       bool       `json:"enabled"`
	Valid         bool       `json:"valid"`
	Message       string     `json:"message"`
	LicenseKey    string     `json:"license_key,omitempty"`
	LicenseID     uint       `json:"license_id,omitempty"`
	Plan          string     `json:"plan,omitempty"`
	Customer      string     `json:"customer,omitempty"`
	Domain        string     `json:"domain,omitempty"`
	SiteURL       string     `json:"site_url,omitempty"`
	VerifyURL     string     `json:"verify_url,omitempty"`
	Product       string     `json:"product,omitempty"`
	Channel       string     `json:"channel,omitempty"`
	ClientVersion string     `json:"client_version,omitempty"`
	VersionStatus string     `json:"version_status,omitempty"`
	VersionMsg    string     `json:"version_message,omitempty"`
	LatestVersion string     `json:"latest_version,omitempty"`
	MinNodeclient string     `json:"min_nodeclient_version,omitempty"`
	MaxNodeclient string     `json:"max_nodeclient_version,omitempty"`
	AuthorizedAt  *time.Time `json:"authorized_at,omitempty"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
	LastCheckedAt *time.Time `json:"last_checked_at,omitempty"`
	LastSuccessAt *time.Time `json:"last_success_at,omitempty"`
	MachineID     string     `json:"machine_id,omitempty"`
}

// Manager 运行时授权管理器。
type Manager struct {
	cfg           config.LicenseConfig
	client        *http.Client
	machineID     string
	serverOrigins []string

	mu     sync.RWMutex
	status Status
}

type verifyRequest struct {
	LicenseKey    string `json:"license_key"`
	MachineID     string `json:"machine_id"`
	Hostname      string `json:"hostname,omitempty"`
	MachineName   string `json:"machine_name,omitempty"`
	Action        string `json:"action,omitempty"`
	Product       string `json:"product,omitempty"`
	ClientVersion string `json:"client_version,omitempty"`
	Channel       string `json:"channel,omitempty"`
	Versions      struct {
		Panel      string `json:"panel,omitempty"`
		Backend    string `json:"backend,omitempty"`
		Frontend   string `json:"frontend,omitempty"`
		Nodeclient string `json:"nodeclient,omitempty"`
	} `json:"versions,omitempty"`
	Branch  string `json:"branch,omitempty"`
	Commit  string `json:"commit,omitempty"`
	Domain  string `json:"domain,omitempty"`
	SiteURL string `json:"site_url,omitempty"`
}

type verifyResponse struct {
	Success bool `json:"success"`
	Data    struct {
		// 旧版字段
		Valid     *bool      `json:"valid,omitempty"`
		Message   string     `json:"message,omitempty"`
		LicenseID uint       `json:"license_id,omitempty"`
		Plan      string     `json:"plan,omitempty"`
		Customer  string     `json:"customer,omitempty"`
		ExpiresAt *time.Time `json:"expires_at,omitempty"`

		// 新版字段
		Verified *bool `json:"verified,omitempty"`
		License  struct {
			Valid     *bool      `json:"valid,omitempty"`
			Status    string     `json:"status,omitempty"`
			Message   string     `json:"message,omitempty"`
			LicenseID uint       `json:"license_id,omitempty"`
			PlanCode  string     `json:"plan_code,omitempty"`
			Customer  string     `json:"customer,omitempty"`
			ExpiresAt *time.Time `json:"expires_at,omitempty"`
		} `json:"license,omitempty"`
		Version struct {
			Compatible *bool  `json:"compatible,omitempty"`
			Status     string `json:"status,omitempty"`
			Message    string `json:"message,omitempty"`
			Product    string `json:"product,omitempty"`
			Channel    string `json:"channel,omitempty"`
			Current    string `json:"current_version,omitempty"`
			Latest     string `json:"latest_version,omitempty"`
		} `json:"version,omitempty"`
		VersionPolicy struct {
			MinNodeclientVersion string `json:"min_nodeclient_version,omitempty"`
			MaxNodeclientVersion string `json:"max_nodeclient_version,omitempty"`
		} `json:"version_policy,omitempty"`
	} `json:"data"`
	Message string `json:"message"`
	Error   *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// NewManager 创建授权管理器。
func NewManager(cfg *config.LicenseConfig, serverCfg *config.ServerConfig) *Manager {
	manager := &Manager{
		client: &http.Client{Timeout: 10 * time.Second},
		status: Status{
			Enabled: false,
			Valid:   true,
			Message: "license check disabled",
		},
	}
	if cfg == nil {
		return manager
	}

	manager.cfg = *cfg
	manager.machineID = detectMachineID(strings.TrimSpace(cfg.MachineID))
	if serverCfg != nil {
		manager.serverOrigins = append(manager.serverOrigins, serverCfg.AllowedOrigins...)
	}
	manager.status.Enabled = cfg.Enabled
	manager.status.MachineID = manager.machineID
	manager.status.LicenseKey = strings.TrimSpace(cfg.LicenseKey)
	manager.status.VerifyURL = resolveVerifyURL(cfg.VerifyURL)
	manager.status.Product = normalizedProduct(cfg.Product)
	manager.status.Channel = normalizedChannel(cfg.Channel)
	manager.status.ClientVersion = resolveClientVersion(cfg.ClientVersion)

	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "nodepass-backend"
	}
	domain, siteURL := manager.resolveDomainAndSiteURL(hostname)
	manager.status.Domain = domain
	manager.status.SiteURL = siteURL
	if cfg.Enabled {
		manager.status.Valid = false
		manager.status.Message = "license check pending"
	}
	return manager
}

// Start 启动后台授权检查任务。
func (m *Manager) Start(ctx context.Context) {
	if m == nil || !m.cfg.Enabled {
		return
	}

	if err := m.verify("runtime"); err != nil {
		zap.L().Warn("启动时授权校验失败", zap.Error(err))
	}

	interval := time.Duration(m.cfg.VerifyIntervalSeconds) * time.Second
	if interval <= 0 {
		interval = defaultVerifyInterval
	}
	ticker := time.NewTicker(interval)

	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := m.verify("runtime"); err != nil {
					zap.L().Warn("定时授权校验失败", zap.Error(err))
				}
			}
		}
	}()
}

// Enabled 返回授权检查是否开启。
func (m *Manager) Enabled() bool {
	if m == nil {
		return false
	}
	return m.cfg.Enabled
}

// IsAllowed 返回当前是否允许业务访问。
func (m *Manager) IsAllowed() bool {
	if m == nil || !m.cfg.Enabled {
		return true
	}
	status := m.Status()
	return status.Valid
}

// Status 返回授权状态快照。
func (m *Manager) Status() Status {
	if m == nil {
		return Status{
			Enabled: false,
			Valid:   true,
			Message: "license manager not initialized",
		}
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status
}

func (m *Manager) verify(action string) error {
	now := time.Now().UTC()
	m.mu.RLock()
	cfg := m.cfg
	machineID := m.machineID
	m.mu.RUnlock()

	if strings.TrimSpace(cfg.LicenseKey) == "" {
		m.setFailure(now, "license license_key 未配置", nil)
		return fmt.Errorf("license license_key 未配置")
	}

	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "nodepass-backend"
	}
	domain, siteURL := m.resolveDomainAndSiteURL(hostname)
	verifyURL := resolveVerifyURL(cfg.VerifyURL)
	product := normalizedProduct(cfg.Product)
	channel := normalizedChannel(cfg.Channel)
	clientVersion := resolveClientVersion(cfg.ClientVersion)

	m.mu.Lock()
	m.status.LicenseKey = strings.TrimSpace(cfg.LicenseKey)
	m.status.Domain = domain
	m.status.SiteURL = siteURL
	m.status.VerifyURL = verifyURL
	m.status.Product = product
	m.status.Channel = channel
	m.status.ClientVersion = clientVersion
	m.mu.Unlock()

	if cfg.RequireDomain && !isUsableLicenseDomain(domain) {
		msg := "license.domain/site_url 未配置或无效，请设置可公开访问的生产域名"
		m.setFailure(now, msg, nil)
		return fmt.Errorf(msg)
	}

	reqPayload := verifyRequest{
		LicenseKey:    strings.TrimSpace(cfg.LicenseKey),
		MachineID:     machineID,
		Hostname:      hostname,
		MachineName:   hostname,
		Action:        action,
		Product:       product,
		ClientVersion: clientVersion,
		Channel:       channel,
		Branch:        strings.TrimSpace(os.Getenv("APP_GIT_BRANCH")),
		Commit:        strings.TrimSpace(os.Getenv("APP_GIT_COMMIT")),
		Domain:        domain,
		SiteURL:       siteURL,
	}
	// 兼容旧授权中心：保留 versions 字段。
	reqPayload.Versions.Panel = clientVersion
	reqPayload.Versions.Backend = clientVersion
	reqPayload.Versions.Frontend = clientVersion
	reqPayload.Versions.Nodeclient = clientVersion

	body, err := json.Marshal(reqPayload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, verifyURL, bytes.NewReader(body))
	if err != nil {
		m.setFailure(now, "构建授权请求失败", nil)
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.client.Do(req)
	if err != nil {
		m.handleVerifyRequestError(now, err)
		return fmt.Errorf("授权中心连接失败")
	}
	defer resp.Body.Close()

	var parsed verifyResponse
	if decodeErr := json.NewDecoder(resp.Body).Decode(&parsed); decodeErr != nil {
		m.handleVerifyRequestError(now, decodeErr)
		return fmt.Errorf("授权响应解析失败")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := fmt.Sprintf("授权接口状态码异常: %d", resp.StatusCode)
		if parsed.Error != nil && strings.TrimSpace(parsed.Error.Message) != "" {
			message = parsed.Error.Message
		}
		m.setFailure(now, message, &parsed)
		return fmt.Errorf(message)
	}

	passed, message := evaluateVerifyResult(&parsed)
	if !passed {
		m.setFailure(now, message, &parsed)
		return fmt.Errorf(message)
	}

	m.mu.Lock()
	m.status.Valid = true
	m.status.Message = message
	m.status.LastCheckedAt = &now
	m.status.LastSuccessAt = &now
	m.status.AuthorizedAt = &now
	applyParsedStatus(&m.status, &parsed)
	m.mu.Unlock()

	return nil
}

func resolveRuntimeVersion() string {
	compiled := strings.TrimSpace(version.Version)
	if !isDevLikeVersion(compiled) {
		return compiled
	}

	candidates := []string{
		strings.TrimSpace(os.Getenv("NODEPASS_RUNTIME_VERSION")),
		strings.TrimSpace(os.Getenv("BACKEND_VERSION")),
		strings.TrimSpace(os.Getenv("PANEL_VERSION")),
		strings.TrimSpace(os.Getenv("APP_VERSION")),
	}
	for _, candidate := range candidates {
		if !isDevLikeVersion(candidate) {
			return candidate
		}
	}
	return "0.1.0"
}

func resolveClientVersion(configured string) string {
	if trimmed := strings.TrimSpace(configured); trimmed != "" {
		return trimmed
	}
	return resolveRuntimeVersion()
}

func normalizedProduct(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "backend"
	}
	return trimmed
}

func normalizedChannel(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "stable"
	}
	return trimmed
}

func isDevLikeVersion(raw string) bool {
	value := strings.TrimSpace(strings.ToLower(raw))
	if value == "" {
		return true
	}
	if value == "dev" || value == "devel" || value == "development" {
		return true
	}
	return strings.HasPrefix(value, "dev-")
}

func evaluateVerifyResult(parsed *verifyResponse) (bool, string) {
	if parsed == nil {
		return false, "授权响应为空"
	}

	messageCandidates := []string{
		strings.TrimSpace(parsed.Data.Message),
		strings.TrimSpace(parsed.Data.License.Message),
		strings.TrimSpace(parsed.Data.Version.Message),
		strings.TrimSpace(parsed.Message),
	}
	message := "授权校验失败"
	for _, candidate := range messageCandidates {
		if candidate != "" {
			message = candidate
			break
		}
	}

	licenseValid := true
	if parsed.Data.License.Valid != nil {
		licenseValid = *parsed.Data.License.Valid
	} else if parsed.Data.Valid != nil {
		licenseValid = *parsed.Data.Valid
	}

	versionCompatible := true
	if parsed.Data.Version.Compatible != nil {
		versionCompatible = *parsed.Data.Version.Compatible
	}

	verified := licenseValid && versionCompatible
	if parsed.Data.Verified != nil {
		verified = *parsed.Data.Verified
	}

	if verified {
		if message == "授权校验失败" {
			message = "license verify passed"
		}
		return true, message
	}
	return false, message
}

func applyParsedStatus(dst *Status, parsed *verifyResponse) {
	if dst == nil || parsed == nil {
		return
	}

	if parsed.Data.License.LicenseID > 0 {
		dst.LicenseID = parsed.Data.License.LicenseID
	} else if parsed.Data.LicenseID > 0 {
		dst.LicenseID = parsed.Data.LicenseID
	}

	if plan := strings.TrimSpace(parsed.Data.License.PlanCode); plan != "" {
		dst.Plan = plan
	} else if plan := strings.TrimSpace(parsed.Data.Plan); plan != "" {
		dst.Plan = plan
	}

	if customer := strings.TrimSpace(parsed.Data.License.Customer); customer != "" {
		dst.Customer = customer
	} else if customer := strings.TrimSpace(parsed.Data.Customer); customer != "" {
		dst.Customer = customer
	}

	if parsed.Data.License.ExpiresAt != nil {
		dst.ExpiresAt = parsed.Data.License.ExpiresAt
	} else if parsed.Data.ExpiresAt != nil {
		dst.ExpiresAt = parsed.Data.ExpiresAt
	}

	if status := strings.TrimSpace(parsed.Data.Version.Status); status != "" {
		dst.VersionStatus = status
	}
	if msg := strings.TrimSpace(parsed.Data.Version.Message); msg != "" {
		dst.VersionMsg = msg
	}
	if current := strings.TrimSpace(parsed.Data.Version.Current); current != "" {
		dst.ClientVersion = current
	}
	if latest := strings.TrimSpace(parsed.Data.Version.Latest); latest != "" {
		dst.LatestVersion = latest
	}
	if product := strings.TrimSpace(parsed.Data.Version.Product); product != "" {
		dst.Product = product
	}
	if channel := strings.TrimSpace(parsed.Data.Version.Channel); channel != "" {
		dst.Channel = channel
	}

	// 运行时保存 nodeclient 版本策略，供节点心跳快速校验。
	dst.MinNodeclient = strings.TrimSpace(parsed.Data.VersionPolicy.MinNodeclientVersion)
	dst.MaxNodeclient = strings.TrimSpace(parsed.Data.VersionPolicy.MaxNodeclientVersion)
}

func (m *Manager) handleVerifyRequestError(now time.Time, requestErr error) {
	m.mu.RLock()
	failOpen := m.cfg.FailOpen
	graceSeconds := m.cfg.OfflineGraceSeconds
	m.mu.RUnlock()

	m.mu.RLock()
	lastSuccess := m.status.LastSuccessAt
	m.mu.RUnlock()

	if failOpen && lastSuccess != nil {
		msg := "授权中心不可达，fail_open 生效"
		m.setPass(now, msg, lastSuccess)
		return
	}

	if graceSeconds < 0 {
		graceSeconds = 0
	}
	if graceSeconds > 0 && lastSuccess != nil {
		if now.Sub(*lastSuccess) <= time.Duration(graceSeconds)*time.Second {
			msg := "授权中心不可达，处于离线宽限期"
			m.setPass(now, msg, lastSuccess)
			return
		}
	}

	_ = requestErr // 避免将底层网络错误原样暴露到状态消息
	m.setFailure(now, "授权校验失败", nil)
}

func (m *Manager) setPass(now time.Time, message string, lastSuccess *time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.status.Valid = true
	m.status.Message = message
	m.status.LastCheckedAt = &now
	if lastSuccess != nil {
		m.status.LastSuccessAt = lastSuccess
		m.status.AuthorizedAt = lastSuccess
	}
}

func (m *Manager) setFailure(now time.Time, message string, parsed *verifyResponse) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.status.Valid = false
	m.status.Message = message
	m.status.LastCheckedAt = &now
	if parsed != nil {
		applyParsedStatus(&m.status, parsed)
	}
}

// ValidateNodeclientVersion 按运行时授权下发的版本策略校验节点端版本。
func (m *Manager) ValidateNodeclientVersion(clientVersion string) error {
	versionValue := strings.TrimSpace(clientVersion)
	if versionValue == "" {
		return fmt.Errorf("client_version 不能为空")
	}
	if m == nil || !m.Enabled() {
		return nil
	}

	status := m.Status()
	minVersion := strings.TrimSpace(status.MinNodeclient)
	maxVersion := strings.TrimSpace(status.MaxNodeclient)
	if minVersion == "" && maxVersion == "" {
		return nil
	}

	if minVersion != "" {
		cmp, err := compareSemanticVersion(versionValue, minVersion)
		if err != nil {
			return fmt.Errorf("版本比较失败: %w", err)
		}
		if cmp < 0 {
			return fmt.Errorf("nodeclient 版本过低: current=%s, min=%s", versionValue, minVersion)
		}
	}

	if maxVersion != "" {
		cmp, err := compareSemanticVersion(versionValue, maxVersion)
		if err != nil {
			return fmt.Errorf("版本比较失败: %w", err)
		}
		if cmp > 0 {
			return fmt.Errorf("nodeclient 版本过高: current=%s, max=%s", versionValue, maxVersion)
		}
	}

	return nil
}

func compareSemanticVersion(current string, target string) (int, error) {
	left, err := parseSemanticVersion(current)
	if err != nil {
		return 0, fmt.Errorf("当前版本 %q 非法: %w", current, err)
	}
	right, err := parseSemanticVersion(target)
	if err != nil {
		return 0, fmt.Errorf("策略版本 %q 非法: %w", target, err)
	}

	for i := 0; i < len(left); i++ {
		if left[i] < right[i] {
			return -1, nil
		}
		if left[i] > right[i] {
			return 1, nil
		}
	}
	return 0, nil
}

func parseSemanticVersion(raw string) ([3]int, error) {
	var result [3]int
	trimmed := strings.TrimSpace(strings.ToLower(raw))
	if trimmed == "" {
		return result, fmt.Errorf("版本为空")
	}

	trimmed = strings.TrimPrefix(trimmed, "v")
	if idx := strings.Index(trimmed, "-"); idx >= 0 {
		trimmed = trimmed[:idx]
	}
	if idx := strings.Index(trimmed, "+"); idx >= 0 {
		trimmed = trimmed[:idx]
	}

	parts := strings.Split(trimmed, ".")
	if len(parts) == 0 {
		return result, fmt.Errorf("版本格式无效")
	}

	for i := 0; i < len(result) && i < len(parts); i++ {
		part := strings.TrimSpace(parts[i])
		if part == "" {
			return result, fmt.Errorf("版本段为空")
		}
		value, err := strconv.Atoi(part)
		if err != nil {
			return result, fmt.Errorf("版本段 %q 非数字", part)
		}
		if value < 0 {
			return result, fmt.Errorf("版本段 %q 非法", part)
		}
		result[i] = value
	}

	return result, nil
}

func detectMachineID(configured string) string {
	if strings.TrimSpace(configured) != "" {
		return strings.TrimSpace(configured)
	}
	if bytes, err := os.ReadFile("/etc/machine-id"); err == nil {
		value := strings.TrimSpace(string(bytes))
		if value != "" {
			return value
		}
	}
	if bytes, err := os.ReadFile("/var/lib/dbus/machine-id"); err == nil {
		value := strings.TrimSpace(string(bytes))
		if value != "" {
			return value
		}
	}
	hostname, _ := os.Hostname()
	if strings.TrimSpace(hostname) != "" {
		return strings.TrimSpace(hostname)
	}
	return "unknown-machine"
}

// UpdateDomain 更新授权域名并在授权开启时触发重新校验。
func (m *Manager) UpdateDomain(domain string, siteURL string) (Status, error) {
	if m == nil {
		return Status{
			Enabled: false,
			Valid:   true,
			Message: "license manager not initialized",
		}, fmt.Errorf("license manager not initialized")
	}

	normalizedDomain := extractHost(domain)
	normalizedSiteURL := strings.TrimSpace(siteURL)
	if normalizedDomain == "" {
		normalizedDomain = extractHost(normalizedSiteURL)
	}

	if normalizedDomain == "" && normalizedSiteURL == "" {
		return m.Status(), fmt.Errorf("domain 与 site_url 至少提供一个")
	}
	if normalizedDomain != "" && !isUsableLicenseDomain(normalizedDomain) {
		return m.Status(), fmt.Errorf("domain 无效，请使用可公开访问的生产域名")
	}
	if normalizedSiteURL == "" && normalizedDomain != "" {
		normalizedSiteURL = fmt.Sprintf("https://%s", normalizedDomain)
	}

	m.mu.Lock()
	m.cfg.Domain = normalizedDomain
	m.cfg.SiteURL = normalizedSiteURL
	m.status.Domain = normalizedDomain
	m.status.SiteURL = normalizedSiteURL
	m.mu.Unlock()

	if !m.Enabled() {
		return m.Status(), nil
	}

	if err := m.verify("domain_change"); err != nil {
		return m.Status(), err
	}
	return m.Status(), nil
}

func (m *Manager) resolveDomainAndSiteURL(hostname string) (string, string) {
	m.mu.RLock()
	rawDomain := m.cfg.Domain
	rawSiteURL := m.cfg.SiteURL
	origins := append([]string(nil), m.serverOrigins...)
	m.mu.RUnlock()

	domain := extractHost(rawDomain)
	siteURL := strings.TrimSpace(rawSiteURL)

	if domain == "" {
		domain = extractHost(siteURL)
	}
	if domain == "" {
		for _, origin := range origins {
			host := extractHost(origin)
			if !isUsableLicenseDomain(host) {
				continue
			}
			domain = host
			break
		}
	}

	if siteURL == "" && domain != "" {
		siteURL = fmt.Sprintf("https://%s", domain)
	}

	return domain, siteURL
}

func extractHost(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "*.") {
		return strings.ToLower(raw)
	}

	if strings.Contains(raw, "://") {
		u, err := url.Parse(raw)
		if err != nil {
			return ""
		}
		return normalizeHost(u.Hostname())
	}

	host := raw
	if strings.Contains(host, "/") {
		host = strings.SplitN(host, "/", 2)[0]
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		return normalizeHost(h)
	}
	return normalizeHost(host)
}

func normalizeHost(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

func isLocalDomain(host string) bool {
	host = extractHost(host)
	if host == "" {
		return true
	}
	if host == "localhost" || host == "::1" || host == "0.0.0.0" {
		return true
	}
	if strings.HasPrefix(host, "127.") {
		return true
	}
	if strings.HasSuffix(host, ".local") || strings.HasSuffix(host, ".test") {
		return true
	}
	return false
}

func isUsableLicenseDomain(host string) bool {
	host = extractHost(host)
	if host == "" {
		return false
	}
	if host == "*" || strings.Contains(host, "*") {
		return false
	}
	return !isLocalDomain(host)
}
