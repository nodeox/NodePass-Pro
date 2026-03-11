package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"nodepass-pro/backend/internal/models"
	"nodepass-pro/backend/internal/utils"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// AuthService 认证服务。
//
// Deprecated: 此服务已被重构为 DDD 架构。
// 新代码请使用以下模块：
//   - Commands: internal/application/auth/commands (LoginHandler, RegisterHandler, ChangePasswordHandler, RefreshTokenHandler)
//   - Queries: internal/application/auth/queries (GetUserHandler)
//   - Repository: internal/infrastructure/persistence/postgres/auth/auth_repository.go
//   - Cache: internal/infrastructure/cache/auth_cache.go
// 通过依赖注入容器获取: container.LoginHandler, container.RegisterHandler 等
// 此服务将在所有旧代码迁移完成后删除。
type AuthService struct {
	db                  *gorm.DB
	vipService          *VIPService
	refreshTokenService *RefreshTokenService
	emailService        *EmailService
	verificationService *VerificationCodeService
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
	AccessToken  string       `json:"access_token"`  // 访问令牌（短期，30分钟）
	RefreshToken string       `json:"refresh_token"` // 刷新令牌（长期，7天）
	ExpiresIn    int          `json:"expires_in"`    // 访问令牌过期时间（秒）
	TokenType    string       `json:"token_type"`    // 令牌类型（Bearer）
	User         *models.User `json:"user"`
}

type EmailChangeCodeResult struct {
	ExpiresAt time.Time `json:"expires_at"`
	DebugCode string    `json:"debug_code,omitempty"`
	Sent      bool      `json:"sent"`
}

const emailChangeCodeTTL = 10 * time.Minute

// NewAuthService 创建认证服务。
func NewAuthService(db *gorm.DB) *AuthService {
	return &AuthService{
		db:                  db,
		vipService:          NewVIPService(db),
		refreshTokenService: NewRefreshTokenService(db),
		emailService:        NewEmailService(db),
		verificationService: NewVerificationCodeService(db),
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
		zap.L().Warn("更新用户登录时间失败",
			zap.Uint("user_id", user.ID),
			zap.Error(err))
		// 不更新内存中的时间，保持与数据库一致
	} else {
		// 只有更新成功才设置内存中的时间
		user.LastLoginAt = &now
	}

	return &LoginResult{
		AccessToken:  token,
		RefreshToken: "",   // 旧接口不返回 refresh token
		ExpiresIn:    3600, // 1 小时
		TokenType:    "Bearer",
		User:         &user,
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

// SendEmailChangeCode 发送邮箱修改验证码。
func (s *AuthService) SendEmailChangeCode(userID uint, password string, newEmail string) (*EmailChangeCodeResult, error) {
	if userID == 0 {
		return nil, fmt.Errorf("%w: 用户 ID 无效", ErrInvalidParams)
	}

	password = strings.TrimSpace(password)
	newEmail = strings.ToLower(strings.TrimSpace(newEmail))
	if password == "" || newEmail == "" {
		return nil, fmt.Errorf("%w: password/new_email 不能为空", ErrInvalidParams)
	}
	if err := utils.ValidateEmail(newEmail); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidParams, err)
	}

	user, err := s.GetMe(userID)
	if err != nil {
		return nil, err
	}
	if compareErr := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); compareErr != nil {
		return nil, fmt.Errorf("%w: 当前密码错误", ErrUnauthorized)
	}
	if strings.EqualFold(strings.TrimSpace(user.Email), newEmail) {
		return nil, fmt.Errorf("%w: 新邮箱不能与当前邮箱相同", ErrInvalidParams)
	}
	if err := s.ensureEmailAvailable(newEmail, userID); err != nil {
		return nil, err
	}

	code, err := s.verificationService.GenerateNumericCode(6)
	if err != nil {
		return nil, fmt.Errorf("生成邮箱验证码失败: %w", err)
	}

	expiresAt := time.Now().Add(emailChangeCodeTTL)
	codeData := &VerificationCodeData{
		UserID:    userID,
		Code:      code,
		Purpose:   "email_change",
		Target:    newEmail,
		ExpiresAt: expiresAt,
	}

	// 使用新的验证码服务存储（优先 Redis，降级数据库）
	ctx := context.Background()
	if err := s.verificationService.StoreCode(ctx, codeData); err != nil {
		return nil, fmt.Errorf("保存邮箱验证码失败: %w", err)
	}

	sent := false
	if s.emailService != nil {
		sendErr := s.emailService.SendEmailChangeCode(newEmail, code, expiresAt)
		if sendErr != nil {
			if !errors.Is(sendErr, ErrSMTPNotEnabled) {
				return nil, fmt.Errorf("发送邮箱验证码失败: %w", sendErr)
			}
			zap.L().Info("SMTP 未启用，返回调试验证码", zap.Uint("user_id", userID))
		} else {
			sent = true
		}
	}

	debugCode := ""
	if !sent {
		debugCode = code
	}
	return &EmailChangeCodeResult{
		ExpiresAt: expiresAt,
		DebugCode: debugCode,
		Sent:      sent,
	}, nil
}

// ChangeEmail 使用验证码修改邮箱。
func (s *AuthService) ChangeEmail(userID uint, newEmail string, code string) error {
	if userID == 0 {
		return fmt.Errorf("%w: 用户 ID 无效", ErrInvalidParams)
	}

	newEmail = strings.ToLower(strings.TrimSpace(newEmail))
	code = strings.TrimSpace(code)
	if newEmail == "" || code == "" {
		return fmt.Errorf("%w: new_email/code 不能为空", ErrInvalidParams)
	}
	if err := utils.ValidateEmail(newEmail); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidParams, err)
	}

	user, err := s.GetMe(userID)
	if err != nil {
		return err
	}
	if strings.EqualFold(strings.TrimSpace(user.Email), newEmail) {
		return fmt.Errorf("%w: 新邮箱不能与当前邮箱相同", ErrInvalidParams)
	}

	// 使用新的验证码服务验证
	ctx := context.Background()
	if err := s.verificationService.VerifyAndDelete(ctx, userID, "email_change", code, newEmail); err != nil {
		return err
	}

	if err := s.ensureEmailAvailable(newEmail, userID); err != nil {
		return err
	}

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.User{}).
			Where("id = ?", userID).
			Update("email", newEmail).Error; err != nil {
			if isDuplicateKeyError(err) {
				return fmt.Errorf("%w: 邮箱已被使用", ErrConflict)
			}
			return fmt.Errorf("更新邮箱失败: %w", err)
		}
		return nil
	}); err != nil {
		return err
	}

	if err := s.refreshTokenService.RevokeUserRefreshTokens(userID); err != nil {
		return fmt.Errorf("撤销历史登录会话失败: %w", err)
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

func (s *AuthService) ensureEmailAvailable(email string, excludeUserID uint) error {
	var count int64
	query := s.db.Model(&models.User{}).Where("email = ?", email)
	if excludeUserID > 0 {
		query = query.Where("id <> ?", excludeUserID)
	}
	if err := query.Count(&count).Error; err != nil {
		return fmt.Errorf("校验邮箱失败: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("%w: 邮箱已被使用", ErrConflict)
	}
	return nil
}
