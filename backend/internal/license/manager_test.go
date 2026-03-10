package license

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"nodepass-pro/backend/internal/config"
)

func TestFixedLicenseVerifyURL(t *testing.T) {
	const expected = "https://key.hahaha.ooo/api/v1/verify"
	if got := fixedLicenseVerifyURL(); got != expected {
		t.Fatalf("verify url mismatch: got=%q want=%q", got, expected)
	}
}

func TestResolveVerifyURL(t *testing.T) {
	custom := "http://127.0.0.1:8091/api/v1/verify"
	if got := resolveVerifyURL(custom); got != custom {
		t.Fatalf("resolve verify url mismatch: got=%q want=%q", got, custom)
	}
	if got := resolveVerifyURL(""); got != fixedLicenseVerifyURL() {
		t.Fatalf("resolve verify url fallback mismatch: got=%q want=%q", got, fixedLicenseVerifyURL())
	}
}

func TestResolveDomainAndSiteURL(t *testing.T) {
	tests := []struct {
		name        string
		licenseCfg  config.LicenseConfig
		serverCfg   config.ServerConfig
		wantDomain  string
		wantSiteURL string
	}{
		{
			name: "explicit domain",
			licenseCfg: config.LicenseConfig{
				Domain: "panel.example.com",
			},
			wantDomain:  "panel.example.com",
			wantSiteURL: "https://panel.example.com",
		},
		{
			name: "explicit domain should normalize host from url",
			licenseCfg: config.LicenseConfig{
				Domain: "https://panel.example.com:8443/path",
			},
			wantDomain:  "panel.example.com",
			wantSiteURL: "https://panel.example.com",
		},
		{
			name: "local domain with port should stay local host",
			licenseCfg: config.LicenseConfig{
				Domain: "localhost:3000",
			},
			wantDomain:  "localhost",
			wantSiteURL: "https://localhost",
		},
		{
			name: "local domain url should stay local host",
			licenseCfg: config.LicenseConfig{
				Domain: "https://localhost",
			},
			wantDomain:  "localhost",
			wantSiteURL: "https://localhost",
		},
		{
			name: "from site url",
			licenseCfg: config.LicenseConfig{
				SiteURL: "https://panel.example.com/path",
			},
			wantDomain:  "panel.example.com",
			wantSiteURL: "https://panel.example.com/path",
		},
		{
			name: "from non-local allowed origins",
			serverCfg: config.ServerConfig{
				AllowedOrigins: []string{"localhost", "https://panel.example.com"},
			},
			wantDomain:  "panel.example.com",
			wantSiteURL: "https://panel.example.com",
		},
		{
			name: "local origins should not be used",
			serverCfg: config.ServerConfig{
				AllowedOrigins: []string{"localhost", "127.0.0.1"},
			},
			wantDomain:  "",
			wantSiteURL: "",
		},
		{
			name: "wildcard origin with prefix should not infer domain",
			serverCfg: config.ServerConfig{
				AllowedOrigins: []string{"*.example.com"},
			},
			wantDomain:  "",
			wantSiteURL: "",
		},
		{
			name: "wildcard origin star should not infer domain",
			serverCfg: config.ServerConfig{
				AllowedOrigins: []string{"*"},
			},
			wantDomain:  "",
			wantSiteURL: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewManager(&tt.licenseCfg, &tt.serverCfg)
			gotDomain, gotSiteURL := manager.resolveDomainAndSiteURL("nodepass-backend")
			if gotDomain != tt.wantDomain {
				t.Fatalf("domain mismatch: got=%q want=%q", gotDomain, tt.wantDomain)
			}
			if gotSiteURL != tt.wantSiteURL {
				t.Fatalf("site_url mismatch: got=%q want=%q", gotSiteURL, tt.wantSiteURL)
			}
		})
	}
}

func TestVerifyRejectsMissingOrLocalDomain(t *testing.T) {
	manager := NewManager(&config.LicenseConfig{
		Enabled:       true,
		LicenseKey:    "LIC-TEST",
		RequireDomain: true,
	}, &config.ServerConfig{
		AllowedOrigins: []string{"localhost", "127.0.0.1"},
	})

	err := manager.verify("runtime")
	if err == nil {
		t.Fatalf("expected verify to fail when domain is missing")
	}
	if !strings.Contains(err.Error(), "license.domain/site_url 未配置或无效") {
		t.Fatalf("unexpected error: %v", err)
	}
	status := manager.Status()
	if status.Valid {
		t.Fatalf("status should be invalid when domain is missing")
	}
}

