package license

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"nodepass-pro/nodeclient/internal/infra/config"
	"nodepass-pro/nodeclient/internal/domain/interfaces"
)

// Verifier nodeclient 统一授权校验器。
type Verifier struct {
	cfg       *config.Config
	client    *http.Client
	machineID string
	hostname  string
}

type verifyRequest struct {
	LicenseKey    string `json:"license_key"`
	MachineID     string `json:"machine_id"`
	Hostname      string `json:"hostname"`
	Product       string `json:"product"`
	ClientVersion string `json:"client_version"`
	Channel       string `json:"channel"`
}

type verifyResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Verified bool   `json:"verified"`
		Status   string `json:"status"`
		License  struct {
			LicenseID uint   `json:"license_id"`
			PlanCode  string `json:"plan_code"`
			Customer  string `json:"customer"`
			ExpiresAt string `json:"expires_at"`
			Message   string `json:"message"`
		} `json:"license"`
		Version struct {
			Status  string `json:"status"`
			Message string `json:"message"`
		} `json:"version"`
	} `json:"data"`
	Message string `json:"message"`
	Error   *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// NewVerifier 创建授权校验器（license_enabled=false 时返回 nil）。
func NewVerifier(cfg *config.Config) *Verifier {
	if cfg == nil || !cfg.LicenseEnabled {
		return nil
	}

	hostname, _ := os.Hostname()
	if strings.TrimSpace(hostname) == "" {
		hostname = "nodeclient"
	}

	return &Verifier{
		cfg: cfg,
		client: &http.Client{
			Timeout: time.Duration(cfg.LicenseTimeout) * time.Second,
		},
		machineID: detectMachineID(cfg.LicenseMachineID, cfg.NodeID, hostname),
		hostname:  hostname,
	}
}

// Enabled 是否启用授权校验。
func (v *Verifier) Enabled() bool {
	return v != nil && v.cfg != nil && v.cfg.LicenseEnabled
}

// Verify 执行一次统一授权校验。
func (v *Verifier) Verify(currentVersion string) (*interfaces.VerifyStatus, error) {
	if !v.Enabled() {
		return &interfaces.VerifyStatus{Allowed: true, Message: "license check disabled", Status: "disabled"}, nil
	}

	payload := verifyRequest{
		LicenseKey:    strings.TrimSpace(v.cfg.LicenseKey),
		MachineID:     v.machineID,
		Hostname:      v.hostname,
		Product:       normalizedProduct(v.cfg.LicenseProduct),
		ClientVersion: strings.TrimSpace(currentVersion),
		Channel:       normalizedChannel(v.cfg.LicenseChannel),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, strings.TrimSpace(v.cfg.LicenseVerifyURL), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := v.client.Do(req)
	if err != nil {
		if v.cfg.LicenseFailOpen {
			return &interfaces.VerifyStatus{Allowed: true, Message: "授权中心不可达，license_fail_open 生效", Status: "fail_open"}, nil
		}
		return nil, fmt.Errorf("请求授权中心失败: %w", err)
	}
	defer resp.Body.Close()

	var parsed verifyResponse
	if err = json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		if v.cfg.LicenseFailOpen {
			return &interfaces.VerifyStatus{Allowed: true, Message: "授权响应解析失败，license_fail_open 生效", Status: "fail_open"}, nil
		}
		return nil, fmt.Errorf("解析授权响应失败: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := fmt.Sprintf("授权接口状态码异常: %d", resp.StatusCode)
		if parsed.Error != nil && strings.TrimSpace(parsed.Error.Message) != "" {
			msg = parsed.Error.Message
		}
		if v.cfg.LicenseFailOpen {
			return &interfaces.VerifyStatus{Allowed: true, Message: msg + "（license_fail_open）", Status: "fail_open"}, nil
		}
		return nil, fmt.Errorf(msg)
	}

	status := &interfaces.VerifyStatus{
		Allowed:       parsed.Success && parsed.Data.Verified,
		Message:       resolvedMessage(&parsed),
		Status:        strings.TrimSpace(parsed.Data.Status),
		LicenseID:     parsed.Data.License.LicenseID,
		Plan:          strings.TrimSpace(parsed.Data.License.PlanCode),
		Customer:      strings.TrimSpace(parsed.Data.License.Customer),
		ExpiresAt:     strings.TrimSpace(parsed.Data.License.ExpiresAt),
		VersionStatus: strings.TrimSpace(parsed.Data.Version.Status),
	}

	if status.Allowed {
		return status, nil
	}
	return status, fmt.Errorf(status.Message)
}

func resolvedMessage(parsed *verifyResponse) string {
	if parsed == nil {
		return "授权响应为空"
	}
	candidates := []string{
		strings.TrimSpace(parsed.Data.License.Message),
		strings.TrimSpace(parsed.Data.Status),
		strings.TrimSpace(parsed.Message),
		strings.TrimSpace(parsed.Data.Version.Message),
	}
	for _, candidate := range candidates {
		if candidate != "" {
			return candidate
		}
	}
	if parsed.Data.Verified {
		return "授权校验通过"
	}
	return "授权校验失败"
}

func normalizedProduct(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "nodeclient"
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

func detectMachineID(configured string, fallbackNodeID string, hostname string) string {
	if trimmed := strings.TrimSpace(configured); trimmed != "" {
		return trimmed
	}
	if bytes, err := os.ReadFile("/etc/machine-id"); err == nil {
		if value := strings.TrimSpace(string(bytes)); value != "" {
			return value
		}
	}
	if bytes, err := os.ReadFile("/var/lib/dbus/machine-id"); err == nil {
		if value := strings.TrimSpace(string(bytes)); value != "" {
			return value
		}
	}
	if trimmed := strings.TrimSpace(fallbackNodeID); trimmed != "" {
		return trimmed
	}
	if trimmed := strings.TrimSpace(hostname); trimmed != "" {
		return trimmed
	}
	return "unknown-nodeclient"
}
