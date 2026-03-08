package database

import (
	"fmt"
	"strings"
	"time"

	"nodepass-license-center/internal/config"
	"nodepass-license-center/internal/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// Init 初始化数据库连接并自动迁移。
func Init(cfg *config.Config) (*gorm.DB, error) {
	if cfg == nil {
		return nil, fmt.Errorf("配置不能为空")
	}

	dsn, err := buildDSN(cfg)
	if err != nil {
		return nil, err
	}

	gormCfg := &gorm.Config{Logger: logger.Default.LogMode(logger.Warn)}

	var db *gorm.DB
	switch strings.ToLower(strings.TrimSpace(cfg.Database.Type)) {
	case "sqlite":
		db, err = gorm.Open(sqlite.Open(dsn), gormCfg)
	case "mysql":
		db, err = gorm.Open(mysql.Open(dsn), gormCfg)
	case "postgres", "postgresql":
		db, err = gorm.Open(postgres.Open(dsn), gormCfg)
	default:
		return nil, fmt.Errorf("不支持的数据库类型: %s", cfg.Database.Type)
	}
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	if err := db.AutoMigrate(
		&models.AdminUser{},
		&models.LicensePlan{},
		&models.LicenseKey{},
		&models.LicenseActivation{},
		&models.VerifyLog{},
		&models.LicenseTag{},
		&models.LicenseKeyTag{},
		&models.WebhookConfig{},
		&models.WebhookLog{},
		&models.Alert{},
		&models.LicenseTransferLog{},
		&models.LicenseDomainBinding{},
		&models.DomainIPBinding{},
	); err != nil {
		return nil, fmt.Errorf("自动迁移失败: %w", err)
	}

	if err := seedDefaults(db, cfg); err != nil {
		return nil, err
	}

	DB = db
	return db, nil
}

func buildDSN(cfg *config.Config) (string, error) {
	if strings.TrimSpace(cfg.Database.DSN) != "" {
		return cfg.Database.DSN, nil
	}

	switch strings.ToLower(strings.TrimSpace(cfg.Database.Type)) {
	case "sqlite":
		return "./data/license-center.db", nil
	case "mysql":
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			cfg.Database.User,
			cfg.Database.Password,
			cfg.Database.Host,
			cfg.Database.Port,
			cfg.Database.DBName,
		), nil
	case "postgres", "postgresql":
		return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable TimeZone=UTC",
			cfg.Database.Host,
			cfg.Database.Port,
			cfg.Database.User,
			cfg.Database.Password,
			cfg.Database.DBName,
		), nil
	default:
		return "", fmt.Errorf("不支持的数据库类型: %s", cfg.Database.Type)
	}
}

func seedDefaults(db *gorm.DB, cfg *config.Config) error {
	if db == nil {
		return fmt.Errorf("数据库未初始化")
	}

	var adminCount int64
	if err := db.Model(&models.AdminUser{}).Count(&adminCount).Error; err != nil {
		return fmt.Errorf("查询管理员失败: %w", err)
	}
	if adminCount == 0 {
		if cfg == nil {
			return fmt.Errorf("配置不能为空")
		}
		cfg.Admin.Username = strings.TrimSpace(cfg.Admin.Username)
		cfg.Admin.Email = strings.TrimSpace(cfg.Admin.Email)
		cfg.Admin.Password = strings.TrimSpace(cfg.Admin.Password)
		if cfg.Admin.Username == "" || cfg.Admin.Email == "" || cfg.Admin.Password == "" {
			return fmt.Errorf("初始化管理员失败: 请完整配置 admin.username/admin.email/admin.password")
		}
		if cfg.Admin.Password == "ChangeMe123!" {
			return fmt.Errorf("初始化管理员失败: 禁止使用默认弱密码")
		}
		if len(cfg.Admin.Password) < 12 {
			return fmt.Errorf("初始化管理员失败: admin.password 长度不能少于 12 位")
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(cfg.Admin.Password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("生成管理员密码哈希失败: %w", err)
		}
		admin := &models.AdminUser{
			Username:     cfg.Admin.Username,
			Email:        cfg.Admin.Email,
			PasswordHash: string(hash),
			Role:         "admin",
			Status:       "active",
		}
		if err := db.Create(admin).Error; err != nil {
			return fmt.Errorf("创建默认管理员失败: %w", err)
		}
	}

	var planCount int64
	if err := db.Model(&models.LicensePlan{}).Count(&planCount).Error; err != nil {
		return fmt.Errorf("查询套餐失败: %w", err)
	}
	if planCount == 0 {
		defaultPlans := []*models.LicensePlan{
			{
				Name:                 "标准版",
				Code:                 "standard",
				Description:          "默认标准授权套餐",
				IsEnabled:            true,
				MaxMachines:          1,
				DurationDays:         365,
				MinPanelVersion:      "0.1.0",
				MaxPanelVersion:      "9.9.9",
				MinBackendVersion:    "0.1.0",
				MaxBackendVersion:    "9.9.9",
				MinFrontendVersion:   "0.1.0",
				MaxFrontendVersion:   "9.9.9",
				MinNodeclientVersion: "0.1.0",
				MaxNodeclientVersion: "9.9.9",
			},
			{
				Name:                 "企业版",
				Code:                 "enterprise",
				Description:          "企业授权套餐",
				IsEnabled:            true,
				MaxMachines:          10,
				DurationDays:         365,
				MinPanelVersion:      "0.1.0",
				MaxPanelVersion:      "9.9.9",
				MinBackendVersion:    "0.1.0",
				MaxBackendVersion:    "9.9.9",
				MinFrontendVersion:   "0.1.0",
				MaxFrontendVersion:   "9.9.9",
				MinNodeclientVersion: "0.1.0",
				MaxNodeclientVersion: "9.9.9",
			},
		}
		if err := db.Create(defaultPlans).Error; err != nil {
			return fmt.Errorf("创建默认套餐失败: %w", err)
		}
	}

	return nil
}

// TouchLastLogin 更新管理员最后登录时间。
func TouchLastLogin(db *gorm.DB, adminID uint) {
	if db == nil || adminID == 0 {
		return
	}
	now := time.Now().UTC()
	_ = db.Model(&models.AdminUser{}).Where("id = ?", adminID).Update("last_login_at", &now).Error
}
