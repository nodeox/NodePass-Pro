package services

import (
	"fmt"
	"time"

	"nodepass-license-center/internal/database"
	"nodepass-license-center/internal/models"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// AuthService 管理员认证服务。
type AuthService struct {
	db          *gorm.DB
	jwtSecret   []byte
	expireHours int
}

// AdminClaims 管理员 JWT 声明。
type AdminClaims struct {
	AdminID  uint   `json:"admin_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// LoginRequest 登录请求。
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResult 登录结果。
type LoginResult struct {
	Token string           `json:"token"`
	Admin models.AdminUser `json:"admin"`
}

// NewAuthService 创建认证服务。
func NewAuthService(db *gorm.DB, secret string, expireHours int) *AuthService {
	if expireHours <= 0 {
		expireHours = 24
	}
	return &AuthService{db: db, jwtSecret: []byte(secret), expireHours: expireHours}
}

// Login 管理员登录。
func (s *AuthService) Login(req *LoginRequest) (*LoginResult, error) {
	if req == nil {
		return nil, fmt.Errorf("请求不能为空")
	}

	var admin models.AdminUser
	if err := s.db.Where("username = ?", req.Username).First(&admin).Error; err != nil {
		return nil, fmt.Errorf("用户名或密码错误")
	}
	if admin.Status != "active" {
		return nil, fmt.Errorf("账号不可用")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(req.Password)); err != nil {
		return nil, fmt.Errorf("用户名或密码错误")
	}

	now := time.Now().UTC()
	expireAt := now.Add(time.Duration(s.expireHours) * time.Hour)
	claims := AdminClaims{
		AdminID:  admin.ID,
		Username: admin.Username,
		Role:     admin.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expireAt),
			IssuedAt:  jwt.NewNumericDate(now),
			Subject:   fmt.Sprintf("admin:%d", admin.ID),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("生成 token 失败: %w", err)
	}

	database.TouchLastLogin(s.db, admin.ID)
	admin.PasswordHash = ""
	return &LoginResult{Token: tokenString, Admin: admin}, nil
}

// ParseToken 解析管理员 token。
func (s *AuthService) ParseToken(tokenString string) (*AdminClaims, error) {
	claims := &AdminClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("签名算法不正确")
		}
		return s.jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("无效或过期 token")
	}
	return claims, nil
}

// GetAdmin 获取管理员信息。
func (s *AuthService) GetAdmin(id uint) (*models.AdminUser, error) {
	var admin models.AdminUser
	if err := s.db.First(&admin, id).Error; err != nil {
		return nil, err
	}
	admin.PasswordHash = ""
	return &admin, nil
}
