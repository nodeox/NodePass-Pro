package services

import (
	"testing"

	"nodepass-pro/backend/internal/models"
)

func TestValidateProtocolConfig(t *testing.T) {
	tests := []struct {
		name    string
		proto   string
		config  *models.ProtocolConfig
		wantErr bool
	}{
		{
			name:    "nil config should pass",
			proto:   "tcp",
			config:  nil,
			wantErr: false,
		},
		{
			name:  "valid TCP config",
			proto: "tcp",
			config: &models.ProtocolConfig{
				TCPKeepalive:      boolPtr(true),
				KeepaliveInterval: intPtr(60),
				ConnectTimeout:    intPtr(10),
				ReadTimeout:       intPtr(30),
			},
			wantErr: false,
		},
		{
			name:  "invalid keepalive interval - too small",
			proto: "tcp",
			config: &models.ProtocolConfig{
				KeepaliveInterval: intPtr(0),
			},
			wantErr: true,
		},
		{
			name:  "invalid keepalive interval - too large",
			proto: "tcp",
			config: &models.ProtocolConfig{
				KeepaliveInterval: intPtr(301),
			},
			wantErr: true,
		},
		{
			name:  "valid UDP config",
			proto: "udp",
			config: &models.ProtocolConfig{
				BufferSize:     intPtr(8192),
				SessionTimeout: intPtr(60),
			},
			wantErr: false,
		},
		{
			name:  "invalid buffer size - too small",
			proto: "udp",
			config: &models.ProtocolConfig{
				BufferSize: intPtr(512),
			},
			wantErr: true,
		},
		{
			name:  "valid WebSocket config",
			proto: "ws",
			config: &models.ProtocolConfig{
				WSPath:         stringPtr("/ws"),
				PingInterval:   intPtr(30),
				MaxMessageSize: intPtr(1024),
				Compression:    boolPtr(true),
			},
			wantErr: false,
		},
		{
			name:  "invalid ws_path - missing leading slash",
			proto: "wss",
			config: &models.ProtocolConfig{
				WSPath: stringPtr("ws"),
			},
			wantErr: true,
		},
		{
			name:  "valid TLS config",
			proto: "tls",
			config: &models.ProtocolConfig{
				TLSVersion: stringPtr("tls1.2"),
				VerifyCert: boolPtr(true),
				SNI:        stringPtr("example.com"),
			},
			wantErr: false,
		},
		{
			name:  "invalid TLS version",
			proto: "tls",
			config: &models.ProtocolConfig{
				TLSVersion: stringPtr("tls1.1"),
			},
			wantErr: true,
		},
		{
			name:  "valid QUIC config",
			proto: "quic",
			config: &models.ProtocolConfig{
				MaxStreams:    intPtr(100),
				InitialWindow: intPtr(256),
				IdleTimeout:   intPtr(30),
				Enable0RTT:    boolPtr(true),
			},
			wantErr: false,
		},
		{
			name:  "invalid max_streams - too large",
			proto: "quic",
			config: &models.ProtocolConfig{
				MaxStreams: intPtr(1001),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateProtocolConfig(tt.proto, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateProtocolConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTunnelNormalizeProtocol(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
		want     string
		wantErr  bool
	}{
		{"tcp", "tcp", "tcp", false},
		{"TCP uppercase", "TCP", "tcp", false},
		{"udp", "udp", "udp", false},
		{"ws", "ws", "ws", false},
		{"wss", "wss", "wss", false},
		{"wss uppercase", "WSS", "wss", false},
		{"tls", "tls", "tls", false},
		{"quic", "quic", "quic", false},
		{"QUIC uppercase", "QUIC", "quic", false},
		{"invalid protocol", "http", "", true},
		{"empty protocol", "", "", true},
		{"whitespace", "  tcp  ", "tcp", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tunnelNormalizeProtocol(tt.protocol)
			if (err != nil) != tt.wantErr {
				t.Errorf("tunnelNormalizeProtocol() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("tunnelNormalizeProtocol() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper functions
func intPtr(i int) *int {
	return &i
}

func boolPtr(b bool) *bool {
	return &b
}

func stringPtr(s string) *string {
	return &s
}
