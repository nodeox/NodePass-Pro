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
	"strings"
	"sync"
	"time"

	"nodepass-pro/backend/internal/config"
	"nodepass-pro/backend/internal/version"

	"go.uber.org/zap"
)

const defaultVerifyInterval = 300 * time.Second

// Status 运行时授权状态。
type Status struct {
	Enabled       bool       `json:"enabled"`
	Valid         bool       `json:"valid"`
	Message       string     `json:"message"`
	LicenseID     uint       `json:"license_id,omitempty"`
	Plan          string     `json:"plan,omitempty"`
	Customer      string     `json:"customer,omitempty"`
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
	LicenseKey  string `json:"license_key"`
	MachineID   string `json:"machine_id"`
	MachineName string `json:"machine_name"`
	Action      string `json:"action"`
	Versions    struct {
		Panel      string `json:"panel"`
		Backend    string `json:"backend"`
		Frontend   string `json:"frontend"`
		Nodeclient string `json:"nodeclient"`
	} `json:"versions"`
	Branch  string `json:"branch,omitempty"`
	Commit  string `json:"commit,omitempty"`
	Domain  string `json:"domain,omitempty"`
	SiteURL string `json:"site_url,omitempty"`
}

type verifyResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Valid     bool       `json:"valid"`
		Message   string     `json:"message"`
		LicenseID uint       `json:"license_id"`
		Plan      string     `json:"plan"`
		Customer  string     `json:"customer"`
		ExpiresAt *time.Time `json:"expires_at"`
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
	if strings.TrimSpace(m.cfg.VerifyURL) == "" || strings.TrimSpace(m.cfg.LicenseKey) == "" {
		m.setFailure(now, "license verify_url/license_key 未配置", nil)
		return fmt.Errorf("license verify_url/license_key 未配置")
	}

	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "nodepass-backend"
	}
	domain, siteURL := m.resolveDomainAndSiteURL(hostname)
	if !isUsableLicenseDomain(domain) {
		msg := "license.domain/site_url 未配置或无效，请设置可公开访问的生产域名"
		m.setFailure(now, msg, nil)
		return fmt.Errorf(msg)
	}

	reqPayload := verifyRequest{
		LicenseKey:  strings.TrimSpace(m.cfg.LicenseKey),
		MachineID:   m.machineID,
		MachineName: hostname,
		Action:      action,
		Branch:      strings.TrimSpace(os.Getenv("APP_GIT_BRANCH")),
		Commit:      strings.TrimSpace(os.Getenv("APP_GIT_COMMIT")),
		Domain:      domain,
		SiteURL:     siteURL,
	}
	reqPayload.Versions.Panel = version.Version
	reqPayload.Versions.Backend = version.Version
	reqPayload.Versions.Frontend = version.Version
	reqPayload.Versions.Nodeclient = version.Version

	body, err := json.Marshal(reqPayload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, strings.TrimSpace(m.cfg.VerifyURL), bytes.NewReader(body))
	if err != nil {
		m.setFailure(now, "构建授权请求失败", nil)
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.client.Do(req)
	if err != nil {
		m.handleVerifyRequestError(now, err)
		return err
	}
	defer resp.Body.Close()

	var parsed verifyResponse
	if decodeErr := json.NewDecoder(resp.Body).Decode(&parsed); decodeErr != nil {
		m.handleVerifyRequestError(now, decodeErr)
		return decodeErr
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := fmt.Sprintf("授权接口状态码异常: %d", resp.StatusCode)
		if parsed.Error != nil && strings.TrimSpace(parsed.Error.Message) != "" {
			message = parsed.Error.Message
		}
		m.setFailure(now, message, nil)
		return fmt.Errorf(message)
	}

	message := strings.TrimSpace(parsed.Data.Message)
	if message == "" {
		message = strings.TrimSpace(parsed.Message)
	}
	if message == "" {
		message = "license verify passed"
	}

	if !parsed.Data.Valid {
		m.setFailure(now, message, &parsed)
		return fmt.Errorf(message)
	}

	m.mu.Lock()
	m.status.Valid = true
	m.status.Message = message
	m.status.LicenseID = parsed.Data.LicenseID
	m.status.Plan = parsed.Data.Plan
	m.status.Customer = parsed.Data.Customer
	m.status.ExpiresAt = parsed.Data.ExpiresAt
	m.status.LastCheckedAt = &now
	m.status.LastSuccessAt = &now
	m.mu.Unlock()

	return nil
}

func (m *Manager) handleVerifyRequestError(now time.Time, requestErr error) {
	m.mu.RLock()
	lastSuccess := m.status.LastSuccessAt
	m.mu.RUnlock()

	if m.cfg.FailOpen && lastSuccess != nil {
		msg := fmt.Sprintf("授权中心不可达，fail_open 生效: %v", requestErr)
		m.setPass(now, msg, lastSuccess)
		return
	}

	graceSeconds := m.cfg.OfflineGraceSeconds
	if graceSeconds < 0 {
		graceSeconds = 0
	}
	if graceSeconds > 0 && lastSuccess != nil {
		if now.Sub(*lastSuccess) <= time.Duration(graceSeconds)*time.Second {
			msg := fmt.Sprintf("授权中心不可达，处于离线宽限期: %v", requestErr)
			m.setPass(now, msg, lastSuccess)
			return
		}
	}

	m.setFailure(now, fmt.Sprintf("授权校验失败: %v", requestErr), nil)
}

func (m *Manager) setPass(now time.Time, message string, lastSuccess *time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.status.Valid = true
	m.status.Message = message
	m.status.LastCheckedAt = &now
	if lastSuccess != nil {
		m.status.LastSuccessAt = lastSuccess
	}
}

func (m *Manager) setFailure(now time.Time, message string, parsed *verifyResponse) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.status.Valid = false
	m.status.Message = message
	m.status.LastCheckedAt = &now
	if parsed != nil {
		m.status.LicenseID = parsed.Data.LicenseID
		m.status.Plan = parsed.Data.Plan
		m.status.Customer = parsed.Data.Customer
		m.status.ExpiresAt = parsed.Data.ExpiresAt
	}
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

func (m *Manager) resolveDomainAndSiteURL(hostname string) (string, string) {
	domain := extractHost(m.cfg.Domain)
	siteURL := strings.TrimSpace(m.cfg.SiteURL)

	if domain == "" {
		domain = extractHost(siteURL)
	}
	if domain == "" {
		for _, origin := range m.serverOrigins {
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
