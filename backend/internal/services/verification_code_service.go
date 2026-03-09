package services

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"nodepass-pro/backend/internal/cache"
	"nodepass-pro/backend/internal/models"

	"gorm.io/gorm"
)

// VerificationCodeService 验证码服务（优先使用 Redis，降级到数据库）
type VerificationCodeService struct {
	db *gorm.DB
}

// VerificationCodeData 验证码数据结构
type VerificationCodeData struct {
	UserID    uint      `json:"user_id"`
	Code      string    `json:"code"`
	Purpose   string    `json:"purpose"` // 用途：email_change, password_reset 等
	Target    string    `json:"target"`  // 目标：新邮箱地址等
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// NewVerificationCodeService 创建验证码服务
func NewVerificationCodeService(db *gorm.DB) *VerificationCodeService {
	return &VerificationCodeService{db: db}
}

// StoreCode 存储验证码（优先使用 Redis，降级到数据库）
func (s *VerificationCodeService) StoreCode(ctx context.Context, data *VerificationCodeData) error {
	if data == nil {
		return fmt.Errorf("验证码数据不能为空")
	}

	data.CreatedAt = time.Now()
	ttl := time.Until(data.ExpiresAt)
	if ttl <= 0 {
		return fmt.Errorf("验证码已过期")
	}

	key := s.buildKey(data.UserID, data.Purpose)

	// 优先使用 Redis
	if cache.Enabled() {
		return cache.SetJSON(ctx, key, data, ttl)
	}

	// 降级到数据库
	return s.storeCodeInDB(key, data)
}

// GetCode 获取验证码（优先从 Redis，降级到数据库）
func (s *VerificationCodeService) GetCode(ctx context.Context, userID uint, purpose string) (*VerificationCodeData, error) {
	if userID == 0 || strings.TrimSpace(purpose) == "" {
		return nil, fmt.Errorf("用户 ID 和用途不能为空")
	}

	key := s.buildKey(userID, purpose)

	// 优先从 Redis 读取
	if cache.Enabled() {
		var data VerificationCodeData
		exists, err := cache.GetJSON(ctx, key, &data)
		if err != nil {
			return nil, fmt.Errorf("读取验证码失败: %w", err)
		}
		if !exists {
			return nil, fmt.Errorf("%w: 验证码不存在", ErrNotFound)
		}

		// 检查是否过期
		if time.Now().After(data.ExpiresAt) {
			_ = cache.Delete(ctx, key)
			return nil, fmt.Errorf("%w: 验证码已过期", ErrUnauthorized)
		}

		return &data, nil
	}

	// 降级到数据库
	return s.getCodeFromDB(key)
}

// VerifyAndDelete 验证并删除验证码
func (s *VerificationCodeService) VerifyAndDelete(ctx context.Context, userID uint, purpose string, code string, target string) error {
	data, err := s.GetCode(ctx, userID, purpose)
	if err != nil {
		return err
	}

	// 验证码匹配
	if data.Code != strings.TrimSpace(code) {
		return fmt.Errorf("%w: 验证码错误", ErrUnauthorized)
	}

	// 验证目标匹配（如果提供）
	if target != "" && !strings.EqualFold(strings.TrimSpace(data.Target), strings.TrimSpace(target)) {
		return fmt.Errorf("%w: 验证码与目标不匹配", ErrUnauthorized)
	}

	// 删除验证码
	return s.DeleteCode(ctx, userID, purpose)
}

// DeleteCode 删除验证码
func (s *VerificationCodeService) DeleteCode(ctx context.Context, userID uint, purpose string) error {
	key := s.buildKey(userID, purpose)

	// 从 Redis 删除
	if cache.Enabled() {
		return cache.Delete(ctx, key)
	}

	// 从数据库删除
	return s.db.Where("key = ?", key).Delete(&models.SystemConfig{}).Error
}

// GenerateNumericCode 生成数字验证码
func (s *VerificationCodeService) GenerateNumericCode(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("验证码长度必须大于 0")
	}
	const digits = "0123456789"
	buf := make([]byte, length)
	random := make([]byte, length)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	for i := range random {
		buf[i] = digits[int(random[i])%len(digits)]
	}
	return string(buf), nil
}

// buildKey 构建验证码存储键
func (s *VerificationCodeService) buildKey(userID uint, purpose string) string {
	return fmt.Sprintf("verification_code:%s:%d", purpose, userID)
}

// storeCodeInDB 存储验证码到数据库（降级方案）
func (s *VerificationCodeService) storeCodeInDB(key string, data *VerificationCodeData) error {
	payloadBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("序列化验证码失败: %w", err)
	}

	value := string(payloadBytes)
	description := fmt.Sprintf("验证码: %s", data.Purpose)

	var systemConfig models.SystemConfig
	queryErr := s.db.Where("key = ?", key).First(&systemConfig).Error
	if queryErr == nil {
		// 更新现有记录
		if err := s.db.Model(&models.SystemConfig{}).Where("id = ?", systemConfig.ID).Updates(map[string]interface{}{
			"value":       value,
			"description": description,
		}).Error; err != nil {
			return fmt.Errorf("更新验证码失败: %w", err)
		}
	} else if errors.Is(queryErr, gorm.ErrRecordNotFound) {
		// 创建新记录
		systemConfig = models.SystemConfig{
			Key:         key,
			Value:       &value,
			Description: &description,
		}
		if err := s.db.Create(&systemConfig).Error; err != nil {
			return fmt.Errorf("创建验证码失败: %w", err)
		}
	} else {
		return fmt.Errorf("查询验证码失败: %w", queryErr)
	}

	return nil
}

// getCodeFromDB 从数据库获取验证码（降级方案）
func (s *VerificationCodeService) getCodeFromDB(key string) (*VerificationCodeData, error) {
	var systemConfig models.SystemConfig
	if err := s.db.Where("key = ?", key).First(&systemConfig).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: 验证码不存在", ErrNotFound)
		}
		return nil, fmt.Errorf("查询验证码失败: %w", err)
	}

	if systemConfig.Value == nil || strings.TrimSpace(*systemConfig.Value) == "" {
		return nil, fmt.Errorf("%w: 验证码无效", ErrInvalidParams)
	}

	var data VerificationCodeData
	if err := json.Unmarshal([]byte(*systemConfig.Value), &data); err != nil {
		return nil, fmt.Errorf("%w: 验证码数据格式错误", ErrInvalidParams)
	}

	// 检查是否过期
	if time.Now().After(data.ExpiresAt) {
		_ = s.db.Where("id = ?", systemConfig.ID).Delete(&models.SystemConfig{}).Error
		return nil, fmt.Errorf("%w: 验证码已过期", ErrUnauthorized)
	}

	return &data, nil
}
