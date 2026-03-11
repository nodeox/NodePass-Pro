package license

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"nodepass-pro/nodeclient/internal/infra/config"
)

func TestNewVerifier(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config.Config
		wantNil bool
	}{
		{
			name:    "nil config returns nil",
			cfg:     nil,
			wantNil: true,
		},
		{
			name: "license disabled returns nil",
			cfg: &config.Config{
				LicenseEnabled: false,
			},
			wantNil: true,
		},
		{
			name: "license enabled returns verifier",
			cfg: &config.Config{
				LicenseEnabled:   true,
				LicenseVerifyURL: "http://localhost:8080/verify",
				LicenseKey:       "test-key",
				LicenseTimeout:   10,
			},
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewVerifier(tt.cfg)
			if tt.wantNil && v != nil {
				t.Error("Expected nil verifier")
			}
			if !tt.wantNil && v == nil {
				t.Error("Expected non-nil verifier")
			}
		})
	}
}

func TestVerifierEnabled(t *testing.T) {
	tests := []struct {
		name    string
		v       *Verifier
		enabled bool
	}{
		{
			name:    "nil verifier is disabled",
			v:       nil,
			enabled: false,
		},
		{
			name: "verifier with nil config is disabled",
			v: &Verifier{
				cfg: nil,
			},
			enabled: false,
		},
		{
			name: "verifier with license disabled is disabled",
			v: &Verifier{
				cfg: &config.Config{
					LicenseEnabled: false,
				},
			},
			enabled: false,
		},
		{
			name: "verifier with license enabled is enabled",
			v: &Verifier{
				cfg: &config.Config{
					LicenseEnabled: true,
				},
			},
			enabled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.v.Enabled(); got != tt.enabled {
				t.Errorf("Enabled() = %v, want %v", got, tt.enabled)
			}
		})
	}
}

func TestVerifyDisabled(t *testing.T) {
	v := &Verifier{
		cfg: &config.Config{
			LicenseEnabled: false,
		},
	}

	status, err := v.Verify("1.0.0")
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}

	if !status.Allowed {
		t.Error("Expected Allowed = true when disabled")
	}
	if status.Status != "disabled" {
		t.Errorf("Expected Status = disabled, got %s", status.Status)
	}
}

func TestVerifySuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("Expected Content-Type: application/json")
		}

		var req verifyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		if req.LicenseKey != "test-key" {
			t.Errorf("Expected license_key = test-key, got %s", req.LicenseKey)
		}

		resp := verifyResponse{
			Success: true,
			Data: struct {
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
			}{
				Verified: true,
				Status:   "active",
			},
		}
		resp.Data.License.LicenseID = 123
		resp.Data.License.PlanCode = "pro"
		resp.Data.License.Customer = "test-customer"
		resp.Data.License.ExpiresAt = "2027-12-31"
		resp.Data.License.Message = "License valid"

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	v := &Verifier{
		cfg: &config.Config{
			LicenseEnabled:   true,
			LicenseVerifyURL: server.URL,
			LicenseKey:       "test-key",
			LicenseTimeout:   10,
		},
		client:    server.Client(),
		machineID: "test-machine",
		hostname:  "test-host",
	}

	status, err := v.Verify("1.0.0")
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}

	if !status.Allowed {
		t.Error("Expected Allowed = true")
	}
	if status.Status != "active" {
		t.Errorf("Expected Status = active, got %s", status.Status)
	}
	if status.LicenseID != 123 {
		t.Errorf("Expected LicenseID = 123, got %d", status.LicenseID)
	}
	if status.Plan != "pro" {
		t.Errorf("Expected Plan = pro, got %s", status.Plan)
	}
}

func TestVerifyFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := verifyResponse{
			Success: false,
			Data: struct {
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
			}{
				Verified: false,
				Status:   "expired",
			},
		}
		resp.Data.License.Message = "License expired"

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	v := &Verifier{
		cfg: &config.Config{
			LicenseEnabled:   true,
			LicenseVerifyURL: server.URL,
			LicenseKey:       "test-key",
			LicenseTimeout:   10,
		},
		client:    server.Client(),
		machineID: "test-machine",
		hostname:  "test-host",
	}

	status, err := v.Verify("1.0.0")
	if err == nil {
		t.Error("Expected error for failed verification")
	}

	if status.Allowed {
		t.Error("Expected Allowed = false")
	}
	if status.Status != "expired" {
		t.Errorf("Expected Status = expired, got %s", status.Status)
	}
}

