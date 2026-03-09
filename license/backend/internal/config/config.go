package config

import (
	"os"
	"strconv"
	"strings"
)

// Config 应用配置。
type Config struct {
	Server struct {
		Port string
		Mode string
	}
	Database struct {
		Driver string
		DSN    string
	}
	JWT struct {
		Secret      string
		ExpireHours int
	}
	Bootstrap struct {
		AdminUsername  string
		AdminEmail     string
		AdminPassword  string
		ResetAdminPass bool
	}
	Payment struct {
		CallbackStrict           bool
		CallbackToleranceSeconds int
		CallbackSecretDefault    string
		CallbackSecretManual     string
		CallbackSecretAlipay     string
		CallbackSecretWechat     string
	}
	Storage struct {
		ReleaseUploadDir string
	}
}

// Load 从环境变量加载配置。
func Load() *Config {
	cfg := &Config{}

	cfg.Server.Port = getEnv("SERVER_PORT", "8091")
	cfg.Server.Mode = getEnv("GIN_MODE", "debug")

	cfg.Database.Driver = getEnv("DB_DRIVER", "sqlite")
	cfg.Database.DSN = getEnv("DB_DSN", "./data/license-unified.db")

	cfg.JWT.Secret = getEnv("JWT_SECRET", "change-this-secret-in-production")
	cfg.JWT.ExpireHours = getEnvInt("JWT_EXPIRE_HOURS", 24)

	cfg.Bootstrap.AdminUsername = getEnv("BOOTSTRAP_ADMIN_USERNAME", "admin")
	cfg.Bootstrap.AdminEmail = getEnv("BOOTSTRAP_ADMIN_EMAIL", "admin@example.com")
	cfg.Bootstrap.AdminPassword = strings.TrimSpace(os.Getenv("BOOTSTRAP_ADMIN_PASSWORD"))
	cfg.Bootstrap.ResetAdminPass = getEnvBool("BOOTSTRAP_RESET_ADMIN_PASSWORD", false)

	cfg.Payment.CallbackStrict = getEnvBool("PAYMENT_CALLBACK_STRICT", false)
	cfg.Payment.CallbackToleranceSeconds = getEnvInt("PAYMENT_CALLBACK_TOLERANCE_SECONDS", 300)
	cfg.Payment.CallbackSecretDefault = getEnv("PAYMENT_CALLBACK_SECRET_DEFAULT", "")
	cfg.Payment.CallbackSecretManual = getEnv("PAYMENT_CALLBACK_SECRET_MANUAL", "")
	cfg.Payment.CallbackSecretAlipay = getEnv("PAYMENT_CALLBACK_SECRET_ALIPAY", "")
	cfg.Payment.CallbackSecretWechat = getEnv("PAYMENT_CALLBACK_SECRET_WECHAT", "")
	cfg.Storage.ReleaseUploadDir = getEnv("RELEASE_UPLOAD_DIR", "./uploads/releases")

	return cfg
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}
