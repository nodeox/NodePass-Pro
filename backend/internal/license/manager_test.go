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
			name: "wildcard origin should not infer domain",
			serverCfg: config.ServerConfig{
				AllowedOrigins: []string{"*.example.com"},
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