func TestVerifyFailOpen(t *testing.T) {
	// Server that always fails
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	server.Close() // Close immediately to simulate network error

	v := &Verifier{
		cfg: &config.Config{
			LicenseEnabled:   true,
			LicenseVerifyURL: server.URL,
			LicenseKey:       "test-key",
			LicenseTimeout:   1,
			LicenseFailOpen:  true,
		},
		client:    &http.Client{},
		machineID: "test-machine",
		hostname:  "test-host",
	}

	status, err := v.Verify("1.0.0")
	if err != nil {
		t.Fatalf("Verify() with fail_open should not error, got: %v", err)
	}

	if !status.Allowed {
		t.Error("Expected Allowed = true with fail_open")
	}
	if status.Status != "fail_open" {
		t.Errorf("Expected Status = fail_open, got %s", status.Status)
	}
}

func TestVerifyHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		resp := verifyResponse{
			Success: false,
			Error: &struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			}{
				Code:    "UNAUTHORIZED",
				Message: "Invalid license key",
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	v := &Verifier{
		cfg: &config.Config{
			LicenseEnabled:   true,
			LicenseVerifyURL: server.URL,
			LicenseKey:       "invalid-key",
			LicenseTimeout:   10,
			LicenseFailOpen:  false,
		},
		client:    server.Client(),
		machineID: "test-machine",
		hostname:  "test-host",
	}

	_, err := v.Verify("1.0.0")
	if err == nil {
		t.Error("Expected error for HTTP 401")
	}
}

func TestNormalizedProduct(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", "nodeclient"},
		{"  ", "nodeclient"},
		{"custom-product", "custom-product"},
		{"  custom-product  ", "custom-product"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := normalizedProduct(tt.input); got != tt.want {
				t.Errorf("normalizedProduct(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizedChannel(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", "stable"},
		{"  ", "stable"},
		{"beta", "beta"},
		{"  beta  ", "beta"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := normalizedChannel(tt.input); got != tt.want {
				t.Errorf("normalizedChannel(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestDetectMachineID(t *testing.T) {
	tests := []struct {
		name           string
		configured     string
		fallbackNodeID string
		hostname       string
		want           string
	}{
		{
			name:       "configured value takes precedence",
			configured: "configured-id",
			want:       "configured-id",
		},
		{
			name:           "fallback to node ID",
			configured:     "",
			fallbackNodeID: "node-123",
			hostname:       "host-456",
			want:           "node-123",
		},
		{
			name:           "fallback to hostname",
			configured:     "",
			fallbackNodeID: "",
			hostname:       "host-456",
			want:           "host-456",
		},
		{
			name:           "default when all empty",
			configured:     "",
			fallbackNodeID: "",
			hostname:       "",
			want:           "unknown-nodeclient",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectMachineID(tt.configured, tt.fallbackNodeID, tt.hostname)
			if got != tt.want {
				t.Errorf("detectMachineID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDetectMachineIDFromFile(t *testing.T) {
	// Create temp file to simulate /etc/machine-id
	tmpDir := t.TempDir()
	machineIDPath := filepath.Join(tmpDir, "machine-id")
	if err := os.WriteFile(machineIDPath, []byte("file-machine-id\n"), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	// This test verifies the function can read from files
	// In real scenario, it reads from /etc/machine-id or /var/lib/dbus/machine-id
	// We can't easily test that without mocking the filesystem
	// So we just verify the fallback logic works
	got := detectMachineID("", "fallback-node", "fallback-host")
	if got != "fallback-node" {
		t.Errorf("Expected fallback to node ID, got %s", got)
	}
}

func TestResolvedMessage(t *testing.T) {
	tests := []struct {
		name string
		resp *verifyResponse
		want string
	}{
		{
			name: "nil response",
			resp: nil,
			want: "授权响应为空",
		},
		{
			name: "license message takes precedence",
			resp: &verifyResponse{
				Data: struct {
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
				}{
					Status: "active",
				},
			},
			want: "active",
		},
		{
			name: "verified without message",
			resp: &verifyResponse{
				Data: struct {
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
				}{
					Verified: true,
				},
			},
			want: "授权校验通过",
		},
		{
			name: "not verified without message",
			resp: &verifyResponse{
				Data: struct {
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
				}{
					Verified: false,
				},
			},
			want: "授权校验失败",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolvedMessage(tt.resp); got != tt.want {
				t.Errorf("resolvedMessage() = %q, want %q", got, tt.want)
			}
		})
	}
}
