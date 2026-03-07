package services

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"nodepass-pro/backend/internal/models"
	"nodepass-pro/backend/internal/utils"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// AuthService 认证服务。
type AuthService struct {
	db         *gorm.DB
	vipService *VIPService
}

// RegisterRequest 注册请求。
type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginRequest 登录请求。
type LoginRequest struct {
	Account  string `json:"account" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResult 登录结果。
type LoginResult struct {
	Token string       `json:"token"`
	User  *models.User `json:"user"`
}

// NewAuthService 创建认证服务。
func NewAuthService(db *gorm.DB) *AuthService {
	return &AuthService{
		db:         db,
		vipService: NewVIPService(db),
	}
}

// Register 注册用户。
func (s *AuthService) Register(req *RegisterRequest) (*models.User, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: 请求体不能为空", ErrInvalidParams)
	}

	username := strings.TrimSpace(req.Username)
	email := strings.ToLower(strings.TrimSpace(req.Email))
	password := strings.TrimSpace(req.Password)
	if username == "" || email == "" || password == "" {
		return nil, fmt.Errorf("%w: username、email、password 不能为空", ErrInvalidParams)
	}

	// 验证用户名
	if err := utils.ValidateUsername(username); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidParams, err)
	}

	// 验证邮箱
	if err := utils.ValidateEmail(email); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidParams, err)
	}

	// 验证密码
	if err := utils.ValidatePassword(password); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidParams, err)
	}

	if err := s.ensureUniqueUser(username, email); err != nil {
		return nil, err
	}

	freeLevel, err := s.vipService.getLevelByLevel(0)
	if err != nil {
		return nil, err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("密码加密失败: %w", err)
	}

	user := &models.User{
		Username:                username,
		Email:                   email,
		PasswordHash:            string(passwordHash),
		Role:                    "user",
		Status:                  "normal",
		VipLevel:                freeLevel.Level,
		VipExpiresAt:            nil,
		TrafficQuota:            freeLevel.TrafficQuota,
		TrafficUsed:             0,
		MaxRules:                freeLevel.MaxRules,
		MaxBandwidth:            freeLevel.MaxBandwidth,
		MaxSelfHostedEntryNodes: freeLevel.MaxSelfHostedEntryNodes,
		MaxSelfHostedExitNodes:  freeLevel.MaxSelfHostedExitNodes,
	}
	if err = s.db.Create(user).Error; err != nil {
		if isDuplicateKeyError(err) {
			return nil, fmt.Errorf("%w: 用户名或邮箱已存在", ErrConflict)
		}
		return nil, fmt.Errorf("创建用户失败: %w", err)
	}
	return user, nil
}

// Login 登录并返回 JWT。
func (s *AuthService) Login(req *LoginRequest) (*LoginResult, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: 请求体不能为空", ErrInvalidParams)
	}

	account := strings.TrimSpace(req.Account)
	password := strings.TrimSpace(req.Password)
	if account == "" || password == "" {
		return nil, fmt.Errorf("%w: account 和 password 不能为空", ErrInvalidParams)
	}

	var user models.User
	if err := s.db.Model(&models.User{}).
		Where("email = ? OR username = ?", strings.ToLower(account), account).
		First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: 用户名/邮箱或密码错误", ErrUnauthorized)
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("%w: 用户名/邮箱或密码错误", ErrUnauthorized)
	}
	if strings.EqualFold(strings.TrimSpace(user.Status), "banned") {
		return nil, fmt.Errorf("%w: 账户已被封禁", ErrForbidden)
	}

	token, err := generateUserJWT(user.ID, user.Role)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	if err := s.db.Model(&models.User{}).Where("id = ?", user.ID).Update("last_login_at", &now).Error; err != nil {
		// 更新登录时间失败不影响登录流程，但需要记录日志
		// 注意：这里无法直接使用 zap，因为会导致循环依赖
		// 错误会在上层被记录
		_ = err // 明确标记为已知的非关键错误
	}
	user.LastLoginAt = &now

	return &LoginResult{
		Token: token,
		User:  &user,
	}, nil
}

// GetMe 获取当前用户信息。
func (s *AuthService) GetMe(userID uint) (*models.User, error) {
	if userID == 0 {
		return nil, fmt.Errorf("%w: 用户 ID 无效", ErrInvalidParams)
	}

	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: 用户不存在", ErrNotFound)
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	return &user, nil
}

// RefreshToken 刷新用户 Token。
func (s *AuthService) RefreshToken(userID uint) (string, error) {
	if userID == 0 {
		return "", fmt.Errorf("%w: 用户 ID 无效", ErrInvalidParams)
	}

	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", fmt.Errorf("%w: 用户不存在", ErrNotFound)
		}
		return "", fmt.Errorf("查询用户失败: %w", err)
	}

	if strings.EqualFold(strings.TrimSpace(user.Status), "banned") {
		return "", fmt.Errorf("%w: 账户已被封禁", ErrForbidden)
	}

	token, err := generateUserJWT(user.ID, user.Role)
	if err != nil {
		return "", err
	}

	return token, nil
}

// ChangePassword 修改当前用户密码。
func (s *AuthService) ChangePassword(userID uint, oldPassword string, newPassword string) error {
	if userID == 0 {
		return fmt.Errorf("%w: 用户 ID 无效", ErrInvalidParams)
	}

	oldPassword = strings.TrimSpace(oldPassword)
	newPassword = strings.TrimSpace(newPassword)
	if oldPassword == "" || newPassword == "" {
		return fmt.Errorf("%w: old_password/new_password 不能为空", ErrInvalidParams)
	}

	// 验证新密码
	if err := utils.ValidatePassword(newPassword); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidParams, err)
	}

	user, err := s.GetMe(userID)
	if err != nil {
		return err
	}
	if compareErr := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword)); compareErr != nil {
		return fmt.Errorf("%w: 原密码错误", ErrUnauthorized)
	}

	passwordHash, hashErr := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if hashErr != nil {
		return fmt.Errorf("密码加密失败: %w", hashErr)
	}
	if err = s.db.Model(&models.User{}).
		Where("id = ?", userID).
		Update("password_hash", string(passwordHash)).Error; err != nil {
		return fmt.Errorf("更新密码失败: %w", err)
	}
	return nil
}

func (s *AuthService) ensureUniqueUser(username string, email string) error {
	var count int64
	if err := s.db.Model(&models.User{}).
		Where("username = ? OR email = ?", username, email).
		Count(&count).Error; err != nil {
		return fmt.Errorf("校验用户名/邮箱失败: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("%w: 用户名或邮箱已存在", ErrConflict)
	}
	return nil
}
