package services

import (
	"errors"
	"fmt"
	"time"

	"nodepass-license-unified/internal/models"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Claims JWT 负载。
type Claims struct {
	AdminID  uint   `json:"admin_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// AuthService 认证服务。
type AuthService struct {
	db          *gorm.DB
	jwtSecret   []byte
	expireHours int
}

// NewAuthService 创建认证服务。
func NewAuthService(db *gorm.DB, secret string, expireHours int) *AuthService {
	return &AuthService{
		db:          db,
		jwtSecret:   []byte(secret),
		expireHours: expireHours,
	}
}

// Login 管理员登录。
func (s *AuthService) Login(username, password string) (string, *models.AdminUser, error) {
	var user models.AdminUser
	if err := s.db.Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil, errors.New("账号或密码错误")
		}
		return "", nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", nil, errors.New("账号或密码错误")
	}

	now := time.Now()
	expiresAt := now.Add(time.Duration(s.expireHours) * time.Hour)

	claims := &Claims{
		AdminID:  user.ID,
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", user.ID),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", nil, err
	}

	return tokenString, &user, nil
}

// ParseToken 解析 JWT。
func (s *AuthService) ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid token signing method")
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

// GetAdminByID 查询管理员。
func (s *AuthService) GetAdminByID(id uint) (*models.AdminUser, error) {
	var user models.AdminUser
	if err := s.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}
