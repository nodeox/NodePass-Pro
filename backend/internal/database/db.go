package database

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"nodepass-panel/backend/internal/config"
	"nodepass-panel/backend/internal/models"

	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

// DB 全局数据库连接。
var DB *gorm.DB

// InitDB 初始化数据库连接并执行自动迁移。
func InitDB(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	if cfg == nil {
		return nil, fmt.Errorf("数据库配置不能为空")
	}

	var (
		dialector gorm.Dialector
		dsn       string
	)

	switch cfg.Type {
	case "sqlite":
		dsn = cfg.DSN
		if dsn == "" {
			dsn = "./data/nodepass.db"
		}
		dir := filepath.Dir(dsn)
		if dir != "." {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, fmt.Errorf("创建 sqlite 目录失败: %w", err)
			}
		}
		dialector = sqlite.Open(dsn)
	case "mysql":
		dsn = cfg.DSN
		if dsn == "" {
			dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
				cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)
		}
		dialector = mysql.Open(dsn)
	case "postgres", "postgresql":
		dsn = cfg.DSN
		if dsn == "" {
			dsn = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
				cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName)
		}
		dialector = postgres.Open(dsn)
	default:
		return nil, fmt.Errorf("不支持的数据库类型: %s", cfg.Type)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("数据库连接失败: %w", err)
	}

	// 配置数据库连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取数据库实例失败: %w", err)
	}

	// 设置连接池参数
	sqlDB.SetMaxIdleConns(10)                  // 最大空闲连接数
	sqlDB.SetMaxOpenConns(100)                 // 最大打开连接数
	sqlDB.SetConnMaxLifetime(time.Hour)        // 连接最大生命周期
	sqlDB.SetConnMaxIdleTime(10 * time.Minute) // 空闲连接最大生命周期

	zap.L().Info("数据库连接池配置完成",
		zap.Int("max_idle_conns", 10),
		zap.Int("max_open_conns", 100),
		zap.Duration("conn_max_lifetime", time.Hour))

	if err := AutoMigrate(db); err != nil {
		return nil, fmt.Errorf("自动迁移失败: %w", err)
	}
	if err := seedDefaultVIPLevels(db); err != nil {
		return nil, fmt.Errorf("初始化默认 VIP 等级失败: %w", err)
	}

	DB = db
	zap.L().Info("数据库初始化完成", zap.String("type", cfg.Type))
	return db, nil
}

// AutoMigrate 注册全部模型并执行迁移。
func AutoMigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&models.User{},
		&models.UserPermission{},
		&models.Node{},
		&models.NodeConfig{},
		&models.NodePair{},
		&models.Rule{},
		&models.TrafficRecord{},
		&models.VIPLevel{},
		&models.BenefitCode{},
		&models.SystemConfig{},
		&models.Announcement{},
		&models.AuditLog{},
	); err != nil {
		return err
	}

	return migrateNodePairIndexes(db)
}

func migrateNodePairIndexes(db *gorm.DB) error {
	migrator := db.Migrator()
	nodePairModel := &models.NodePair{}

	legacyIndexes := []string{
		"uk_node_pairs_entry_exit",
	}
	for _, indexName := range legacyIndexes {
		if migrator.HasIndex(nodePairModel, indexName) {
			if err := migrator.DropIndex(nodePairModel, indexName); err != nil {
				return fmt.Errorf("移除旧节点配对唯一索引失败(%s): %w", indexName, err)
			}
		}
	}

	const scopedUniqueIndex = "uk_node_pairs_user_entry_exit"
	if !migrator.HasIndex(nodePairModel, scopedUniqueIndex) {
		if err := migrator.CreateIndex(nodePairModel, scopedUniqueIndex); err != nil {
			return fmt.Errorf("创建节点配对唯一索引失败(%s): %w", scopedUniqueIndex, err)
		}
	}

	return nil
}

func seedDefaultVIPLevels(db *gorm.DB) error {
	gb := int64(1024 * 1024 * 1024)
	tb := gb * 1024

	defaultLevels := []models.VIPLevel{
		{
			Level:                   0,
			Name:                    "免费",
			TrafficQuota:            10 * gb,
			MaxRules:                5,
			MaxBandwidth:            100,
			MaxSelfHostedEntryNodes: 0,
			MaxSelfHostedExitNodes:  0,
			AccessibleNodeLevel:     0,
			TrafficMultiplier:       1.0,
		},
		{
			Level:                   1,
			Name:                    "基础",
			TrafficQuota:            100 * gb,
			MaxRules:                20,
			MaxBandwidth:            500,
			MaxSelfHostedEntryNodes: 1,
			MaxSelfHostedExitNodes:  1,
			AccessibleNodeLevel:     1,
			TrafficMultiplier:       1.0,
		},
		{
			Level:                   2,
			Name:                    "高级",
			TrafficQuota:            500 * gb,
			MaxRules:                50,
			MaxBandwidth:            1000,
			MaxSelfHostedEntryNodes: 3,
			MaxSelfHostedExitNodes:  3,
			AccessibleNodeLevel:     2,
			TrafficMultiplier:       1.0,
		},
		{
			Level:                   3,
			Name:                    "专业",
			TrafficQuota:            2 * tb,
			MaxRules:                100,
			MaxBandwidth:            -1,
			MaxSelfHostedEntryNodes: 5,
			MaxSelfHostedExitNodes:  5,
			AccessibleNodeLevel:     3,
			TrafficMultiplier:       1.0,
		},
		{
			Level:                   4,
			Name:                    "企业",
			TrafficQuota:            10 * tb,
			MaxRules:                -1,
			MaxBandwidth:            -1,
			MaxSelfHostedEntryNodes: 10,
			MaxSelfHostedExitNodes:  10,
			AccessibleNodeLevel:     4,
			TrafficMultiplier:       1.0,
		},
	}

	for _, level := range defaultLevels {
		if err := db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "level"}},
			DoNothing: true,
		}).Create(&level).Error; err != nil {
			return err
		}
	}

	return nil
}

// Close 关闭数据库连接。
func Close() error {
	if DB == nil {
		return nil
	}
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
