package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

const (
	defaultJWTSecret       = "change-this-license-secret"
	defaultAdminPassword   = "ChangeMe123!"
	defaultSignatureSecret = "change-this-signature-secret"
)

// Config 应用配置。
type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
	Database   DatabaseConfig   `mapstructure:"database"`
	Redis      RedisConfig      `mapstructure:"redis"`
	JWT        JWTConfig        `mapstructure:"jwt"`
	Admin      AdminConfig      `mapstructure:"admin"`
	Security   SecurityConfig   `mapstructure:"security"`
	Monitoring MonitoringConfig `mapstructure:"monitoring"`
}

// ServerConfig 服务配置。
type ServerConfig struct {
	Port string `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

// DatabaseConfig 数据库配置。
type DatabaseConfig struct {
	Type     string `mapstructure:"type"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"db_name"`
	DSN      string `mapstructure:"dsn"`
}

// JWTConfig JWT 配置。
type JWTConfig struct {
	Secret      string `mapstructure:"secret"`
	ExpireHours int    `mapstructure:"expire_hours"`
}

// AdminConfig 默认管理员配置。
type AdminConfig struct {
	Username string `mapstructure:"username"`
	Email    string `mapstructure:"email"`
	Password string `mapstructure:"password"`
}

// RedisConfig Redis 配置。
type RedisConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	Prefix   string `mapstructure:"prefix"`
}

// SecurityConfig 安全配置。
type SecurityConfig struct {
	RateLimit   RateLimitConfig   `mapstructure:"rate_limit"`
	Signature   SignatureConfig   `mapstructure:"signature"`
	IPWhitelist IPWhitelistConfig `mapstructure:"ip_whitelist"`
}

// RateLimitConfig 限流配置。
type RateLimitConfig struct {
	Enabled           bool `mapstructure:"enabled"`
	RequestsPerSecond int  `mapstructure:"requests_per_second"`
	Burst             int  `mapstructure:"burst"`
}

// SignatureConfig 签名配置。
type SignatureConfig struct {
	Enabled    bool   `mapstructure:"enabled"`
	Secret     string `mapstructure:"secret"`
	TimeWindow int64  `mapstructure:"time_window"`
}

// IPWhitelistConfig IP 白名单配置。
type IPWhitelistConfig struct {
	Enabled      bool     `mapstructure:"enabled"`
	AllowedIPs   []string `mapstructure:"allowed_ips"`
	AllowedCIDRs []string `mapstructure:"allowed_cidrs"`
}

// MonitoringConfig 监控配置。
type MonitoringConfig struct {
	Alert   AlertConfig   `mapstructure:"alert"`
	Cleanup CleanupConfig `mapstructure:"cleanup"`
}

// AlertConfig 告警配置。
type AlertConfig struct {
	Enabled       bool `mapstructure:"enabled"`
	CheckInterval int  `mapstructure:"check_interval"`
	ExpiringDays  int  `mapstructure:"expiring_days"`
}

// CleanupConfig 清理配置。
type CleanupConfig struct {
	Enabled        bool `mapstructure:"enabled"`
	VerifyLogDays  int  `mapstructure:"verify_log_days"`
	WebhookLogDays int  `mapstructure:"webhook_log_days"`
	AlertDays      int  `mapstructure:"alert_days"`
}

// Load 加载配置。
func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	v.AutomaticEnv()
	v.SetEnvPrefix("LICENSE")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	if cfg.Server.Port == "" {
		cfg.Server.Port = "8090"
	}
	if cfg.Server.Mode == "" {
		cfg.Server.Mode = "release"
	}
	if cfg.Database.Type == "" {
		cfg.Database.Type = "postgres"
	}
	if cfg.JWT.ExpireHours <= 0 {
		cfg.JWT.ExpireHours = 24
	}

	cfg.JWT.Secret = strings.TrimSpace(cfg.JWT.Secret)
	if cfg.JWT.Secret == "" || cfg.JWT.Secret == defaultJWTSecret {
		return nil, fmt.Errorf("jwt.secret 未配置或仍为默认值，请设置强随机密钥")
	}
	if len(cfg.JWT.Secret) < 32 {
		return nil, fmt.Errorf("jwt.secret 长度不能少于 32 位")
	}

	cfg.Admin.Username = strings.TrimSpace(cfg.Admin.Username)
	cfg.Admin.Email = strings.TrimSpace(cfg.Admin.Email)
	cfg.Admin.Password = strings.TrimSpace(cfg.Admin.Password)
	if cfg.Admin.Username == "" {
		cfg.Admin.Username = "admin"
	}
	if cfg.Admin.Email == "" {
		cfg.Admin.Email = "admin@license.local"
	}
	if cfg.Admin.Password == defaultAdminPassword {
		return nil, fmt.Errorf("admin.password 仍为默认值，请修改为强密码")
	}
	if cfg.Admin.Password != "" && len(cfg.Admin.Password) < 12 {
		return nil, fmt.Errorf("admin.password 长度不能少于 12 位")
	}

	cfg.Security.Signature.Secret = strings.TrimSpace(cfg.Security.Signature.Secret)
	if cfg.Security.Signature.TimeWindow <= 0 {
		cfg.Security.Signature.TimeWindow = 300
	}
	if cfg.Security.Signature.Enabled {
		if cfg.Security.Signature.Secret == "" || cfg.Security.Signature.Secret == defaultSignatureSecret {
			return nil, fmt.Errorf("security.signature.secret 未配置或仍为默认值")
		}
		if len(cfg.Security.Signature.Secret) < 32 {
			return nil, fmt.Errorf("security.signature.secret 长度不能少于 32 位")
		}
	}

	return cfg, nil
}