func TestVerifyRejectsLocalDomainVariants(t *testing.T) {
	cases := []struct {
		name   string
		domain string
	}{
		{name: "localhost with port", domain: "localhost:3000"},
		{name: "localhost url", domain: "https://localhost"},
		{name: "loopback ip with port", domain: "127.0.0.1:8080"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			manager := NewManager(&config.LicenseConfig{
				Enabled:       true,
				LicenseKey:    "LIC-TEST",
				Domain:        tc.domain,
				RequireDomain: true,
			}, &config.ServerConfig{})

			err := manager.verify("runtime")
			if err == nil {
				t.Fatalf("expected verify to fail for local domain variant %q", tc.domain)
			}
			if !strings.Contains(err.Error(), "license.domain/site_url 未配置或无效") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestUpdateDomainWhenLicenseDisabled(t *testing.T) {
	manager := NewManager(&config.LicenseConfig{
		Enabled:    false,
		LicenseKey: "LIC-TEST",
	}, &config.ServerConfig{})

	status, err := manager.UpdateDomain("panel.example.com", "")
	if err != nil {
		t.Fatalf("expected update domain success, got err: %v", err)
	}
	if status.Domain != "panel.example.com" {
		t.Fatalf("domain mismatch: got=%q want=%q", status.Domain, "panel.example.com")
	}
	if status.SiteURL != "https://panel.example.com" {
		t.Fatalf("site_url mismatch: got=%q want=%q", status.SiteURL, "https://panel.example.com")
	}
}

func TestUpdateDomainRejectsInvalidDomain(t *testing.T) {
	manager := NewManager(&config.LicenseConfig{
		Enabled:    false,
		LicenseKey: "LIC-TEST",
	}, &config.ServerConfig{})

	_, err := manager.UpdateDomain("localhost:3000", "")
	if err == nil {
		t.Fatalf("expected invalid domain to be rejected")
	}
	if !strings.Contains(err.Error(), "domain 无效") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVerifyWithUnifiedResponse(t *testing.T) {
	var captured verifyRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request failed: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"success": true,
			"data": {
				"verified": true,
				"status": "ok",
				"license": {
					"valid": true,
					"license_id": 12,
					"plan_code": "NP-STD",
					"customer": "DemoCorp",
					"expires_at": "2027-03-01T00:00:00Z",
					"message": "授权有效"
				},
				"version": {
					"compatible": true,
					"status": "upgrade_available",
					"message": "建议升级",
					"latest_version": "1.2.0"
				}
			},
			"message": "ok"
		}`))
	}))
	defer server.Close()

	manager := NewManager(&config.LicenseConfig{
		Enabled:       true,
		LicenseKey:    "NP-TEST-001",
		VerifyURL:     server.URL,
		Product:       "backend",
		Channel:       "stable",
		ClientVersion: "1.1.0",
	}, &config.ServerConfig{})

	if err := manager.verify("runtime"); err != nil {
		t.Fatalf("expected verify success, got err: %v", err)
	}
	if captured.Product != "backend" {
		t.Fatalf("unexpected product: %s", captured.Product)
	}
	if captured.Channel != "stable" {
		t.Fatalf("unexpected channel: %s", captured.Channel)
	}
	if captured.ClientVersion != "1.1.0" {
		t.Fatalf("unexpected client_version: %s", captured.ClientVersion)
	}

	status := manager.Status()
	if !status.Valid {
		t.Fatalf("expected status valid, got invalid: %s", status.Message)
	}
	if status.LicenseID != 12 {
		t.Fatalf("unexpected license_id: %d", status.LicenseID)
	}
	if status.Plan != "NP-STD" {
		t.Fatalf("unexpected plan: %s", status.Plan)
	}
	if status.VersionStatus != "upgrade_available" {
		t.Fatalf("unexpected version_status: %s", status.VersionStatus)
	}
}

func TestValidateNodeclientVersion(t *testing.T) {
	manager := NewManager(&config.LicenseConfig{
		Enabled: true,
	}, &config.ServerConfig{})
	manager.status.MinNodeclient = "1.2.0"
	manager.status.MaxNodeclient = "2.0.0"

	if err := manager.ValidateNodeclientVersion("1.1.9"); err == nil || !strings.Contains(err.Error(), "版本过低") {
		t.Fatalf("expected low version error, got: %v", err)
	}

	if err := manager.ValidateNodeclientVersion("1.2.0"); err != nil {
		t.Fatalf("expected 1.2.0 to pass, got err: %v", err)
	}

	if err := manager.ValidateNodeclientVersion("v1.5.3"); err != nil {
		t.Fatalf("expected v1.5.3 to pass, got err: %v", err)
	}

	if err := manager.ValidateNodeclientVersion("2.0.1"); err == nil || !strings.Contains(err.Error(), "版本过高") {
		t.Fatalf("expected high version error, got: %v", err)
	}
}
