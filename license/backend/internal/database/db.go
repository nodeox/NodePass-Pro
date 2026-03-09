package database

import (
	"fmt"
	"os"
	"path/filepath"

	"nodepass-license-unified/internal/config"
	"nodepass-license-unified/internal/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Init 初始化数据库连接与迁移。
func Init(cfg *config.Config) (*gorm.DB, error) {
	var dialector gorm.Dialector

	switch cfg.Database.Driver {
	case "mysql":
		dialector = mysql.Open(cfg.Database.DSN)
	case "postgres", "postgresql":
		dialector = postgres.Open(cfg.Database.DSN)
	case "sqlite", "sqlite3":
		if err := os.MkdirAll(filepath.Dir(cfg.Database.DSN), 0o755); err != nil {
			return nil, fmt.Errorf("创建 SQLite 数据目录失败: %w", err)
		}
		dialector = sqlite.Open(cfg.Database.DSN)
	default:
		return nil, fmt.Errorf("不支持的数据库驱动: %s", cfg.Database.Driver)
	}

	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	if err = db.AutoMigrate(
		&models.AdminUser{},
		&models.AdminAuditLog{},
		&models.LicensePlan{},
		&models.License{},
		&models.LicenseActivation{},
		&models.ProductRelease{},
		&models.VersionSyncConfig{},
		&models.VersionPolicy{},
		&models.VerifyLog{},
		&models.BillingOrder{},
		&models.BillingOrderEvent{},
		&models.LicenseTransferLog{},
		&models.LicenseReminder{},
		&models.TrialIssue{},
	); err != nil {
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}

	if err = bootstrapAdmin(db, cfg); err != nil {
		return nil, err
	}

	if err = bootstrapPlan(db); err != nil {
		return nil, err
	}

	return db, nil
}

func bootstrapAdmin(db *gorm.DB, cfg *config.Config) error {
	var admin models.AdminUser
	err := db.Where("username = ?", cfg.Bootstrap.AdminUsername).First(&admin).Error
	if err == nil {
		if cfg.Bootstrap.ResetAdminPass {
			hash, hashErr := bcrypt.GenerateFromPassword([]byte(cfg.Bootstrap.AdminPassword), bcrypt.DefaultCost)
			if hashErr != nil {
				return fmt.Errorf("重置管理员密码失败: %w", hashErr)
			}
			if updateErr := db.Model(&admin).Updates(map[string]interface{}{
				"password_hash": string(hash),
				"email":         cfg.Bootstrap.AdminEmail,
			}).Error; updateErr != nil {
				return fmt.Errorf("更新管理员信息失败: %w", updateErr)
			}
		}
		return nil
	}
	if err != gorm.ErrRecordNotFound {
		return fmt.Errorf("查询管理员失败: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(cfg.Bootstrap.AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("生成管理员密码失败: %w", err)
	}

	admin = models.AdminUser{
		Username:     cfg.Bootstrap.AdminUsername,
		Email:        cfg.Bootstrap.AdminEmail,
		PasswordHash: string(hash),
	}
	if err = db.Create(&admin).Error; err != nil {
		return fmt.Errorf("初始化管理员失败: %w", err)
	}
	return nil
}

func bootstrapPlan(db *gorm.DB) error {
	var count int64
	if err := db.Model(&models.LicensePlan{}).Count(&count).Error; err != nil {
		return fmt.Errorf("查询默认套餐失败: %w", err)
	}
	if count > 0 {
		return nil
	}

	plan := models.LicensePlan{
		Code:         "NP-STD",
		Name:         "NodePass Standard",
		Description:  "默认标准套餐",
		MaxMachines:  3,
		DurationDays: 365,
		Status:       "active",
	}
	if err := db.Create(&plan).Error; err != nil {
		return fmt.Errorf("创建默认套餐失败: %w", err)
	}
	return nil
}
