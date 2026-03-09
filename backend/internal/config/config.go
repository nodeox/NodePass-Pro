package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config 应用配置。
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Telegram TelegramConfig `mapstructure:"telegram"`
	License  LicenseConfig  `mapstructure:"license"`
}

// ServerConfig 服务配置。
type ServerConfig struct {
	Port                  string   `mapstructure:"port"`
	Mode                  string   `mapstructure:"mode"`
	AllowedOrigins        []string `mapstructure:"allowed_origins"`
	TrustForwardedHeaders bool     `mapstructure:"trust_forwarded_headers"` // 是否信任 X-Forwarded-* 头
	StrictCSRF            bool     `mapstructure:"strict_csrf"`             // 是否启用严格 CSRF 模式（不跳过无 Origin/Referer 的请求）
}

// DatabaseConfig 数据库配置。
type DatabaseConfig struct {
	Type            string `mapstructure:"type"`
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	User            string `mapstructure:"user"`
	Password        string `mapstructure:"password"`
	DBName          string `mapstructure:"db_name"`
	DSN             string `mapstructure:"dsn"`
	SSLMode         string `mapstructure:"ssl_mode"`           // PostgreSQL/MySQL SSL 模式
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`     // 最大空闲连接数
	MaxOpenConns    int    `mapstructure:"max_open_conns"`     // 最大打开连接数
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime"`  // 连接最大生命周期（秒）
	ConnMaxIdleTime int    `mapstructure:"conn_max_idle_time"` // 空闲连接最大生命周期（秒）
}

// RedisConfig Redis 缓存配置。
type RedisConfig struct {
	Enabled    bool   `mapstructure:"enabled"`
	Addr       string `mapstructure:"addr"`
	Password   string `mapstructure:"password"`
	DB         int    `mapstructure:"db"`
	KeyPrefix  string `mapstructure:"key_prefix"`
	DefaultTTL int    `mapstructure:"default_ttl"`
}

// JWTConfig JWT 配置。
type JWTConfig struct {
	Secret     string `mapstructure:"secret"`
	ExpireTime int    `mapstructure:"expire_time"`
}

// TelegramConfig Telegram 配置。
type TelegramConfig struct {
	BotToken    string `mapstructure:"bot_token"`
	BotUsername string `mapstructure:"bot_username"`
	WebhookURL  string `mapstructure:"webhook_url"`
	SecretToken string `mapstructure:"secret_token"`
}

// LicenseConfig 运行时授权配置。
type LicenseConfig struct {
	Enabled               bool   `mapstructure:"enabled"`
	LicenseKey            string `mapstructure:"license_key"`
	MachineID             string `mapstructure:"machine_id"`
	Domain                string `mapstructure:"domain"`
	SiteURL               string `mapstructure:"site_url"`
	VerifyIntervalSeconds int    `mapstructure:"verify_interval"`
	FailOpen              bool   `mapstructure:"fail_open"`
	OfflineGraceSeconds   int    `mapstructure:"offline_grace_seconds"`
}

// GlobalConfig 全局配置缓存。
var GlobalConfig *Config

// LoadConfig 加载配置文件。
func LoadConfig(configPath string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")
	v.AutomaticEnv()
	v.SetEnvPrefix("NODEPASS")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	if cfg.Server.Port == "" {
		cfg.Server.Port = "8080"
	}
	if cfg.Server.Mode == "" {
		cfg.Server.Mode = "debug"
	}
	if len(cfg.Server.AllowedOrigins) == 0 {
		cfg.Server.AllowedOrigins = []string{"localhost", "127.0.0.1"}
	}
	if cfg.Database.Type == "" {
		cfg.Database.Type = "postgres"
	}
	if cfg.Redis.Addr == "" {
		cfg.Redis.Addr = "127.0.0.1:6379"
	}
	if cfg.Redis.KeyPrefix == "" {
		cfg.Redis.KeyPrefix = "nodepass:panel"
	}
	if cfg.Redis.DefaultTTL <= 0 {
		cfg.Redis.DefaultTTL = 300
	}
	if cfg.JWT.ExpireTime <= 0 {
		cfg.JWT.ExpireTime = 24
	}
	if cfg.License.VerifyIntervalSeconds <= 0 {
		cfg.License.VerifyIntervalSeconds = 300
	}
	if cfg.License.OfflineGraceSeconds <= 0 {
		cfg.License.OfflineGraceSeconds = 600
	}

	GlobalConfig = cfg
	return cfg, nil
}
