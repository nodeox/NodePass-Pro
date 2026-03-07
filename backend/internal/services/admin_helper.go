package services

import (
	"errors"
	"fmt"
	"strings"

	"nodepass-pro/backend/internal/models"

	"gorm.io/gorm"
)

func ensureAdminUser(db *gorm.DB, userID uint) (*models.User, error) {
	if userID == 0 {
		return nil, fmt.Errorf("%w: 用户 ID 无效", ErrInvalidParams)
	}

	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: 用户不存在", ErrNotFound)
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	if !strings.EqualFold(strings.TrimSpace(user.Role), "admin") {
		return nil, fmt.Errorf("%w: 仅管理员可执行此操作", ErrForbidden)
	}

	return &user, nil
}
