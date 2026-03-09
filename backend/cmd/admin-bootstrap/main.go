package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"nodepass-pro/backend/internal/config"
	"nodepass-pro/backend/internal/database"
	"nodepass-pro/backend/internal/models"
	"nodepass-pro/backend/internal/services"
	"nodepass-pro/backend/internal/utils"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func main() {
	var (
		configPath string
		username   string
		email      string
		password   string
	)

	flag.StringVar(&configPath, "config", "configs/config.yaml", "配置文件路径")
	flag.StringVar(&username, "username", "", "管理员用户名")
	flag.StringVar(&email, "email", "", "管理员邮箱")
	flag.StringVar(&password, "password", "", "管理员密码")
	flag.Parse()

	username = strings.TrimSpace(username)
	email = strings.ToLower(strings.TrimSpace(email))
	password = strings.TrimSpace(password)

	if err := validateInput(username, email, password); err != nil {
		exitWithError(err)
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		exitWithError(fmt.Errorf("加载配置失败: %w", err))
	}

	db, err := database.InitDB(&cfg.Database)
	if err != nil {
		exitWithError(fmt.Errorf("初始化数据库失败: %w", err))
	}
	defer func() {
		_ = database.Close()
	}()

	user, created, err := upsertAdminUser(db, username, email, password)
	if err != nil {
		exitWithError(err)
	}

	action := "更新"
	if created {
		action = "创建"
	}
	fmt.Printf("管理员账号已%s成功: username=%s email=%s role=%s\n", action, user.Username, user.Email, user.Role)
}

func validateInput(username string, email string, password string) error {
	if username == "" || email == "" || password == "" {
		return fmt.Errorf("username、email、password 均不能为空")
	}
	if err := utils.ValidateUsername(username); err != nil {
		return err
	}
	if err := utils.ValidateEmail(email); err != nil {
		return err
	}
	if err := utils.ValidatePassword(password); err != nil {
		return err
	}
	return nil
}

func upsertAdminUser(db *gorm.DB, username string, email string, password string) (*models.User, bool, error) {
	var (
		user    models.User
		created bool
	)

	if err := db.Where("email = ?", email).First(&user).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, fmt.Errorf("查询邮箱失败: %w", err)
		}
		if err = db.Where("username = ?", username).First(&user).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, false, fmt.Errorf("查询用户名失败: %w", err)
			}

			authService := services.NewAuthService(db)
			createdUser, registerErr := authService.Register(&services.RegisterRequest{
				Username: username,
				Email:    email,
				Password: password,
			})
			if registerErr != nil {
				return nil, false, fmt.Errorf("创建管理员账号失败: %w", registerErr)
			}
			user = *createdUser
			created = true
		}
	}

	// 准备更新字段
	updates := map[string]interface{}{
		"role":   "admin",
		"status": "normal",
	}

	// 只有在用户已存在时才更新密码（新创建的用户密码已经在 Register 中设置）
	if !created {
		passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return nil, false, fmt.Errorf("加密管理员密码失败: %w", err)
		}
		updates["password_hash"] = string(passwordHash)
	}

	if canUpdateUsername(db, user.ID, username) {
		updates["username"] = username
	}
	if canUpdateEmail(db, user.ID, email) {
		updates["email"] = email
	}

	if err := db.Model(&models.User{}).Where("id = ?", user.ID).Updates(updates).Error; err != nil {
		return nil, false, fmt.Errorf("更新管理员信息失败: %w", err)
	}

	if err := db.First(&user, user.ID).Error; err != nil {
		return nil, false, fmt.Errorf("读取管理员信息失败: %w", err)
	}

	return &user, created, nil
}

func canUpdateUsername(db *gorm.DB, userID uint, username string) bool {
	var count int64
	if err := db.Model(&models.User{}).
		Where("username = ? AND id <> ?", username, userID).
		Count(&count).Error; err != nil {
		return false
	}
	return count == 0
}

func canUpdateEmail(db *gorm.DB, userID uint, email string) bool {
	var count int64
	if err := db.Model(&models.User{}).
		Where("email = ? AND id <> ?", email, userID).
		Count(&count).Error; err != nil {
		return false
	}
	return count == 0
}

func exitWithError(err error) {
	fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
	os.Exit(1)
}
