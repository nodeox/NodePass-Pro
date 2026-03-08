package license

import (
	"strings"
	"testing"

	"nodepass-pro/backend/internal/config"
)

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
		Enabled:    true,
		VerifyURL:  "https://license.example.com/api/v1/license/verify",
		LicenseKey: "LIC-TEST",
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
				Enabled:    true,
				VerifyURL:  "https://license.example.com/api/v1/license/verify",
				LicenseKey: "LIC-TEST",
				Domain:     tc.domain,
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
